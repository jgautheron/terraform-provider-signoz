// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// Dashboard is the SigNoz dashboard envelope. The `data` field carries the
// full dashboard definition (title, layout, widgets, variables, …) as opaque
// JSON — we round-trip it verbatim rather than modeling every nested field.
type Dashboard struct {
	UUID string          `json:"id"`
	Data json.RawMessage `json:"data"`
}

// dashboardEnvelope is SigNoz's standard {"status","data"} response wrapper.
type dashboardEnvelope struct {
	Data Dashboard `json:"data"`
}

// CreateDashboard POSTs a new dashboard. data is the dashboard definition JSON.
func (c *Client) CreateDashboard(ctx context.Context, data json.RawMessage) (*Dashboard, error) {
	body := map[string]json.RawMessage{"data": data}
	var env dashboardEnvelope
	if err := c.do(ctx, http.MethodPost, "/api/v1/dashboards", body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// GetDashboard fetches a dashboard by UUID.
func (c *Client) GetDashboard(ctx context.Context, uuid string) (*Dashboard, error) {
	var env dashboardEnvelope
	if err := c.do(ctx, http.MethodGet, "/api/v1/dashboards/"+uuid, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// UpdateDashboard replaces a dashboard's definition.
func (c *Client) UpdateDashboard(ctx context.Context, uuid string, data json.RawMessage) (*Dashboard, error) {
	body := map[string]json.RawMessage{"data": data}
	var env dashboardEnvelope
	if err := c.do(ctx, http.MethodPut, "/api/v1/dashboards/"+uuid, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// DeleteDashboard removes a dashboard by UUID.
func (c *Client) DeleteDashboard(ctx context.Context, uuid string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/dashboards/"+uuid, nil, nil)
}
