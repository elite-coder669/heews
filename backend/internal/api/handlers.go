package api

import (
	"encoding/json"
	"io"
	"net/http"

	"heatwave/backend/internal/config"
	"heatwave/backend/internal/orchestration"
)

type Handlers struct {
	Cfg  *config.Config
	Orch *orchestration.Orchestrator
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{
		"ok":    false,
		"error": map[string]string{"code": code, "message": msg},
	})
}

func (h *Handlers) Health(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": map[string]any{"status": "ok", "mode": h.Cfg.AppMode},
	})
}

func (h *Handlers) Version(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok": true,
		"data": map[string]any{
			"service":            "0.1.0",
			"physics_version":    "wbgt-liljegren-1.0",
			"risk_model_version": "risk-v1.0.0",
			"agent_version":      "agent-v1.0.0",
		},
	})
}

func (h *Handlers) ListWards(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": map[string]any{"wards": h.Orch.ListWards()},
	})
}

func (h *Handlers) GetWard(c *ctx, id string) {
	w, ok := h.Orch.GetWard(id)
	if !ok {
		writeError(c.w, http.StatusNotFound, "WARD_NOT_FOUND", "ward not found")
		return
	}
	writeJSON(c.w, http.StatusOK, map[string]any{"ok": true, "data": w})
}

func (h *Handlers) GetWardWeather(c *ctx, id string) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": h.Orch.GetWardWeather(id),
	})
}

func (h *Handlers) GetWardThermal(c *ctx, id string) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": h.Orch.GetWardThermal(id),
	})
}

func (h *Handlers) GetWardRisk(c *ctx, id string) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": h.Orch.GetWardRisk(id),
	})
}

func (h *Handlers) GetForecast(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": h.Orch.GetForecast(),
	})
}

func (h *Handlers) ListAlerts(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": map[string]any{"alerts": h.Orch.ListAlerts()},
	})
}

func (h *Handlers) AgentActionPlan(c *ctx) {
	var req struct {
		WardIDs []string `json:"ward_ids"`
		Context string   `json:"context"`
	}
	body, _ := io.ReadAll(c.r.Body)
	_ = json.Unmarshal(body, &req)
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": h.Orch.AgentActionPlan(req.WardIDs, req.Context),
	})
}

func (h *Handlers) RunPipeline(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": h.Orch.RunOnce(),
	})
}