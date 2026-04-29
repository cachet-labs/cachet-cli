package llm

import (
	"context"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicAdapter calls the Anthropic Messages API.
type AnthropicAdapter struct {
	client anthropic.Client
	model  string
}

// NewAnthropicAdapter creates an adapter for the given API key and model.
func NewAnthropicAdapter(apiKey, model string) *AnthropicAdapter {
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &AnthropicAdapter{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// Ask sends prompt to the Anthropic API and returns the text response.
func (a *AnthropicAdapter) Ask(prompt string) (string, error) {
	msg, err := a.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic request: %w", err)
	}
	for _, block := range msg.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("no text content in anthropic response")
}
