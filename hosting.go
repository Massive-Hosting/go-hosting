// Package hosting provides a Go client for the Hosting Platform control panel API.
// It authenticates via Personal Access Tokens (PATs) and handles JSON encoding,
// error responses, list envelopes, and async resource polling.
package hosting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client wraps the control panel API with PAT authentication.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// New creates a new API client.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListResponse is the standard list envelope returned by all list endpoints.
type ListResponse[T any] struct {
	Items   []T  `json:"items"`
	HasMore bool `json:"has_more"`
}

// ErrorResponse is an API error.
type ErrorResponse struct {
	StatusCode int
	Message    string `json:"message"`
	Detail     string `json:"detail"`
}

func (e *ErrorResponse) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("API error %d: %s — %s", e.StatusCode, e.Message, e.Detail)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// Do performs an HTTP request with auth headers and JSON body encoding.
// It returns the raw response; callers are responsible for closing the body.
// On 4xx/5xx responses, it returns an *ErrorResponse.
func (c *Client) Do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	u := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var apiErr ErrorResponse
		apiErr.StatusCode = resp.StatusCode
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, &ErrorResponse{StatusCode: resp.StatusCode, Message: "unknown error"}
		}
		return nil, &apiErr
	}

	return resp, nil
}

// Get performs a GET request and decodes the JSON response into T.
func Get[T any](ctx context.Context, c *Client, path string) (*T, error) {
	resp, err := c.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// List performs a GET request and unwraps the standard list envelope.
func List[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	result, err := Get[ListResponse[T]](ctx, c, path)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// Post performs a POST request and decodes the JSON response into T.
func Post[T any](ctx context.Context, c *Client, path string, body any) (*T, error) {
	resp, err := c.Do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// Put performs a PUT request and decodes the JSON response into T.
func Put[T any](ctx context.Context, c *Client, path string, body any) (*T, error) {
	resp, err := c.Do(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// Patch performs a PATCH request and decodes the JSON response into T.
func Patch[T any](ctx context.Context, c *Client, path string, body any) (*T, error) {
	resp, err := c.Do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// Delete performs a DELETE request (no response body expected).
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.Do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// IsNotFound returns true if the error is a 404 API response.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*ErrorResponse); ok {
		return apiErr.StatusCode == 404
	}
	return false
}

// StatusCode extracts the HTTP status code from an error, or 0 if not an API error.
func StatusCode(err error) int {
	if apiErr, ok := err.(*ErrorResponse); ok {
		return apiErr.StatusCode
	}
	return 0
}

// WaitForStatus polls a resource until it reaches an active/running status or fails.
func WaitForStatus[T any](ctx context.Context, c *Client, path string, getStatus func(*T) string, timeout time.Duration) (*T, error) {
	deadline := time.Now().Add(timeout)
	interval := 2 * time.Second

	for {
		result, err := Get[T](ctx, c, path)
		if err != nil {
			return nil, err
		}

		status := getStatus(result)
		switch status {
		case "active", "running":
			return result, nil
		case "error", "failed":
			return result, fmt.Errorf("resource entered %s state", status)
		}

		if time.Now().After(deadline) {
			return result, fmt.Errorf("timed out waiting for resource (current status: %s)", status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// QueryEscape escapes a string for use in URL query parameters.
func QueryEscape(s string) string {
	return url.QueryEscape(s)
}
