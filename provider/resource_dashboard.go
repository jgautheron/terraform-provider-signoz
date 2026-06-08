// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jgautheron/terraform-provider-signoz/internal/client"
)

var (
	_ resource.Resource                = &dashboardResource{}
	_ resource.ResourceWithConfigure   = &dashboardResource{}
	_ resource.ResourceWithImportState = &dashboardResource{}
)

// NewDashboardResource is the resource factory.
func NewDashboardResource() resource.Resource { return &dashboardResource{} }

type dashboardResource struct {
	client *client.Client
}

// dashboardModel is the Terraform state model. The full dashboard definition
// is a single JSON blob (`data`) round-tripped verbatim — far simpler and more
// churn-resistant than modeling SigNoz's many nested dashboard fields.
type dashboardModel struct {
	ID   types.String         `tfsdk:"id"`
	Data jsontypes.Normalized `tfsdk:"data"`
}

func (r *dashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (r *dashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A SigNoz dashboard. The full dashboard definition is supplied as JSON in `data`; " +
			"export an existing dashboard from the SigNoz UI to bootstrap it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned dashboard UUID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"data": schema.StringAttribute{
				MarkdownDescription: "Dashboard definition as JSON (title, layout, widgets, variables, …). " +
					"Compared semantically, so key ordering and whitespace don't cause diffs.",
				CustomType: jsontypes.NormalizedType{},
				Required:   true,
			},
		},
	}
}

func (r *dashboardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *dashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateDashboard(ctx, json.RawMessage(plan.Data.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Creating SigNoz dashboard", err.Error())
		return
	}

	plan.ID = types.StringValue(created.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No id yet (e.g. a Crossplane/Upjet Observe before Create) means the
	// resource does not exist — don't issue a malformed GET .../dashboards/.
	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	got, err := r.client.GetDashboard(ctx, state.ID.ValueString())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading SigNoz dashboard", err.Error())
		return
	}

	// Preserve the user's `data` unless the server's differs semantically;
	// jsontypes.Normalized equality avoids spurious diffs from reserialization.
	state.Data = normalizedFromServer(state.Data, got.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dashboardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.UpdateDashboard(ctx, plan.ID.ValueString(), json.RawMessage(plan.Data.ValueString())); err != nil {
		resp.Diagnostics.AddError("Updating SigNoz dashboard", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dashboardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDashboard(ctx, state.ID.ValueString()); err != nil {
		if client.NotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Deleting SigNoz dashboard", err.Error())
	}
}

func (r *dashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
