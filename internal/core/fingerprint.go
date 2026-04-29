package core

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	uuidRe    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	numericRe = regexp.MustCompile(`^\d+$`)
	hashRe    = regexp.MustCompile(`^[a-zA-Z0-9]{25,}$`)
)

// NormalizeRoute replaces dynamic path segments (UUIDs, numbers, hashes) with :id.
func NormalizeRoute(rawPath string) string {
	// Strip scheme+host if present.
	if u, err := url.Parse(rawPath); err == nil && u.Path != "" && u.Host != "" {
		rawPath = u.Path
	}
	// Strip query string.
	if i := strings.IndexByte(rawPath, '?'); i != -1 {
		rawPath = rawPath[:i]
	}
	segments := strings.Split(rawPath, "/")
	for i, seg := range segments {
		if seg == "" {
			continue
		}
		if uuidRe.MatchString(seg) || numericRe.MatchString(seg) || hashRe.MatchString(seg) {
			segments[i] = ":id"
		}
	}
	return strings.Join(segments, "/")
}

// Fingerprint generates METHOD:NORMALIZED_ROUTE:STATUS:ERROR_TYPE.
func Fingerprint(method, rawPath string, status int, errorType string) string {
	return fmt.Sprintf("%s:%s:%d:%s",
		strings.ToUpper(method),
		NormalizeRoute(rawPath),
		status,
		strings.ToLower(errorType),
	)
}
