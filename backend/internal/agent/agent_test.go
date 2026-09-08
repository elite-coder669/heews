package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNarrateRoundTrip(t *testing.T) {
	var gotAuth, gotPath string
	var gotPrompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(body.Messages) > 0 {
			gotPrompt = body.Messages[len(body.Messages)-1].Content
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"why\":\"w\",\"reasoning\":\"r\",\"actions\":[{\"id\":\"ISSUE_ADVISORY\",\"priority\":\"HIGH\"}],\"precedent_event_ids\":[\"evt-1\"]}"}}]}`))
	}))
	defer srv.Close()

	c := &Client{APIKey: "test-key", Model: "openrouter/auto", BaseURL: srv.URL}
	n, err := c.Narrate(context.Background(), NarrativeInput{
		Severity:   "HIGH",
		WardName:   "L B Nagar",
		MRI:        48,
		WBGT:       26.1,
		UTCI:       21.7,
		Confidence: "HIGH",
		PrecedentFound: true,
		Candidates: []CandidateAction{
			{ID: "ISSUE_ADVISORY", Text: "Issue advisory", Priority: "LOW", Evidence: []string{"current_risk"}},
			{ID: "REPLICATE_PRECEDENT", Text: "Replicate precedent", Priority: "MEDIUM"},
		},
		Precedents: []Precedent{{EventID: "evt-1", Score: 0.9, Band: "HIGH"}},
	})
	if err != nil {
		t.Fatalf("Narrate: %v", err)
	}
	if n.Why != "w" || n.Reasoning != "r" {
		t.Fatalf("unexpected narrative: %+v", n)
	}
	if len(n.Actions) != 1 || n.Actions[0].ID != "ISSUE_ADVISORY" || n.Actions[0].Priority != "HIGH" {
		t.Fatalf("unexpected actions: %+v", n.Actions)
	}
	if len(n.PrecedentEventIDs) != 1 || n.PrecedentEventIDs[0] != "evt-1" {
		t.Fatalf("unexpected precedents: %+v", n.PrecedentEventIDs)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("auth = %q", gotAuth)
	}
	for _, want := range []string{"Severity: HIGH", "L B Nagar", "MRI: 48", "WBGT: 26.1", "ISSUE_ADVISORY: Issue advisory (current priority LOW", "evt-1"} {
		if !strings.Contains(gotPrompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, gotPrompt)
		}
	}
}

func TestNarrateNoCandidates(t *testing.T) {
	c := &Client{APIKey: "k", Model: "m", BaseURL: "http://unused"}
	if _, err := c.Narrate(context.Background(), NarrativeInput{}); err == nil {
		t.Fatal("expected error for empty candidate pool")
	}
}

func TestClientEnabled(t *testing.T) {
	if (&Client{}).Enabled() {
		t.Error("empty client must not be enabled")
	}
	if (&Client{APIKey: "k"}).Enabled() {
		t.Error("key alone must not enable")
	}
	if !(&Client{APIKey: "k", Model: "m"}).Enabled() {
		t.Error("key+model must enable")
	}
}

func TestNarrateBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", Model: "m", BaseURL: srv.URL}
	if _, err := c.Narrate(context.Background(), NarrativeInput{Candidates: []CandidateAction{{ID: "A"}}}); err == nil {
		t.Fatal("expected parse error")
	}
}

// TestNarrateRejectsUnknownFields proves the model cannot smuggle extra fields
// (e.g. numbers or invented values) past the strict-JSON parser: any unknown
// key turns the whole completion into a parse failure → deterministic fallback.
func TestNarrateRejectsUnknownFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"why\":\"w\",\"reasoning\":\"r\",\"actions\":[{\"id\":\"A\"}],\"precedent_event_ids\":[],\"MRI\":99,\"deaths\":500}"}}]}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", Model: "m", BaseURL: srv.URL}
	if _, err := c.Narrate(context.Background(), NarrativeInput{Candidates: []CandidateAction{{ID: "A"}}}); err == nil {
		t.Fatal("expected reject of unknown numeric fields")
	}
}
