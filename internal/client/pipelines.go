// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// Log pipelines on /api/v1/logs/pipelines are managed as a VERSIONED SET, not
// as individually-addressable resources: a POST publishes a new version
// containing the full ordered list of pipelines, and GET returns the latest
// version. There is no per-pipeline id-based create/delete. The Terraform
// resource therefore models the whole managed set (see resource_log_pipeline).
//
// NOTE: this endpoint's exact shape (latest-version path, payload key) is
// confirmed against a live SigNoz instance during acceptance testing; if the
// API proves to lack usable set-replacement semantics, the resource ships
// disabled with a documented gap rather than guessing.

type pipelineEnvelope struct {
	Data json.RawMessage `json:"data"`
}

// GetPipelines returns the latest published pipeline set.
func (c *Client) GetPipelines(ctx context.Context) (json.RawMessage, error) {
	var env pipelineEnvelope
	if err := c.do(ctx, http.MethodGet, "/api/v1/logs/pipelines/latest", nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// SetPipelines publishes a new pipeline-set version. payload is the full
// {"pipelines":[...]} body assembled by the resource layer.
func (c *Client) SetPipelines(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var env pipelineEnvelope
	if err := c.do(ctx, http.MethodPost, "/api/v1/logs/pipelines", payload, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}
