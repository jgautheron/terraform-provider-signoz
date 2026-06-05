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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jgautheron/terraform-provider-signoz/internal/client"
)

var (
	_ resource.Resource                = &alertResource{}
	_ resource.ResourceWithConfigure   = &alertResource{}
	_ resource.ResourceWithImportState = &alertResource{}
)

// NewAlertResource is the resource factory.
func NewAlertResource() resource.Resource { return &alertResource{} }

type alertResource struct {
	client *client.Client
}

// alertModel mixes typed top-level fields with JSON-blob bodies for the
// version-churny parts (condition, evaluation, notification_settings).
// Defaults target SigNoz >= 0.125 which requires the v5 / v2alpha1 schema.
type alertModel struct {
	ID                   types.String         `tfsdk:"id"`
	Alert                types.String         `tfsdk:"alert"`
	AlertType            types.String         `tfsdk:"alert_type"`
	Condition            jsontypes.Normalized `tfsdk:"condition"`
	RuleType             types.String         `tfsdk:"rule_type"`
	Version              types.String         `tfsdk:"version"`
	SchemaVersion        types.String         `tfsdk:"schema_version"`
	EvalWindow           types.String         `tfsdk:"eval_window"`
	Frequency            types.String         `tfsdk:"frequency"`
	Severity             types.String         `tfsdk:"severity"`
	Description          types.String         `tfsdk:"description"`
	Source               types.String         `tfsdk:"source"`
	Disabled             types.Bool           `tfsdk:"disabled"`
	Labels               types.Map            `tfsdk:"labels"`
	PreferredChannels    types.List           `tfsdk:"preferred_channels"`
	Evaluation           jsontypes.Normalized `tfsdk:"evaluation"`
	NotificationSettings jsontypes.Normalized `tfsdk:"notification_settings"`
}

func (r *alertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert"
}

func (r *alertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A SigNoz alert rule. Top-level fields are typed; the query `condition`, " +
			"`evaluation`, and `notification_settings` are JSON blobs (export from the SigNoz UI to adapt). " +
			"Defaults target the v5 / v2alpha1 schema required by SigNoz >= 0.125.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned alert rule id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"alert": schema.StringAttribute{
				MarkdownDescription: "Alert name.",
				Required:            true,
			},
			"alert_type": schema.StringAttribute{
				MarkdownDescription: "One of `METRIC_BASED_ALERT`, `LOGS_BASED_ALERT`, `TRACES_BASED_ALERT`, `EXCEPTIONS_BASED_ALERT`.",
				Required:            true,
			},
			"condition": schema.StringAttribute{
				MarkdownDescription: "Query condition as JSON (compositeQuery + thresholds). Compared semantically.",
				CustomType:          jsontypes.NormalizedType{},
				Required:            true,
			},
			"rule_type": schema.StringAttribute{
				MarkdownDescription: "Rule engine type: `threshold_rule` (default) or `promql_rule`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("threshold_rule"),
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Alert API version. Defaults to `v5` (required by SigNoz >= 0.125).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("v5"),
			},
			"schema_version": schema.StringAttribute{
				MarkdownDescription: "Alert schema version. Defaults to `v2alpha1`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("v2alpha1"),
			},
			"eval_window": schema.StringAttribute{
				MarkdownDescription: "Evaluation window (e.g. `5m0s`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("5m0s"),
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "Evaluation frequency (e.g. `1m0s`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("1m0s"),
			},
			"severity": schema.StringAttribute{
				MarkdownDescription: "Severity label (e.g. `info`, `warning`, `critical`).",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Human-readable description.",
				Optional:            true,
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Source URL recorded on the rule.",
				Optional:            true,
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the rule is disabled. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Custom labels attached to the alert.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"preferred_channels": schema.ListAttribute{
				MarkdownDescription: "Notification channel names the alert routes to.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"evaluation": schema.StringAttribute{
				MarkdownDescription: "Evaluation block as JSON (v2alpha1+). Compared semantically.",
				CustomType:          jsontypes.NormalizedType{},
				Optional:            true,
			},
			"notification_settings": schema.StringAttribute{
				MarkdownDescription: "Notification settings as JSON (renotify, group_by, use_policy). Compared semantically.",
				CustomType:          jsontypes.NormalizedType{},
				Optional:            true,
			},
		},
	}
}

func (r *alertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// buildRule assembles the API rule body from the model. Only set fields are
// included so the server applies its own defaults for anything omitted.
func (r *alertResource) buildRule(ctx context.Context, m *alertModel) (json.RawMessage, error) {
	rule := map[string]any{
		"alert":         m.Alert.ValueString(),
		"alertType":     m.AlertType.ValueString(),
		"ruleType":      m.RuleType.ValueString(),
		"version":       m.Version.ValueString(),
		"schemaVersion": m.SchemaVersion.ValueString(),
		"evalWindow":    m.EvalWindow.ValueString(),
		"frequency":     m.Frequency.ValueString(),
		"disabled":      m.Disabled.ValueBool(),
	}

	rule["condition"] = json.RawMessage(m.Condition.ValueString())
	if !m.Evaluation.IsNull() {
		rule["evaluation"] = json.RawMessage(m.Evaluation.ValueString())
	}
	if !m.NotificationSettings.IsNull() {
		rule["notificationSettings"] = json.RawMessage(m.NotificationSettings.ValueString())
	}
	if !m.Severity.IsNull() {
		rule["severity"] = m.Severity.ValueString()
	}
	if !m.Description.IsNull() {
		rule["description"] = m.Description.ValueString()
	}
	if !m.Source.IsNull() {
		rule["source"] = m.Source.ValueString()
	}
	if !m.Labels.IsNull() {
		labels := map[string]string{}
		_ = m.Labels.ElementsAs(ctx, &labels, false)
		rule["labels"] = labels
	}
	if !m.PreferredChannels.IsNull() {
		var channels []string
		_ = m.PreferredChannels.ElementsAs(ctx, &channels, false)
		rule["preferredChannels"] = channels
	}

	return json.Marshal(rule)
}

func (r *alertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.buildRule(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Assembling alert rule", err.Error())
		return
	}

	created, err := r.client.CreateAlert(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Creating SigNoz alert", err.Error())
		return
	}

	id, err := extractRuleID(created)
	if err != nil {
		resp.Diagnostics.AddError("Reading created alert id", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.GetAlert(ctx, state.ID.ValueString()); err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading SigNoz alert", err.Error())
		return
	}
	// Config-as-code is authoritative for alert bodies (see normalizedFromServer
	// rationale); a successful GET confirms existence and we keep state as-is.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *alertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.buildRule(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Assembling alert rule", err.Error())
		return
	}
	if _, err := r.client.UpdateAlert(ctx, plan.ID.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("Updating SigNoz alert", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAlert(ctx, state.ID.ValueString()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Deleting SigNoz alert", err.Error())
	}
}

func (r *alertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// extractRuleID pulls the rule id out of the create/update response, which
// SigNoz returns either as {"id":"…"} or as a bare string id.
func extractRuleID(data json.RawMessage) (string, error) {
	var obj struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && len(obj.ID) > 0 {
		var s string
		if json.Unmarshal(obj.ID, &s) == nil && s != "" {
			return s, nil
		}
		var n json.Number
		if json.Unmarshal(obj.ID, &n) == nil {
			return n.String(), nil
		}
	}
	// Fall back to a bare string id.
	var s string
	if json.Unmarshal(data, &s) == nil && s != "" {
		return s, nil
	}
	return "", &client.APIError{Message: "could not locate rule id in create response: " + string(data)}
}
