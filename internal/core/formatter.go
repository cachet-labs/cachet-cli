package core

import (
	"fmt"
	"strings"
)

// BuildPrompt constructs the LLM prompt from a redacted failure and optional similar cases.
func BuildPrompt(f *Failure, cases []*Case) string {
	var sb strings.Builder

	sb.WriteString("You are debugging a runtime API failure. Analyze the failure below and provide a structured diagnosis.\n\n")

	sb.WriteString("== REQUEST ==\n")
	fmt.Fprintf(&sb, "Method: %s\n", f.Request.Method)
	fmt.Fprintf(&sb, "URL: %s\n", f.Request.URL)
	if f.Request.Body != "" {
		fmt.Fprintf(&sb, "Body: %s\n", f.Request.Body)
	}
	sb.WriteString("\n")

	sb.WriteString("== RESPONSE ==\n")
	fmt.Fprintf(&sb, "Status: %d\n", f.Response.Status)
	if f.Response.Body != "" {
		fmt.Fprintf(&sb, "Body: %s\n", f.Response.Body)
	}
	sb.WriteString("\n")

	sb.WriteString("== ERROR ==\n")
	fmt.Fprintf(&sb, "Type: %s\n", f.Error.Type)
	fmt.Fprintf(&sb, "Message: %s\n", f.Error.Message)
	if f.Error.Stack != "" {
		fmt.Fprintf(&sb, "Stack: %s\n", f.Error.Stack)
	}
	sb.WriteString("\n")

	if len(cases) > 0 {
		sb.WriteString("== SIMILAR PAST ISSUES ==\n")
		for _, c := range cases {
			fmt.Fprintf(&sb, "- Fingerprint: %s\n", c.Fingerprint)
			fmt.Fprintf(&sb, "  Root Cause: %s\n", c.RootCause)
			fmt.Fprintf(&sb, "  Fix: %s\n\n", c.Fix)
		}
	}

	sb.WriteString("== TASK ==\n")
	sb.WriteString("1. Identify the root cause (1 sentence)\n")
	sb.WriteString("2. Suggest a concrete fix (1–3 sentences)\n")
	sb.WriteString("3. List edge cases or related failure modes to watch for\n")
	sb.WriteString("4. Assign a category: timeout | auth | not_found | rate_limit | validation | upstream | config | unknown\n")

	return sb.String()
}
