package api

import (
	"net/http"
	"strings"

	"heatwave/backend/internal/config"
	"heatwave/backend/internal/orchestration"
)

func NewRouter(cfg *config.Config, orch *orchestration.Orchestrator) *http.ServeMux {
	mux := http.NewServeMux()
	h := &Handlers{Cfg: cfg, Orch: orch}

	mux.HandleFunc("/api/health", wrap(h.Health))
	mux.HandleFunc("/api/version", wrap(h.Version))
	mux.HandleFunc("/api/config", wrap(h.Config))
	mux.HandleFunc("/api/wards", wrap(h.ListWards))
	mux.HandleFunc("/api/wards/", wrap(func(c *ctx) {
		rest := strings.TrimPrefix(c.r.URL.Path, "/api/wards/")
		parts := strings.Split(rest, "/")
		if len(parts) == 1 {
			h.GetWard(c, parts[0])
			return
		}
		switch parts[1] {
		case "weather":
			h.GetWardWeather(c, parts[0])
		case "thermal":
			h.GetWardThermal(c, parts[0])
		case "risk":
			h.GetWardRisk(c, parts[0])
		default:
			http.NotFound(c.w, c.r)
		}
	}))
	mux.HandleFunc("/api/forecast", wrap(h.GetForecast))
	mux.HandleFunc("/api/alerts", wrap(func(c *ctx) {
		switch c.r.Method {
		case http.MethodGet:
			h.ListAlerts(c)
		case http.MethodPost:
			h.CreateAlert(c)
		default:
			http.Error(c.w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/agent/priority", wrap(h.AgentPriority))
	mux.HandleFunc("/api/agent/action-plan", wrap(h.AgentActionPlan))
	mux.HandleFunc("/api/agent/memory", wrap(h.AgentMemory))
	mux.HandleFunc("/api/pipeline/run", wrap(h.RunPipeline))

	return mux
}

type ctx struct {
	w http.ResponseWriter
	r *http.Request
}

func wrap(fn func(*ctx)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(&ctx{w: w, r: r})
	}
}
