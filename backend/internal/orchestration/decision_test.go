package orchestration

import (
	"fmt"
	"testing"

	"heatwave/backend/internal/agent"
	"heatwave/backend/internal/config"
)

func testOrch(risks map[string]Risk, memory *HistoricalMemory) *Orchestrator {
	return &Orchestrator{
		Cfg: &config.Config{AgentLLMProvider: "mock"},
		wards: []Ward{
			{WardID: "ward_001", WardName: "Somajiguda", Zone: "Central",
				Demographics: Demographics{ElderlyRatio: 0.15, OutdoorWorkerShare: 0.20, InformalHousingShare: 0.10, Population: 30000}},
			{WardID: "ward_002", WardName: "Banjara Hills", Zone: "West",
				Demographics: Demographics{}},
		},
		risks:    risks,
		forecast: CityForecast{City: "Hyderabad", HorizonHours: 120},
		degraded: false,
		memory:   memory,
	}
}

func demoFixtureEvents() []HistoricalEvent {
	return []HistoricalEvent{
		{EventID: "demo-2025-24", Date: "2025-06-24", WardID: "ward_006", WardName: "Secunderabad",
			MRI: 52, Band: "HIGH", WBGT: 26.9, UTCI: 23.4, Vulnerability: 0.27,
			Season: "monsoon", Source: "demo-memory-fixture", Synthetic: true,
			ActionsTaken: []string{"Issue municipal heat-health advisory"}, Outcome: "contained"},
		{EventID: "demo-2024-29", Date: "2024-05-29", WardID: "ward_001", WardName: "Somajiguda",
			MRI: 87, Band: "EXTREME", WBGT: 31.2, UTCI: 41.0, Vulnerability: 0.31,
			Season: "pre-monsoon", Source: "demo-memory-fixture", Synthetic: true},
	}
}

func riskMap(score, wbgt, utci float64) map[string]Risk {
	return map[string]Risk{
		"ward_001": {WardID: "ward_001", WBGT: wbgt, UTCI: utci,
			Risk: RiskPayload{Score: score, Band: bandFor(score), TopFactors: []string{"high WBGT", "elderly exposure"}}},
	}
}

func TestDecisionNoHallucinatedSources(t *testing.T) {
	o := testOrch(riskMap(46, 26.76, 23.38), &HistoricalMemory{})
	plan := o.AgentActionPlan(nil, "")
	if plan.Evidence.ExternalReference {
		t.Fatal("external_reference must be false: this prototype has no external retrieval")
	}
	if plan.HistoricalContext.Found {
		t.Fatal("no memory loaded, so no historical precedent should be found")
	}
	if plan.HistoricalContext.Type != "none" {
		t.Fatalf("expected type none, got %s", plan.HistoricalContext.Type)
	}
	if !plan.Evidence.GeneralGuidance {
		t.Fatal("general guidance must fill in when no precedent exists")
	}
	for _, l := range plan.Limitations {
		if l == "" {
			t.Fatal("limitation strings must not be empty")
		}
	}
}

func TestDecisionMatchingPrecedent(t *testing.T) {
	o := testOrch(riskMap(46, 26.76, 23.38), &HistoricalMemory{Demo: demoFixtureEvents()})
	plan := o.AgentActionPlan(nil, "")
	if !plan.HistoricalContext.Found {
		t.Fatal("demo-2025-24 (ward_006 monsoon HIGH 52) should match ward_001 monsoon HIGH 46")
	}
	if !plan.Evidence.HistoricalPrecedent || !plan.Evidence.CurrentRisk || !plan.Evidence.Vulnerability {
		t.Fatalf("all three evidence bases must be satisfied, got %+v", plan.Evidence)
	}
	if plan.HistoricalContext.Type != "demo" || !plan.HistoricalContext.Synthetic {
		t.Fatalf("match must be labeled demo/synthetic, got %s synthetic=%t", plan.HistoricalContext.Type, plan.HistoricalContext.Synthetic)
	}
	if plan.HistoricalContext.SimilarityReason == "" {
		t.Fatal("similarity_reason must name the dominant dimensions")
	}
	if plan.Confidence != "HIGH" {
		t.Fatalf("strong match + live risk + vuln must yield HIGH confidence, got %s", plan.Confidence)
	}
	applied := []string{}
	for _, m := range plan.MatchingEvents {
		if m.EventID == "" || m.Similarity <= 0 || m.Score == 0 {
			t.Fatalf("matching event malformed: %+v", m)
		}
		applied = append(applied, m.EventID)
	}
	hasJourney := false
	for _, a := range plan.RecommendedActions {
		if a.Reason == "" {
			t.Fatal("every action must carry a reason")
		}
		if len(a.Evidence) == 0 {
			t.Fatal("every action must carry evidence tags")
		}
		hasJourney = hasJourney || a.Action == "Replicate outcome-backed precedent actions from memory"
	}
	if len(applied) > 0 && !hasJourney {
		t.Fatal("precedent actions must surface when memory matched")
	}
}

