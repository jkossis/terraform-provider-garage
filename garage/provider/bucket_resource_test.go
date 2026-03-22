// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBucketResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccBucketResourceConfig_basic("test-bucket-basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "test-bucket-basic"),
					resource.TestCheckResourceAttrSet("garage_bucket.test", "id"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "garage_bucket.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccBucketResourceConfig_basic("test-bucket-basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "test-bucket-basic"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccBucketResource_website(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with website disabled
			{
				Config: testAccBucketResourceConfig_basic("test-bucket-website"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "test-bucket-website"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_enabled", "false"),
				),
			},
			// Update to enable website
			{
				Config: testAccBucketResourceConfig_website("test-bucket-website", true, "index.html", "error.html"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "test-bucket-website"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_enabled", "true"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_index_document", "index.html"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_error_document", "error.html"),
				),
			},
			// Update website documents
			{
				Config: testAccBucketResourceConfig_website("test-bucket-website", true, "home.html", "404.html"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "website_enabled", "true"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_index_document", "home.html"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_error_document", "404.html"),
				),
			},
			// Disable website
			{
				Config: testAccBucketResourceConfig_website("test-bucket-website", false, "", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "website_enabled", "false"),
				),
			},
		},
	})
}

func TestAccBucketResource_quotas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without quotas
			{
				Config: testAccBucketResourceConfig_basic("test-bucket-quotas"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "test-bucket-quotas"),
					resource.TestCheckNoResourceAttr("garage_bucket.test", "max_size"),
					resource.TestCheckNoResourceAttr("garage_bucket.test", "max_objects"),
				),
			},
			// Update to add quotas
			{
				Config: testAccBucketResourceConfig_quotas("test-bucket-quotas", 1073741824, 10000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "max_size", "1073741824"),
					resource.TestCheckResourceAttr("garage_bucket.test", "max_objects", "10000"),
				),
			},
			// Update quota values
			{
				Config: testAccBucketResourceConfig_quotas("test-bucket-quotas", 2147483648, 20000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "max_size", "2147483648"),
					resource.TestCheckResourceAttr("garage_bucket.test", "max_objects", "20000"),
				),
			},
		},
	})
}

func TestAccBucketResource_full(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all features
			{
				Config: testAccBucketResourceConfig_full("test-bucket-full", true, "index.html", "error.html", 5368709120, 50000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "test-bucket-full"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_enabled", "true"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_index_document", "index.html"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_error_document", "error.html"),
					resource.TestCheckResourceAttr("garage_bucket.test", "max_size", "5368709120"),
					resource.TestCheckResourceAttr("garage_bucket.test", "max_objects", "50000"),
					resource.TestCheckResourceAttrSet("garage_bucket.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "garage_bucket.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccBucketResource_nameChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create bucket
			{
				Config: testAccBucketResourceConfig_basic("test-bucket-original"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "test-bucket-original"),
				),
			},
			// Change name (should force replacement)
			{
				Config: testAccBucketResourceConfig_basic("test-bucket-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "test-bucket-renamed"),
				),
			},
		},
	})
}

// Test configuration functions

func testAccBucketResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
}
`, name)
}

func testAccBucketResourceConfig_website(name string, enabled bool, indexDoc, errorDoc string) string {
	config := fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias    = %[1]q
  website_enabled = %[2]t
`, name, enabled)

	if indexDoc != "" {
		config += fmt.Sprintf(`  website_index_document = %[1]q
`, indexDoc)
	}

	if errorDoc != "" {
		config += fmt.Sprintf(`  website_error_document = %[1]q
`, errorDoc)
	}

	config += "}\n"
	return config
}

func testAccBucketResourceConfig_quotas(name string, maxSize, maxObjects int) string {
	return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
  max_size     = %[2]d
  max_objects  = %[3]d
}
`, name, maxSize, maxObjects)
}

func testAccBucketResourceConfig_full(name string, websiteEnabled bool, indexDoc, errorDoc string, maxSize, maxObjects int) string {
	return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias           = %[1]q
  website_enabled        = %[2]t
  website_index_document = %[3]q
  website_error_document = %[4]q
  max_size               = %[5]d
  max_objects            = %[6]d
}
`, name, websiteEnabled, indexDoc, errorDoc, maxSize, maxObjects)
}
