// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// Alert rules use mixed API versions: writes go to /api/v2/rules, reads to
// /api/v1/rules/{id}. The rule body is assembled by the resource layer (typed
// top-level fields + JSON-blob condition/evaluation/notification_settings) and
// passed through here as opaque JSON.

// alertEnvelope wraps SigNoz's {"status","data"} response. The data object
// carries the rule including its server-assigned id.
type alertEnvelope struct {
	Data json.RawMessage `json:"data"`
}

// CreateAlert POSTs a new alert rule and returns the created rule JSON
// (including its id) so the resource can extract the id and refresh state.
func (c *Client) CreateAlert(ctx context.Context, rule json.RawMessage) (json.RawMessage, error) {
	var env alertEnvelope
	if err := c.do(ctx, http.MethodPost, "/api/v2/rules", rule, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// GetAlert fetches an alert rule by id.
func (c *Client) GetAlert(ctx context.Context, id string) (json.RawMessage, error) {
	var env alertEnvelope
	if err := c.do(ctx, http.MethodGet, "/api/v1/rules/"+id, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// UpdateAlert replaces an alert rule by id.
func (c *Client) UpdateAlert(ctx context.Context, id string, rule json.RawMessage) (json.RawMessage, error) {
	var env alertEnvelope
	if err := c.do(ctx, http.MethodPut, "/api/v2/rules/"+id, rule, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// DeleteAlert removes an alert rule by id.
func (c *Client) DeleteAlert(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v2/rules/"+id, nil, nil)
}
