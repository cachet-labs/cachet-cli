package core

import (
	"fmt"
	"strconv"
	"strings"
)

// toonFailure encodes a Failure as a TOON object for LLM prompt injection.
// Keeps JSON on disk untouched — TOON is only for the LLM input layer.
func toonFailure(f *Failure) string {
	var sb strings.Builder
	sb.WriteString("failure:\n")
	if f.Fingerprint != "" {
		fmt.Fprintf(&sb, "  fingerprint: %s\n", f.Fingerprint)
	}
	sb.WriteString("  request:\n")
	fmt.Fprintf(&sb, "    method: %s\n", f.Request.Method)
	fmt.Fprintf(&sb, "    url: %s\n", f.Request.URL)
	if f.Request.Body != "" {
		fmt.Fprintf(&sb, "    body: %s\n", toonInline(f.Request.Body))
	}
	sb.WriteString("  response:\n")
	fmt.Fprintf(&sb, "    status: %d\n", f.Response.Status)
	if f.Response.Body != "" {
		fmt.Fprintf(&sb, "    body: %s\n", toonInline(f.Response.Body))
	}
	sb.WriteString("  error:\n")
	fmt.Fprintf(&sb, "    type: %s\n", f.Error.Type)
	fmt.Fprintf(&sb, "    message: %s\n", toonInline(f.Error.Message))
	if f.Error.Stack != "" {
		fmt.Fprintf(&sb, "    stack: %s\n", toonInline(f.Error.Stack))
	}
	return sb.String()
}

// toonCases encodes a slice of Cases as a TOON tabular array.
// Each row is: fingerprint,rootCause,fix,category,confidence
func toonCases(cases []*Case) string {
	if len(cases) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "similar_cases[%d]{fingerprint,rootCause,fix,category,confidence}:\n", len(cases))
	for _, c := range cases {
		fmt.Fprintf(&sb, "  %s,%s,%s,%s,%s\n",
			toonCell(c.Fingerprint),
			toonCell(c.RootCause),
			toonCell(c.Fix),
			toonCell(c.Category),
			strconv.FormatFloat(c.Confidence, 'f', 2, 64),
		)
	}
	return sb.String()
}

// toonCell quotes a CSV cell value if it contains a comma, double-quote, or newline.
func toonCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if strings.ContainsAny(s, `,"`) {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// toonInline collapses a potentially multi-line value to a single line.
func toonInline(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
}
