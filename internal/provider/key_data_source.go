// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-garage/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &KeyDataSource{}

func NewKeyDataSource() datasource.DataSource {
	return &KeyDataSource{}
}

// KeyDataSource defines the data source implementation.
type KeyDataSource struct {
	client *client.Client
}

// KeyDataSourceModel describes the data source data model.
type KeyDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Expired      types.Bool   `tfsdk:"expired"`
	Created      types.String `tfsdk:"created"`
	Expiration   types.String `tfsdk:"expiration"`
	CreateBucket types.Bool   `tfsdk:"create_bucket"`
}

func (d *KeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key"
}

func (d *KeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Garage access key.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The access key ID to look up.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "A human-friendly name for the access key.",
			},
			"expired": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the access key has expired.",
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The date and time the access key was created.",
			},
			"expiration": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The date and time the access key expires, if set.",
			},
			"create_bucket": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the access key is allowed to create buckets.",
			},
		},
	}
}

func (d *KeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *KeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KeyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading key data source", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	// Fetch key info
	key, err := d.client.GetKeyInfo(ctx, client.GetKeyInfoRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read access key, got error: %s", err))
		return
	}

	if key == nil {
		resp.Diagnostics.AddError(
			"Key Not Found",
			fmt.Sprintf("The specified access key (%s) could not be found.", data.ID.ValueString()),
		)
		return
	}

	// Populate data model
	data.ID = types.StringValue(key.AccessKeyID)
	data.Name = types.StringValue(key.Name)
	data.Expired = types.BoolValue(key.Expired)

	if key.Created != nil {
		data.Created = types.StringValue(*key.Created)
	} else {
		data.Created = types.StringNull()
	}

	if key.Expiration != nil {
		data.Expiration = types.StringValue(*key.Expiration)
	} else {
		data.Expiration = types.StringNull()
	}

	data.CreateBucket = types.BoolValue(key.Permissions.CreateBucket)

	tflog.Trace(ctx, "Read key data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
