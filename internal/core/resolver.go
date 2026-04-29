package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildResolverPrompt creates the LLM prompt used in `verify` to extract a structured Case.
func BuildResolverPrompt(f *Failure, diff string) string {
	var sb strings.Builder

	sb.WriteString("A bug was fixed. Given the failure context and the git diff below, generate a structured resolution.\n\n")

	sb.WriteString("== ORIGINAL FAILURE ==\n")
	fmt.Fprintf(&sb, "Fingerprint: %s\n", f.Fingerprint)
	fmt.Fprintf(&sb, "Error Type: %s\n", f.Error.Type)
	fmt.Fprintf(&sb, "Error Message: %s\n", f.Error.Message)
	fmt.Fprintf(&sb, "Status: %d\n", f.Response.Status)
	sb.WriteString("\n")

	sb.WriteString("== GIT DIFF ==\n")
	if diff == "" {
		sb.WriteString("(no diff provided)\n")
	} else {
		sb.WriteString(diff)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	sb.WriteString("== OUTPUT FORMAT (strict — no extra text) ==\n")
	sb.WriteString("Root Cause: <one sentence identifying the root cause>\n")
	sb.WriteString("Fix: <one sentence describing the fix applied>\n")
	sb.WriteString("Category: <timeout|auth|not_found|rate_limit|validation|upstream|config|unknown>\n")
	sb.WriteString("Confidence: <float 0.0–1.0>\n")

	return sb.String()
}

// ParseResolverResponse parses the strict-format LLM output into a Case.
// It is tolerant of leading/trailing whitespace and extra blank lines.
func ParseResolverResponse(response, fingerprint string) (*Case, error) {
	fields := map[string]string{}
	for _, line := range strings.Split(response, "\n") {
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:idx]))
		val := strings.TrimSpace(line[idx+1:])
		if val != "" {
			fields[key] = val
		}
	}

	rootCause := fields["root cause"]
	if rootCause == "" {
		return nil, fmt.Errorf("resolver response is missing 'Root Cause'")
	}
	fix := fields["fix"]
	if fix == "" {
		return nil, fmt.Errorf("resolver response is missing 'Fix'")
	}

	category := strings.ToLower(fields["category"])
	if category == "" {
		category = "unknown"
	}

	confidence := 0.75
	if raw := fields["confidence"]; raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			if v > 1.0 {
				v = 1.0
			} else if v < 0.0 {
				v = 0.0
			}
			confidence = v
		}
	}

	return &Case{
		ID:          "c_" + uuid.New().String(),
		Fingerprint: fingerprint,
		RootCause:   rootCause,
		Fix:         fix,
		Category:    category,
		Confidence:  confidence,
		CreatedAt:   time.Now().UTC(),
	}, nil
}
