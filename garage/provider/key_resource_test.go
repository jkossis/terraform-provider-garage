// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// generateGarageKeyID generates a random Garage key ID (GK + 24 hex characters).
func generateGarageKeyID() string {
	bytes := make([]byte, 12) // 12 bytes = 24 hex characters
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random key ID: %v", err))
	}
	return "GK" + hex.EncodeToString(bytes)
}

// generateGarageSecret generates a random secret key (64 hex characters).
func generateGarageSecret() string {
	bytes := make([]byte, 32) // 32 bytes = 64 hex characters
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random secret: %v", err))
	}
	return hex.EncodeToString(bytes)
}

func TestAccKeyResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccKeyResourceConfig_basic("test-key-basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "name", "test-key-basic"),
					resource.TestCheckResourceAttrSet("garage_key.test", "id"),
					resource.TestCheckResourceAttrSet("garage_key.test", "secret_access_key"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "garage_key.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Note: We need to ignore both secret_access_key (only on creation) and name (computed field)
				ImportStateVerifyIgnore: []string{"secret_access_key"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccKeyResource_withoutName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create key without explicit name
			{
				Config: testAccKeyResourceConfig_noName(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("garage_key.test", "id"),
					// Note: name might be empty string or the API might auto-generate it
					// Just verify the key was created successfully
					resource.TestCheckResourceAttrSet("garage_key.test", "secret_access_key"),
				),
			},
		},
	})
}

func TestAccKeyResource_multipleKeys(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple keys
			{
				Config: testAccKeyResourceConfig_multiple("test-key-1", "test-key-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check first key
					resource.TestCheckResourceAttr("garage_key.test1", "name", "test-key-1"),
					resource.TestCheckResourceAttrSet("garage_key.test1", "id"),
					resource.TestCheckResourceAttrSet("garage_key.test1", "secret_access_key"),
					// Check second key
					resource.TestCheckResourceAttr("garage_key.test2", "name", "test-key-2"),
					resource.TestCheckResourceAttrSet("garage_key.test2", "id"),
					resource.TestCheckResourceAttrSet("garage_key.test2", "secret_access_key"),
				),
			},
		},
	})
}

func TestAccKeyResource_withBucketPermission(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create key and bucket with permissions
			{
				Config: testAccKeyResourceConfig_withBucket("test-key-with-bucket", "test-bucket-for-key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check key
					resource.TestCheckResourceAttr("garage_key.test", "name", "test-key-with-bucket"),
					resource.TestCheckResourceAttrSet("garage_key.test", "id"),
					// Check bucket
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "test-bucket-for-key"),
					// Check permission
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "read", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "write", "true"),
					resource.TestCheckResourceAttr("garage_bucket_permission.test", "owner", "false"),
				),
			},
		},
	})
}

func TestAccKeyResource_secretNotAvailableAfterCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create key
			{
				Config: testAccKeyResourceConfig_basic("test-key-secret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("garage_key.test", "secret_access_key"),
				),
			},
			// Update triggers refresh - secret should still be in state (but won't be available via API)
			{
				Config: testAccKeyResourceConfig_basic("test-key-secret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("garage_key.test", "secret_access_key"),
				),
			},
		},
	})
}

func TestAccKeyResource_importWithPredefinedCredentials(t *testing.T) {
	keyID := generateGarageKeyID()
	secret := generateGarageSecret()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Import key with predefined credentials
			{
				Config: testAccKeyResourceConfig_import(keyID, secret, "test-imported-key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "id", keyID),
					resource.TestCheckResourceAttr("garage_key.test", "secret_access_key", secret),
					resource.TestCheckResourceAttr("garage_key.test", "name", "test-imported-key"),
				),
			},
			// Verify idempotency - applying again should not change anything
			{
				Config: testAccKeyResourceConfig_import(keyID, secret, "test-imported-key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "id", keyID),
					resource.TestCheckResourceAttr("garage_key.test", "secret_access_key", secret),
					resource.TestCheckResourceAttr("garage_key.test", "name", "test-imported-key"),
				),
			},
		},
	})
}

func TestAccKeyResource_importWithoutName(t *testing.T) {
	keyID := generateGarageKeyID()
	secret := generateGarageSecret()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Import key without name
			{
				Config: testAccKeyResourceConfig_importNoName(keyID, secret),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "id", keyID),
					resource.TestCheckResourceAttr("garage_key.test", "secret_access_key", secret),
					// Name is computed and will be empty string when not provided during import
					resource.TestCheckResourceAttr("garage_key.test", "name", ""),
				),
			},
		},
	})
}

func TestAccKeyResource_importRequiresBothCredentials(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Only ID provided - should fail
			{
				Config:      testAccKeyResourceConfig_onlyID("GK123456789abcdef01234567"),
				ExpectError: regexp.MustCompile("Both 'id' and 'secret_access_key' must be provided together"),
			},
		},
	})
}

func TestAccKeyResource_changesRequireReplacement(t *testing.T) {
	keyID1 := generateGarageKeyID()
	secret1 := generateGarageSecret()
	keyID2 := generateGarageKeyID()
	secret2 := generateGarageSecret()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create key with predefined credentials
			{
				Config: testAccKeyResourceConfig_import(keyID1, secret1, "test-replace-key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "id", keyID1),
				),
			},
			// Change ID - should require replacement
			{
				Config: testAccKeyResourceConfig_import(keyID2, secret2, "test-replace-key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "id", keyID2),
				),
			},
		},
	})
}

// Test configuration functions

func testAccKeyResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "garage_key" "test" {
  name = %[1]q
}
`, name)
}

func testAccKeyResourceConfig_noName() string {
	return `
resource "garage_key" "test" {
}
`
}

func testAccKeyResourceConfig_multiple(name1, name2 string) string {
	return fmt.Sprintf(`
resource "garage_key" "test1" {
  name = %[1]q
}

resource "garage_key" "test2" {
  name = %[2]q
}
`, name1, name2)
}

func testAccKeyResourceConfig_withBucket(keyName, bucketName string) string {
	return fmt.Sprintf(`
resource "garage_key" "test" {
  name = %[1]q
}

resource "garage_bucket" "test" {
  global_alias = %[2]q
}

resource "garage_bucket_permission" "test" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.test.id
  read          = true
  write         = true
  owner         = false
}
`, keyName, bucketName)
}

func testAccKeyResourceConfig_import(id, secret, name string) string {
	return fmt.Sprintf(`
resource "garage_key" "test" {
  id                = %[1]q
  secret_access_key = %[2]q
  name              = %[3]q
}
`, id, secret, name)
}

func testAccKeyResourceConfig_importNoName(id, secret string) string {
	return fmt.Sprintf(`
resource "garage_key" "test" {
  id                = %[1]q
  secret_access_key = %[2]q
}
`, id, secret)
}

func testAccKeyResourceConfig_onlyID(id string) string {
	return fmt.Sprintf(`
resource "garage_key" "test" {
  id = %[1]q
}
`, id)
}
