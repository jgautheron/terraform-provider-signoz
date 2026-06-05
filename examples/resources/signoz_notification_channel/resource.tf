# Notification channels require an ADMIN SigNoz service-account token.
resource "signoz_notification_channel" "slack" {
  name = "team-alerts"
  data = jsonencode({
    slack_configs = [{
      send_resolved = true
      api_url       = "https://hooks.slack.com/services/T000/B000/XXXX"
      channel       = "#alerts"
    }]
  })
}
