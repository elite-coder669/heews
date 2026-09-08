package orchestration

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"heatwave/backend/internal/agent"
)

const agentVersion = "agent-v2.0.0"

const (
	simMinScore = 0.45
	simStrong   = 0.65
	maxMatches  = 3
)

// buildActionPlan assembles the enriched ActionPlan for the top-risk ward.
// It is deterministic and never invents values: every number comes from
// o.risks / o.thermal / o.forecast / o.wards, and precedent from HistoricalMemory.
// When exactly one ward is requested with horizonHours>0, the reasoning is
// scoped to that ward's forecast cell at the requested horizon; otherwise it
// uses current-risk metrics. priority_wards always reflects current risk.
// Lock must be held (RLock) by the caller.
func (o *Orchestrator) buildActionPlan(wardIDs []string, ctx string, horizonHours int) ActionPlan {
	names := map[string]string{}
	zones := map[string]string{}
	demos := map[string]Demographics{}
	for _, w := range o.wards {
		names[w.WardID] = w.WardName
		zones[w.WardID] = w.Zone
		demos[w.WardID] = w.Demographics
	}

	candidates := []Risk{}
	include := map[string]bool{}
	if len(wardIDs) > 0 {
		for _, id := range wardIDs {
			include[id] = true
		}
	}
	for id, r := range o.risks {
		if len(include) == 0 || include[id] {
			candidates = append(candidates, r)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Risk.Score > candidates[j].Risk.Score })

	if len(candidates) == 0 {
		return o.emptyPlan(candidates)
	}

	scope := o.planScope(candidates, horizonHours)
	top := scope.metric // the risk/metric carrier for severity/numbers
	severity := top.Band
	topName := names[scope.wardID]
	ward := demos[scope.wardID]
	vuln := vulnerabilityIndex(ward)
	forecast, hasForecast := o.forecastSeries(scope.wardID)
	trajectory := forecastTrajectory(forecast)

	matchCtx := EventContext{
		WardID: scope.wardID, WardName: topName,
		MRI: top.MRI, Band: top.Band,
		WBGT: top.WBGT, UTCI: top.UTCI, Vulnerability: vuln,
		Month: int(time.Now().Month()),
	}
	var matches []MatchResult
	if o.memory != nil {
		matches = o.memory.Match(matchCtx, maxMatches, simMinScore)
	}

	histCtx := HistoricalContext{Found: false, Type: "none"}
	matching := []MatchingEvent{}
	if len(matches) > 0 {
		best := matches[0]
		histCtx = HistoricalContext{
			Found: true, Type: matchType(best.Event), EventID: best.Event.EventID,
			SimilarityReason: WhyMatches(best.Dimensions), Source: best.Event.Source, Synthetic: best.Event.Synthetic,
		}
		for _, m := range matches {
			matching = append(matching, MatchingEvent{
				EventID: m.Event.EventID, Date: m.Event.Date, WardName: m.Event.WardName,
				Score: m.Event.MRI, Band: m.Event.Band, Similarity: m.Score, Dimensions: m.Dimensions,
				ActionsTaken: m.Event.ActionsTaken, Outcome: m.Event.Outcome,
			})
		}
	}

	evidence := o.evidenceFor(scope.wardID, top, vuln, hasForecast, histCtx.Found)
	simScore := 0.0
	if len(matches) > 0 {
		simScore = matches[0].Score
	}
	confidence := confidenceRule(evidence, simScore)
	limitations := o.limitationsFor(histCtx, hasForecast, vuln, evidence)

	actions := o.actionsFor(top, topName, severity, histCtx, evidence, vuln)
	recheck := 360
	if severity == "EXTREME" || severity == "VERY_HIGH" {
		recheck = 60
	}

	priorityWards := make([]PriorityWard, 0, len(candidates))
	for _, r := range candidates {
		priorityWards = append(priorityWards, PriorityWard{WardID: r.WardID, WardName: names[r.WardID], Score: r.Risk.Score, Band: r.Risk.Band})
	}

	why := o.whyThisMatters(top, topName, vuln, trajectory, histCtx.Type, scope)
	reasoning := fmt.Sprintf("Priority: %s at %s risk (MRI %.0f). %s",
		topName, strings.ToLower(severity), top.MRI, strings.Join(scope.keyFactors, "; "))

	advisory := "Stay hydrated, avoid midday sun, and check on elderly neighbours."
	if severity == "EXTREME" {
		advisory = "EXTREME heat risk: limit outdoor exposure, open cooling centres, prioritise vulnerable residents."
	}

	plan := ActionPlan{
		Scope: scope.kind, ScopeWardID: scope.wardID,
		ScopeHorizonHours: scope.horizonHours, ScopeLabel: scope.label,
		Severity: severity, PriorityWards: priorityWards,
		ReasoningSummary: reasoning, KeyFactors: scope.keyFactors,
		RecommendedActions: actions, PublicAdvisory: advisory,
		RecheckIntervalMinutes: recheck, AgentRunID: newUUID(),
		FallbackUsed: o.degraded, AgentVersion: agentVersion,
		WhyThisMatters: why, HistoricalContext: histCtx,
		MatchingEvents: matching, Evidence: evidence,
		Confidence: confidence, Limitations: limitations,
	}

	o.applyLLMNarratives(&plan, top, ctx, scope)
	return plan
}

