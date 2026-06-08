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

// logPipelineID is the fixed id for the singleton pipeline-set resource.
const logPipelineID = "signoz-log-pipelines"

var (
	_ resource.Resource                = &logPipelineResource{}
	_ resource.ResourceWithConfigure   = &logPipelineResource{}
	_ resource.ResourceWithImportState = &logPipelineResource{}
)

// NewLogPipelineResource is the resource factory.
func NewLogPipelineResource() resource.Resource { return &logPipelineResource{} }

type logPipelineResource struct {
	client *client.Client
}

// logPipelineModel models the SigNoz log-pipeline SET. SigNoz versions log
// pipelines as one ordered collection (not individually-addressable), so this
// is a SINGLETON resource: declare at most one per SigNoz instance. `pipelines`
// is the full ordered array as JSON.
type logPipelineModel struct {
	ID        types.String         `tfsdk:"id"`
	Pipelines jsontypes.Normalized `tfsdk:"pipelines"`
}

func (r *logPipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_pipeline"
}

func (r *logPipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The SigNoz log-processing pipeline set. SigNoz manages log pipelines as a single " +
			"versioned, ordered collection, so this is a **singleton** resource — declare at most one. " +
			"Applying publishes a new pipeline-set version; destroying publishes an empty set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Constant singleton id (`" + logPipelineID + "`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pipelines": schema.StringAttribute{
				MarkdownDescription: "Ordered array of pipeline definitions as JSON. Compared semantically.",
				CustomType:          jsontypes.NormalizedType{},
				Required:            true,
			},
		},
	}
}

func (r *logPipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// buildPipelineSet wraps the pipelines array in the API's {"pipelines":[…]} body.
func buildPipelineSet(pipelines jsontypes.Normalized) (json.RawMessage, error) {
	var arr json.RawMessage = []byte("[]")
	if !pipelines.IsNull() && pipelines.ValueString() != "" {
		arr = json.RawMessage(pipelines.ValueString())
	}
	return json.Marshal(map[string]json.RawMessage{"pipelines": arr})
}

func (r *logPipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan logPipelineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := buildPipelineSet(plan.Pipelines)
	if err != nil {
		resp.Diagnostics.AddError("Assembling log pipeline set", err.Error())
		return
	}
	if _, err := r.client.SetPipelines(ctx, body); err != nil {
		resp.Diagnostics.AddError("Publishing SigNoz log pipelines", err.Error())
		return
	}
	plan.ID = types.StringValue(logPipelineID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *logPipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state logPipelineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}
	if _, err := r.client.GetPipelines(ctx); err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading SigNoz log pipelines", err.Error())
		return
	}
	if state.ID.IsNull() {
		state.ID = types.StringValue(logPipelineID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *logPipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan logPipelineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := buildPipelineSet(plan.Pipelines)
	if err != nil {
		resp.Diagnostics.AddError("Assembling log pipeline set", err.Error())
		return
	}
	if _, err := r.client.SetPipelines(ctx, body); err != nil {
		resp.Diagnostics.AddError("Publishing SigNoz log pipelines", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *logPipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Destroying the set publishes an empty pipeline collection.
	empty, _ := json.Marshal(map[string]any{"pipelines": []any{}})
	if _, err := r.client.SetPipelines(ctx, empty); err != nil {
		resp.Diagnostics.AddError("Clearing SigNoz log pipelines", err.Error())
	}
}

func (r *logPipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
