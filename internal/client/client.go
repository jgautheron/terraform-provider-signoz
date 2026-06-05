// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

// Package client is a thin HTTP client for the SigNoz REST API.
//
// SigNoz mixes API versions across resources (e.g. dashboards on /api/v1,
// alert-rule writes on /api/v2), so each resource file owns its exact paths;
// this file only handles transport, auth, and error decoding.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultEndpoint is SigNoz's local query-service address.
	DefaultEndpoint = "http://localhost:3301"
	// DefaultTimeout for a single HTTP call.
	DefaultTimeout = 35 * time.Second
	// DefaultMaxRetry for transient (5xx / network) failures.
	DefaultMaxRetry = 3

	// authHeader is the SigNoz access-token header (Service Account key).
	authHeader = "SIGNOZ-API-KEY"
)

// Client talks to a single SigNoz instance.
type Client struct {
	httpClient  *http.Client
	endpoint    string // normalized, no trailing slash
	accessToken string
	maxRetry    int
	userAgent   string
}

// Config configures a Client.
type Config struct {
	Endpoint    string
	AccessToken string
	Timeout     time.Duration
	MaxRetry    int
	UserAgent   string
}

// New builds a Client, applying defaults for unset fields.
func New(cfg Config) *Client {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	endpoint = strings.TrimRight(endpoint, "/")

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	maxRetry := cfg.MaxRetry
	if maxRetry < 0 {
		maxRetry = 0
	} else if maxRetry == 0 {
		maxRetry = DefaultMaxRetry
	}
	ua := cfg.UserAgent
	if ua == "" {
		ua = "terraform-provider-signoz"
	}

	return &Client{
		httpClient:  &http.Client{Timeout: timeout},
		endpoint:    endpoint,
		accessToken: cfg.AccessToken,
		maxRetry:    maxRetry,
		userAgent:   ua,
	}
}

// APIError is a non-2xx response from SigNoz. SigNoz error bodies are usually
// {"status":"error","errorType":"...","error":"..."}; we surface the message
// when present and fall back to the raw body otherwise.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("signoz API error: status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("signoz API error: status %d: %s", e.StatusCode, e.Body)
}

// NotFound reports whether err is a 404 APIError (used by Read to drop state).
func NotFound(err error) bool {
	var apiErr *APIError
	if e, ok := err.(*APIError); ok {
		apiErr = e
	}
	return apiErr != nil && apiErr.StatusCode == http.StatusNotFound
}

// do issues an HTTP request. body is JSON-marshaled when non-nil; out, when
// non-nil, receives the decoded JSON response. Retries transient failures.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
	}

	url := c.endpoint + path

	var lastErr error
	for attempt := 0; attempt <= c.maxRetry; attempt++ {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if c.accessToken != "" {
			req.Header.Set(authHeader, c.accessToken)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%s %s: %w", method, path, err)
			continue // network error → retry
		}

		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = decodeAPIError(resp.StatusCode, respBody)
			continue // server error → retry
		}
		if resp.StatusCode >= 400 {
			return decodeAPIError(resp.StatusCode, respBody) // client error → no retry
		}

		if out != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decoding %s %s response: %w", method, path, err)
			}
		}
		return nil
	}
	return lastErr
}

func decodeAPIError(status int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: status, Body: string(body)}
	var parsed struct {
		Status    string `json:"status"`
		ErrorType string `json:"errorType"`
		// SigNoz uses both a string `error` and, on newer endpoints, an
		// object {code,message}; handle both via RawMessage.
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		apiErr.Type = parsed.ErrorType
		if len(parsed.Error) > 0 {
			var msg string
			if json.Unmarshal(parsed.Error, &msg) == nil {
				apiErr.Message = msg
			} else {
				var obj struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}
				if json.Unmarshal(parsed.Error, &obj) == nil && obj.Message != "" {
					apiErr.Message = obj.Message
				}
			}
		}
	}
	return apiErr
}
