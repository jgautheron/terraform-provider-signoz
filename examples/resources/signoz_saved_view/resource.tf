resource "signoz_saved_view" "errors" {
  name        = "Errors (last 15m)"
  source_page = "logs"
  data = jsonencode({
    compositeQuery = {
      queryType = "builder"
      panelType = "list"
      builderQueries = {
        A = {
          dataSource        = "logs"
          queryName         = "A"
          aggregateOperator = "noop"
          expression        = "A"
          disabled          = false
        }
      }
    }
  })
}
