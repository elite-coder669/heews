package orchestration

import (
	"sync"
	"time"

	"heatwave/backend/internal/config"
)

// Orchestrator coordinates weather → physics → risk → agent and persists state.
// In MVP mode, all state is in-memory; replace with PostgreSQL repository.
type Orchestrator struct {
	Cfg *config.Config

	mu    sync.RWMutex
	wards []Ward
	risks map[string]Risk
	state State
}

type State struct {
	LastRun       time.Time
	PipelineRunID string
	Status        string
}

func New(cfg *config.Config) (*Orchestrator, error) {
	o := &Orchestrator{Cfg: cfg, risks: map[string]Risk{}}
	if err := o.LoadFixtures(); err != nil {
		return nil, err
	}
	o.RunOnce()
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
	o.state.LastRun = time.Now()
	o.state.PipelineRunID = newUUID()
	o.state.Status = "ok"
	return StageReport{
		PipelineRunID: o.state.PipelineRunID,
		Stages: []Stage{
			{Stage: "weather", Status: "ok", LatencyMs: 320},
			{Stage: "physics", Status: "ok", LatencyMs: 180},
			{Stage: "risk", Status: "ok", LatencyMs: 60},
			{Stage: "agent", Status: "ok", LatencyMs: 1450},
		},
	}
}

func (o *Orchestrator) ListWards() []Ward     { o.mu.RLock(); defer o.mu.RUnlock(); return o.wards }
func (o *Orchestrator) GetWard(id string) (Ward, bool) {
	o.mu.RLock(); defer o.mu.RUnlock()
	for _, w := range o.wards {
		if w.WardID == id {
			return w, true
		}
	}
	return Ward{}, false
}
func (o *Orchestrator) GetWardRisk(id string) Risk {
	o.mu.RLock(); defer o.mu.RUnlock()
	return o.risks[id]
}
func (o *Orchestrator) GetForecast() any      { return map[string]any{"city": "Hyderabad", "by_ward": []any{}} }
func (o *Orchestrator) ListAlerts() []any      { return []any{} }
func (o *Orchestrator) GetWardWeather(id string) any { return map[string]any{"ward_id": id, "current": map[string]any{}} }
func (o *Orchestrator) GetWardThermal(id string) any { return map[string]any{"ward_id": id, "current": map[string]any{}} }

func (o *Orchestrator) AgentActionPlan(wardIDs []string, ctx string) any {
	return map[string]any{
		"severity":              "MODERATE",
		"priority_wards":        wardIDs,
		"reasoning_summary":     "MVP stub; replace with real agent.",
		"key_factors":           []string{},
		"recommended_actions":   []any{},
		"public_advisory":       "Heat-health risk awareness recommended.",
		"recheck_interval_minutes": 60,
		"fallback_used":         true,
	}
}