package yily

import "strings"

// normalizeBaseURL ensures the base URL is trimmed and properly formatted with the /api/v1 prefix.
// It removes any trailing slashes and the default API prefix if present, then appends the default API prefix.
func normalizeBaseURL(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	raw = strings.TrimSuffix(raw, defaultAPIPrefix)
	return raw + defaultAPIPrefix
}

// normalizeVariableName matches the server's variable naming convention.
// It converts the name to uppercase, replaces spaces with underscores, and trims whitespace.
func normalizeVariableName(name string) string {
	return strings.Join(strings.Fields(strings.ToUpper(name)), "_")
}
