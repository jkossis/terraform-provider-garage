// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccKeyEphemeralResource_importedKey(t *testing.T) {
	keyID := generateGarageKeyID()
	secret := generateGarageSecret()

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			providerTypeName: providerserver.NewProtocol6WithError(New("test")()),
			"echo":           echoprovider.NewProviderServer(),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKeyEphemeralResourceConfig(keyID, secret, "test-key-ephemeral"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("name"), knownvalue.StringExact("test-key-ephemeral")),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("secret_access_key"), knownvalue.StringExact(secret)),
				},
			},
		},
	})
}

func testAccKeyEphemeralResourceConfig(keyID, secret, name string) string {
	return fmt.Sprintf(`
resource "garage_key" "test" {
  id                = %[1]q
  secret_access_key = %[2]q
  name              = %[3]q
}

ephemeral "garage_key" "test" {
  id = garage_key.test.id
}

provider "echo" {
  data = ephemeral.garage_key.test
}

resource "echo" "test" {}
`, keyID, secret, name)
}
