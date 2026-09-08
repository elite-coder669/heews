package orchestration

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MemoryStatus summarises what the decision agent can retrieve, for /api/agent/memory.
type MemoryStatus struct {
	LocalEvents int    `json:"local_events"`
	DemoEvents  int    `json:"demo_events"`
	Enabled     bool   `json:"enabled"`
	Synthetic   bool   `json:"synthetic"`
	LocalFile   string `json:"local_file"`
	DemoFile    string `json:"demo_file"`
}

// HistoricalEvent is one past heat event the decision agent uses as precedent.
// Local journal entries carry Source "local-journal" (real pipeline observations
// persisted across restarts); demo entries carry Source "demo-memory-fixture" and
// Synthetic=true. Every field except the identity/predictor core is optional so
// incomplete records are handled gracefully by the weighted similarity.
type HistoricalEvent struct {
	EventID       string   `json:"event_id"`
	Date          string   `json:"date"`
	Start         string   `json:"start,omitempty"`
	End           string   `json:"end,omitempty"`
	WardID        string   `json:"ward_id"`
	WardName      string   `json:"ward_name,omitempty"`
	TemperatureC  float64  `json:"temperature_c,omitempty"`
	WBGT          float64  `json:"wbgt_c,omitempty"`
	UTCI          float64  `json:"utci_c,omitempty"`
	MRI           float64  `json:"mri"`
	Band          string   `json:"risk_band"`
	Vulnerability float64  `json:"vulnerability_index,omitempty"`
	DurationDays  int      `json:"duration_days,omitempty"`
	ActionsTaken  []string `json:"actions_taken,omitempty"`
	AlertLevel    string   `json:"alert_level,omitempty"`
	Outcome       string   `json:"outcome,omitempty"`
	Season        string   `json:"season,omitempty"`
	Source        string   `json:"source"`
	Synthetic     bool     `json:"synthetic"`
}

// HistoricalMemory combines the local journal (real history accumulated through
// pipeline runs) and the demo fixture (synthetic, gated to MOCK/opt-in).
type HistoricalMemory struct {
	journalPath string
	demoPath    string
	Local       []HistoricalEvent
	Demo        []HistoricalEvent
}

const (
	localJournalRel = "data/historical/alert_history.json"
	demoFixtureRel  = "data/historical/demo_heat_events.json"
	localJournalCap = 200
)

func (o *Orchestrator) loadHistorical(repoRoot, appMode string, demoMemory bool) *HistoricalMemory {
	m := &HistoricalMemory{
		journalPath: filepath.Join(repoRoot, localJournalRel),
		demoPath:    filepath.Join(repoRoot, demoFixtureRel),
	}
	if data, err := os.ReadFile(m.journalPath); err == nil {
		var events []HistoricalEvent
		if json.Unmarshal(data, &events) == nil {
			m.Local = events
		}
	}
	if appMode == "MOCK" || demoMemory {
		if data, err := os.ReadFile(m.demoPath); err == nil {
			var doc struct {
				Events []HistoricalEvent `json:"events"`
			}
			if json.Unmarshal(data, &doc) == nil {
				m.Demo = doc.Events
			}
		}
	}
	return m
}

// persistLocalJournal records each distinct high-severity alert episode as a real
// local precedent. Dedupes on ward+date+band+mri bucket so repeated pipeline runs
// do not spam identical entries; caps history at the newest localJournalCap events.
func (o *Orchestrator) persistLocalJournal() {
	if o.memory == nil || len(o.alerts) == 0 {
		return
	}
	now := time.Now().Format(time.DateOnly)
	existing := map[string]bool{}
	for _, e := range o.memory.Local {
		existing[string(e.WardID)+"|"+e.Date+"|"+e.Band] = true
	}
	for _, a := range o.alerts {
		key := a.WardID + "|" + now + "|" + a.Severity
		if existing[key] {
			continue
		}
		existing[key] = true
		r, riskOK := o.risks[a.WardID]
		t, thermalOK := o.thermal[a.WardID]
		w, wardOK := o.wardLocked(a.WardID)
		ev := HistoricalEvent{
			EventID:      "local-" + newUUID()[:8],
			Date:         now,
			Start:        now,
			End:          now,
			WardID:       a.WardID,
			Band:         a.Severity,
			ActionsTaken: make([]string, 0, len(a.RecommendedActions)),
			AlertLevel:   strings.ToUpper(a.Severity),
			Season:       seasonOfMonth(int(time.Now().Month())),
			Source:       "local-journal",
		}
		if wardOK {
			ev.WardName = w.WardName
			ev.Vulnerability = vulnerabilityIndex(w.Demographics)
		}
		if riskOK {
			ev.MRI = r.Risk.Score
			ev.WBGT = r.WBGT
			ev.UTCI = r.UTCI
		}
		if thermalOK {
			ev.WBGT = t.Current.WBGT
			ev.UTCI = t.Current.UTCI
		}
		for _, ac := range a.RecommendedActions {
			ev.ActionsTaken = append(ev.ActionsTaken, ac.Action)
		}
		o.memory.Local = append(o.memory.Local, ev)
	}
	if len(o.memory.Local) > localJournalCap {
		o.memory.Local = o.memory.Local[len(o.memory.Local)-localJournalCap:]
	}
	o.writeLocalJournal()
}

