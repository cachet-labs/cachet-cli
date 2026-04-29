package core

import "time"

// Failure is a captured API error stored in .cachet/recent/<id>.json.
type Failure struct {
	ID          string    `json:"id"`
	CapturedAt  time.Time `json:"captured_at"`
	Request     Request   `json:"request"`
	Response    Response  `json:"response"`
	Error       ErrorInfo `json:"error"`
	Fingerprint string    `json:"fingerprint"`
}

// Request holds the outgoing HTTP request fields.
type Request struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// Response holds the HTTP response fields.
type Response struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// ErrorInfo holds structured error metadata.
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

// Case is a resolved failure stored in ~/.cachet/cases/<id>.json.
type Case struct {
	ID          string    `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	RootCause   string    `json:"root_cause"`
	Fix         string    `json:"fix"`
	Category    string    `json:"category"`
	Confidence  float64   `json:"confidence"`
	CreatedAt   time.Time `json:"created_at"`
}
