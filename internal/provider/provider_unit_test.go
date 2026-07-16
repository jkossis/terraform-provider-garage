// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestGarageProviderMetadata_usesProviderTypeNameConstant_whenCalled(t *testing.T) {
	// Given
	garageProvider := &GarageProvider{version: "test-version"}
	resp := &provider.MetadataResponse{}

	// When
	garageProvider.Metadata(context.Background(), provider.MetadataRequest{}, resp)

	// Then
	if resp.TypeName != providerTypeName {
		t.Fatalf("TypeName = %q, want %q", resp.TypeName, providerTypeName)
	}
	if resp.Version != "test-version" {
		t.Fatalf("Version = %q, want %q", resp.Version, "test-version")
	}
}

func TestGarageProviderSchema_exposesOptionalSensitiveEnvBackedConfig_whenBuilt(t *testing.T) {
	// Given
	garageProvider := &GarageProvider{}
	resp := &provider.SchemaResponse{}

	// When
	garageProvider.Schema(context.Background(), provider.SchemaRequest{}, resp)

	// Then
	endpointAttr, ok := resp.Schema.Attributes["endpoint"].(providerschema.StringAttribute)
	if !ok {
		t.Fatalf("endpoint attribute type = %T, want provider schema.StringAttribute", resp.Schema.Attributes["endpoint"])
	}
	if !endpointAttr.Optional || endpointAttr.Required || endpointAttr.Sensitive {
		t.Fatalf("endpoint flags Optional=%t Required=%t Sensitive=%t, want optional non-sensitive", endpointAttr.Optional, endpointAttr.Required, endpointAttr.Sensitive)
	}
	if !containsAll(endpointAttr.MarkdownDescription, garageEndpointEnv) {
		t.Fatalf("endpoint description %q does not mention %s", endpointAttr.MarkdownDescription, garageEndpointEnv)
	}

	tokenAttr, ok := resp.Schema.Attributes["token"].(providerschema.StringAttribute)
	if !ok {
		t.Fatalf("token attribute type = %T, want provider schema.StringAttribute", resp.Schema.Attributes["token"])
	}
	if !tokenAttr.Optional || tokenAttr.Required || !tokenAttr.Sensitive {
		t.Fatalf("token flags Optional=%t Required=%t Sensitive=%t, want optional sensitive", tokenAttr.Optional, tokenAttr.Required, tokenAttr.Sensitive)
	}
	if !containsAll(tokenAttr.MarkdownDescription, garageTokenEnv) {
		t.Fatalf("token description %q does not mention %s", tokenAttr.MarkdownDescription, garageTokenEnv)
	}
}

func TestGarageProviderConfigFrom_prefersConfigOverEnvironment_whenBothSet(t *testing.T) {
	// Given
	data := GarageProviderModel{
		Endpoint: types.StringValue("http://config.example"),
		Token:    types.StringValue("config-token"),
	}

	// When
	config, diags := providerConfigFrom(data, func(name string) string {
		return map[string]string{
			garageEndpointEnv: "http://env.example",
			garageTokenEnv:    "env-token",
		}[name]
	})

	// Then
	if diags.HasError() {
		t.Fatalf("providerConfigFrom returned diagnostics: %v", diags)
	}
	if config.endpoint != "http://config.example" {
		t.Fatalf("endpoint = %q, want config value", config.endpoint)
	}
	if config.token != "config-token" {
		t.Fatalf("token = %q, want config value", config.token)
	}
}

func TestGarageProviderConfigFrom_usesEnvironmentFallback_whenConfigMissing(t *testing.T) {
	// Given
	data := GarageProviderModel{
		Endpoint: types.StringNull(),
		Token:    types.StringNull(),
	}

	// When
	config, diags := providerConfigFrom(data, func(name string) string {
		return map[string]string{
			garageEndpointEnv: "http://env.example",
			garageTokenEnv:    "env-token",
		}[name]
	})

	// Then
	if diags.HasError() {
		t.Fatalf("providerConfigFrom returned diagnostics: %v", diags)
	}
	if config.endpoint != "http://env.example" {
		t.Fatalf("endpoint = %q, want environment value", config.endpoint)
	}
	if config.token != "env-token" {
		t.Fatalf("token = %q, want environment value", config.token)
	}
}

func TestGarageProviderConfigFrom_reportsAttributeDiagnostics_whenMissingCredentials(t *testing.T) {
	// Given
	data := GarageProviderModel{
		Endpoint: types.StringNull(),
		Token:    types.StringNull(),
	}

	// When
	_, diags := providerConfigFrom(data, func(string) string { return "" })

	// Then
	if !diags.HasError() {
		t.Fatal("providerConfigFrom returned no errors, want endpoint and token errors")
	}
	assertDiagnosticPath(t, diags, path.Root("endpoint"))
	assertDiagnosticPath(t, diags, path.Root("token"))
}

