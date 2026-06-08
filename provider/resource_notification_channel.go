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
	_ resource.Resource                = &notificationChannelResource{}
	_ resource.ResourceWithConfigure   = &notificationChannelResource{}
	_ resource.ResourceWithImportState = &notificationChannelResource{}
)

// NewNotificationChannelResource is the resource factory.
func NewNotificationChannelResource() resource.Resource { return &notificationChannelResource{} }

type notificationChannelResource struct {
	client *client.Client
}

// notificationChannelModel models a SigNoz notification channel. The receiver
// configuration (the Alertmanager-style *_configs block: slack_configs,
// webhook_configs, pagerduty_configs, …) is a JSON blob; `name` is lifted out
// as a typed field because it's the channel's stable identifier.
//
// NOTE: write operations are ADMIN-ONLY in SigNoz.
type notificationChannelModel struct {
	ID   types.String         `tfsdk:"id"`
	Name types.String         `tfsdk:"name"`
	Data jsontypes.Normalized `tfsdk:"data"`
}

func (r *notificationChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_channel"
}

func (r *notificationChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A SigNoz notification channel (Slack, webhook, PagerDuty, email, OpsGenie, MS Teams). " +
			"**Write operations require an admin Service Account token.** The receiver config goes in `data` as JSON.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned channel id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Unique channel name.",
				Required:            true,
			},
			"data": schema.StringAttribute{
				MarkdownDescription: "Receiver configuration as JSON, e.g. " +
					"`{\"slack_configs\":[{\"api_url\":\"…\",\"channel\":\"#alerts\"}]}`. Compared semantically.",
				CustomType: jsontypes.NormalizedType{},
				Required:   true,
			},
		},
	}
}

func (r *notificationChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// buildChannel merges the typed name into the receiver-config blob.
func buildChannel(name string, data jsontypes.Normalized) (json.RawMessage, error) {
	body := map[string]any{}
	if !data.IsNull() && data.ValueString() != "" {
		if err := json.Unmarshal([]byte(data.ValueString()), &body); err != nil {
			return nil, err
		}
	}
	body["name"] = name
	return json.Marshal(body)
}

func (r *notificationChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := buildChannel(plan.Name.ValueString(), plan.Data)
	if err != nil {
		resp.Diagnostics.AddError("Assembling notification channel", err.Error())
		return
	}
	created, err := r.client.CreateChannel(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Creating SigNoz notification channel", err.Error())
		return
	}
	id, err := extractRuleID(created)
	if err != nil {
		resp.Diagnostics.AddError("Reading created channel id", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *notificationChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}
	if _, err := r.client.GetChannel(ctx, state.ID.ValueString()); err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading SigNoz notification channel", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *notificationChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan notificationChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := buildChannel(plan.Name.ValueString(), plan.Data)
	if err != nil {
		resp.Diagnostics.AddError("Assembling notification channel", err.Error())
		return
	}
	if _, err := r.client.UpdateChannel(ctx, plan.ID.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("Updating SigNoz notification channel", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *notificationChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state notificationChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteChannel(ctx, state.ID.ValueString()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Deleting SigNoz notification channel", err.Error())
	}
}

func (r *notificationChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
