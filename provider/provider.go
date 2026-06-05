// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

// Package provider implements the SigNoz Terraform provider. It is a public
// (non-internal) package so it can be bridged into Pulumi via
// pulumi-terraform-bridge's plugin-framework path.
package provider

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jgautheron/terraform-provider-signoz/internal/client"
)

// Ensure SignozProvider satisfies the provider.Provider interface.
var _ provider.Provider = &SignozProvider{}

// Environment variables consulted when the matching attribute is unset.
const (
	envAccessToken = "SIGNOZ_ACCESS_TOKEN"
	envEndpoint    = "SIGNOZ_ENDPOINT"
	envHTTPTimeout = "SIGNOZ_HTTP_TIMEOUT"
	envHTTPRetry   = "SIGNOZ_HTTP_MAX_RETRY"
)

// SignozProvider is the provider implementation.
type SignozProvider struct {
	// version is set to the release version on build, "dev" locally, "test"
	// under acceptance testing.
	version string
}

// SignozProviderModel maps provider configuration to Go.
type SignozProviderModel struct {
	Endpoint     types.String `tfsdk:"endpoint"`
	AccessToken  types.String `tfsdk:"access_token"`
	HTTPTimeout  types.Int64  `tfsdk:"http_timeout"`
	HTTPMaxRetry types.Int64  `tfsdk:"http_max_retry"`
}

func (p *SignozProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "signoz"
	resp.Version = p.version
}

func (p *SignozProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage SigNoz dashboards, alerts, notification channels, saved views, and log pipelines as code.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL of the SigNoz query-service API (e.g. `https://signoz.example.com`). " +
					"Defaults to `" + client.DefaultEndpoint + "`. May also be set via the `SIGNOZ_ENDPOINT` environment variable.",
				Optional: true,
			},
			"access_token": schema.StringAttribute{
				MarkdownDescription: "SigNoz Service Account access token, sent as the `SIGNOZ-API-KEY` header. " +
					"May also be set via the `SIGNOZ_ACCESS_TOKEN` environment variable. Required for any real API call.",
				Optional:  true,
				Sensitive: true,
			},
			"http_timeout": schema.Int64Attribute{
				MarkdownDescription: "Per-request HTTP timeout in seconds. Defaults to 35. May also be set via `SIGNOZ_HTTP_TIMEOUT`.",
				Optional:            true,
			},
			"http_max_retry": schema.Int64Attribute{
				MarkdownDescription: "Maximum retries for transient (5xx / network) failures. Defaults to 3. May also be set via `SIGNOZ_HTTP_MAX_RETRY`.",
				Optional:            true,
			},
		},
	}
}

func (p *SignozProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg SignozProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := firstNonEmpty(cfg.Endpoint, os.Getenv(envEndpoint))
	accessToken := firstNonEmpty(cfg.AccessToken, os.Getenv(envAccessToken))

	timeout := client.DefaultTimeout
	if !cfg.HTTPTimeout.IsNull() {
		timeout = time.Duration(cfg.HTTPTimeout.ValueInt64()) * time.Second
	} else if v := os.Getenv(envHTTPTimeout); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			timeout = time.Duration(secs) * time.Second
		}
	}

	maxRetry := 0 // 0 → client applies DefaultMaxRetry
	if !cfg.HTTPMaxRetry.IsNull() {
		maxRetry = int(cfg.HTTPMaxRetry.ValueInt64())
	} else if v := os.Getenv(envHTTPRetry); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxRetry = n
		}
	}

	c := client.New(client.Config{
		Endpoint:    endpoint,
		AccessToken: accessToken,
		Timeout:     timeout,
		MaxRetry:    maxRetry,
		UserAgent:   "terraform-provider-signoz/" + p.version,
	})
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *SignozProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDashboardResource,
		NewAlertResource,
		NewNotificationChannelResource,
		NewSavedViewResource,
		NewLogPipelineResource,
	}
}

func (p *SignozProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// New returns the provider factory consumed by providerserver.Serve and by
// pulumi-terraform-bridge (which calls New(version)()).
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SignozProvider{version: version}
	}
}

// firstNonEmpty returns the attribute value if set & non-empty, else fallback.
func firstNonEmpty(attr types.String, fallback string) string {
	if !attr.IsNull() && attr.ValueString() != "" {
		return attr.ValueString()
	}
	return fallback
}
