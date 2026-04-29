package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const defaultWidth = 80

// isTTY reports whether stdout is an interactive terminal.
var isTTY = func() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}()

// IsTTY returns true when stdout is an interactive terminal.
func IsTTY() bool { return isTTY }

// ── Palette ───────────────────────────────────────────────────────────────────

var (
	purple = lipgloss.Color("#7C3AED")
	cyan   = lipgloss.Color("#06B6D4")
	green  = lipgloss.Color("#10B981")
	red    = lipgloss.Color("#EF4444")
	amber  = lipgloss.Color("#F59E0B")
	gray   = lipgloss.Color("#6B7280")
	white  = lipgloss.Color("#F1F5F9")
	pink   = lipgloss.Color("#EC4899")
	violet = lipgloss.Color("#8B5CF6")
)

// ── Base styles ───────────────────────────────────────────────────────────────

var (
	boldPurple = lipgloss.NewStyle().Foreground(purple).Bold(true)
	boldCyan   = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	boldGreen  = lipgloss.NewStyle().Foreground(green).Bold(true)
	boldRed    = lipgloss.NewStyle().Foreground(red).Bold(true)
	boldAmber  = lipgloss.NewStyle().Foreground(amber).Bold(true)
	muted      = lipgloss.NewStyle().Foreground(gray)
	plain      = lipgloss.NewStyle().Foreground(white)
)

// ── Banner ────────────────────────────────────────────────────────────────────

// PrintBanner prints the cachet wordmark to stdout.
func PrintBanner() {
	if !isTTY {
		return
	}
	diamond := lipgloss.NewStyle().Foreground(purple).Bold(true).Render("◆")
	name := lipgloss.NewStyle().Foreground(white).Bold(true).Render("cachet")
	sub := muted.Render("runtime failure intelligence")
	fmt.Printf("\n  %s %s  %s\n\n", diamond, name, sub)
}

// ── Status lines ──────────────────────────────────────────────────────────────

// Success prints a green success line to stdout.
func Success(msg string) {
	icon := boldGreen.Render("✓")
	fmt.Printf("  %s  %s\n", icon, plain.Render(msg))
}

// Warn prints an amber warning line to stdout.
func Warn(msg string) {
	icon := boldAmber.Render("!")
	fmt.Printf("  %s  %s\n", icon, plain.Render(msg))
}

// Error prints a red error line to stderr.
func Error(msg string) {
	icon := boldRed.Render("✗")
	fmt.Fprintf(os.Stderr, "  %s  %s\n", icon, msg)
}

// Info prints a dim informational line to stderr (safe to pipe stdout).
func Info(msg string) {
	fmt.Fprintf(os.Stderr, "  %s  %s\n", boldCyan.Render("→"), muted.Render(msg))
}

// ── Key-value display ─────────────────────────────────────────────────────────

// KV prints aligned key → value pairs to stdout.
// pairs must alternate key, value, key, value, ...
func KV(pairs ...string) {
	maxLen := 0
	for i := 0; i < len(pairs)-1; i += 2 {
		if l := len(pairs[i]); l > maxLen {
			maxLen = l
		}
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		k := pairs[i]
		v := pairs[i+1]
		pad := strings.Repeat(" ", maxLen-len(k))
		label := boldCyan.Render(k)
		fmt.Printf("  %s%s   %s\n", label, pad, plain.Render(v))
	}
}

// ── Section header ────────────────────────────────────────────────────────────

// SectionHeader prints a titled horizontal rule to stdout.
func SectionHeader(title string) {
	rule := muted.Render(strings.Repeat("─", defaultWidth-4-len(title)-3))
	if isTTY {
		fmt.Printf("\n  %s  %s\n\n", boldPurple.Render(title), rule)
	} else {
		fmt.Printf("\n  %s\n\n", title)
	}
}

// ── Rounded box ───────────────────────────────────────────────────────────────

var boxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(purple).
	Padding(1, 2).
	Width(defaultWidth - 2)

