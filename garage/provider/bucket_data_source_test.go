// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBucketDataSource_byAlias(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketDataSourceConfig_byAlias("test-bucket-datasource-alias"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.garage_bucket.test", "global_alias", "test-bucket-datasource-alias"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "id"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "website_enabled", "false"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "objects"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "bytes"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "unfinished_uploads"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "global_aliases.#"),
				),
			},
		},
	})
}

func TestAccBucketDataSource_byID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketDataSourceConfig_byID("test-bucket-datasource-id"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.garage_bucket.test", "global_alias", "test-bucket-datasource-id"),
					resource.TestCheckResourceAttrPair(
						"data.garage_bucket.test", "id",
						"garage_bucket.source", "id",
					),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "objects"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "bytes"),
				),
			},
		},
	})
}

func TestAccBucketDataSource_withWebsite(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketDataSourceConfig_withWebsite("test-bucket-datasource-website"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.garage_bucket.test", "global_alias", "test-bucket-datasource-website"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "website_enabled", "true"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "website_index_document", "index.html"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "website_error_document", "error.html"),
				),
			},
		},
	})
}

func TestAccBucketDataSource_withQuotas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketDataSourceConfig_withQuotas("test-bucket-datasource-quotas"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.garage_bucket.test", "global_alias", "test-bucket-datasource-quotas"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "max_size", "1073741824"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "max_objects", "10000"),
				),
			},
		},
	})
}

func TestAccBucketDataSource_full(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketDataSourceConfig_full("test-bucket-datasource-full"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.garage_bucket.test", "global_alias", "test-bucket-datasource-full"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "id"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "website_enabled", "true"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "website_index_document", "index.html"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "website_error_document", "error.html"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "max_size", "5368709120"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "max_objects", "50000"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "objects"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "bytes"),
					resource.TestCheckResourceAttrSet("data.garage_bucket.test", "unfinished_uploads"),
				),
			},
		},
	})
}

func TestAccBucketDataSource_multipleAliases(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketDataSourceConfig_byAlias("test-bucket-datasource-multi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.garage_bucket.test", "global_alias", "test-bucket-datasource-multi"),
					// Check that global_aliases list is populated
					resource.TestCheckResourceAttr("data.garage_bucket.test", "global_aliases.#", "1"),
					resource.TestCheckResourceAttr("data.garage_bucket.test", "global_aliases.0", "test-bucket-datasource-multi"),
				),
			},
		},
	})
}

// Test configuration functions

func testAccBucketDataSourceConfig_byAlias(name string) string {
	return fmt.Sprintf(`
resource "garage_bucket" "source" {
  global_alias = %[1]q
}

data "garage_bucket" "test" {
  global_alias = garage_bucket.source.global_alias
}
`, name)
}

func testAccBucketDataSourceConfig_byID(name string) string {
	return fmt.Sprintf(`
resource "garage_bucket" "source" {
  global_alias = %[1]q
}

data "garage_bucket" "test" {
  id = garage_bucket.source.id
}
`, name)
}

func testAccBucketDataSourceConfig_withWebsite(name string) string {
	return fmt.Sprintf(`
resource "garage_bucket" "source" {
  global_alias           = %[1]q
  website_enabled        = true
  website_index_document = "index.html"
  website_error_document = "error.html"
}

data "garage_bucket" "test" {
  global_alias = garage_bucket.source.global_alias
}
`, name)
}

func testAccBucketDataSourceConfig_withQuotas(name string) string {
	return fmt.Sprintf(`
resource "garage_bucket" "source" {
  global_alias = %[1]q
  max_size     = 1073741824
  max_objects  = 10000
}

data "garage_bucket" "test" {
  global_alias = garage_bucket.source.global_alias
}
`, name)
}

func testAccBucketDataSourceConfig_full(name string) string {
	return fmt.Sprintf(`
resource "garage_bucket" "source" {
  global_alias           = %[1]q
  website_enabled        = true
  website_index_document = "index.html"
  website_error_document = "error.html"
  max_size               = 5368709120
  max_objects            = 50000
}

data "garage_bucket" "test" {
  global_alias = garage_bucket.source.global_alias
}
`, name)
}
