# Terraform Provider for SigNoz

A [Terraform](https://www.terraform.io) provider for self-hosted
[SigNoz](https://signoz.io), built on the
[Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework).

Manage your SigNoz observability config as code: dashboards, alerts,
notification channels, saved views, and log pipelines.

> **Status: early (v0.x).** Targets SigNoz Community >= 0.125 (v5 / v2alpha1
> alert schema). Complex nested bodies are modeled as JSON strings (see below).

## Resources

| Resource | Manages | Notes |
| --- | --- | --- |
| `signoz_dashboard` | Dashboards | Full definition as a single `data` JSON blob |
| `signoz_alert` | Alert rules | Typed top-level fields + JSON `condition`/`evaluation`/`notification_settings`; v5 defaults |
| `signoz_notification_channel` | Slack/webhook/PagerDuty/email/OpsGenie/MS Teams channels | **Admin token required** |
| `signoz_saved_view` | Logs/traces explorer saved views | |
| `signoz_log_pipeline` | The log-processing pipeline set | **Singleton** — one per instance |

## Design: JSON-blob fields

SigNoz's nested request bodies (dashboard layouts, alert conditions, pipeline
definitions) change shape across versions. Rather than model every nested
field — brittle against that churn — this provider keeps the deep bodies as
JSON strings, validated with a semantic-JSON type so reordering keys or
whitespace never produces a spurious plan diff. Export an object from the
SigNoz UI ("Edit" -> "Show JSON") to bootstrap any blob.

Config-as-code is authoritative: `Read` confirms a resource still exists but
does not overwrite your config from the server (avoiding the round-trip drift
that plagues round-tripping providers).

## Configuration

```hcl
provider "signoz" {
  endpoint     = "https://signoz.example.com" # or SIGNOZ_ENDPOINT
  access_token = var.signoz_token             # or SIGNOZ_ACCESS_TOKEN (sensitive)
}
```

| Argument | Env | Default | Purpose |
| --- | --- | --- | --- |
| `endpoint` | `SIGNOZ_ENDPOINT` | `http://localhost:3301` | Query-service base URL |
| `access_token` | `SIGNOZ_ACCESS_TOKEN` | — | Service Account token (`SIGNOZ-API-KEY` header) |
| `http_timeout` | `SIGNOZ_HTTP_TIMEOUT` | `35` | Per-request timeout (seconds) |
| `http_max_retry` | `SIGNOZ_HTTP_MAX_RETRY` | `3` | Retries on 5xx/network errors |

Generate a token in the SigNoz UI: **Settings -> Service Accounts -> Add -> Keys -> Add Key**.
Notification-channel management needs an **admin**-role account.

See [`examples/`](./examples/) for per-resource usage and [`docs/`](./docs/) for the full schema reference.

## Developing

```bash
go build ./...                 # compile
go test ./...                  # unit tests
make generate                  # regenerate docs/ (needs terraform on PATH)
make testacc                   # acceptance tests — needs a live SigNoz + TF_ACC
```

Acceptance tests require `SIGNOZ_ENDPOINT` + `SIGNOZ_ACCESS_TOKEN` pointing at a
reachable SigNoz instance.

## License

Apache-2.0.