func (o *Orchestrator) writeLocalJournal() {
	data, err := json.MarshalIndent(o.memory.Local, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(o.memory.journalPath), 0o755)
	_ = os.WriteFile(o.memory.journalPath, data, 0o644)
}

// EventContext is the current ward situation matched against precedent events.
type EventContext struct {
	WardID        string
	WardName      string
	MRI           float64
	Band          string
	WBGT          float64
	UTCI          float64
	Vulnerability float64
	Month         int
}

// MatchResult is one ranked precedent with its per-dimension similarity.
type MatchResult struct {
	Event      HistoricalEvent
	Score      float64
	Dimensions map[string]float64
}

// Match ranks precedent events (demo then local) against the current context by a
// deterministic weighted similarity and returns the top n above minScore.
func (m *HistoricalMemory) Match(ctx EventContext, n int, minScore float64) []MatchResult {
	candidates := make([]HistoricalEvent, 0, len(m.Demo)+len(m.Local))
	candidates = append(candidates, m.Demo...)
	candidates = append(candidates, m.Local...)
	ranked := make([]MatchResult, 0, len(candidates))
	for _, e := range candidates {
		dims := similarityDims(ctx, e)
		score := weightedSim(dims)
		if score < minScore {
			continue
		}
		ranked = append(ranked, MatchResult{Event: e, Score: score, Dimensions: dims})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })
	if len(ranked) > n {
		ranked = ranked[:n]
	}
	return ranked
}

var simWeights = map[string]float64{
	"mri":    0.30,
	"wbgt":   0.15,
	"utci":   0.10,
	"vuln":   0.10,
	"ward":   0.15,
	"season": 0.10,
}

func similarityDims(ctx EventContext, e HistoricalEvent) map[string]float64 {
	dims := map[string]float64{}
	if e.MRI > 0 {
		dims["mri"] = 1 - math.Min(math.Abs(ctx.MRI-e.MRI), 60)/60
	}
	if e.WBGT > 0 {
		dims["wbgt"] = 1 - math.Min(math.Abs(ctx.WBGT-e.WBGT), 20)/20
	}
	if e.UTCI > 0 {
		dims["utci"] = 1 - math.Min(math.Abs(ctx.UTCI-e.UTCI), 20)/20
	}
	if e.Vulnerability > 0 && ctx.Vulnerability > 0 {
		dims["vuln"] = 1 - math.Min(math.Abs(ctx.Vulnerability-e.Vulnerability), 0.5)/0.5
	}
	if e.WardID == ctx.WardID {
		dims["ward"] = 1.0
	}
	if e.Season != "" && seasonOfMonth(ctx.Month) != "" {
		if e.Season == seasonOfMonth(ctx.Month) {
			dims["season"] = 1.0
		} else if (e.Season == "pre-monsoon" || e.Season == "monsoon") && (seasonOfMonth(ctx.Month) == "pre-monsoon" || seasonOfMonth(ctx.Month) == "monsoon") {
			dims["season"] = 0.6
		}
	}
	return dims
}

// weightedSim renormalises against the weights of the dimensions actually present,
// so events with missing fields still get a fair, conservative score.
func weightedSim(dims map[string]float64) float64 {
	total, wsum := 0.0, 0.0
	for k, d := range dims {
		w := simWeights[k]
		if w == 0 {
			continue
		}
		total += w * d
		wsum += w
	}
	if wsum == 0 {
		return 0
	}
	return total / wsum
}

// WhyMatches explains the ranking by the dominant dimensions.
func WhyMatches(dims map[string]float64) string {
	type kv struct {
		k string
		v float64
	}
	entries := make([]kv, 0, len(dims))
	for k, v := range dims {
		entries = append(entries, kv{k, v})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].v > entries[j].v })
	parts := []string{}
	for _, e := range entries {
		if len(parts) >= 2 || e.v < 0.5 {
			break
		}
		parts = append(parts, labelDim(e.k))
	}
	if len(parts) == 0 {
		return "weak dimensional overlap"
	}
	return strings.Join(parts, ", ")
}

func labelDim(k string) string {
	switch k {
	case "mri":
		return "risk score proximity"
	case "wbgt":
		return "WBGT proximity"
	case "utci":
		return "UTCI proximity"
	case "vuln":
		return "matching vulnerability profile"
	case "ward":
		return "same ward"
	case "season":
		return "same season"
	}
	return k
}

func seasonOfMonth(month int) string {
	switch {
	case month >= 3 && month <= 5:
		return "pre-monsoon"
	case month >= 6 && month <= 9:
		return "monsoon"
	case month >= 10 && month <= 11:
		return "post-monsoon"
	default:
		return "winter"
	}
}

func vulnerabilityIndex(d Demographics) float64 {
	return 0.40*d.ElderlyRatio + 0.30*d.OutdoorWorkerShare + 0.20*d.InformalHousingShare + 0.10*math.Min(float64(d.Population)/30000, 1)
}
