package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const openAIEndpoint = "https://api.openai.com/v1/chat/completions"

// OpenAIAdapter calls the OpenAI chat completions API using raw HTTP (no SDK).
type OpenAIAdapter struct {
	apiKey string
	model  string
	client *http.Client
}

// NewOpenAIAdapter creates an adapter for the given API key and model.
func NewOpenAIAdapter(apiKey, model string) *OpenAIAdapter {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIAdapter{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// Ask sends prompt to the OpenAI chat completions API and returns the text response.
func (o *OpenAIAdapter) Ask(prompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":      o.model,
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, openAIEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build openai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read openai response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai returned %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse openai response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return result.Choices[0].Message.Content, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
