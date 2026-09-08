package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"heatwave/backend/internal/config"
)

// Orchestrator coordinates weather → physics → risk → agent and persists state.
// In MVP mode, all state is in-memory; replace with PostgreSQL repository.
type Orchestrator struct {
	Cfg *config.Config

	mu       sync.RWMutex
	wards    []Ward
	risks    map[string]Risk
	thermal  map[string]WardThermal
	weather  map[string]WardWeather
	forecast CityForecast
	alerts   []Alert
	// manualAlerts holds municipal-approved alerts separately from pipeline
	// alerts so applyDoc's o.alerts overwrite never wipes them.
	manualAlerts []Alert
	state        State
	lastDoc  *PipelineDoc
	repoRoot string
	degraded bool
	memory   *HistoricalMemory
}

type State struct {
	LastRun       time.Time
	PipelineRunID string
	Status        string
}

const pipelineRelPath = "ml/run_pipeline.py"

func findRepoRoot() string {
	if root := os.Getenv("HEEWS_ROOT"); root != "" {
		return root
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, pipelineRelPath)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

func New(cfg *config.Config) (*Orchestrator, error) {
	o := &Orchestrator{
		Cfg:      cfg,
		risks:    map[string]Risk{},
		thermal:  map[string]WardThermal{},
		weather:  map[string]WardWeather{},
		forecast: CityForecast{City: "Hyderabad", HorizonHours: 120},
		repoRoot: findRepoRoot(),
	}
	o.memory = o.loadHistorical(o.repoRoot, cfg.AppMode, cfg.DemoMemory)
	if err := o.LoadFixtures(); err != nil {
		return nil, err
	}
	o.mu.Lock()
	o.applyDoc(o.syntheticDocLocked())
	o.state.Status = "ok"
	o.mu.Unlock()
	go o.RunOnce()
	return o, nil
}

func (o *Orchestrator) RunPipelineLoop(interval string) {
	d, _ := time.ParseDuration(interval)
	if d == 0 {
		d = 15 * time.Minute
	}
	t := time.NewTicker(d)
	for range t.C {
		o.RunOnce()
	}
}

func (o *Orchestrator) RunOnce() StageReport {
	o.mu.Lock()
	defer o.mu.Unlock()
	start := time.Now()
	o.state.LastRun = time.Now()
	o.state.PipelineRunID = newUUID()
	runID := o.state.PipelineRunID

	var stages []Stage
	if o.Cfg.AppMode == "LIVE" {
		doc, docStages, err := o.runPythonPipeline()
		stages = docStages
		if err != nil {
			o.degraded = true
			o.state.Status = "degraded"
			if o.lastDoc == nil {
				o.applyDoc(o.syntheticDocLocked())
			}
		} else {
			o.degraded = false
			o.state.Status = "ok"
			o.applyDoc(doc)
			o.persistLocalJournal()
		}
	} else {
		o.degraded = false
		o.state.Status = "ok"
		o.applyDoc(o.syntheticDocLocked())
		stages = []Stage{
			{Stage: "weather", Status: "ok", LatencyMs: 320},
			{Stage: "physics", Status: "ok", LatencyMs: 180},
			{Stage: "risk", Status: "ok", LatencyMs: 60},
			{Stage: "agent", Status: "ok", LatencyMs: 1450},
		}
	}

	return StageReport{
		PipelineRunID:  runID,
		Stages:         stages,
		LatencyTotalMs: int(time.Since(start).Milliseconds()),
		Degraded:       o.degraded,
	}
}

func (o *Orchestrator) runPythonPipeline() (*PipelineDoc, []Stage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tStart := time.Now()
	cmd := exec.CommandContext(ctx, "python3", pipelineRelPath,
		"--mode", "LIVE",
		"--lat", fmt.Sprintf("%.4f", o.Cfg.HyderabadLat),
		"--lon", fmt.Sprintf("%.4f", o.Cfg.HyderabadLon),
		"--base", o.Cfg.OpenMeteoBase,
	)
	cmd.Dir = o.repoRoot
	out, err := cmd.CombinedOutput()
	weatherMs := int(time.Since(tStart).Milliseconds())
	if err != nil {
		return nil, nil, fmt.Errorf("pipeline failed (%v): %s", err, firstLine(string(out)))
	}

	var doc PipelineDoc
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, nil, fmt.Errorf("pipeline output unparseable: %v", err)
	}

	stages := []Stage{
		{Stage: "weather", Status: "ok", LatencyMs: weatherMs},
		{Stage: "physics", Status: "ok", LatencyMs: 180},
		{Stage: "risk", Status: "ok", LatencyMs: 60},
		{Stage: "agent", Status: "ok", LatencyMs: 1450},
	}
	return &doc, stages, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func (o *Orchestrator) applyDoc(doc *PipelineDoc) {
	if len(doc.Wards) > 0 {
		o.wards = doc.Wards
	}
	if doc.Risks != nil {
		o.risks = doc.Risks
	}
	if doc.Thermal != nil {
		o.thermal = doc.Thermal
	}
	if doc.Weather != nil {
		o.weather = doc.Weather
	}
	if len(doc.Forecast.ByWard) > 0 {
		o.forecast = doc.Forecast
	}
	if doc.Alerts != nil {
		o.alerts = doc.Alerts
	}
	o.lastDoc = doc
}

func (o *Orchestrator) syntheticDocLocked() *PipelineDoc {
	thermal := map[string]WardThermal{}
	weather := map[string]WardWeather{}
	forecast := CityForecast{City: "Hyderabad", HorizonHours: 120}
	alerts := []Alert{}
	risks := map[string]Risk{}

	for _, w := range o.wards {
		r := syntheticRisk(w)
		risks[w.WardID] = r
		thermal[w.WardID] = WardThermal{
			WardID: w.WardID, PhysicsVersion: "wbgt-liljegren-1.0",
			Current: ThermalCurrent{WBGT: 32.5, UTCI: 42.1, WBGTQuality: "ok"},
		}
		weather[w.WardID] = WardWeather{
			WardID: w.WardID, Source: "synthetic", FetchedAt: time.Now().Format(time.RFC3339),
			Current: WeatherCurrent{
				TemperatureC: 38.0, HumidityPct: 45.0, WindMS: 2.5,
				ShortwaveWm2: 850.0, PressurePa: 100300.0, CloudCoverPct: 10.0,
			},
		}
		series := make([]ForecastCell, 5)
		for i := 0; i < 5; i++ {
			dayScore := r.Risk.Score + float64(i%3)*3
			series[i] = ForecastCell{
				Day: dayLabel(i), WBGT: 32.5 + float64(i), UTCI: 42.1 + float64(i),
				MRI: dayScore, Band: bandFor(dayScore),
			}
		}
		forecast.ByWard = append(forecast.ByWard, ForecastSeries{WardID: w.WardID, Series: series})
		if r.Risk.Score >= 60 {
			alerts = append(alerts, Alert{
				ID: newUUID(), WardID: w.WardID, CreatedAt: time.Now().Format(time.RFC3339),
				Severity: r.Risk.Band, Headline: w.WardName + " — heat risk " + strings.ToLower(r.Risk.Band),
				Body: fmt.Sprintf("Peak WBGT %.1f °C, UTCI %.1f °C.", r.WBGT, r.UTCI),
				RecommendedActions: []AlertAction{
					{Action: "Pre-position cooling materials", Priority: "MEDIUM", Target: w.WardName},
				},
				Status: "active",
			})
		}
	}
	return &PipelineDoc{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Mode:        "MOCK",
		Wards:       o.wards,
		Risks:       risks,
		Thermal:     thermal,
		Weather:     weather,
		Forecast:    forecast,
		Alerts:      alerts,
	}
}

func dayLabel(i int) string {
	switch i {
	case 0:
		return "Today"
	case 1:
		return "Tomorrow"
	default:
		return time.Now().AddDate(0, 0, i).Format("Mon")
	}
}

func bandFor(score float64) string {
	switch {
	case score >= 80:
		return "EXTREME"
	case score >= 60:
		return "VERY_HIGH"
	case score >= 40:
		return "HIGH"
	default:
		return "MODERATE"
	}
}

func (o *Orchestrator) ListWards() []Ward {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.wards
}

func (o *Orchestrator) GetWard(id string) (Ward, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, w := range o.wards {
		if w.WardID == id {
			return w, true
		}
	}
	return Ward{}, false
}

func (o *Orchestrator) GetWardRisk(id string) (Risk, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	r, ok := o.risks[id]
	return r, ok
}

func (o *Orchestrator) GetForecast() CityForecast {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.forecast
}

func (o *Orchestrator) ListAlerts() []Alert {
	o.mu.RLock()
	defer o.mu.RUnlock()
	out := make([]Alert, 0, len(o.manualAlerts)+len(o.alerts))
	out = append(out, o.manualAlerts...)
	out = append(out, o.alerts...)
	return out
}

func (o *Orchestrator) GetWardWeather(id string) (WardWeather, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	w, ok := o.weather[id]
	return w, ok
}

func (o *Orchestrator) GetWardThermal(id string) (WardThermal, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	t, ok := o.thermal[id]
	return t, ok
}

func (o *Orchestrator) Degraded() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.degraded
}

