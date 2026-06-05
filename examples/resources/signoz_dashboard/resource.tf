resource "signoz_dashboard" "overview" {
  data = jsonencode({
    title       = "App Overview"
    description = "Managed by Terraform"
    tags        = ["managed-by:terraform"]
    layout      = []
    widgets     = []
  })
}
