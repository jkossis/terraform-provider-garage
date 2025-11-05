// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-garage/internal/client"
)

// Ensure GarageProvider satisfies various provider interfaces.
var _ provider.Provider = &GarageProvider{}
var _ provider.ProviderWithFunctions = &GarageProvider{}
var _ provider.ProviderWithEphemeralResources = &GarageProvider{}

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

func (p *GarageProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "garage"
	resp.Version = p.version
}

func (p *GarageProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing Garage S3 buckets via the Garage Admin API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The Garage Admin API endpoint URL. Can also be set via the GARAGE_ENDPOINT environment variable.",
				Optional:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The Garage Admin API bearer token. Can also be set via the GARAGE_TOKEN environment variable.",
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

	// Check for environment variables if not set in config
	endpoint := data.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("GARAGE_ENDPOINT")
	}

	token := data.Token.ValueString()
	if token == "" {
		token = os.Getenv("GARAGE_TOKEN")
	}

	// Validate required configuration
	if endpoint == "" {
		resp.Diagnostics.AddError(
			"Missing Garage Endpoint",
			"The provider cannot create the Garage API client as there is a missing or empty value for the Garage endpoint. "+
				"Set the endpoint value in the configuration or use the GARAGE_ENDPOINT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Garage Token",
			"The provider cannot create the Garage API client as there is a missing or empty value for the Garage admin token. "+
				"Set the token value in the configuration or use the GARAGE_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create Garage API client
	garageClient := client.NewClient(endpoint, token)
	resp.DataSourceData = garageClient
	resp.ResourceData = garageClient
}

func (p *GarageProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBucketResource,
		NewBucketPermissionResource,
		NewKeyResource,
	}
}

func (p *GarageProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *GarageProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewBucketDataSource,
	}
}

func (p *GarageProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GarageProvider{
			version: version,
		}
	}
}
