// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKeyDataSource_byID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeyDataSourceConfig("test-key-datasource"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.garage_key.test", "id",
						"garage_key.source", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.garage_key.test", "name",
						"garage_key.source", "name",
					),
					resource.TestCheckResourceAttr("data.garage_key.test", "expired", "false"),
					resource.TestCheckResourceAttrSet("data.garage_key.test", "created"),
					resource.TestCheckResourceAttr("data.garage_key.test", "create_bucket", "false"),
				),
			},
		},
	})
}

func testAccKeyDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "garage_key" "source" {
  name = %[1]q
}

data "garage_key" "test" {
  id = garage_key.source.id
}
`, name)
}
