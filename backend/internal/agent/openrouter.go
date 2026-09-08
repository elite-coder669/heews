// Package agent owns the optional LLM decision-support layer for the
// decision agent. It talks to OpenRouter's OpenAI-compatible chat completions
// endpoint and nothing else; orchestration decides when (and whether) a
// proposal is requested. The LLM may only select and rank from a candidate
// pool orchestration provides — it never builds the plan.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// OpenRouterBase is the OpenAI-compatible chat completions endpoint.
const OpenRouterBase = "https://openrouter.ai/api/v1"

// Client is a minimal OpenRouter chat-completions client.
type Client struct {
	APIKey string
	Model  string
	// BaseURL overrides the default OpenRouter endpoint (tests, self-hosted
	// OpenAI-compatible gateways). Empty means OpenRouterBase.
	BaseURL string
	HTTP    *http.Client
}

// Enabled reports whether the client can make a request.
func (c *Client) Enabled() bool {
	return c != nil && c.APIKey != "" && c.Model != ""
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return OpenRouterBase
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// chatCompletion posts a system+user prompt and returns the raw completion.
func (c *Client) chatCompletion(ctx context.Context, prompt string) (string, error) {
	payload := map[string]any{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are the Municipal Heat Decision Agent. Respond only with strict JSON. Never invent numbers, events, or outcomes."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := c.http().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openrouter http %d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(out.Choices) == 0 || out.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty chat completion")
	}
	return out.Choices[0].Message.Content, nil
}