func (o *Orchestrator) GetState() State {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.state
}

const (
	actionCoolingCouncil = "Activate cooling council centres and night shelters"
	actionOutdoorWork    = "Issue outdoor-work advisory (shift timings during peak heat)"
	actionPrePosition    = "Pre-position ORS, water, and cooling materials at public points"
	actionMunicipalAlert = "Issue municipal heat-health advisory"
	actionOutreach       = "Direct vulnerable-population outreach in priority wards"
)

func (o *Orchestrator) AgentActionPlan(wardIDs []string, ctx string, horizonHours ...int) ActionPlan {
	o.mu.RLock()
	defer o.mu.RUnlock()
	horizon := 0
	if len(horizonHours) > 0 {
		horizon = horizonHours[0]
	}
	return o.buildActionPlan(wardIDs, ctx, horizon)
}

// PriorityWards is the single deterministic risk ranking of all wards, used
// by the Top Priority Wards panel. horizon is always current (horizon 0).
func (o *Orchestrator) PriorityWards() []PriorityWard {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.buildActionPlan(nil, "", 0).PriorityWards
}

// CreateManualAlert records a human-approved municipal alert. The plan it is
// built from is passed through so the public advisory stays downstream of an
// approved plan. Returns an error if the ward does not exist.
func (o *Orchestrator) CreateManualAlert(in ManualAlertInput) (Alert, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, ok := o.wardLocked(in.WardID); !ok {
		return Alert{}, ErrWardNotFound
	}
	a := Alert{
		ID:                 newUUID(),
		WardID:             in.WardID,
		CreatedAt:          time.Now().Format(time.RFC3339),
		Severity:           in.Severity,
		Headline:           in.Headline,
		Body:               in.Body,
		RecommendedActions: in.RecommendedActions,
		Status:             "ACTIVE",
		Source:             "municipal",
		AgentRunID:         in.AgentRunID,
		AgentVersion:       in.AgentVersion,
		FallbackUsed:       in.FallbackUsed,
	}
	o.manualAlerts = append([]Alert{a}, o.manualAlerts...)
	if len(o.manualAlerts) > 100 {
		o.manualAlerts = o.manualAlerts[:100]
	}
	return a, nil
}

// wardLocked returns a ward by ID; the caller must hold o.mu (any lock type).
func (o *Orchestrator) wardLocked(id string) (Ward, bool) {
	for _, w := range o.wards {
		if w.WardID == id {
			return w, true
		}
	}
	return Ward{}, false
}

func (o *Orchestrator) MemoryStatus() MemoryStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()
	st := MemoryStatus{LocalFile: localJournalRel, DemoFile: demoFixtureRel}
	if o.memory == nil {
		return st
	}
	st.LocalEvents = len(o.memory.Local)
	st.DemoEvents = len(o.memory.Demo)
	st.Enabled = st.DemoEvents > 0
	st.Synthetic = st.Enabled
	st.LocalFile = o.memory.journalPath
	st.DemoFile = o.memory.demoPath
	return st
}
