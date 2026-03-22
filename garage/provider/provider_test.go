// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"garage": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the garage provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
var _ = map[string]func() (tfprotov6.ProviderServer, error){
	"garage": providerserver.NewProtocol6WithError(New("test")()),
	"echo":   echoprovider.NewProviderServer(),
}

func testAccPreCheck(t *testing.T) {
	// Verify that required environment variables are set for acceptance tests
	if v := os.Getenv("GARAGE_ENDPOINT"); v == "" {
		t.Fatal("GARAGE_ENDPOINT must be set for acceptance tests")
	}

	if v := os.Getenv("GARAGE_TOKEN"); v == "" {
		t.Fatal("GARAGE_TOKEN must be set for acceptance tests")
	}
}
