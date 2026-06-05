// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecodeAPIError_StringError(t *testing.T) {
	body := []byte(`{"status":"error","errorType":"bad_data","error":"version: only v5 is supported, got \"v4\""}`)
	e := decodeAPIError(http.StatusBadRequest, body)
	if e.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", e.StatusCode)
	}
	if e.Type != "bad_data" {
		t.Fatalf("type: got %q", e.Type)
	}
	if e.Message != `version: only v5 is supported, got "v4"` {
		t.Fatalf("message: got %q", e.Message)
	}
}

func TestDecodeAPIError_ObjectError(t *testing.T) {
	body := []byte(`{"status":"error","error":{"code":"authz_forbidden","message":"only admins can access this resource"}}`)
	e := decodeAPIError(http.StatusForbidden, body)
	if e.Message != "only admins can access this resource" {
		t.Fatalf("message: got %q", e.Message)
	}
}

func TestNotFound(t *testing.T) {
	if !NotFound(&APIError{StatusCode: http.StatusNotFound}) {
		t.Fatal("expected NotFound true for 404")
	}
	if NotFound(&APIError{StatusCode: http.StatusInternalServerError}) {
		t.Fatal("expected NotFound false for 500")
	}
	if NotFound(nil) {
		t.Fatal("expected NotFound false for nil")
	}
}

func TestDo_SetsAuthHeaderAndDecodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(authHeader); got != "tok123" {
			t.Errorf("auth header: got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"uuid": "abc"}})
	}))
	defer srv.Close()

	c := New(Config{Endpoint: srv.URL, AccessToken: "tok123"})
	d, err := c.CreateDashboard(context.Background(), json.RawMessage(`{"title":"t"}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if d.UUID != "abc" {
		t.Fatalf("uuid: got %q", d.UUID)
	}
}

func TestDo_RetriesOn5xx(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"uuid": "ok"}})
	}))
	defer srv.Close()

	c := New(Config{Endpoint: srv.URL, MaxRetry: 3})
	if _, err := c.GetDashboard(context.Background(), "x"); err != nil {
		t.Fatalf("unexpected err after retries: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls (2 retries), got %d", calls)
	}
}
