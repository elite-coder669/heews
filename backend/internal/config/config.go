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
	AgentLLMProvider string
	AgentAPIKey      string
	LogLevel         string
	AutoRun          bool
	PipelineInterval string
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		AppMode:          getEnv("APP_MODE", "MOCK"),
		OpenMeteoBase:    getEnv("OPEN_METEO_BASE", "https://api.open-meteo.com/v1/forecast"),
		HyderabadLat:     17.3850,
		HyderabadLon:     78.4867,
		DBURL:            getEnv("DB_URL", ""),
		AgentLLMProvider: getEnv("AGENT_LLM_PROVIDER", "mock"),
		AgentAPIKey:      getEnv("AGENT_API_KEY", ""),
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