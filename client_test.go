package yily

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		client, err := NewClient(Config{
			BaseURL:       "http://localhost:16529",
			EnvironmentID: "env_test",
			Token:         "yily_test_token",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		if client.baseURL != "http://localhost:16529/api/v1" {
			t.Errorf("got %q, want %q", client.baseURL, "http://localhost:16529/api/v1")
		}
		if client.timeout != defaultTimeout {
			t.Errorf("got timeout %v, want %v", client.timeout, defaultTimeout)
		}
	})

	t.Run("custom HTTP client and timeout", func(t *testing.T) {
		customHTTP := &http.Client{Timeout: 10 * time.Second}
		client, err := NewClient(Config{
			BaseURL:       "http://localhost:16529",
			EnvironmentID: "env_test",
			Token:         "yily_test_token",
			HTTPClient:    customHTTP,
			Timeout:       10 * time.Second,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.httpClient != customHTTP {
			t.Error("expected custom HTTP client to be preserved")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		_, err := NewClient(Config{})
		if err == nil {
			t.Fatal("expected error for empty config")
		}
	})
}

func TestClientDo(t *testing.T) {
	t.Run("successful request with auth header and json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer yily_my_token" {
				t.Errorf("got Authorization %q, want %q", authHeader, "Bearer yily_my_token")
			}
			if r.Header.Get("Accept") != "application/json" {
				t.Errorf("got Accept %q, want application/json", r.Header.Get("Accept"))
			}

			if r.URL.Query().Get("key") != "val" {
				t.Errorf("got query param %q, want val", r.URL.Query().Get("key"))
			}

			resp := apiResponse[map[string]string]{
				Status:  http.StatusOK,
				Message: "success",
				Data:    map[string]string{"foo": "bar"},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := NewClient(Config{
			BaseURL:       server.URL,
			EnvironmentID: "env_1",
			Token:         "yily_my_token",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var target apiResponse[map[string]string]
		q := url.Values{}
		q.Set("key", "val")
		err = client.do(context.Background(), http.MethodGet, "/test", q, nil, &target)
		if err != nil {
			t.Fatalf("do() unexpected error: %v", err)
		}

		if target.Data["foo"] != "bar" {
			t.Errorf("got data foo = %q, want bar", target.Data["foo"])
		}
	})

	t.Run("401 unauthorized error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := apiResponse[any]{
				Status:  http.StatusUnauthorized,
				Code:    "AUTH_UNAUTHORIZED",
				Message: "invalid or expired token",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, _ := NewClient(Config{
			BaseURL:       server.URL,
			EnvironmentID: "env_1",
			Token:         "yily_bad_token",
		})

		err := client.do(context.Background(), http.MethodGet, "/test", nil, nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrUnauthorized) {
			t.Errorf("expected error to match ErrUnauthorized, got %v", err)
		}
	})

	t.Run("404 not found error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := apiResponse[any]{
				Status:  http.StatusNotFound,
				Code:    "RESOURCE_NOT_FOUND",
				Message: "resource does not exist",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, _ := NewClient(Config{
			BaseURL:       server.URL,
			EnvironmentID: "env_1",
			Token:         "yily_token",
		})

		err := client.do(context.Background(), http.MethodGet, "/test", nil, nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected error to match ErrNotFound, got %v", err)
		}
	})

	t.Run("context cancelled / timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := NewClient(Config{
			BaseURL:       server.URL,
			EnvironmentID: "env_1",
			Token:         "yily_token",
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := client.do(ctx, http.MethodGet, "/test", nil, nil, nil)
		if err == nil {
			t.Fatal("expected timeout/cancellation error, got nil")
		}
	})
}

func TestClientGet(t *testing.T) {
	t.Run("successfully retrieves variable value", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/environments/env_123/variables/values" {
				t.Errorf("got path %q, want expected environment path", r.URL.Path)
			}
			if r.URL.Query().Get("name") != "DB_HOST" {
				t.Errorf("got query name = %q, want DB_HOST", r.URL.Query().Get("name"))
			}

			resp := apiResponse[map[string]string]{
				Status:  http.StatusOK,
				Message: "Variable values retrieved successfully",
				Data: map[string]string{
					"DB_HOST": "localhost",
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
			t.Fatalf("unexpected error: %v", err)
		}

		val, err := client.Get(context.Background(), "DB_HOST")
		if err != nil {
			t.Fatalf("unexpected Get error: %v", err)
		}
		if val != "localhost" {
			t.Errorf("got %q, want %q", val, "localhost")
		}
	})

	t.Run("retrieves specific version", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("version") != "2" {
				t.Errorf("got query version = %q, want 2", r.URL.Query().Get("version"))
			}

			resp := apiResponse[map[string]string]{
				Status:  http.StatusOK,
				Message: "Variable values retrieved successfully",
				Data: map[string]string{
					"DB_HOST": "legacy-host",
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

		val, err := client.Get(context.Background(), "DB_HOST", Version(2))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "legacy-host" {
			t.Errorf("got %q, want legacy-host", val)
		}
	})

	t.Run("returns ErrVariableNotFound when variable is missing in response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := apiResponse[map[string]string]{
				Status:  http.StatusOK,
				Message: "Variable values retrieved successfully",
				Data:    map[string]string{},
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

		_, err := client.Get(context.Background(), "NON_EXISTING")
		if !errors.Is(err, ErrVariableNotFound) {
			t.Errorf("got error %v, want ErrVariableNotFound", err)
		}
	})

	t.Run("empty variable name returns ErrInvalidVariableName", func(t *testing.T) {
		client, _ := NewClient(Config{
			BaseURL:       "http://localhost:16529",
			EnvironmentID: "env_123",
			Token:         "yily_token",
		})

		_, err := client.Get(context.Background(), "   ")
		if !errors.Is(err, ErrInvalidVariableName) {
			t.Fatalf("expected ErrInvalidVariableName, got %v", err)
		}
	})

	t.Run("invalid version returns ErrInvalidVersion", func(t *testing.T) {
		client, _ := NewClient(Config{
			BaseURL:       "http://localhost:16529",
			EnvironmentID: "env_123",
			Token:         "yily_token",
		})

		_, err := client.Get(context.Background(), "DB_HOST", Version(0))
		if !errors.Is(err, ErrInvalidVersion) {
			t.Fatalf("expected ErrInvalidVersion, got %v", err)
		}

		_, err = client.Get(context.Background(), "DB_HOST", Version(-1))
		if !errors.Is(err, ErrInvalidVersion) {
			t.Fatalf("expected ErrInvalidVersion, got %v", err)
		}
	})
}
