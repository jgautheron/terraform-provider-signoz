// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/jgautheron/terraform-provider-signoz/internal/client"
)

// configureResourceClient extracts the shared *client.Client placed in
// ProviderData by the provider's Configure. It tolerates a nil ProviderData
// (which happens during early validation walks) and reports a typed error
// otherwise.
func configureResourceClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData),
		)
		return nil
	}
	return c
}

// normalizedFromServer reconciles an opaque JSON blob attribute on Read.
//
// SigNoz echoes back blobs with server-added/computed fields (timestamps,
// defaults, reordered keys), so blindly storing the server's value would cause
// perpetual diffs against a config that is the source of truth. We therefore
// keep the user's existing config value when it is present, and only adopt the
// server's value when state is empty — i.e. during `terraform import`, where
// there is no prior config to preserve. Out-of-band edits are intentionally
// not surfaced for these opaque blobs (config-as-code is authoritative).
func normalizedFromServer(current jsontypes.Normalized, server json.RawMessage) jsontypes.Normalized {
	if current.IsNull() || current.IsUnknown() {
		return jsontypes.NewNormalizedValue(string(server))
	}
	return current
}
