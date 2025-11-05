// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBucketPermissionResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create bucket and grant read permission
			{
				Config: testAccBucketPermissionResourceConfig_basic("test-perm-bucket", "test-perm-key", true, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("garage_bucket_permission.test", "id"),
					resource.TestCheckResourceAttrSet("garage_bucket_permission.test", "bucket_id"),
					resource.TestCheckResourceAttrSet("garage_bucket_permission.test", "access_key_id"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "read", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "write", "false"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "owner", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "garage_bucket_permission.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update to grant write permission as well
			{
				Config: testAccBucketPermissionResourceConfig_basic("test-perm-bucket", "test-perm-key", true, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "read", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "write", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "owner", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccBucketPermissionResource_allPermissions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all permissions
			{
				Config: testAccBucketPermissionResourceConfig_basic("test-all-perm-bucket", "test-all-perm-key", true, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "read", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "write", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "owner", "true"),
				),
			},
			// Remove write permission
			{
				Config: testAccBucketPermissionResourceConfig_basic("test-all-perm-bucket", "test-all-perm-key", true, false, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "read", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "write", "false"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "owner", "true"),
				),
			},
			// Remove all permissions
			{
				Config: testAccBucketPermissionResourceConfig_basic("test-all-perm-bucket", "test-all-perm-key", false, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "read", "false"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "write", "false"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "owner", "false"),
				),
			},
		},
	})
}

func TestAccBucketPermissionResource_ownerOnly(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with owner permission only
			{
				Config: testAccBucketPermissionResourceConfig_basic("test-owner-bucket", "test-owner-key", false, false, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "read", "false"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "write", "false"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "owner", "true"),
				),
			},
		},
	})
}

func TestAccBucketPermissionResource_multipleKeys(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create permissions for multiple keys on the same bucket
			{
				Config: testAccBucketPermissionResourceConfig_multiple("test-multi-bucket", "test-key-1", "test-key-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check first key permissions
					resource.TestCheckResourceAttr("garage_bucket_permission.test1", "read", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test1", "write", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test1", "owner", "false"),
					// Check second key permissions
					resource.TestCheckResourceAttr("garage_bucket_permission.test2", "read", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test2", "write", "false"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test2", "owner", "false"),
				),
			},
		},
	})
}

// Test configuration functions

func testAccBucketPermissionResourceConfig_basic(bucketName, keyName string, read, write, owner bool) string {
	return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
}

resource "garage_key" "test" {
  name = %[2]q
}

resource "garage_bucket_permission" "test" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.test.id
  read          = %[3]t
  write         = %[4]t
  owner         = %[5]t
}
`, bucketName, keyName, read, write, owner)
}

func testAccBucketPermissionResourceConfig_multiple(bucketName, key1Name, key2Name string) string {
	return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
}

resource "garage_key" "test1" {
  name = %[2]q
}

resource "garage_key" "test2" {
  name = %[3]q
}

resource "garage_bucket_permission" "test1" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.test1.id
  read          = true
  write         = true
  owner         = false
}

resource "garage_bucket_permission" "test2" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.test2.id
  read          = true
  write         = false
  owner         = false
}
`, bucketName, key1Name, key2Name)
}
