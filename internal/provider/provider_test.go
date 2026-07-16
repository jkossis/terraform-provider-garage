// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var requiredAcceptanceEnvVars = []string{
	garageEndpointEnv,
	garageTokenEnv,
}

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	providerTypeName: providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC must be set to run acceptance tests")
	}

	missing := missingAcceptanceEnvVars(os.Getenv)
	if len(missing) > 0 {
		t.Fatalf("acceptance tests require environment variables: %s", strings.Join(missing, ", "))
	}
}

func missingAcceptanceEnvVars(lookupEnv func(string) string) []string {
	missing := make([]string, 0, len(requiredAcceptanceEnvVars))
	for _, envVar := range requiredAcceptanceEnvVars {
		if lookupEnv(envVar) == "" {
			missing = append(missing, envVar)
		}
	}
	return missing
}
