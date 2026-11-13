package yily

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Defaults applied by NewClient and normalizeBaseURL.
const (
	defaultTimeout   = 5 * time.Second // Default per-request timeout.
	defaultAPIPrefix = "/api/v1"       // Base path of the Yily API.
)

// Config holds the options used by NewClient.
type Config struct {
	BaseURL       string        // Root URL of the Yily instance.
	EnvironmentID string        // Environment to read variables from.
	Token         string        // Read-scoped PAT prefixed with "yily_".
	HTTPClient    *http.Client  // Optional HTTP client; defaults to one using Timeout.
	Timeout       time.Duration // Per-request timeout; defaults to 5 seconds.
}

// validate checks that all required fields are set and that BaseURL is a safe
// absolute HTTP or HTTPS URL. It is called by NewClient.
func (c *Config) validate() error {
	baseURL := strings.TrimSpace(c.BaseURL)
	if baseURL == "" {
		return ErrMissingBaseURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return ErrInvalidBaseURL
	}
	if !parsedURL.IsAbs() || parsedURL.Host == "" || parsedURL.User != nil ||
		(!strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https")) ||
		parsedURL.RawQuery != "" || parsedURL.ForceQuery || parsedURL.Fragment != "" {
		return ErrInvalidBaseURL
	}

	token := strings.TrimSpace(c.Token)
	if token == "" {
		return ErrMissingToken
	}
	if !strings.HasPrefix(token, "yily_") || len(token) == len("yily_") {
		return ErrInvalidToken
	}

	if strings.TrimSpace(c.EnvironmentID) == "" {
		return ErrMissingEnvironmentID
	}

	return nil
}
