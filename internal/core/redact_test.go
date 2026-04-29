package core

import (
	"testing"
)

func TestRedactHeaders(t *testing.T) {
	r, err := NewRedactor(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	headers := map[string]string{
		"Authorization": "Bearer tok_secret",
		"Content-Type":  "application/json",
		"Cookie":        "session=abc123",
		"X-Api-Key":     "key_12345",
		"X-Request-Id":  "req-001",
	}
	got := r.RedactHeaders(headers)

	redacted := []string{"Authorization", "Cookie", "X-Api-Key"}
	for _, h := range redacted {
		if got[h] != "[REDACTED]" {
			t.Errorf("header %q not redacted, got %q", h, got[h])
		}
	}
	kept := []string{"Content-Type", "X-Request-Id"}
	for _, h := range kept {
		if got[h] == "[REDACTED]" {
			t.Errorf("header %q should not be redacted", h)
		}
	}
}

func TestRedactString(t *testing.T) {
	r, err := NewRedactor(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		in   string
		want string
	}{
		{"Bearer abc123DEF", "[REDACTED]"},
		{"user@example.com logged in", "[REDACTED] logged in"},
		{"key: AKIA1234567890ABCDEF", "key: [REDACTED]"},
		{"no secrets here", "no secrets here"},
	}
	for _, tt := range tests {
		got := r.RedactString(tt.in)
		if got != tt.want {
			t.Errorf("RedactString(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRedactFailure(t *testing.T) {
	r, err := NewRedactor(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	f := &Failure{
		Request: Request{
			Headers: map[string]string{"Authorization": "Bearer secret"},
			Body:    `{"email":"user@example.com"}`,
		},
		Error: ErrorInfo{
			Message: "token AKIA1234567890ABCDEF expired",
		},
	}

	safe := r.RedactFailure(f)

	if safe.Request.Headers["Authorization"] != "[REDACTED]" {
		t.Error("Authorization header not redacted")
	}
	if safe.Request.Body == f.Request.Body {
		t.Error("request body not redacted")
	}
	if safe.Error.Message == f.Error.Message {
		t.Error("error message not redacted")
	}
	// Original must be untouched.
	if f.Request.Headers["Authorization"] != "Bearer secret" {
		t.Error("original failure mutated")
	}
}