// planScope describes whether reasoning runs on current-risk or forecast-cell
// metrics. The metric carrier (scopedMetric) unifies Risk and ForecastCell for
// the downstream functions.
type planScope struct {
	kind         string // "current" | "forecast"
	wardID       string
	horizonHours int
	label        string        // human label e.g. "Forecast day: Tomorrow"
	keyFactors   []string      // evidence reasons for this scope
	metric       scopedMetric  // severity/MRI/WBGT/UTCI carrier
	band         string
}

type scopedMetric struct {
	MRI     float64
	Band    string
	WBGT    float64
	UTCI    float64
	Factors []string
}

// planScope resolves which metrics drive the plan. Exactly one requested ward
// plus horizon>0 selects that ward's forecast cell; everything else uses the
// top current risk. priority_wards stays current regardless (handled by the
// caller via candidates).
func (o *Orchestrator) planScope(candidates []Risk, horizonHours int) planScope {
	top := candidates[0]
	if len(candidates) == 1 && horizonHours > 0 {
		if series, ok := o.forecastSeries(top.WardID); ok && len(series) > 0 {
			idx := horizonHours / 24
			if idx < 0 {
				idx = 0
			}
			if idx >= len(series) {
				idx = len(series) - 1
			}
			cell := series[idx]
			return planScope{
				kind: "forecast", wardID: top.WardID, horizonHours: cellIdxHours(idx),
				label: forecastLabel(cell, idx),
				keyFactors: []string{
					fmt.Sprintf("MRI at %s = %.1f (%s)", forecastDayName(cell), cell.MRI, cell.Band),
					fmt.Sprintf("WBGT at %s = %.1f °C", forecastDayName(cell), cell.WBGT),
					fmt.Sprintf("UTCI at %s = %.1f °C", forecastDayName(cell), cell.UTCI),
					fmt.Sprintf("Forecast trajectory: %s", forecastTrajectory(series)),
				},
				metric: scopedMetric{MRI: cell.MRI, Band: cell.Band, WBGT: cell.WBGT, UTCI: cell.UTCI, Factors: []string{}},
				band:   cell.Band,
			}
		}
	}
	return planScope{
		kind: "current", wardID: top.WardID, horizonHours: 0, label: "Current conditions",
		keyFactors: top.Risk.TopFactors,
		metric:     scopedMetric{MRI: top.Risk.Score, Band: top.Risk.Band, WBGT: top.WBGT, UTCI: top.UTCI, Factors: top.Risk.TopFactors},
		band:       top.Risk.Band,
	}
}

func cellIdxHours(idx int) int { return idx * 24 }

func forecastDayName(c ForecastCell) string {
	if c.Day != "" {
		return c.Day
	}
	return "this day"
}

