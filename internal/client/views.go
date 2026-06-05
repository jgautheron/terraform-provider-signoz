// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// Saved (explorer) views live on /api/v1/explorer/views. The view body
// (name, sourcePage, compositeQuery, …) is passed through as opaque JSON.
//
// NOTE: exact envelope confirmed against a live SigNoz instance during
// acceptance testing.

type viewEnvelope struct {
	Data json.RawMessage `json:"data"`
}

// CreateView POSTs a new saved view.
func (c *Client) CreateView(ctx context.Context, view json.RawMessage) (json.RawMessage, error) {
	var env viewEnvelope
	if err := c.do(ctx, http.MethodPost, "/api/v1/explorer/views", view, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// GetView fetches a saved view by id.
func (c *Client) GetView(ctx context.Context, id string) (json.RawMessage, error) {
	var env viewEnvelope
	if err := c.do(ctx, http.MethodGet, "/api/v1/explorer/views/"+id, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// UpdateView replaces a saved view by id.
func (c *Client) UpdateView(ctx context.Context, id string, view json.RawMessage) (json.RawMessage, error) {
	var env viewEnvelope
	if err := c.do(ctx, http.MethodPut, "/api/v1/explorer/views/"+id, view, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// DeleteView removes a saved view by id.
func (c *Client) DeleteView(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/explorer/views/"+id, nil, nil)
}
