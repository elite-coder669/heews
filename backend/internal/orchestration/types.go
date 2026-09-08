package orchestration

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
)

type Ward struct {
	WardID      string    `json:"ward_id"`
	WardNumber  int       `json:"ward_number"`
	WardName    string    `json:"ward_name"`
	Zone        string    `json:"zone"`
	Centroid    GeoPoint  `json:"centroid"`
	Demographics Demographics `json:"demographics"`
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Demographics struct {
	ElderlyRatio          float64 `json:"elderly_ratio"`
	OutdoorWorkerShare    float64 `json:"outdoor_worker_share"`
	InformalHousingShare  float64 `json:"informal_housing_share"`
	Population            int     `json:"population"`
	SourceYear            int     `json:"source_year"`
}

type Risk struct {
	WardID       string      `json:"ward_id"`
	ComputedAt   string      `json:"computed_at"`
	Risk         RiskPayload `json:"risk"`
	WBGT         float64     `json:"wbgt_c"`
	UTCI         float64     `json:"utci_c"`
	WBGTQuality  string      `json:"wbgt_quality"`
	RiskModelVersion string  `json:"risk_model_version"`
}

type RiskPayload struct {
	Score      float64  `json:"score"`
	Band       string   `json:"band"`
	TopFactors []string `json:"top_factors"`
}

type Stage struct {
	Stage     string `json:"stage"`
	Status    string `json:"status"`
	LatencyMs int    `json:"latency_ms"`
}

type StageReport struct {
	PipelineRunID string  `json:"pipeline_run_id"`
	Stages        []Stage `json:"stages"`
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// LoadFixtures loads deterministic Hyderabad wards for MOCK mode.
// In LIVE mode, replace with real GeoJSON loader.
func (o *Orchestrator) LoadFixtures() error {
	path := "../../data/fixtures/wards/hyderabad_wards.geojson"
	if o.Cfg.AppMode == "LIVE" {
		path = "../../data/fixtures/wards/hyderabad_wards.geojson"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// Fallback to inline minimal fixture
		o.wards = inlineWards()
		for _, w := range o.wards {
			o.risks[w.WardID] = syntheticRisk(w)
		}
		return nil
	}
	var fc struct {
		Features []struct {
			Properties struct {
				WardNumber int    `json:"ward_number"`
				WardName   string `json:"ward_name"`
				Zone       string `json:"zone"`
			} `json:"properties"`
			Geometry json.RawMessage `json:"geometry"`
		} `json:"features"`
	}
	_ = json.Unmarshal(data, &fc)
	for _, f := range fc.Features {
		w := Ward{
			WardID:     "ward_" + itoa3(f.Properties.WardNumber),
			WardNumber: f.Properties.WardNumber,
			WardName:   f.Properties.WardName,
			Zone:       f.Properties.Zone,
			Centroid:   GeoPoint{Lat: 17.3850 + float64(f.Properties.WardNumber%10)*0.01, Lon: 78.4867 + float64(f.Properties.WardNumber%7)*0.01},
			Demographics: Demographics{
				ElderlyRatio: 0.10, OutdoorWorkerShare: 0.15,
				InformalHousingShare: 0.20, Population: 30000 + f.Properties.WardNumber*100, SourceYear: 2021,
			},
		}
		o.wards = append(o.wards, w)
		o.risks[w.WardID] = syntheticRisk(w)
	}
	return nil
}

func syntheticRisk(w Ward) Risk {
	score := 40 + float64(w.WardNumber%5)*10
	band := "MODERATE"
	switch {
	case score >= 80:
		band = "EXTREME"
	case score >= 60:
		band = "VERY_HIGH"
	case score >= 40:
		band = "HIGH"
	}
	return Risk{
		WardID: w.WardID, ComputedAt: "2026-09-07T12:00:00+05:30",
		Risk: RiskPayload{
			Score: score, Band: band,
			TopFactors: []string{
				"Peak WBGT next 72h = 32.5 °C",
				"UTCI now = 42.1 °C",
				"Outdoor worker share = 15%",
			},
		},
		WBGT: 32.5, UTCI: 42.1, WBGTQuality: "ok",
		RiskModelVersion: "risk-v1.0.0",
	}
}

func itoa3(n int) string {
	if n < 10 {
		return "00" + string(rune('0'+n))
	}
	if n < 100 {
		return "0" + string(rune('0'+n/10)) + string(rune('0'+n%10))
	}
	return string(rune('0'+n/100)) + string(rune('0'+(n/10)%10)) + string(rune('0'+n%10))
}

func inlineWards() []Ward {
	ws := []Ward{}
	for i := 1; i <= 30; i++ {
		ws = append(ws, Ward{
			WardID: "ward_" + itoa3(i), WardNumber: i, WardName: "Ward " + itoa3(i),
			Zone: "Central", Centroid: GeoPoint{Lat: 17.3850, Lon: 78.4867},
			Demographics: Demographics{
				ElderlyRatio: 0.10, OutdoorWorkerShare: 0.15,
				InformalHousingShare: 0.20, Population: 30000 + i*100, SourceYear: 2021,
			},
		})
	}
	return ws
}