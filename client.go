package yily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client reads configuration variables from a Yily environment.
//
// Create a Client with NewClient. A Client is safe for concurrent use by
// multiple goroutines.
type Client struct {
	baseURL       string
	environmentID string
	token         string
	httpClient    *http.Client
	timeout       time.Duration
}

// NewClient validates cfg and returns a Client for the configured environment.
//
// If cfg.HTTPClient is nil, NewClient creates an http.Client using cfg.Timeout
// (5 seconds when Timeout is zero or negative).
func NewClient(cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	return &Client{
		baseURL:       normalizeBaseURL(cfg.BaseURL),
		environmentID: strings.TrimSpace(cfg.EnvironmentID),
		token:         strings.TrimSpace(cfg.Token),
		httpClient:    httpClient,
		timeout:       timeout,
	}, nil
}

// Load populates target with configuration variables retrieved from Yily.
//
// Target must be a non-nil pointer to a struct. Exported fields are loaded when
// they carry a `yily` tag:
//
//	type AppConfig struct {
//		Host string `yily:"DB_HOST"`
//		Port int    `yily:"DB_PORT"`
//	}
//
// Tagged fields are fetched in one batch request. Values are assigned only
// after every retrieval and conversion has succeeded, so target is left
// unchanged on error. Supported types are string, int, int64, float64, bool
// and time.Duration.
func (c *Client) Load(ctx context.Context, target any) error {
	fields, err := inspectStruct(target)
	if err != nil {
		return err
	}

	if len(fields) == 0 {
		return nil
	}

	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.varName)
	}

	values, err := c.fetchBatchValues(ctx, names)
	if err != nil {
		return err
	}

	return decodeFields(fields, values)
}

// Get returns the value of a single configuration variable.
//
// The name is normalized to Yily's convention (upper case, whitespace replaced
// by underscores). Get returns ErrInvalidVariableName when the normalized name
// is empty and ErrVariableNotFound when the variable does not exist.
func (c *Client) Get(ctx context.Context, name string) (string, error) {
	name = normalizeVariableName(name)
	if name == "" {
		return "", ErrInvalidVariableName
	}

	values, err := c.fetchBatchValues(ctx, []string{name})
	if err != nil {
		return "", err
	}

	value, ok := values[name]
	if !ok {
		return "", ErrVariableNotFound
	}

	return value, nil
}

// GetAll returns every variable of the client's environment as a map of
// variable name to active value.
//
// GetAll performs two requests: it lists the variable names, then reads their
// values in one batch. Changes made between those requests may affect the
// result, so the map is not an atomic snapshot.
func (c *Client) GetAll(ctx context.Context) (map[string]string, error) {
	path := fmt.Sprintf("/environments/%s/variables", c.environmentID)

	var resp apiResponse[[]struct {
		Name string `json:"name"`
	}]
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return make(map[string]string), nil
	}

	names := make([]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		names = append(names, item.Name)
	}

	return c.fetchBatchValues(ctx, names)
}

// ------------------ HTTP request helpers ------------------

// apiResponse is Yily's standard JSON response envelope.
type apiResponse[T any] struct {
	Status  int    `json:"status"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// batchValuesRequest is the JSON body for the batch variable values endpoint.
type batchValuesRequest struct {
	Variables []batchVariable `json:"variables"`
}

// batchVariable selects one variable. An omitted version requests its active
// version.
type batchVariable struct {
	Name string `json:"name"`
}

// fetchBatchValues retrieves the active values of names from the batch endpoint.
//
// Names are normalized and deduplicated before being sent. Unknown names are
// omitted by the server and therefore absent from the returned map.
func (c *Client) fetchBatchValues(ctx context.Context, names []string) (map[string]string, error) {
	if len(names) == 0 {
		return make(map[string]string), nil
	}

	requested := make(map[string]struct{}, len(names))
	selections := make([]batchVariable, 0, len(names))
	for _, name := range names {
		name = normalizeVariableName(name)
		if name == "" {
			continue
		}
		if _, exists := requested[name]; exists {
			continue
		}
		requested[name] = struct{}{}
		selections = append(selections, batchVariable{Name: name})
	}

	if len(selections) == 0 {
		return make(map[string]string), nil
	}

	body, err := json.Marshal(batchValuesRequest{Variables: selections})
	if err != nil {
		return nil, fmt.Errorf("yily: failed to encode request: %w", err)
	}

	path := fmt.Sprintf("/environments/%s/variables/values", c.environmentID)

	var resp apiResponse[map[string]string]
	if err := c.do(ctx, http.MethodPost, path, nil, bytes.NewReader(body), &resp); err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return make(map[string]string), nil
	}

	return resp.Data, nil
}

// do sends an HTTP request, adds the Bearer token, applies the client timeout,
// and decodes the response envelope into out.
// Non-2xx responses are converted to ErrUnauthorized, ErrNotFound, or an *APIError.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader, out any) error {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return fmt.Errorf("yily: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("yily: request failed: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("yily: failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errEnv apiResponse[any]

		if err := json.Unmarshal(respBody, &errEnv); err == nil && (errEnv.Message != "" || errEnv.Code != "") {
			apiErr := &APIError{
				StatusCode: resp.StatusCode,
				Code:       errEnv.Code,
				Message:    errEnv.Message,
			}

			switch resp.StatusCode {
			case http.StatusUnauthorized:
				return fmt.Errorf("%w: %w", ErrUnauthorized, apiErr)
			case http.StatusNotFound:
				return fmt.Errorf("%w: %w", ErrNotFound, apiErr)
			default:
				return apiErr
			}
		}

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return ErrUnauthorized
		case http.StatusNotFound:
			return ErrNotFound
		default:
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    http.StatusText(resp.StatusCode),
			}
		}
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("yily: failed to decode response: %w", err)
		}
	}

	return nil
}
