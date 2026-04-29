package core

import "testing"

func TestNormalizeRoute(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/pay", "/pay"},
		{"/users/123", "/users/:id"},
		{"/users/123/orders/456", "/users/:id/orders/:id"},
		{"/users/550e8400-e29b-41d4-a716-446655440000", "/users/:id"},
		{"/items/abc123def456ghi789jkl012mno345", "/items/:id"}, // >24 char alphanum
		{"/search?q=foo", "/search"},
		{"https://api.example.com/users/42/orders", "/users/:id/orders"},
		{"/v1/payments/pay_1234567890abcdef", "/v1/payments/pay_1234567890abcdef"}, // mixed prefix, short
	}
	for _, tt := range tests {
		got := NormalizeRoute(tt.in)
		if got != tt.want {
			t.Errorf("NormalizeRoute(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFingerprint(t *testing.T) {
	tests := []struct {
		method, path string
		status       int
		errType      string
		want         string
	}{
		{"POST", "/pay", 500, "timeout", "POST:/pay:500:timeout"},
		{"GET", "/users/123", 404, "not_found", "GET:/users/:id:404:not_found"},
		{"get", "/items", 200, "OK", "GET:/items:200:ok"},
	}
	for _, tt := range tests {
		got := Fingerprint(tt.method, tt.path, tt.status, tt.errType)
		if got != tt.want {
			t.Errorf("Fingerprint(%q,%q,%d,%q) = %q, want %q",
				tt.method, tt.path, tt.status, tt.errType, got, tt.want)
		}
	}
}
