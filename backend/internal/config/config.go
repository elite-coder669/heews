package config

import (
	"os"
)

type Config struct {
	Port             string
	AppMode          string // LIVE | MOCK
	OpenMeteoBase    string
	HyderabadLat     float64
	HyderabadLon     float64
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
	return &Config{
		Port:             getEnv("PORT", "8080"),
		AppMode:          getEnv("APP_MODE", "LIVE"),
		OpenMeteoBase:    getEnv("OPEN_METEO_BASE", "https://api.open-meteo.com/v1/forecast"),
		HyderabadLat:     17.3850,
		HyderabadLon:     78.4867,
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
