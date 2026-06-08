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
	_ resource.Resource                = &savedViewResource{}
	_ resource.ResourceWithConfigure   = &savedViewResource{}
	_ resource.ResourceWithImportState = &savedViewResource{}
)

// NewSavedViewResource is the resource factory.
func NewSavedViewResource() resource.Resource { return &savedViewResource{} }

type savedViewResource struct {
	client *client.Client
}

// savedViewModel models a saved explorer view. `name` and `source_page` are
// typed; the query state (compositeQuery + UI options) is a JSON blob.
type savedViewModel struct {
	ID         types.String         `tfsdk:"id"`
	Name       types.String         `tfsdk:"name"`
	SourcePage types.String         `tfsdk:"source_page"`
	Data       jsontypes.Normalized `tfsdk:"data"`
}

func (r *savedViewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_view"
}

func (r *savedViewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A saved SigNoz explorer view (logs or traces). The query state goes in `data` as JSON.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned view id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "View name.",
				Required:            true,
			},
			"source_page": schema.StringAttribute{
				MarkdownDescription: "Explorer the view belongs to: `logs` or `traces`.",
				Required:            true,
			},
			"data": schema.StringAttribute{
				MarkdownDescription: "Saved query state as JSON (compositeQuery, filters, columns, …). Compared semantically.",
				CustomType:          jsontypes.NormalizedType{},
				Required:            true,
			},
		},
	}
}

func (r *savedViewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// buildView merges the typed name + source_page into the query-state blob.
func buildView(name, sourcePage string, data jsontypes.Normalized) (json.RawMessage, error) {
	body := map[string]any{}
	if !data.IsNull() && data.ValueString() != "" {
		if err := json.Unmarshal([]byte(data.ValueString()), &body); err != nil {
			return nil, err
		}
	}
	body["name"] = name
	body["sourcePage"] = sourcePage
	return json.Marshal(body)
}

func (r *savedViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan savedViewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := buildView(plan.Name.ValueString(), plan.SourcePage.ValueString(), plan.Data)
	if err != nil {
		resp.Diagnostics.AddError("Assembling saved view", err.Error())
		return
	}
	created, err := r.client.CreateView(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Creating SigNoz saved view", err.Error())
		return
	}
	id, err := extractRuleID(created)
	if err != nil {
		resp.Diagnostics.AddError("Reading created view id", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *savedViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state savedViewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}
	if _, err := r.client.GetView(ctx, state.ID.ValueString()); err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading SigNoz saved view", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *savedViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan savedViewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := buildView(plan.Name.ValueString(), plan.SourcePage.ValueString(), plan.Data)
	if err != nil {
		resp.Diagnostics.AddError("Assembling saved view", err.Error())
		return
	}
	if _, err := r.client.UpdateView(ctx, plan.ID.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("Updating SigNoz saved view", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *savedViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state savedViewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteView(ctx, state.ID.ValueString()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Deleting SigNoz saved view", err.Error())
	}
}

func (r *savedViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
