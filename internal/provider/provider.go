// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jkossis/terraform-provider-garage/internal/client"
)

const (
	providerTypeName  = "garage"
	typeNamePrefix    = providerTypeName
	garageEndpointEnv = "GARAGE_ENDPOINT"
	garageTokenEnv    = "GARAGE_TOKEN"
)

// Ensure GarageProvider satisfies various provider interfaces.
var _ provider.Provider = &GarageProvider{}

// GarageProvider defines the provider implementation.
type GarageProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// GarageProviderModel describes the provider data model.
type GarageProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

type providerConfig struct {
	endpoint string
	token    string
}

type providerStringConfigSource struct {
	value       types.String
	attribute   string
	envVar      string
	displayName string
}

func (p *GarageProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = providerTypeName
	resp.Version = p.version
}

func (p *GarageProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing Garage S3 buckets via the Garage Admin API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The Garage Admin API endpoint URL. Can also be set via the " + garageEndpointEnv + " environment variable.",
				Optional:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The Garage Admin API bearer token. Can also be set via the " + garageTokenEnv + " environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *GarageProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data GarageProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	config, diags := providerConfigFrom(data, os.Getenv)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	garageClient := client.NewClient(config.endpoint, config.token)
	resp.DataSourceData = garageClient
	resp.ResourceData = garageClient
}

func providerConfigFrom(data GarageProviderModel, lookupEnv func(string) string) (providerConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	config := providerConfig{
		endpoint: stringConfigValue(providerStringConfigSource{
			value:       data.Endpoint,
			attribute:   "endpoint",
			envVar:      garageEndpointEnv,
			displayName: "Garage Endpoint",
		}, lookupEnv, &diags),
		token: stringConfigValue(providerStringConfigSource{
			value:       data.Token,
			attribute:   "token",
			envVar:      garageTokenEnv,
			displayName: "Garage Token",
		}, lookupEnv, &diags),
	}

	return config, diags
}

func stringConfigValue(source providerStringConfigSource, lookupEnv func(string) string, diags *diag.Diagnostics) string {
	if source.value.IsUnknown() {
		diags.AddAttributeError(
			path.Root(source.attribute),
			"Unknown "+source.displayName,
			"The "+source.attribute+" provider attribute must be known before the provider can be configured.",
		)
		return ""
	}

	if !source.value.IsNull() {
		value := source.value.ValueString()
		if value == "" {
			addMissingProviderConfigDiagnostic(source, diags)
		}
		return value
	}

	value := lookupEnv(source.envVar)
	if value == "" {
		addMissingProviderConfigDiagnostic(source, diags)
	}
	return value
}

func addMissingProviderConfigDiagnostic(source providerStringConfigSource, diags *diag.Diagnostics) {
	diags.AddAttributeError(
		path.Root(source.attribute),
		"Missing "+source.displayName,
		"Set the "+source.attribute+" provider attribute or the "+source.envVar+" environment variable.",
	)
}

func (p *GarageProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBucketResource,
		NewBucketPermissionResource,
		NewKeyResource,
	}
}

func (p *GarageProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewBucketDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GarageProvider{
			version: version,
		}
	}
}
