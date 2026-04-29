package core

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	sensitiveHeaders = map[string]bool{
		"authorization": true,
		"cookie":        true,
		"set-cookie":    true,
		"x-api-key":     true,
		"x-auth-token":  true,
	}

	defaultPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)Bearer [A-Za-z0-9\-._~+\/]+=*`),
		regexp.MustCompile(`eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`),
		regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	}
)

// Redactor applies header stripping and value masking to Failure objects.
type Redactor struct {
	extraHeaders  map[string]bool
	extraPatterns []*regexp.Regexp
}

// NewRedactor creates a Redactor with user-supplied extra headers and regex patterns
// appended to the built-in defaults.
func NewRedactor(headers []string, patterns []string) (*Redactor, error) {
	r := &Redactor{extraHeaders: make(map[string]bool)}
	for _, h := range headers {
		r.extraHeaders[strings.ToLower(h)] = true
	}
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("compile redact pattern %q: %w", p, err)
		}
		r.extraPatterns = append(r.extraPatterns, re)
	}
	return r, nil
}

// RedactHeaders returns a copy of the header map with sensitive values replaced.
func (r *Redactor) RedactHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if sensitiveHeaders[strings.ToLower(k)] || r.extraHeaders[strings.ToLower(k)] {
			out[k] = "[REDACTED]"
		} else {
			out[k] = v
		}
	}
	return out
}

// RedactString masks sensitive values within a string.
func (r *Redactor) RedactString(s string) string {
	for _, re := range defaultPatterns {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	for _, re := range r.extraPatterns {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}

// RedactFailure returns a deep copy of f with all sensitive data removed.
// Must be called before any prompt building, LLM send, or disk write.
func (r *Redactor) RedactFailure(f *Failure) *Failure {
	out := *f

	req := f.Request
	req.Headers = r.RedactHeaders(req.Headers)
	req.Body = r.RedactString(req.Body)
	out.Request = req

	resp := f.Response
	resp.Headers = r.RedactHeaders(resp.Headers)
	resp.Body = r.RedactString(resp.Body)
	out.Response = resp

	info := f.Error
	info.Message = r.RedactString(info.Message)
	info.Stack = r.RedactString(info.Stack)
	out.Error = info

	return &out
}