func forecastLabel(c ForecastCell, idx int) string {
	if c.Day != "" {
		return "Forecast day: " + c.Day
	}
	return fmt.Sprintf("Forecast horizon %dh", idx*24)
}

// emptyPlan is returned when no ward risk data exists yet.
func (o *Orchestrator) emptyPlan(candidates []Risk) ActionPlan {
	return ActionPlan{
		Severity: "MODERATE", PriorityWards: []PriorityWard{},
		ReasoningSummary: "No ward risk data yet.",
		RecommendedActions: []AlertAction{
			{Action: actionMunicipalAlert, Priority: "LOW", Target: "Hyderabad",
				Reason:   "No live risk is currently available; a baseline advisory keeps heat-health awareness on record.",
				Evidence: []string{"general_guidance"}},
		},
		PublicAdvisory:         "Heat-health risk awareness recommended.",
		RecheckIntervalMinutes: 360, AgentRunID: newUUID(), FallbackUsed: true,
		AgentVersion: agentVersion, HistoricalContext: HistoricalContext{Found: false, Type: "none"},
		Evidence:   Evidence{CurrentRisk: false, GeneralGuidance: true},
		Confidence: "LOW",
		Limitations: []string{
			"No ward risk data is currently available; numbers are not populated",
			"External (cited-source) historical examples are not available in this prototype",
		},
	}
}

// matchType distinguishes demo vs local precedent.
func matchType(e HistoricalEvent) string {
	if e.Synthetic {
		return "demo"
	}
	return "local"
}

// evidenceFor reports which evidence bases are satisfied for the top ward.
func (o *Orchestrator) evidenceFor(wardID string, top scopedMetric, vuln float64, hasForecast, hasPrecedent bool) Evidence {
	ev := Evidence{
		CurrentRisk:         top.MRI > 0,
		Vulnerability:       vuln > 0,
		Forecast:            hasForecast,
		HistoricalPrecedent: hasPrecedent,
		ExternalReference:   false,
	}
	ev.GeneralGuidance = !hasPrecedent
	return ev
}

