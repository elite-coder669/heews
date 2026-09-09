package config

import (
	"os"
)

type Config struct {
	Port          string
	AppMode       string // LIVE | MOCK
	OpenMeteoBase string

	// City selects which municipality's ward data (and weather reference
	// point) this instance runs for. Set via CITY=<slug>, e.g. CITY=mumbai.
	// See backend/internal/config/cities.go for the full supported list.
	City      string  // slug, e.g. "hyderabad" — used to build data file paths
	CityLabel string  // display name, e.g. "Hyderabad"
	CityState string  // e.g. "Telangana"
	CityLat   float64 // city-wide reference point for the weather feed
	CityLon   float64

	DBURL            string
	AgentLLMProvider string // mock | openrouter
	OpenRouterAPIKey string // OPENROUTER_API_KEY
	LLMModel         string // LLM_MODEL
	DemoMemory       bool
	LogLevel         string
	AutoRun          bool
	PipelineInterval string
}

func Load() *Config {
	citySlug, city := CityOrDefault(getEnv("CITY", "hyderabad"))
	return &Config{
		Port:             getEnv("PORT", "8080"),
		AppMode:          getEnv("APP_MODE", "LIVE"),
		OpenMeteoBase:    getEnv("OPEN_METEO_BASE", "https://api.open-meteo.com/v1/forecast"),
		City:             citySlug,
		CityLabel:        city.Label,
		CityState:        city.State,
		CityLat:          city.Lat,
		CityLon:          city.Lon,
		DBURL:            getEnv("DB_URL", ""),
		AgentLLMProvider: getEnv("AGENT_LLM_PROVIDER", "mock"),
		OpenRouterAPIKey: getEnv("OPENROUTER_API_KEY", ""),
		LLMModel:         getEnv("LLM_MODEL", "nvidia/nemotron-3.5-lightning:free"),
		DemoMemory:       getEnv("HEAT_DEMO_MEMORY", "false") == "true",
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		AutoRun:          getEnv("AUTO_RUN", "false") == "true",
		PipelineInterval: getEnv("PIPELINE_INTERVAL", "15m"),
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