func TestDecisionNoMatchFallsBack(t *testing.T) {
	o := testOrch(riskMap(46, 26.76, 23.38), &HistoricalMemory{Local: []HistoricalEvent{
		// Dissimilar on every dimension: extreme MRI far from current HIGH, winter vs
		// monsoon, different ward, and no thermal/vulnerability values to compensate.
		{EventID: "local-1", Date: "2025-01-05", WardID: "ward_002", Band: "EXTREME",
			MRI: 91, Season: "winter", Source: "local-journal"},
	}})
	plan := o.AgentActionPlan(nil, "")
	if plan.HistoricalContext.Found {
		t.Fatal("a winter EXTREME record in a different ward must not match a monsoon HIGH situation above threshold")
	}
	if plan.Evidence.GeneralGuidance != true {
		t.Fatal("general guidance should substitute when no precedent clears the threshold")
	}
	if plan.Confidence != "MEDIUM" {
		t.Fatalf("live risk + vuln with no precedent must be MEDIUM, got %s", plan.Confidence)
	}
}

func TestDecisionMissingVulnerability(t *testing.T) {
	risks := map[string]Risk{
		"ward_002": {WardID: "ward_002", WBGT: 26.8, UTCI: 23.0,
			Risk: RiskPayload{Score: 44, Band: "HIGH", TopFactors: []string{"high WBGT"}}},
	}
	o := testOrch(risks, nil)
	plan := o.AgentActionPlan(nil, "")
	if plan.Evidence.Vulnerability {
		t.Fatal("empty demographics must mean vulnerability evidence is absent")
	}
	foundLim := false
	for _, l := range plan.Limitations {
		if contains(l, "vulnerability") {
			foundLim = true
		}
	}
	if !foundLim {
		t.Fatal("missing vulnerability must be surfaced in limitations")
	}
}

func TestDecisionMultipleMatchesCapped(t *testing.T) {
	o := testOrch(riskMap(46, 26.76, 23.38), &HistoricalMemory{Demo: demoFixtureEvents()})
	plan := o.AgentActionPlan(nil, "")
	if len(plan.MatchingEvents) > 3 {
		t.Fatalf("matching events must be capped at 3, got %d", len(plan.MatchingEvents))
	}
	for i := 1; i < len(plan.MatchingEvents); i++ {
		if plan.MatchingEvents[i-1].Similarity < plan.MatchingEvents[i].Similarity {
			t.Fatal("matching events must be ranked by similarity descending")
		}
	}
}

func TestDecisionLowRiskAdvisory(t *testing.T) {
	o := testOrch(riskMap(32, 24.5, 22.0), &HistoricalMemory{})
	plan := o.AgentActionPlan(nil, "")
	if plan.Severity != "MODERATE" {
		t.Fatalf("expected MODERATE severity, got %s", plan.Severity)
	}
	if plan.RecheckIntervalMinutes != 360 {
		t.Fatalf("low risk must use 360 min recheck, got %d", plan.RecheckIntervalMinutes)
	}
	if plan.FallbackUsed {
		t.Fatal("live but low risk must NOT be flagged as fallback")
	}
}