func TestGarageProviderConfigFrom_doesNotUseEnvironmentForExplicitEmptyOrUnknownValues(t *testing.T) {
	t.Parallel()

	testCases := map[string]GarageProviderModel{
		"empty endpoint": {
			Endpoint: types.StringValue(""),
			Token:    types.StringValue("config-token"),
		},
		"empty token": {
			Endpoint: types.StringValue("http://config.example"),
			Token:    types.StringValue(""),
		},
		"unknown endpoint": {
			Endpoint: types.StringUnknown(),
			Token:    types.StringValue("config-token"),
		},
		"unknown token": {
			Endpoint: types.StringValue("http://config.example"),
			Token:    types.StringUnknown(),
		},
	}

	for name, data := range testCases {
		t.Run(name, func(t *testing.T) {
			config, diags := providerConfigFrom(data, func(name string) string {
				return map[string]string{
					garageEndpointEnv: "http://env.example",
					garageTokenEnv:    "env-token",
				}[name]
			})

			if !diags.HasError() {
				t.Fatal("providerConfigFrom returned no diagnostics")
			}
			if config.endpoint == "http://env.example" || config.token == "env-token" {
				t.Fatalf("providerConfigFrom used environment fallback: %#v", config)
			}
		})
	}
}

func TestGarageProviderResources_registerGarageTypeNames_whenCalled(t *testing.T) {
	// Given
	garageProvider := &GarageProvider{}

	// When
	resources := garageProvider.Resources(context.Background())

	// Then
	wantNames := []string{"garage_bucket", "garage_bucket_permission", "garage_key"}
	if len(resources) != len(wantNames) {
		t.Fatalf("resource count = %d, want %d", len(resources), len(wantNames))
	}
	for index, resourceFactory := range resources {
		assertResourceTypeName(t, resourceFactory, wantNames[index])
	}
}

func TestGarageProviderDataSources_registerGarageTypeNames_whenCalled(t *testing.T) {
	// Given
	garageProvider := &GarageProvider{}

	// When
	dataSources := garageProvider.DataSources(context.Background())

	// Then
	wantNames := []string{"garage_bucket"}
	if len(dataSources) != len(wantNames) {
		t.Fatalf("data source count = %d, want %d", len(dataSources), len(wantNames))
	}
	for index, dataSourceFactory := range dataSources {
		assertDataSourceTypeName(t, dataSourceFactory, wantNames[index])
	}
}

func TestGarageProviderConfigure_assignsClientData_whenConfigValid(t *testing.T) {
	// Given
	garageProvider := &GarageProvider{}
	req := provider.ConfigureRequest{
		Config: testProviderConfig(t, "http://config.example", "config-token"),
	}
	resp := &provider.ConfigureResponse{}

	// When
	garageProvider.Configure(context.Background(), req, resp)

	// Then
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure returned diagnostics: %v", resp.Diagnostics)
	}
	if resp.ResourceData == nil {
		t.Fatal("ResourceData is nil, want configured client")
	}
	if resp.DataSourceData == nil {
		t.Fatal("DataSourceData is nil, want configured client")
	}
	if resp.ResourceData != resp.DataSourceData {
		t.Fatal("ResourceData and DataSourceData differ, want shared configured client")
	}
}

func assertResourceTypeName(t *testing.T, factory func() resource.Resource, want string) {
	t.Helper()

	resp := &resource.MetadataResponse{}
	factory().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: typeNamePrefix}, resp)
	if resp.TypeName != want {
		t.Fatalf("resource TypeName = %q, want %q", resp.TypeName, want)
	}
}

func assertDataSourceTypeName(t *testing.T, factory func() datasource.DataSource, want string) {
	t.Helper()

	resp := &datasource.MetadataResponse{}
	factory().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: typeNamePrefix}, resp)
	if resp.TypeName != want {
		t.Fatalf("data source TypeName = %q, want %q", resp.TypeName, want)
	}
}

func assertDiagnosticPath(t *testing.T, diagnostics diag.Diagnostics, want path.Path) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		pathDiagnostic, ok := diagnostic.(diag.DiagnosticWithPath)
		if !ok {
			continue
		}
		if pathDiagnostic.Path().Equal(want) {
			return
		}
	}

	t.Fatalf("diagnostics did not include path %s: %v", want.String(), diagnostics)
}

func containsAll(value string, want string) bool {
	return strings.Contains(value, want)
}

func testProviderConfig(t *testing.T, endpoint string, token string) tfsdk.Config {
	t.Helper()

	garageProvider := &GarageProvider{}
	schemaResp := &provider.SchemaResponse{}
	garageProvider.Schema(context.Background(), provider.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"endpoint": tftypes.String,
					"token":    tftypes.String,
				},
			},
			map[string]tftypes.Value{
				"endpoint": tftypes.NewValue(tftypes.String, endpoint),
				"token":    tftypes.NewValue(tftypes.String, token),
			},
		),
		Schema: schemaResp.Schema,
	}
}
