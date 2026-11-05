package yily

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type FullConfig struct {
	Host     string  `yily:"DB_HOST"`
	Port     int     `yily:"DB_PORT"`
	MaxConns int64   `yily:"MAX_CONNS"`
	Timeout  float64 `yily:"TIMEOUT"`
	Debug    bool    `yily:"DEBUG"`
	Optional string  `yily:"EMPTY_VAL"`
	Ignored  string  `yily:"-"`
	Untagged string
}

func TestClientLoad(t *testing.T) {
	t.Run("successfully loads all supported types and empty values", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/environments/env_123/variables/values" {
				t.Errorf("got path %q, want expected environment path", r.URL.Path)
			}

			names := r.URL.Query()["name"]
			if len(names) != 6 {
				t.Errorf("expected 6 variable names queried, got %d", len(names))
			}

			resp := apiResponse[map[string]string]{
				Status:  http.StatusOK,
				Message: "Variable values retrieved successfully",
				Data: map[string]string{
					"DB_HOST":   "127.0.0.1",
					"DB_PORT":   "5432",
					"MAX_CONNS": "100",
					"TIMEOUT":   "2.5",
					"DEBUG":     "true",
					"EMPTY_VAL": "",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := NewClient(Config{
			BaseURL:       server.URL,
			EnvironmentID: "env_123",
			Token:         "yily_token",
		})
		if err != nil {
			t.Fatalf("unexpected NewClient error: %v", err)
		}

		var cfg FullConfig
		cfg.Ignored = "original_ignored"
		cfg.Untagged = "original_untagged"

		err = client.Load(context.Background(), &cfg)
		if err != nil {
			t.Fatalf("unexpected Load error: %v", err)
		}

		if cfg.Host != "127.0.0.1" {
			t.Errorf("got Host %q, want 127.0.0.1", cfg.Host)
		}
		if cfg.Port != 5432 {
			t.Errorf("got Port %d, want 5432", cfg.Port)
		}
		if cfg.MaxConns != 100 {
			t.Errorf("got MaxConns %d, want 100", cfg.MaxConns)
		}
		if cfg.Timeout != 2.5 {
			t.Errorf("got Timeout %f, want 2.5", cfg.Timeout)
		}
		if !cfg.Debug {
			t.Errorf("got Debug false, want true")
		}
		if cfg.Optional != "" {
			t.Errorf("got Optional %q, want empty string", cfg.Optional)
		}
		if cfg.Ignored != "original_ignored" {
			t.Errorf("got Ignored %q, want original_ignored", cfg.Ignored)
		}
		if cfg.Untagged != "original_untagged" {
			t.Errorf("got Untagged %q, want original_untagged", cfg.Untagged)
		}
	})

	t.Run("missing variable returns ErrVariableNotFound", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := apiResponse[map[string]string]{
				Status:  http.StatusOK,
				Message: "Variable values retrieved successfully",
				Data: map[string]string{
					"DB_HOST": "127.0.0.1",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, _ := NewClient(Config{
			BaseURL:       server.URL,
			EnvironmentID: "env_123",
			Token:         "yily_token",
		})

		type SmallConfig struct {
			Host string `yily:"DB_HOST"`
			Port int    `yily:"DB_PORT"`
		}

		var cfg SmallConfig
		err := client.Load(context.Background(), &cfg)
		if !errors.Is(err, ErrVariableNotFound) {
			t.Errorf("expected ErrVariableNotFound, got %v", err)
		}
	})

	t.Run("conversion failure leaves struct fields unchanged", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := apiResponse[map[string]string]{
				Status:  http.StatusOK,
				Message: "Variable values retrieved successfully",
				Data: map[string]string{
					"DB_HOST": "new_host",
					"DB_PORT": "not_a_valid_number",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, _ := NewClient(Config{
			BaseURL:       server.URL,
			EnvironmentID: "env_123",
			Token:         "yily_token",
		})

		type SmallConfig struct {
			Host string `yily:"DB_HOST"`
			Port int    `yily:"DB_PORT"`
		}

		cfg := SmallConfig{
			Host: "initial_host",
			Port: 8080,
		}

		err := client.Load(context.Background(), &cfg)
		if err == nil {
			t.Fatal("expected conversion error, got nil")
		}

		if cfg.Host != "initial_host" {
			t.Errorf("Host was modified to %q despite conversion error; wanted initial_host", cfg.Host)
		}
		if cfg.Port != 8080 {
			t.Errorf("Port was modified to %d, wanted 8080", cfg.Port)
		}
	})

	t.Run("unsupported struct field type returns ErrUnsupportedType", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := apiResponse[map[string]string]{
				Status:  http.StatusOK,
				Message: "Variable values retrieved successfully",
				Data: map[string]string{
					"DATA": "raw",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, _ := NewClient(Config{
			BaseURL:       server.URL,
			EnvironmentID: "env_123",
			Token:         "yily_token",
		})

		type UnsupportedConfig struct {
			Data []byte `yily:"DATA"`
		}

		var cfg UnsupportedConfig
		err := client.Load(context.Background(), &cfg)
		if !errors.Is(err, ErrUnsupportedType) {
			t.Errorf("expected ErrUnsupportedType, got %v", err)
		}
	})

	t.Run("invalid target returns ErrInvalidTarget", func(t *testing.T) {
		client, _ := NewClient(Config{
			BaseURL:       "http://localhost:16529",
			EnvironmentID: "env_123",
			Token:         "yily_token",
		})

		err := client.Load(context.Background(), FullConfig{})
		if !errors.Is(err, ErrInvalidTarget) {
			t.Errorf("expected ErrInvalidTarget for non-pointer, got %v", err)
		}

		var nilPtr *FullConfig
		err = client.Load(context.Background(), nilPtr)
		if !errors.Is(err, ErrInvalidTarget) {
			t.Errorf("expected ErrInvalidTarget for nil pointer, got %v", err)
		}

		num := 42
		err = client.Load(context.Background(), &num)
		if !errors.Is(err, ErrInvalidTarget) {
			t.Errorf("expected ErrInvalidTarget for pointer to non-struct, got %v", err)
		}
	})

	t.Run("tag version parsing", func(t *testing.T) {
		name, ver, err := parseTag("API_KEY,version=3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "API_KEY" || ver != 3 {
			t.Errorf("got name %q, ver %d, want API_KEY and 3", name, ver)
		}

		_, _, err = parseTag("API_KEY,version=abc")
		if err == nil {
			t.Fatal("expected error for invalid version, got nil")
		}

		_, _, err = parseTag("")
		if err == nil {
			t.Fatal("expected error for empty tag, got nil")
		}
	})
}
