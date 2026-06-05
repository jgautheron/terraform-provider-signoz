# Singleton: declare at most one signoz_log_pipeline per SigNoz instance.
# Applying publishes a new pipeline-set version; destroying clears the set.
resource "signoz_log_pipeline" "main" {
  pipelines = jsonencode([
    {
      orderId = 1
      name    = "drop-health-checks"
      alias   = "drop-health-checks"
      enabled = true
      filter = {
        op    = "AND"
        items = [{ key = { key = "http.target" }, op = "=", value = "/healthz" }]
      }
      config = []
    }
  ])
}
