package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// CandidateAction is one deterministic action in the pool the model may select.
// The model never invents actions: it can only choose, order, and (within
// LOW/MEDIUM/HIGH) re-prioritise entries from this pool.
type CandidateAction struct {
	ID       string
	Text     string
	Priority string
	Evidence []string
}

// Precedent is one matched historical event offered for context.
type Precedent struct {
	EventID      string
	Score        float64
	Band         string
	ActionsTaken []string
	Outcome      string
}

// NarrativeInput is the pure, pre-computed context handed to the model. Every
// number was fixed by the deterministic plan before this struct was built; the
// schema forces the model to answer by referencing candidate IDs only, so it
// cannot modify or invent quantitative values.
type NarrativeInput struct {
	CityLabel          string // e.g. "Hyderabad" — which municipality this agent is deciding for
	Severity           string
	WardName           string
	MRI                float64
	WBGT               float64
	UTCI               float64
	VulnerabilityKnown bool
	Confidence         string
	PrecedentFound     bool
	Candidates         []CandidateAction
	Precedents         []Precedent
	ForecastTrajectory string
	MunicipalContext   string
	ScopeLabel         string // "Current conditions" or "Forecast day: <label>"
}

// SelectedAction references one candidate by ID and may adjust its priority.
type SelectedAction struct {
	ID       string `json:"id"`
	Priority string `json:"priority"`
}

// Narrative is the strict-JSON shape the model must return. Unknown fields are
// rejected at parse time; precedent_event_ids reference real matched events.
type Narrative struct {
	Why               string           `json:"why"`
	Reasoning         string           `json:"reasoning"`
	Actions           []SelectedAction `json:"actions"`
	PrecedentEventIDs []string         `json:"precedent_event_ids"`
}

// Narrate asks the model to select and rank the candidate action pool.
// The caller owns gating and validation; any error here means the
// deterministic plan stands.
func (c *Client) Narrate(ctx context.Context, in NarrativeInput) (Narrative, error) {
	if len(in.Candidates) == 0 {
		return Narrative{}, fmt.Errorf("no candidate actions to select from")
	}
	raw, err := c.chatCompletion(ctx, buildPrompt(in))
	if err != nil {
		return Narrative{}, err
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.DisallowUnknownFields()
	var n Narrative
	if err := dec.Decode(&n); err != nil {
		return Narrative{}, fmt.Errorf("parse narrative json: %w", err)
	}
	return n, nil
}

func buildPrompt(in NarrativeInput) string {
	var b strings.Builder
	city := in.CityLabel
	if city == "" {
		city = "the city"
	}
	b.WriteString(fmt.Sprintf("You are the municipal heat decision agent for %s. Select and rank municipal actions from the candidate pool below. You may not invent actions, numbers, evidence, or events.\n", city))
	b.WriteString(fmt.Sprintf("Severity: %s. Ward: %s. MRI: %.0f. WBGT: %.1f C. UTCI: %.1f C. Vulnerability: available=%t. Confidence: %s.\n",
		in.Severity, in.WardName, in.MRI, in.WBGT, in.UTCI, in.VulnerabilityKnown, in.Confidence))
	b.WriteString("Forecast trajectory: " + forecastStr(in.ForecastTrajectory) + "\n")
	b.WriteString("Candidate actions (choose a subset, ordered by priority):\n")
	for _, a := range in.Candidates {
		b.WriteString(fmt.Sprintf("- %s: %s (current priority %s, evidence: %s)\n", a.ID, a.Text, a.Priority, strings.Join(a.Evidence, ", ")))
	}
	if in.PrecedentFound {
		b.WriteString("Historical precedents (may reference by event id in precedent_event_ids):\n")
		for _, p := range in.Precedents {
			b.WriteString(fmt.Sprintf("- %s (similarity %.2f, band %s): took %s; outcome %s\n",
				p.EventID, p.Score, p.Band, strings.Join(p.ActionsTaken, "; "), p.Outcome))
		}
	} else {
		b.WriteString("No historical precedent is available; precedent_event_ids must be empty.\n")
	}
	if in.MunicipalContext != "" {
		b.WriteString("Municipal context: " + in.MunicipalContext + "\n")
	}
	b.WriteString(`Return strict JSON only: {"why":"...","reasoning":"...","actions":[{"id":"<candidate id>","priority":"LOW|MEDIUM|HIGH"},...],"precedent_event_ids":["<event id>",...]}. actions must be a non-empty subset of the candidates, best first. No other fields.`)
	return b.String()
}

func forecastStr(t string) string {
	if t == "" {
		return "unknown"
	}
	return t
}
