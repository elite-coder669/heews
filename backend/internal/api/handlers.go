package api

import (
	"encoding/json"
	"errors"
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
	mode := h.Cfg.AppMode
	status := "ok"
	if h.Orch.Degraded() {
		mode = "DEGRADED"
		status = "degraded"
	}
	st := h.Orch.GetState()
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": map[string]any{"status": status, "mode": mode, "last_run": st.LastRun.Format("2006-01-02T15:04:05Z07:00"), "pipeline_run_id": st.PipelineRunID},
	})
}

func (h *Handlers) Version(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok": true,
		"data": map[string]any{
			"service":            "0.1.0",
			"physics_version":    "wbgt-liljegren-1.0",
			"risk_model_version": "risk-v1.0.0",
			"agent_version":      "agent-v2.0.0",
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
	v, ok := h.Orch.GetWardWeather(id)
	if !ok {
		writeError(c.w, http.StatusNotFound, "WARD_NOT_FOUND", "ward weather not available")
		return
	}
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": v,
	})
}

func (h *Handlers) GetWardThermal(c *ctx, id string) {
	v, ok := h.Orch.GetWardThermal(id)
	if !ok {
		writeError(c.w, http.StatusNotFound, "WARD_NOT_FOUND", "ward thermal not available")
		return
	}
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": v,
	})
}

func (h *Handlers) GetWardRisk(c *ctx, id string) {
	v, ok := h.Orch.GetWardRisk(id)
	if !ok {
		writeError(c.w, http.StatusNotFound, "WARD_NOT_FOUND", "ward risk not available")
		return
	}
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": v,
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

func (h *Handlers) CreateAlert(c *ctx) {
	var req struct {
		WardID             string                      `json:"ward_id"`
		Severity           string                      `json:"severity"`
		Headline           string                      `json:"headline"`
		Body               string                      `json:"body"`
		RecommendedActions []orchestration.AlertAction `json:"recommended_actions"`
		AgentRunID         string                      `json:"agent_run_id"`
		AgentVersion       string                      `json:"agent_version"`
		FallbackUsed       bool                        `json:"fallback_used"`
	}
	body, _ := io.ReadAll(c.r.Body)
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c.w, http.StatusBadRequest, "BAD_REQUEST", "invalid alert payload: "+err.Error())
		return
	}
	if req.WardID == "" || req.Headline == "" {
		writeError(c.w, http.StatusBadRequest, "BAD_REQUEST", "ward_id and headline are required")
		return
	}
	switch req.Severity {
	case "EXTREME", "VERY_HIGH", "HIGH", "MODERATE", "LOW", "":
	default:
		writeError(c.w, http.StatusBadRequest, "BAD_REQUEST", "invalid severity")
		return
	}
	in := orchestration.ManualAlertInput{
		WardID:             req.WardID,
		Severity:           req.Severity,
		Headline:           req.Headline,
		Body:               req.Body,
		RecommendedActions: req.RecommendedActions,
		AgentRunID:         req.AgentRunID,
		AgentVersion:       req.AgentVersion,
		FallbackUsed:       req.FallbackUsed,
	}
	alert, err := h.Orch.CreateManualAlert(in)
	if err != nil {
		if errors.Is(err, orchestration.ErrWardNotFound) {
			writeError(c.w, http.StatusNotFound, "WARD_NOT_FOUND", "ward not found")
			return
		}
		writeError(c.w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	writeJSON(c.w, http.StatusCreated, map[string]any{"ok": true, "data": alert})
}

func (h *Handlers) AgentPriority(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": map[string]any{"priority_wards": h.Orch.PriorityWards()},
	})
}

func (h *Handlers) AgentActionPlan(c *ctx) {
	var req struct {
		WardIDs      []string `json:"ward_ids"`
		Context      string   `json:"context"`
		HorizonHours *int     `json:"horizon_hours"`
	}
	body, _ := io.ReadAll(c.r.Body)
	_ = json.Unmarshal(body, &req)
	horizon := 0
	if req.HorizonHours != nil {
		horizon = *req.HorizonHours
	}
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": h.Orch.AgentActionPlan(req.WardIDs, req.Context, horizon),
	})
}

func (h *Handlers) AgentMemory(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": h.Orch.MemoryStatus(),
	})
}

func (h *Handlers) RunPipeline(c *ctx) {
	writeJSON(c.w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": h.Orch.RunOnce(),
	})
}
