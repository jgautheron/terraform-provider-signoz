resource "signoz_saved_view" "errors" {
  name        = "Errors (last 15m)"
  source_page = "logs"
  data = jsonencode({
    compositeQuery = {
      queryType = "builder"
      builderQueries = {
        A = { dataSource = "logs", aggregateOperator = "count", expression = "A" }
      }
    }
  })
}