func TestDecisionExtremeWorsening(t *testing.T) {
	o := testOrch(map[string]Risk{
		"ward_001": {WardID: "ward_001", WBGT: 31.2, UTCI: 41.0,
			Risk: RiskPayload{Score: 87, Band: "EXTREME", TopFactors: []string{"extreme WBGT"}}},
	}, &HistoricalMemory{})
	o.forecast = CityForecast{City: "Hyderabad", HorizonHours: 120, ByWard: []ForecastSeries{
		{WardID: "ward_001", Series: []ForecastCell{
			{Day: "Today", WBGT: 31.2, UTCI: 41.0, MRI: 87, Band: "EXTREME"},
			{Day: "Tomorrow", WBGT: 33.0, UTCI: 43.0, MRI: 91, Band: "EXTREME"},
		}},
	}}
	plan := o.AgentActionPlan(nil, "")
	if plan.Severity != "EXTREME" || plan.RecheckIntervalMinutes != 60 {
		t.Fatalf("EXTREME must recheck in 60m, got severity=%s recheck=%d", plan.Severity, plan.RecheckIntervalMinutes)
	}
	if !contains(plan.WhyThisMatters, "worsening") {
		t.Fatalf("worsening trajectory must be stated, got %q", plan.WhyThisMatters)
	}
	found := map[string]bool{}
	for _, a := range plan.RecommendedActions {
		found[a.Action] = true
	}
	if !found[actionCoolingCouncil] || !found[actionOutdoorWork] || !found[actionOutreach] {
		t.Fatalf("EXTREME must trigger cooling council, outdoor work, and outreach: %v", found)
	}
}

func TestDecisionDegradedFallback(t *testing.T) {
	o := testOrch(nil, nil)
	plan := o.AgentActionPlan(nil, "")
	if !plan.FallbackUsed {
		t.Fatal("no risk data => fallback plan")
	}
	if plan.Confidence != "LOW" {
		t.Fatalf("fallback plan must be LOW confidence, got %s", plan.Confidence)
	}
	if plan.PublicAdvisory == "" {
		t.Fatal("advisory must remain populated")
	}
}

func TestDecisionNumbersFromPipeline(t *testing.T) {
	o := testOrch(riskMap(46, 26.76, 23.38), nil)
	plan := o.AgentActionPlan(nil, "")
	if !containsFloat(plan.WhyThisMatters, 46.0) || !containsFloat(plan.WhyThisMatters, 26.76) {
		t.Fatalf("why_this_matters must quote the pipeline numbers verbatim, got %q", plan.WhyThisMatters)
	}
	if plan.PriorityWards[0].Score != 46 {
		t.Fatalf("priority ward score must mirror pipeline, got %f", plan.PriorityWards[0].Score)
	}
	for _, a := range plan.RecommendedActions {
		if a.Action == "" || a.Priority == "" || a.Target == "" {
			t.Fatalf("action must be fully populated: %+v", a)
		}
	}
	if plan.AgentVersion != agentVersion {
		t.Fatalf("agent_version must be %s, got %s", agentVersion, plan.AgentVersion)
	}
}

