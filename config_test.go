package yily

import (
	"errors"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid config",
			config: Config{
				BaseURL:       "http://localhost:16529",
				Token:         "yily_test_token",
				EnvironmentID: "env_123",
			},
			wantErr: nil,
		},
		{
			name: "missing base URL",
			config: Config{
				Token:         "yily_test_token",
				EnvironmentID: "env_123",
			},
			wantErr: ErrMissingBaseURL,
		},
		{
			name: "invalid base URL (relative)",
			config: Config{
				BaseURL:       "/relative/path",
				Token:         "yily_test_token",
				EnvironmentID: "env_123",
			},
			wantErr: ErrInvalidBaseURL,
		},
		{
			name: "invalid base URL (ftp scheme)",
			config: Config{
				BaseURL:       "ftp://localhost:16529",
				Token:         "yily_test_token",
				EnvironmentID: "env_123",
			},
			wantErr: ErrInvalidBaseURL,
		},
		{
			name: "missing token",
			config: Config{
				BaseURL:       "http://localhost:16529",
				EnvironmentID: "env_123",
			},
			wantErr: ErrMissingToken,
		},
		{
			name: "invalid token prefix",
			config: Config{
				BaseURL:       "http://localhost:16529",
				Token:         "invalid_prefix",
				EnvironmentID: "env_123",
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "token with only prefix",
			config: Config{
				BaseURL:       "http://localhost:16529",
				Token:         "yily_",
				EnvironmentID: "env_123",
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "missing environment ID",
			config: Config{
				BaseURL: "http://localhost:16529",
				Token:   "yily_test_token",
			},
			wantErr: ErrMissingEnvironmentID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "http://localhost:16529",
			expected: "http://localhost:16529/api/v1",
		},
		{
			input:    "http://localhost:16529/",
			expected: "http://localhost:16529/api/v1",
		},
		{
			input:    "http://localhost:16529/api/v1",
			expected: "http://localhost:16529/api/v1",
		},
		{
			input:    "http://localhost:16529/api/v1/",
			expected: "http://localhost:16529/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeBaseURL(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeBaseURL(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeVariableName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"db_host", "DB_HOST"},
		{"DB_HOST", "DB_HOST"},
		{"  db host  ", "DB_HOST"},
		{"db host", "DB_HOST"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeVariableName(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeVariableName(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAPIErrorFormat(t *testing.T) {
	errWithCode := &APIError{
		StatusCode: 400,
		Code:       "INVALID_REQUEST",
		Message:    "bad payload",
	}
	expectedWithCode := "yily: api error (status 400, code INVALID_REQUEST): bad payload"
	if errWithCode.Error() != expectedWithCode {
		t.Errorf("got %q, expected %q", errWithCode.Error(), expectedWithCode)
	}

	errWithoutCode := &APIError{
		StatusCode: 500,
		Message:    "server failure",
	}
	expectedWithoutCode := "yily: api error (status 500): server failure"
	if errWithoutCode.Error() != expectedWithoutCode {
		t.Errorf("got %q, expected %q", errWithoutCode.Error(), expectedWithoutCode)
	}
}
