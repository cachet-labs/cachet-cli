package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildResolverPrompt creates the LLM prompt used in `verify` to extract a structured Case.
// Failure data is TOON-encoded to reduce tokens; the output format section is kept verbatim
// because ParseResolverResponse depends on its exact line structure.
func BuildResolverPrompt(f *Failure, diff string) string {
	var sb strings.Builder

	sb.WriteString("A bug was fixed. Extract a structured resolution from the failure and git diff below.\n\n")

	sb.WriteString(toonFailure(f))
	sb.WriteString("\n")

	sb.WriteString("git_diff:\n")
	if diff == "" {
		sb.WriteString("  (none)\n")
	} else {
		sb.WriteString(diff)
	}
	sb.WriteString("\n")

	// Output format kept verbatim — ParseResolverResponse parses these exact keys.
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
