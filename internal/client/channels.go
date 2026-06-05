// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// Notification channels live on /api/v1/channels. Write operations are
// admin-only in SigNoz. The channel body (type + type-specific config) is
// passed through as opaque JSON assembled by the resource layer.
//
// NOTE: exact request/response envelope confirmed against a live SigNoz
// instance during provider acceptance testing.

type channelEnvelope struct {
	Data json.RawMessage `json:"data"`
}

// CreateChannel POSTs a new notification channel.
func (c *Client) CreateChannel(ctx context.Context, channel json.RawMessage) (json.RawMessage, error) {
	var env channelEnvelope
	if err := c.do(ctx, http.MethodPost, "/api/v1/channels", channel, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// GetChannel fetches a notification channel by id.
func (c *Client) GetChannel(ctx context.Context, id string) (json.RawMessage, error) {
	var env channelEnvelope
	if err := c.do(ctx, http.MethodGet, "/api/v1/channels/"+id, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// UpdateChannel replaces a notification channel by id.
func (c *Client) UpdateChannel(ctx context.Context, id string, channel json.RawMessage) (json.RawMessage, error) {
	var env channelEnvelope
	if err := c.do(ctx, http.MethodPut, "/api/v1/channels/"+id, channel, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// DeleteChannel removes a notification channel by id.
func (c *Client) DeleteChannel(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/channels/"+id, nil, nil)
}
