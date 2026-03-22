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

	"github.com/jkossis/terraform-provider-garage/garage/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BucketDataSource{}

func NewBucketDataSource() datasource.DataSource {
	return &BucketDataSource{}
}

// BucketDataSource defines the data source implementation.
type BucketDataSource struct {
	client *client.Client
}

// BucketDataSourceModel describes the data source data model.
type BucketDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	GlobalAlias       types.String `tfsdk:"global_alias"`
	GlobalAliases     types.List   `tfsdk:"global_aliases"`
	WebsiteEnabled    types.Bool   `tfsdk:"website_enabled"`
	WebsiteIndex      types.String `tfsdk:"website_index_document"`
	WebsiteError      types.String `tfsdk:"website_error_document"`
	MaxSize           types.Int64  `tfsdk:"max_size"`
	MaxObjects        types.Int64  `tfsdk:"max_objects"`
	Objects           types.Int64  `tfsdk:"objects"`
	Bytes             types.Int64  `tfsdk:"bytes"`
	UnfinishedUploads types.Int64  `tfsdk:"unfinished_uploads"`
}

func (d *BucketDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (d *BucketDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Garage S3 bucket.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The unique identifier of the bucket. Either id or global_alias must be specified.",
			},
			"global_alias": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The primary global alias (name) of the bucket. Either id or global_alias must be specified.",
			},
			"global_aliases": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "All global aliases for this bucket.",
			},
			"website_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether website hosting is enabled for this bucket.",
			},
			"website_index_document": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The index document for website hosting.",
			},
			"website_error_document": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The error document for website hosting.",
			},
			"max_size": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Maximum size of the bucket in bytes.",
			},
			"max_objects": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Maximum number of objects in the bucket.",
			},
			"objects": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Current number of objects in the bucket.",
			},
			"bytes": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Current size of the bucket in bytes.",
			},
			"unfinished_uploads": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of unfinished multipart uploads.",
			},
		},
	}
}

func (d *BucketDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BucketDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that either ID or GlobalAlias is provided
	if data.ID.IsNull() && data.GlobalAlias.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'global_alias' must be specified.",
		)
		return
	}

	tflog.Debug(ctx, "Reading bucket data source", map[string]interface{}{
		"id":           data.ID.ValueString(),
		"global_alias": data.GlobalAlias.ValueString(),
	})

	// Build request
	getBucketReq := client.GetBucketInfoRequest{}

	if !data.ID.IsNull() {
		id := data.ID.ValueString()
		getBucketReq.ID = &id
	}

	if !data.GlobalAlias.IsNull() {
		alias := data.GlobalAlias.ValueString()
		getBucketReq.GlobalAlias = &alias
	}

	// Fetch bucket info
	bucket, err := d.client.GetBucketInfo(ctx, getBucketReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket, got error: %s", err))
		return
	}

	if bucket == nil {
		resp.Diagnostics.AddError(
			"Bucket Not Found",
			"The specified bucket could not be found.",
		)
		return
	}

	// Populate data model
	data.ID = types.StringValue(bucket.ID)

	if len(bucket.GlobalAliases) > 0 {
		data.GlobalAlias = types.StringValue(bucket.GlobalAliases[0])

		aliases := make([]types.String, 0, len(bucket.GlobalAliases))
		for _, alias := range bucket.GlobalAliases {
			aliases = append(aliases, types.StringValue(alias))
		}
		aliasList, diags := types.ListValueFrom(ctx, types.StringType, aliases)
		resp.Diagnostics.Append(diags...)
		data.GlobalAliases = aliasList
	} else {
		data.GlobalAlias = types.StringNull()
		aliasList, diags := types.ListValueFrom(ctx, types.StringType, []types.String{})
		resp.Diagnostics.Append(diags...)
		data.GlobalAliases = aliasList
	}

	data.WebsiteEnabled = types.BoolValue(bucket.WebsiteAccess)

	if bucket.WebsiteConfig != nil {
		data.WebsiteIndex = types.StringValue(bucket.WebsiteConfig.IndexDocument)
		data.WebsiteError = types.StringValue(bucket.WebsiteConfig.ErrorDocument)
	} else {
		data.WebsiteIndex = types.StringNull()
		data.WebsiteError = types.StringNull()
	}

	if bucket.Quotas != nil {
		if bucket.Quotas.MaxSize != nil {
			data.MaxSize = types.Int64Value(*bucket.Quotas.MaxSize)
		} else {
			data.MaxSize = types.Int64Null()
		}

		if bucket.Quotas.MaxObjects != nil {
			data.MaxObjects = types.Int64Value(*bucket.Quotas.MaxObjects)
		} else {
			data.MaxObjects = types.Int64Null()
		}
	} else {
		data.MaxSize = types.Int64Null()
		data.MaxObjects = types.Int64Null()
	}

	data.Objects = types.Int64Value(bucket.Objects)
	data.Bytes = types.Int64Value(bucket.Bytes)
	data.UnfinishedUploads = types.Int64Value(bucket.UnfinishedUploads)

	tflog.Trace(ctx, "Read bucket data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