func TestDecisionResourcesNotInvented(t *testing.T) {
	o := testOrch(riskMap(46, 26.76, 23.38), &HistoricalMemory{Demo: demoFixtureEvents()})
	plan := o.AgentActionPlan(nil, "")
	for _, l := range plan.Limitations {
		if contains(l, "capacity") && !contains(l, "not yet modelled") {
			t.Fatalf("resource availability must be stated as unavailable, not assumed: %q", l)
		}
	}
	if contains(plan.WhyThisMatters, "hospital") || contains(plan.WhyThisMatters, "ambulance") {
		t.Fatalf("agent must not invent specific resources in narratives: %q", plan.WhyThisMatters)
	}
	for _, a := range plan.RecommendedActions {
		if contains(a.Action, "hospital") || contains(a.Action, "ambulance") {
			t.Fatalf("agent must not invent resource-specific actions: %q", a.Action)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func containsFloat(s string, f float64) bool {
	return contains(s, fmt.Sprintf("%.0f", f)) || contains(s, fmt.Sprintf("%.1f", f))
}

// --- LLM proposal validation gate ---

func proposalPlan() *ActionPlan {
	return &ActionPlan{
		Severity: "EXTREME",
		RecommendedActions: []AlertAction{
			{Action: actionCoolingCouncil, Priority: "HIGH", Target: "Somajiguda",
				Reason: "physiological cooling", Evidence: []string{"current_risk", "forecast"}},
			{Action: actionOutdoorWork, Priority: "HIGH", Target: "Somajiguda",
				Reason: "peak WBGT", Evidence: []string{"current_risk"}},
			{Action: actionPrePosition, Priority: "MEDIUM", Target: "Somajiguda",
				Reason: "pre-position", Evidence: []string{"current_risk"}},
		},
		HistoricalContext: HistoricalContext{Found: false, Type: "none"},
		MatchingEvents:    []MatchingEvent{},
	}
}

func TestProposalValidReordersWithinPool(t *testing.T) {
	plan := proposalPlan()
	n := agent.Narrative{
		Why:       "Extreme stress due.",
		Reasoning: "Cooling and shade first, supply cover later.",
		Actions: []agent.SelectedAction{
			{ID: "PRE_POSITION_SUPPLIES", Priority: "MEDIUM"},
			{ID: "ACTIVATE_COOLING_CENTERS", Priority: "HIGH"},
			{ID: "ISSUE_OUTDOOR_WORK_ADVISORY", Priority: "HIGH"},
		},
	}
	if err := applyLLMProposal(plan, n); err != nil {
		t.Fatalf("valid proposal rejected: %v", err)
	}
	if plan.WhyThisMatters != n.Why || plan.ReasoningSummary != n.Reasoning {
		t.Fatal("prose must be accepted on a valid proposal")
	}
	if len(plan.RecommendedActions) != 3 {
		t.Fatalf("all three pool actions must stay, got %d", len(plan.RecommendedActions))
	}
	if plan.RecommendedActions[0].Action != actionPrePosition {
		t.Fatalf("actions must follow model order, got %q first", plan.RecommendedActions[0].Action)
	}
	if plan.RecommendedActions[1].Priority != "HIGH" {
		t.Fatalf("priority must update from model, got %q", plan.RecommendedActions[1].Priority)
	}
	if plan.RecommendedActions[1].Evidence[0] != "current_risk" {
		t.Fatal("evidence must be preserved from the deterministic pool entry")
	}
}

// TestProposalRejectsHallucination is the advisor's headline failure case:
// an invented action, a numeric hallucination, and a fictional casualty count
// must all fail validation and leave the plan deterministic.
func TestProposalRejectsHallucination(t *testing.T) {
	plan := proposalPlan()
	n := agent.Narrative{
		Why:       "Assume a mass evacuation is needed.",
		Reasoning: "500 deaths expected.",
		Actions: []agent.SelectedAction{
			{ID: "CREATE_MASS_EVACUATION", Priority: "HIGH"},
			{ID: "ACTIVATE_COOLING_CENTERS", Priority: "HIGH"},
		},
	}
	if err := applyLLMProposal(plan, n); err == nil {
		t.Fatal("invented action id must be rejected")
	}
	if len(plan.RecommendedActions) != 3 {
		t.Fatal("rejected proposal must not mutate the plan")
	}
}

func TestProposalRejectsBadPriority(t *testing.T) {
	if err := applyLLMProposal(proposalPlan(), agent.Narrative{
		Why: "w", Reasoning: "r",
		Actions: []agent.SelectedAction{{ID: "ACTIVATE_COOLING_CENTERS", Priority: "URGENT"}},
	}); err == nil {
		t.Fatal("priority outside LOW/MEDIUM/HIGH must be rejected")
	}
}

func TestProposalRejectsDuplicate(t *testing.T) {
	if err := applyLLMProposal(proposalPlan(), agent.Narrative{
		Why: "w", Reasoning: "r",
		Actions: []agent.SelectedAction{
			{ID: "ACTIVATE_COOLING_CENTERS", Priority: "HIGH"},
			{ID: "ACTIVATE_COOLING_CENTERS", Priority: "LOW"},
		},
	}); err == nil {
		t.Fatal("duplicate action id must be rejected")
	}
}

func TestProposalRejectsPrecedentWithoutHistory(t *testing.T) {
	n := agent.Narrative{
		Why: "w", Reasoning: "r",
		Actions:           []agent.SelectedAction{{ID: "ACTIVATE_COOLING_CENTERS", Priority: "HIGH"}},
		PrecedentEventIDs: []string{"demo-2025-24"},
	}
	if err := applyLLMProposal(proposalPlan(), n); err == nil {
		t.Fatal("citing a precedent when none exists must be rejected")
	}
}

func TestProposalRejectsUnknownPrecedentID(t *testing.T) {
	plan := proposalPlan()
	plan.HistoricalContext = HistoricalContext{Found: true, Type: "demo"}
	plan.MatchingEvents = []MatchingEvent{{EventID: "demo-2025-24"}}
	n := agent.Narrative{
		Why: "w", Reasoning: "r",
		Actions:           []agent.SelectedAction{{ID: "ACTIVATE_COOLING_CENTERS", Priority: "HIGH"}},
		PrecedentEventIDs: []string{"events-of-2002"},
	}
	if err := applyLLMProposal(plan, n); err == nil {
		t.Fatal("precedent id must reference a real matched event")
	}
}

func TestProposalAcceptsRealPrecedentID(t *testing.T) {
	plan := proposalPlan()
	plan.HistoricalContext = HistoricalContext{Found: true, Type: "demo"}
	plan.MatchingEvents = []MatchingEvent{{EventID: "demo-2025-24"}}
	n := agent.Narrative{
		Why: "w", Reasoning: "r",
		Actions:           []agent.SelectedAction{{ID: "ACTIVATE_COOLING_CENTERS", Priority: "HIGH"}},
		PrecedentEventIDs: []string{"demo-2025-24"},
	}
	if err := applyLLMProposal(plan, n); err != nil {
		t.Fatalf("present valid precedent reference: %v", err)
	}
	if plan.MatchingEvents[0].EventID != "demo-2025-24" {
		t.Fatal("preserved matching events must reflect the cited precedent")
	}
}

func TestProposalRejectsEmptyProseOrSelection(t *testing.T) {
	if err := applyLLMProposal(proposalPlan(), agent.Narrative{Why: "", Reasoning: "r",
		Actions: []agent.SelectedAction{{ID: "ACTIVATE_COOLING_CENTERS", Priority: "HIGH"}}}); err == nil {
		t.Fatal("empty why must be rejected")
	}
	if err := applyLLMProposal(proposalPlan(), agent.Narrative{Why: "w", Reasoning: "r"}); err == nil {
		t.Fatal("empty action selection must be rejected")
	}
}

// TestDecisionHorizonScope drives the forecast-cell reasoning path: a single-ward
// request with horizon_hours>0 must scope the whole plan to that forecast cell
// (severity/MRI/key_factors/label), while priority_wards stay current-risk.
func TestDecisionHorizonScope(t *testing.T) {
	o := testOrch(riskMap(87, 31.2, 41.0), &HistoricalMemory{})
	o.forecast = CityForecast{City: "Hyderabad", HorizonHours: 120, ByWard: []ForecastSeries{
		{WardID: "ward_001", Series: []ForecastCell{
			{Day: "Today", MRI: 87, Band: "EXTREME", WBGT: 31.2, UTCI: 41.0},
			{Day: "Tomorrow", MRI: 44, Band: "HIGH", WBGT: 26.7, UTCI: 23.3},
		}},
	}}
	plan := o.AgentActionPlan([]string{"ward_001"}, "", 24)
	if plan.Scope != "forecast" {
		t.Fatalf("single ward + horizon must use forecast scope, got %q", plan.Scope)
	}
	if plan.ScopeWardID != "ward_001" || plan.ScopeHorizonHours != 24 {
		t.Fatalf("scope ward/horizon not plumbed: %s %d", plan.ScopeWardID, plan.ScopeHorizonHours)
	}
	if plan.Severity != "HIGH" {
		t.Fatalf("severity must come from forecast cell (HIGH 44), got %q", plan.Severity)
	}
	// priority wards must stay current-risk (EXTREME 87), never the future cell.
	if len(plan.PriorityWards) == 0 || plan.PriorityWards[0].Score != 87 {
		t.Fatalf("priority wards must rank current risk, got %+v", plan.PriorityWards)
	}
	// key_factors must carry the labeled forecast numbers, not current.
	joined := ""
	for _, k := range plan.KeyFactors {
		joined += k + " "
	}
	if !contains(joined, "Tomorrow") || !containsFloat(joined, 44) {
		t.Fatalf("key factors must quote the forecast cell, got %q", joined)
	}
	if !contains(plan.WhyThisMatters, "forecast") {
		t.Fatalf("why_this_matters must state the forecast nature, got %q", plan.WhyThisMatters)
	}
}

// Single ward but horizon 0 (or absent) must keep the current-risk behavior.
func TestDecisionHorizonZeroIsCurrent(t *testing.T) {
	o := testOrch(riskMap(87, 31.2, 41.0), &HistoricalMemory{})
	plan := o.AgentActionPlan([]string{"ward_001"}, "")
	if plan.Scope != "current" {
		t.Fatalf("no horizon must stay current scope, got %q", plan.Scope)
	}
	if plan.Severity != "EXTREME" {
		t.Fatalf("current scope severity must be current risk, got %q", plan.Severity)
	}
}

// Multi-ward requests (or nil => all) must not enter forecast scope; they stay
// a current-risk ranking across wards.
func TestDecisionMultiWardStaysCurrent(t *testing.T) {
	risks := riskMap(87, 31.2, 41.0)
	risks["ward_002"] = Risk{WardID: "ward_002", WBGT: 25.0, UTCI: 22.0,
		Risk: RiskPayload{Score: 40, Band: "HIGH", TopFactors: []string{"moderate WBGT"}}}
	o := testOrch(risks, &HistoricalMemory{})
	o.forecast = CityForecast{City: "Hyderabad", HorizonHours: 120, ByWard: []ForecastSeries{
		{WardID: "ward_001", Series: []ForecastCell{
			{Day: "Today", MRI: 87, Band: "EXTREME"},
			{Day: "Tomorrow", MRI: 30, Band: "MODERATE"},
		}},
	}}
	plan := o.AgentActionPlan(nil, "", 48)
	if plan.Scope != "current" {
		t.Fatalf("multi/all-ward request must stay current scope, got %q", plan.Scope)
	}
}

// Manual alerts are stored municipally first, capped, and reject unknown wards.
func TestCreateManualAlert(t *testing.T) {
	o := testOrch(riskMap(87, 31.2, 41.0), &HistoricalMemory{})
	a, err := o.CreateManualAlert(ManualAlertInput{
		WardID:             "ward_001",
		Severity:           "EXTREME",
		Headline:           "Extreme heat advisory",
		Body:               "Residents should stay indoors.",
		RecommendedActions: []AlertAction{{Action: "Open cooling centres", Priority: "HIGH", Target: "Somajiguda"}},
		AgentRunID:         "run-123",
		AgentVersion:       agentVersion,
		FallbackUsed:       false,
	})
	if err != nil {
		t.Fatalf("create manual alert: %v", err)
	}
	if a.Source != "municipal" || a.Status != "ACTIVE" || a.ID == "" {
		t.Fatalf("manual alert must be municipal/ACTIVE with id: %+v", a)
	}
	if a.AgentRunID != "run-123" || a.AgentVersion != agentVersion || a.FallbackUsed {
		t.Fatalf("agent metadata must be preserved: %+v", a)
	}
	if len(o.ListAlerts()) == 0 || o.ListAlerts()[0].ID != a.ID {
		t.Fatal("manual alert must be listed first")
	}

	if _, err := o.CreateManualAlert(ManualAlertInput{WardID: "ward_999", Severity: "HIGH", Headline: "x"}); err != ErrWardNotFound {
		t.Fatalf("unknown ward must yield ErrWardNotFound, got %v", err)
	}
}

// The manual slice must be capped at 100 so the UI never grows unbounded.
func TestManualAlertCap(t *testing.T) {
	o := testOrch(riskMap(87, 31.2, 41.0), &HistoricalMemory{})
	for i := 0; i < 120; i++ {
		_, _ = o.CreateManualAlert(ManualAlertInput{WardID: "ward_001", Severity: "HIGH", Headline: "h"})
	}
	if got := len(o.ListAlerts()); got != 100 {
		t.Fatalf("manual alerts must be capped at 100, got %d", got)
	}
}