// Box renders content inside a rounded, purple-bordered box to stdout.
func Box(title, content string) {
	if !isTTY {
		fmt.Printf("=== %s ===\n%s\n", title, content)
		return
	}
	header := boldPurple.Render(title)
	divider := muted.Render(strings.Repeat("─", defaultWidth-8))
	body := header + "\n" + divider + "\n\n" + content
	fmt.Println(boxStyle.Render(body))
}

// DiagnosisBox prints the LLM response inside a styled box.
// When stdout is not a TTY the raw content is printed without decoration.
func DiagnosisBox(fingerprint, content string) {
	if !isTTY {
		fmt.Println(content)
		return
	}
	header := boldPurple.Render("Diagnosis")
	fp := muted.Render(fingerprint)
	divider := muted.Render(strings.Repeat("─", defaultWidth-8))
	body := fmt.Sprintf("%s  %s\n%s\n\n%s", header, fp, divider, content)
	fmt.Println(boxStyle.Render(body))
}

// ── Table ─────────────────────────────────────────────────────────────────────

// Table renders an aligned table with styled borders and column headers.
// rows contains raw (unstyled) text; styling is applied internally.
func Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println(muted.Render("  No entries."))
		return
	}

	// Compute column widths from raw content.
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i := range headers {
			if i < len(row) && len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}

	// Build separator lines.
	topLine := func(left, mid, right, fill string) string {
		parts := make([]string, len(widths))
		for i, w := range widths {
			parts[i] = strings.Repeat(fill, w+2)
		}
		return "  " + left + strings.Join(parts, mid) + right
	}

	// Render a row with per-cell padding applied BEFORE styling.
	renderRow := func(cells []string, style lipgloss.Style) string {
		parts := make([]string, len(widths))
		for i := range widths {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			padded := " " + cell + strings.Repeat(" ", widths[i]-len(cell)+1)
			if isTTY {
				parts[i] = style.Render(padded)
			} else {
				parts[i] = padded
			}
		}
		return "  │" + strings.Join(parts, "│") + "│"
	}

	sep := muted.Render

	fmt.Println()
	fmt.Println(sep(topLine("┌", "┬", "┐", "─")))
	fmt.Println(renderRow(headers, boldCyan))
	fmt.Println(sep(topLine("├", "┼", "┤", "─")))
	for _, row := range rows {
		fmt.Println(renderRow(row, plain))
	}
	fmt.Println(sep(topLine("└", "┴", "┘", "─")))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// CategoryBadge returns the category string with a contextual color when in TTY mode.
func CategoryBadge(cat string) string {
	if !isTTY {
		return cat
	}
	colors := map[string]lipgloss.Color{
		"timeout":    amber,
		"auth":       red,
		"not_found":  gray,
		"rate_limit": amber,
		"validation": violet,
		"upstream":   pink,
		"config":     cyan,
		"unknown":    gray,
	}
	c, ok := colors[strings.ToLower(cat)]
	if !ok {
		c = gray
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true).Render(cat)
}

// ConfidencePct formats confidence as a percentage with a color reflecting quality.
func ConfidencePct(f float64) string {
	pct := fmt.Sprintf("%.0f%%", f*100)
	if !isTTY {
		return pct
	}
	var c lipgloss.Color
	switch {
	case f >= 0.85:
		c = green
	case f >= 0.6:
		c = amber
	default:
		c = gray
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true).Render(pct)
}

// ShortID truncates an ID to 20 chars with a trailing ellipsis.
func ShortID(id string) string {
	if len(id) <= 20 {
		return id
	}
	return id[:20] + "…"
}

// Muted renders s in gray when in TTY mode.
func Muted(s string) string {
	if !isTTY {
		return s
	}
	return muted.Render(s)
}

// Bold renders s bold when in TTY mode.
func Bold(s string) string {
	if !isTTY {
		return s
	}
	return lipgloss.NewStyle().Bold(true).Render(s)
}

// Hint prints a dimmed hint line to stdout.
func Hint(msg string) {
	fmt.Printf("  %s\n", muted.Render(msg))
}
