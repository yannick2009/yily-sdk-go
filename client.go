package yily

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is the Yily SDK client for retrieving configuration variables.
type Client struct {
	baseURL       string
	environmentID string
	token         string
	httpClient    *http.Client
	timeout       time.Duration
}

// apiResponse represents Yily's standard JSON response envelope.
type apiResponse[T any] struct {
	Status  int    `json:"status"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// NewClient initializes and returns a new Yily SDK Client.
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

// do sends an HTTP request, injects the Bearer authorization header,
// and decodes the response body into the provided envelope target.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader, out any) error {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	reqCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(reqCtx, method, reqURL, body)

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

// GetOption configures an individual variable retrieval.
type GetOption func(*getOptions)

type getOptions struct {
	version    int
	versionSet bool
}

// Version specifies a specific variable version number to retrieve.
func Version(v int) GetOption {
	return func(o *getOptions) {
		o.version = v
		o.versionSet = true
	}
}

// Get retrieves the value of an individual configuration variable by name.
// By default, it retrieves the active version. Use yily.Version(n) to request a specific version.
func (c *Client) Get(ctx context.Context, name string, opts ...GetOption) (string, error) {
	name = normalizeVariableName(name)
	if name == "" {
		return "", ErrInvalidVariableName
	}

	var options getOptions
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&options)
	}
	if options.versionSet && options.version <= 0 {
		return "", ErrInvalidVersion
	}

	values, err := c.fetchBatchValues(ctx, []string{name}, options.version)
	if err != nil {
		return "", err
	}

	value, ok := values[name]
	if !ok {
		return "", ErrVariableNotFound
	}

	return value, nil
}

// GetAll returns a map of variable names to their active values for the client's
// environment. It first lists the variables, then fetches their values in one batch
// request; changes made between these requests may affect the result.
func (c *Client) GetAll(ctx context.Context) (map[string]string, error) {
	path := fmt.Sprintf("/environments/%s/variables", url.PathEscape(c.environmentID))

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

	return c.fetchBatchValues(ctx, names, 0)
}
