package yily

import (
	"errors"
	"fmt"
)

// Configuration, validation, and loading errors.
var (
	ErrInvalidBaseURL             = errors.New("yily: base URL must be an absolute HTTP or HTTPS URL")         // ErrInvalidBaseURL is returned when the base URL format is invalid
	ErrInvalidToken               = errors.New("yily: token must be a Yily PAT")                               // ErrInvalidToken is returned when the token is not a valid Yily PAT
	ErrInvalidValue               = errors.New("yily: variable value has an invalid format")                   // ErrInvalidValue is returned when a variable value cannot be processed
	ErrInvalidVariableName        = errors.New("yily: variable name cannot be empty")                          // ErrInvalidVariableName is returned when the variable name is empty
	ErrInvalidTag                 = errors.New("yily: invalid yily struct tag")                                // ErrInvalidTag is returned when a yily struct tag is malformed
	ErrConflictingVariableVersion = errors.New("yily: a variable cannot use different versions in one struct") // ErrConflictingVariableVersion is returned when fields request different versions of the same variable
	ErrMissingBaseURL             = errors.New("yily: base URL is required")                                   // ErrMissingBaseURL is returned when the base URL is missing
	ErrMissingToken               = errors.New("yily: token is required")                                      // ErrMissingToken is returned when PAT is missing
	ErrMissingEnvironmentID       = errors.New("yily: environment ID is required")                             // ErrMissingEnvironmentID is returned when the environment ID is missing
	ErrInvalidTarget              = errors.New("yily: target must be a non-nil pointer to a struct")           // ErrInvalidTarget is returned when Load receives anything other than a non-nil pointer to a struct
	ErrVariableNotFound           = errors.New("yily: variable not found")                                     // ErrVariableNotFound is returned when requested variable does not exist
	ErrUnsupportedType            = errors.New("yily: unsupported field type")                                 // ErrUnsupportedType is returned when struct field type is not supported for decoding
)

// API-level errors returned by the HTTP transport.
var (
	ErrUnauthorized = errors.New("yily: unauthorized") // ErrUnauthorized indicates an authentication failure (HTTP 401)
	ErrNotFound     = errors.New("yily: not found")    // ErrNotFound indicates the requested resource was not found (HTTP 404)
)

// APIError represents an error response returned by the Yily API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("yily: api error (status %d, code %s): %s", e.StatusCode, e.Code, e.Message)
	}

	return fmt.Sprintf("yily: api error (status %d): %s", e.StatusCode, e.Message)
}
