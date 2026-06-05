// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDashboardResource covers create → read → import → update → destroy
// against a live SigNoz. Requires TF_ACC=1 plus SIGNOZ_ENDPOINT /
// SIGNOZ_ACCESS_TOKEN (Editor role is sufficient for dashboards).
func TestAccDashboardResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig("tf-acc dashboard", "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("signoz_dashboard.test", "id"),
					resource.TestCheckResourceAttrSet("signoz_dashboard.test", "data"),
				),
			},
			{
				ResourceName:      "signoz_dashboard.test",
				ImportState:       true,
				ImportStateVerify: false, // server adds computed fields to data; id round-trips
			},
			{
				Config: testAccDashboardConfig("tf-acc dashboard", "second"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("signoz_dashboard.test", "id"),
				),
			},
		},
	})
}

func testAccDashboardConfig(title, desc string) string {
	return fmt.Sprintf(`
resource "signoz_dashboard" "test" {
  data = jsonencode({
    title       = %[1]q
    description = %[2]q
    tags        = ["tf-acc"]
    layout      = []
    widgets     = []
  })
}
`, title, desc)
}