// confidenceRule ties confidence to evidence availability, not prose tone.
func confidenceRule(ev Evidence, simScore float64) string {
	switch {
	case simScore >= simStrong:
		return "HIGH"
	case ev.CurrentRisk && ev.Vulnerability:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// limitationsFor lists honest caveats; it never invents missing data.
func (o *Orchestrator) limitationsFor(histCtx HistoricalContext, hasForecast bool, vuln float64, ev Evidence) []string {
	lim := []string{}
	if !ev.ExternalReference {
		lim = append(lim, "External (cited-source) historical examples are not available in this prototype; no external event was claimed.")
	}
	if hasForecast && (o.memory == nil || (len(o.memory.Local)+len(o.memory.Demo) == 0)) {
		lim = append(lim, "Historical memory is empty; recommendations rest on current risk and general heat-health guidance only.")
	}
	if !hasForecast {
		lim = append(lim, "Forecast trajectory is unavailable for this ward; actions assume conditions do not worsen sharply.")
	}
	if vuln == 0 {
		lim = append(lim, "Ward vulnerability index is unavailable (demographics not populated).")
	}
	lim = append(lim, "Hospital capacity and cooling-centre availability are not yet modelled; resource-dependent actions assume municipal capacity decisions.")
	if ev.HistoricalPrecedent && histCtx.Synthetic {
		lim = append(lim, "The matched precedent is a synthetic demo record (demo-memory-fixture), not a real observation.")
	}
	if o.degraded {
		lim = append(lim, "Pipeline is degraded; risk values may be synthetic placeholder data.")
	}
	return lim
}

// actionsFor builds band-based municipal actions, each tied to evidence.
func (o *Orchestrator) actionsFor(top scopedMetric, topName, severity string, histCtx HistoricalContext, ev Evidence, vuln float64) []AlertAction {
	base := evidenceNames(ev, histCtx)
	actions := []AlertAction{}
	switch severity {
	case "EXTREME":
		actions = append(actions,
			AlertAction{Action: actionCoolingCouncil, Priority: "HIGH", Target: topName,
				Reason:   fmt.Sprintf("Sustained %s risk (MRI %.0f, WBGT %.1f °C, UTCI %.1f °C) makes physiological cooling the top municipal priority.", strings.ToLower(severity), top.MRI, top.WBGT, top.UTCI),
				Evidence: base},
			AlertAction{Action: actionOutdoorWork, Priority: "HIGH", Target: topName,
				Reason:   "Peak WBGT during working hours can push outdoor workers past safe thresholds; shifting timings cuts exposure.",
				Evidence: base},
			AlertAction{Action: actionPrePosition, Priority: "MEDIUM", Target: topName,
				Reason:   "Pre-positioning ORS and water at public points covers the highest-exposure hours cheaply and fast.",
				Evidence: base},
		)
	case "VERY_HIGH":
		actions = append(actions,
			AlertAction{Action: actionPrePosition, Priority: "MEDIUM", Target: topName,
				Reason:   "Very high risk with heat expected to continue; pre-positioning limits the number of people needing to travel for relief.",
				Evidence: base},
		)
	case "HIGH":
		actions = append(actions,
			AlertAction{Action: actionMunicipalAlert, Priority: "LOW", Target: topName,
				Reason:   "High risk is below the threshold that justifies disruptive measures; a municipal advisory keeps the ward informed.",
				Evidence: base},
		)
	default:
		actions = append(actions, AlertAction{Action: "Publish heat-health awareness advisory", Priority: "LOW", Target: "Hyderabad",
			Reason:   "No extreme or high risk at present; routine awareness messaging applies city-wide.",
			Evidence: base})
	}
	if severity == "EXTREME" || severity == "VERY_HIGH" {
		actions = append(actions, AlertAction{Action: actionOutreach, Priority: "MEDIUM", Target: topName,
			Reason:   fmt.Sprintf("Vulnerability index %.2f means elderly and outdoor workers carry most of the exposure; targeted outreach reaches them directly.", vuln),
			Evidence: base})
	}
	if histCtx.Found {
		actions = append(actions, AlertAction{Action: "Replicate outcome-backed precedent actions from memory", Priority: "MEDIUM", Target: topName,
			Reason:   "A similar previous situation is recorded; repeating its effective actions while monitoring duration is evidence-based.",
			Evidence: base})
	}
	return actions
}

func evidenceNames(ev Evidence, histCtx HistoricalContext) []string {
	out := []string{}
	if ev.CurrentRisk {
		out = append(out, "current_risk")
	}
	if ev.Vulnerability {
		out = append(out, "vulnerability")
	}
	if ev.Forecast {
		out = append(out, "forecast")
	}
	if ev.HistoricalPrecedent {
		out = append(out, "historical_precedent")
	}
	if len(out) == 0 {
		out = append(out, "general_guidance")
	}
	return out
}

// whyThisMatters explains the risk using only pipeline outputs + precedent.
func (o *Orchestrator) whyThisMatters(top scopedMetric, topName string, vuln float64, trajectory, histCtxNote string, scope planScope) string {
	sb := strings.Builder{}
	if scope.kind == "forecast" {
		sb.WriteString(fmt.Sprintf("%s is forecast to be at %s heat risk at the %s horizon (MRI %.0f, WBGT %.1f °C, UTCI %.1f °C).",
			topName, strings.ToLower(top.Band), strings.ToLower(scope.label), top.MRI, top.WBGT, top.UTCI))
	} else {
		sb.WriteString(fmt.Sprintf("%s is at %s heat risk (MRI %.0f, WBGT %.1f °C, UTCI %.1f °C).",
			topName, strings.ToLower(top.Band), top.MRI, top.WBGT, top.UTCI))
	}
	sb.WriteString(" Physiological stress is driven by WBGT and UTCI (heat stress indices). ")
	sb.WriteString(fmt.Sprintf("The health-risk estimate (MRI band) at this horizon is %s. ", strings.ToLower(top.Band)))
	if vuln > 0 {
		sb.WriteString(fmt.Sprintf("Ward vulnerability index is %.2f (elderly, outdoor workers, informal housing) — this raises who is at risk. ", vuln))
	} else {
		sb.WriteString("Ward vulnerability data is unavailable. ")
	}
	sb.WriteString(fmt.Sprintf("Forecast trajectory: %s. ", trajectory))
	if histCtxNote == "demo" {
		sb.WriteString("Historical precedent: a synthetic demo record matched (not a real observation).")
	} else if histCtxNote == "none" {
		sb.WriteString("Historical precedent: no sufficiently similar event found; general heat-health guidance applies.")
	}
	return sb.String()
}

func (o *Orchestrator) forecastSeries(wardID string) ([]ForecastCell, bool) {
	for _, s := range o.forecast.ByWard {
		if s.WardID == wardID {
			if len(s.Series) == 0 {
				return nil, false
			}
			return s.Series, true
		}
	}
	return nil, false
}

func forecastTrajectory(series []ForecastCell) string {
	if len(series) == 0 {
		return "unknown"
	}
	current := series[0].MRI
	rest := 0.0
	for _, c := range series[1:] {
		if c.MRI > rest {
			rest = c.MRI
		}
	}
	switch {
	case rest > current+3:
		return "worsening over the horizon"
	case rest < current-3:
		return "improving over the horizon"
	default:
		return "roughly stable over the horizon"
	}
}

// actionSlug maps deterministic action text to a stable candidate-pool ID the
// LLM can reference. Unknown text suggests a future action const: the fallback
// is the pool position, still valid because validation runs against the pool.
var actionSlug = map[string]string{
	actionCoolingCouncil:                     "ACTIVATE_COOLING_CENTERS",
	actionOutdoorWork:                        "ISSUE_OUTDOOR_WORK_ADVISORY",
	actionPrePosition:                        "PRE_POSITION_SUPPLIES",
	actionMunicipalAlert:                     "ISSUE_MUNICIPAL_ADVISORY",
	actionOutreach:                           "DIRECT_VULNERABLE_OUTREACH",
	"Publish heat-health awareness advisory": "CITY_WIDE_AWARENESS",
	"Replicate outcome-backed precedent actions from memory": "REPLICATE_PRECEDENT_ACTIONS",
}

// applyLLMNarratives is the optional contextual decision layer. The model only
// selects and ranks from the deterministic candidate pool; it never generates
// actions, numbers, evidence, or events. Any failure or invalid proposal falls
// back to the deterministic plan and flags fallback_used.
func (o *Orchestrator) applyLLMNarratives(plan *ActionPlan, top scopedMetric, userCtx string, scope planScope) {
	cfg := o.Cfg
	if cfg == nil || cfg.AgentLLMProvider != "openrouter" || len(plan.RecommendedActions) == 0 {
		return
	}
	client := &agent.Client{APIKey: cfg.OpenRouterAPIKey, Model: cfg.LLMModel}
	if plan.FallbackUsed || !client.Enabled() {
		return
	}
	series, _ := o.forecastSeries(scope.wardID)
	// The LLM context is built from the SAME scoped metrics that drive the
	// deterministic reasoning, so prose never mixes current and future values.
	severityLabel := plan.Severity
	wardLabel := scope.wardID
	if len(plan.PriorityWards) > 0 {
		wardLabel = plan.PriorityWards[0].WardName
	}
	input := agent.NarrativeInput{
		Severity:           severityLabel,
		WardName:           wardLabel,
		MRI:                top.MRI,
		WBGT:               top.WBGT,
		UTCI:               top.UTCI,
		VulnerabilityKnown: plan.Evidence.Vulnerability,
		Confidence:         plan.Confidence,
		PrecedentFound:     plan.HistoricalContext.Found,
		Candidates:         plan.narrativeCandidates(),
		Precedents:         plan.narrativePrecedents(),
		ForecastTrajectory: forecastTrajectory(series),
		MunicipalContext:   userCtx,
		ScopeLabel:         scope.label,
	}
	connCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	narr, err := client.Narrate(connCtx, input)
	if err != nil {
		plan.FallbackUsed = true
		plan.Limitations = append(plan.Limitations, "LLM narrative unavailable; deterministic reasoning used.")
		return
	}
	if err := applyLLMProposal(plan, narr); err != nil {
		plan.FallbackUsed = true
		plan.Limitations = append(plan.Limitations, "LLM proposal rejected by validation ("+err.Error()+"); deterministic reasoning used.")
		return
	}
}

// narrativeCandidates exposes the deterministic actions as a referenceable pool.
func (p *ActionPlan) narrativeCandidates() []agent.CandidateAction {
	out := make([]agent.CandidateAction, 0, len(p.RecommendedActions))
	for i, a := range p.RecommendedActions {
		out = append(out, agent.CandidateAction{ID: actionSlugFor(a.Action, i), Text: a.Action, Priority: a.Priority, Evidence: a.Evidence})
	}
	return out
}

// narrativePrecedents exposes matched historical events the model may cite.
func (p *ActionPlan) narrativePrecedents() []agent.Precedent {
	out := make([]agent.Precedent, 0, len(p.MatchingEvents))
	for _, m := range p.MatchingEvents {
		out = append(out, agent.Precedent{EventID: m.EventID, Score: m.Similarity, Band: m.Band, ActionsTaken: m.ActionsTaken, Outcome: m.Outcome})
	}
	return out
}

func actionSlugFor(text string, i int) string {
	if id, ok := actionSlug[text]; ok {
		return id
	}
	return fmt.Sprintf("ACTION_%d", i+1)
}

// applyLLMProposal validates a model proposal against the deterministic plan
// and, only if every reference checks out, reorders/re-prioritises the actions
// and accepts the prose. Validation gates: no invented actions, allowed
// priorities, no duplicates, precedent ids must reference real matched events
// (and be empty when no precedent exists), non-empty actions and prose.
func applyLLMProposal(plan *ActionPlan, n agent.Narrative) error {
	if strings.TrimSpace(n.Why) == "" || strings.TrimSpace(n.Reasoning) == "" {
		return fmt.Errorf("empty prose")
	}
	if len(n.Actions) == 0 {
		return fmt.Errorf("empty action selection")
	}
	pool := plan.narrativeCandidates()
	if len(n.Actions) > len(pool) {
		return fmt.Errorf("more actions selected than pool allows")
	}
	idxByID := map[string]int{}
	for i, c := range pool {
		idxByID[c.ID] = i
	}
	used := map[string]bool{}
	ordered := make([]AlertAction, 0, len(n.Actions))
	for _, s := range n.Actions {
		i, ok := idxByID[s.ID]
		if !ok {
			return fmt.Errorf("unrecognised action id %q", s.ID)
		}
		if used[s.ID] {
			return fmt.Errorf("duplicate action id %q", s.ID)
		}
		switch s.Priority {
		case "LOW", "MEDIUM", "HIGH":
		default:
			return fmt.Errorf("invalid priority %q for %s", s.Priority, s.ID)
		}
		used[s.ID] = true
		ordered = append(ordered, plan.RecommendedActions[i])
		ordered[len(ordered)-1].Priority = s.Priority
	}
	// Precedents: must be empty when none are available, else reference real
	// matched events only.
	if !plan.HistoricalContext.Found && len(n.PrecedentEventIDs) > 0 {
		return fmt.Errorf("precedent ids cited with no historical context")
	}
	realEvent := map[string]bool{}
	for _, m := range plan.MatchingEvents {
		realEvent[m.EventID] = true
	}
	for _, id := range n.PrecedentEventIDs {
		if !realEvent[id] {
			return fmt.Errorf("precedent id %q does not match a known event", id)
		}
	}
	plan.RecommendedActions = ordered
	plan.ReasoningSummary = n.Reasoning
	plan.WhyThisMatters = n.Why
	return nil
}
