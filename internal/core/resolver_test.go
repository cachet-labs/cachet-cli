package core

import (
	"strings"
	"testing"
)

func TestParseResolverResponse(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		fingerprint string
		wantCat     string
		wantConf    float64
		wantErr     bool
	}{
		{
			name: "well-formed response",
			response: `Root Cause: Connection pool limit was not set, causing upstream timeouts
Fix: Added MAX_CONNECTIONS=20 to payment service config with exponential backoff
Category: timeout
Confidence: 0.92`,
			fingerprint: "POST:/pay:500:timeout",
			wantCat:     "timeout",
			wantConf:    0.92,
		},
		{
			name: "extra whitespace and blank lines",
			response: `
  Root Cause:  Session token not invalidated on user deletion
  Fix:  Added deletion hook to invalidate active sessions
  Category: auth
  Confidence: 0.85
`,
			fingerprint: "GET:/users/:id:404:not_found",
			wantCat:     "auth",
			wantConf:    0.85,
		},
		{
			name:        "missing root cause",
			response:    "Fix: something\nCategory: unknown\nConfidence: 0.5",
			fingerprint: "X",
			wantErr:     true,
		},
		{
			name:        "missing fix",
			response:    "Root Cause: something broke\nCategory: unknown\nConfidence: 0.5",
			fingerprint: "X",
			wantErr:     true,
		},
		{
			name: "missing confidence defaults to 0.75",
			response: `Root Cause: bad config
Fix: fixed config
Category: config`,
			fingerprint: "X",
			wantCat:     "config",
			wantConf:    0.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseResolverResponse(tt.response, tt.fingerprint)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.Category != tt.wantCat {
				t.Errorf("Category = %q, want %q", c.Category, tt.wantCat)
			}
			if c.Confidence != tt.wantConf {
				t.Errorf("Confidence = %v, want %v", c.Confidence, tt.wantConf)
			}
			if c.Fingerprint != tt.fingerprint {
				t.Errorf("Fingerprint = %q, want %q", c.Fingerprint, tt.fingerprint)
			}
			if !strings.HasPrefix(c.ID, "c_") {
				t.Errorf("ID should start with 'c_', got %q", c.ID)
			}
		})
	}
}

func TestBuildResolverPrompt(t *testing.T) {
	f := &Failure{
		Fingerprint: "POST:/pay:500:timeout",
		Error:       ErrorInfo{Type: "timeout", Message: "upstream timed out"},
		Response:    Response{Status: 500},
	}
	prompt := BuildResolverPrompt(f, "diff --git a/config.go\n+MAX_CONNECTIONS=20")

	for _, want := range []string{"POST:/pay:500:timeout", "GIT DIFF", "Root Cause:", "Confidence:"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}
