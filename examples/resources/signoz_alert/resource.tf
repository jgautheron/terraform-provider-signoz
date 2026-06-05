# A logs-based alert on the v5 / v2alpha1 schema (SigNoz >= 0.125).
# The threshold references a notification channel by name.
resource "signoz_alert" "high_error_logs" {
  alert      = "High error log volume"
  alert_type = "LOGS_BASED_ALERT"
  severity   = "warning"

  condition = jsonencode({
    compositeQuery = {
      queryType = "builder"
      panelType = "graph"
      queries = [{
        type = "builder_query"
        spec = {
          name         = "A"
          signal       = "logs"
          stepInterval = 0
          aggregations = [{ expression = "count()" }]
          filter       = { expression = "severity_text = 'ERROR'" }
          having       = { expression = "" }
        }
      }]
    }
    selectedQueryName = "A"
    thresholds = {
      kind = "basic"
      spec = [{
        name      = "warning"
        target    = 100
        matchType = "1"
        op        = "1"
        channels  = [signoz_notification_channel.slack.name]
      }]
    }
  })

  evaluation = jsonencode({
    kind = "rolling"
    spec = { evalWindow = "5m0s", frequency = "1m0s" }
  })

  preferred_channels = [signoz_notification_channel.slack.name]
}
