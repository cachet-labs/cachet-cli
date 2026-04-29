package core

import (
	"strings"
)

// BuildPrompt constructs the LLM prompt from a redacted failure and optional similar cases.
// Failure data and past cases are encoded as TOON to reduce token usage.
func BuildPrompt(f *Failure, cases []*Case) string {
	var sb strings.Builder

	sb.WriteString("You are debugging a runtime API failure. Analyze the failure and provide a structured diagnosis.\n\n")

	sb.WriteString(toonFailure(f))
	sb.WriteString("\n")

	if len(cases) > 0 {
		sb.WriteString(toonCases(cases))
		sb.WriteString("\n")
	}

	sb.WriteString("task:\n")
	sb.WriteString("  1. identify the root cause (1 sentence)\n")
	sb.WriteString("  2. suggest a concrete fix (1-3 sentences)\n")
	sb.WriteString("  3. list edge cases or related failure modes to watch for\n")
	sb.WriteString("  4. assign a category: timeout|auth|not_found|rate_limit|validation|upstream|config|unknown\n")

	return sb.String()
}
