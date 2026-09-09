package orchestration

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrWardNotFound signals a CreateManualAlert against a ward that does not
// exist; handlers map it to HTTP 404 WARD_NOT_FOUND.
var ErrWardNotFound = errors.New("ward not found")

type Ward struct {
	WardID       string       `json:"ward_id"`
	WardNumber   int          `json:"ward_number"`
	WardName     string       `json:"ward_name"`
	Zone         string       `json:"zone"`
	Centroid     GeoPoint     `json:"centroid"`
	Demographics Demographics `json:"demographics"`
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Demographics struct {
	ElderlyRatio         float64 `json:"elderly_ratio"`
	OutdoorWorkerShare   float64 `json:"outdoor_worker_share"`
	InformalHousingShare float64 `json:"informal_housing_share"`
	Population           int     `json:"population"`
	SourceYear           int     `json:"source_year"`
}

type Risk struct {
	WardID           string      `json:"ward_id"`
	ComputedAt       string      `json:"computed_at"`
	Risk             RiskPayload `json:"risk"`
	WBGT             float64     `json:"wbgt_c"`
	UTCI             float64     `json:"utci_c"`
	WBGTQuality      string      `json:"wbgt_quality"`
	RiskModelVersion string      `json:"risk_model_version"`
}

type RiskPayload struct {
	Score      float64  `json:"score"`
	Band       string   `json:"band"`
	TopFactors []string `json:"top_factors"`
}

type WardThermal struct {
	WardID         string         `json:"ward_id"`
	PhysicsVersion string         `json:"physics_version"`
	Current        ThermalCurrent `json:"current"`
}

type ThermalCurrent struct {
	WBGT        float64 `json:"wbgt_c"`
	UTCI        float64 `json:"utci_c"`
	WBGTQuality string  `json:"wbgt_quality"`
}

type WardWeather struct {
	WardID    string         `json:"ward_id"`
	Source    string         `json:"source"`
	FetchedAt string         `json:"fetched_at"`
	Current   WeatherCurrent `json:"current"`
}

type WeatherCurrent struct {
	TemperatureC  float64 `json:"temperature_c"`
	HumidityPct   float64 `json:"humidity_pct"`
	WindMS        float64 `json:"wind_ms"`
	ShortwaveWm2  float64 `json:"shortwave_rad_wm2"`
	PressurePa    float64 `json:"pressure_pa"`
	CloudCoverPct float64 `json:"cloud_cover_pct"`
}

type ForecastCell struct {
	Day  string  `json:"day"`
	WBGT float64 `json:"wbgt_c"`
	UTCI float64 `json:"utci_c"`
	MRI  float64 `json:"mri"`
	Band string  `json:"band"`
}

type ForecastSeries struct {
	WardID string         `json:"ward_id"`
	Series []ForecastCell `json:"series"`
}

type CityForecast struct {
	City         string           `json:"city"`
	HorizonHours int              `json:"horizon_hours"`
	ByWard       []ForecastSeries `json:"by_ward"`
}

type AlertAction struct {
	Action   string   `json:"action"`
	Priority string   `json:"priority"`
	Target   string   `json:"target"`
	Reason   string   `json:"reason,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
}

// ManualAlertInput is the structured payload for a human-approved municipal
// alert. Plan provenance fields are optional; when set they link the alert to
// the validated decision plan it was issued from.
type ManualAlertInput struct {
	WardID             string        `json:"ward_id"`
	Severity           string        `json:"severity"`
	Headline           string        `json:"headline"`
	Body               string        `json:"body"`
	RecommendedActions []AlertAction `json:"recommended_actions"`
	AgentRunID         string        `json:"agent_run_id"`
	AgentVersion       string        `json:"agent_version"`
	FallbackUsed       bool          `json:"fallback_used"`
}

type Alert struct {
	ID                 string        `json:"id"`
	WardID             string        `json:"ward_id"`
	CreatedAt          string        `json:"created_at"`
	Severity           string        `json:"severity"`
	Headline           string        `json:"headline"`
	Body               string        `json:"body"`
	RecommendedActions []AlertAction `json:"recommended_actions"`
	Status             string        `json:"status"`
	Source             string        `json:"source,omitempty"` // "pipeline" | "municipal"
	AgentRunID         string        `json:"agent_run_id,omitempty"`
	AgentVersion       string        `json:"agent_version,omitempty"`
	FallbackUsed       bool          `json:"fallback_used,omitempty"`
}

type PriorityWard struct {
	WardID   string  `json:"ward_id"`
	WardName string  `json:"ward_name"`
	Score    float64 `json:"score"`
	Band     string  `json:"band"`
}

type ActionPlan struct {
	Scope                  string            `json:"scope,omitempty"` // "current" | "forecast"
	ScopeWardID            string            `json:"scope_ward_id,omitempty"`
	ScopeHorizonHours      int               `json:"scope_horizon_hours,omitempty"`
	ScopeLabel             string            `json:"scope_label,omitempty"` // e.g. "48h", "Forecast day: Tomorrow"
	Severity               string            `json:"severity"`
	PriorityWards          []PriorityWard    `json:"priority_wards"`
	ReasoningSummary       string            `json:"reasoning_summary"`
	KeyFactors             []string          `json:"key_factors"`
	RecommendedActions     []AlertAction     `json:"recommended_actions"`
	PublicAdvisory         string            `json:"public_advisory"`
	RecheckIntervalMinutes int               `json:"recheck_interval_minutes"`
	AgentRunID             string            `json:"agent_run_id"`
	FallbackUsed           bool              `json:"fallback_used"`
	AgentVersion           string            `json:"agent_version"`
	WhyThisMatters         string            `json:"why_this_matters"`
	HistoricalContext      HistoricalContext `json:"historical_context"`
	MatchingEvents         []MatchingEvent   `json:"matching_events,omitempty"`
	Evidence               Evidence          `json:"evidence"`
	Confidence             string            `json:"confidence"`
	Limitations            []string          `json:"limitations"`
}

type HistoricalContext struct {
	Found            bool   `json:"found"`
	Type             string `json:"type"` // local | demo | none
	EventID          string `json:"event_id,omitempty"`
	SimilarityReason string `json:"similarity_reason,omitempty"`
	Source           string `json:"source,omitempty"`
	Synthetic        bool   `json:"synthetic"`
}

type MatchingEvent struct {
	EventID      string             `json:"event_id"`
	Date         string             `json:"date"`
	WardName     string             `json:"ward_name,omitempty"`
	Score        float64            `json:"score"`
	Band         string             `json:"band"`
	Similarity   float64            `json:"similarity"`
	Dimensions   map[string]float64 `json:"dimensions,omitempty"`
	ActionsTaken []string           `json:"actions_taken,omitempty"`
	Outcome      string             `json:"outcome,omitempty"`
}

type Evidence struct {
	CurrentRisk         bool `json:"current_risk"`
	Vulnerability       bool `json:"vulnerability"`
	Forecast            bool `json:"forecast"`
	HistoricalPrecedent bool `json:"historical_precedent"`
	ExternalReference   bool `json:"external_reference"`
	GeneralGuidance     bool `json:"general_guidance"`
}

type PipelineDoc struct {
	GeneratedAt string                 `json:"generated_at"`
	Mode        string                 `json:"mode"`
	Wards       []Ward                 `json:"wards"`
	Risks       map[string]Risk        `json:"risks"`
	Thermal     map[string]WardThermal `json:"thermal"`
	Weather     map[string]WardWeather `json:"weather"`
	Forecast    CityForecast           `json:"forecast"`
	Alerts      []Alert                `json:"alerts"`
}

type Stage struct {
	Stage     string `json:"stage"`
	Status    string `json:"status"`
	LatencyMs int    `json:"latency_ms"`
}

type StageReport struct {
	PipelineRunID  string  `json:"pipeline_run_id"`
	Stages         []Stage `json:"stages"`
	LatencyTotalMs int     `json:"latency_total_ms"`
	Degraded       bool    `json:"degraded"`
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type geoFeatureCollection struct {
	Features []struct {
		Properties struct {
			WardNumber int    `json:"ward_number"`
			WardName   string `json:"ward_name"`
			Zone       string `json:"zone"`
		} `json:"properties"`
		Geometry json.RawMessage `json:"geometry"`
	} `json:"features"`
}

// centroidFromGeometry computes the bounding-box centre of the first ring.
// Supports Polygon (coords[0]) and MultiPolygon (coords[0][0]).
func centroidFromGeometry(raw json.RawMessage) (GeoPoint, error) {
	var geom struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal(raw, &geom); err != nil {
		return GeoPoint{}, err
	}
	var ring [][]float64
	switch geom.Type {
	case "Polygon":
		var polys [][][]float64
		if err := json.Unmarshal(geom.Coordinates, &polys); err != nil {
			return GeoPoint{}, err
		}
		if len(polys) == 0 || len(polys[0]) < 3 {
			return GeoPoint{}, fmt.Errorf("empty polygon ring")
		}
		ring = polys[0]
	case "MultiPolygon":
		var mp [][][][]float64
		if err := json.Unmarshal(geom.Coordinates, &mp); err != nil {
			return GeoPoint{}, err
		}
		if len(mp) == 0 || len(mp[0]) == 0 || len(mp[0][0]) < 3 {
			return GeoPoint{}, fmt.Errorf("empty multipolygon ring")
		}
		ring = mp[0][0]
	default:
		return GeoPoint{}, fmt.Errorf("unsupported geometry %q", geom.Type)
	}
	minLat, minLon, maxLat, maxLon := ring[0][1], ring[0][0], ring[0][1], ring[0][0]
	for _, p := range ring[1:] {
		if p[1] < minLat {
			minLat = p[1]
		}
		if p[1] > maxLat {
			maxLat = p[1]
		}
		if p[0] < minLon {
			minLon = p[0]
		}
		if p[0] > maxLon {
			maxLon = p[0]
		}
	}
	return GeoPoint{Lat: (minLat + maxLat) / 2, Lon: (minLon + maxLon) / 2}, nil
}

// LoadFixtures loads real ward geometry + demographics for the configured
// city (o.Cfg.City, e.g. "hyderabad", "mumbai", "delhi" — see CITY env var
// and backend/internal/config/cities.go) for all modes.
func (o *Orchestrator) LoadFixtures() error {
	city := "hyderabad"
	if o.Cfg != nil && o.Cfg.City != "" {
		city = o.Cfg.City
	}
	data, err := os.ReadFile(filepath.Join(o.repoRoot, "data/fixtures/wards/"+city+"_wards.geojson"))
	if err != nil {
		o.wards = inlineWards()
		for _, w := range o.wards {
			o.risks[w.WardID] = syntheticRisk(w)
		}
		return nil
	}
	var fc geoFeatureCollection
	if err := json.Unmarshal(data, &fc); err != nil {
		o.wards = inlineWards()
		for _, w := range o.wards {
			o.risks[w.WardID] = syntheticRisk(w)
		}
		return nil
	}
	demographics := o.loadDemographics()
	for _, f := range fc.Features {
		c, err := centroidFromGeometry(f.Geometry)
		if err != nil {
			continue
		}
		wid := "ward_" + itoa3(f.Properties.WardNumber)
		d := Demographics{ElderlyRatio: 0.10, OutdoorWorkerShare: 0.15, InformalHousingShare: 0.20, Population: 30000 + f.Properties.WardNumber*100, SourceYear: 2021}
		if dm, ok := demographics[wid]; ok {
			d = dm
		}
		w := Ward{
			WardID: wid, WardNumber: f.Properties.WardNumber,
			WardName: f.Properties.WardName, Zone: f.Properties.Zone,
			Centroid: c, Demographics: d,
		}
		o.wards = append(o.wards, w)
		o.risks[w.WardID] = syntheticRisk(w)
	}
	return nil
}

func (o *Orchestrator) loadDemographics() map[string]Demographics {
	city := "hyderabad"
	if o.Cfg != nil && o.Cfg.City != "" {
		city = o.Cfg.City
	}
	data, err := os.ReadFile(filepath.Join(o.repoRoot, "data/fixtures/demographics/"+city+"_demo.json"))
	if err != nil {
		return nil
	}
	var rows []struct {
		Demographics
		WardID string `json:"ward_id"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil
	}
	out := make(map[string]Demographics, len(rows))
	for _, r := range rows {
		out[r.WardID] = r.Demographics
	}
	return out
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
