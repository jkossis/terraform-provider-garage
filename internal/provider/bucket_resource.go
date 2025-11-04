// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-garage/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BucketResource{}
var _ resource.ResourceWithImportState = &BucketResource{}

func NewBucketResource() resource.Resource {
	return &BucketResource{}
}

// BucketResource defines the resource implementation.
type BucketResource struct {
	client *client.Client
}

// BucketResourceModel describes the resource data model.
type BucketResourceModel struct {
	ID             types.String `tfsdk:"id"`
	GlobalAlias    types.String `tfsdk:"global_alias"`
	WebsiteEnabled types.Bool   `tfsdk:"website_enabled"`
	WebsiteIndex   types.String `tfsdk:"website_index_document"`
	WebsiteError   types.String `tfsdk:"website_error_document"`
	MaxSize        types.Int64  `tfsdk:"max_size"`
	MaxObjects     types.Int64  `tfsdk:"max_objects"`
}

func (r *BucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *BucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Garage S3 bucket.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the bucket.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"global_alias": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The global alias (name) for the bucket.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"website_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Enable website hosting for this bucket.",
			},
			"website_index_document": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The index document for website hosting (e.g., 'index.html').",
			},
			"website_error_document": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The error document for website hosting (e.g., 'error.html').",
			},
			"max_size": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum size of the bucket in bytes. Leave unset for unlimited.",
			},
			"max_objects": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of objects in the bucket. Leave unset for unlimited.",
			},
		},
	}
}

func (r *BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating bucket", map[string]interface{}{
		"global_alias": data.GlobalAlias.ValueString(),
	})

	// Create bucket with global alias
	globalAlias := data.GlobalAlias.ValueString()
	createReq := client.CreateBucketRequest{
		GlobalAlias: &globalAlias,
	}

	bucket, err := r.client.CreateBucket(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket, got error: %s", err))
		return
	}

	data.ID = types.StringValue(bucket.ID)

	// Update bucket with additional configuration if needed
	updateReq := client.UpdateBucketRequest{}
	needsUpdate := false

	// Configure website settings
	if !data.WebsiteEnabled.IsNull() || !data.WebsiteIndex.IsNull() || !data.WebsiteError.IsNull() {
		websiteEnabled := data.WebsiteEnabled.ValueBool()
		updateReq.WebsiteAccess = &struct {
			Enabled       bool    `json:"enabled"`
			IndexDocument *string `json:"indexDocument,omitempty"`
			ErrorDocument *string `json:"errorDocument,omitempty"`
		}{
			Enabled: websiteEnabled,
		}

		if !data.WebsiteIndex.IsNull() {
			indexDoc := data.WebsiteIndex.ValueString()
			updateReq.WebsiteAccess.IndexDocument = &indexDoc
		}

		if !data.WebsiteError.IsNull() {
			errorDoc := data.WebsiteError.ValueString()
			updateReq.WebsiteAccess.ErrorDocument = &errorDoc
		}

		needsUpdate = true
	}

	// Configure quotas
	if !data.MaxSize.IsNull() || !data.MaxObjects.IsNull() {
		updateReq.Quotas = &client.BucketQuotas{}

		if !data.MaxSize.IsNull() {
			maxSize := data.MaxSize.ValueInt64()
			updateReq.Quotas.MaxSize = &maxSize
		}

		if !data.MaxObjects.IsNull() {
			maxObjects := data.MaxObjects.ValueInt64()
			updateReq.Quotas.MaxObjects = &maxObjects
		}

		needsUpdate = true
	}

	if needsUpdate {
		_, err = r.client.UpdateBucket(ctx, bucket.ID, updateReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket, got error: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "Created bucket resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	bucketID := data.ID.ValueString()
	bucket, err := r.client.GetBucketInfo(ctx, client.GetBucketInfoRequest{
		ID: &bucketID,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket, got error: %s", err))
		return
	}

	if bucket == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with bucket information
	data.ID = types.StringValue(bucket.ID)

	if len(bucket.GlobalAliases) > 0 {
		data.GlobalAlias = types.StringValue(bucket.GlobalAliases[0])
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	bucketID := data.ID.ValueString()

	updateReq := client.UpdateBucketRequest{}

	// Configure website settings
	websiteEnabled := data.WebsiteEnabled.ValueBool()
	updateReq.WebsiteAccess = &struct {
		Enabled       bool    `json:"enabled"`
		IndexDocument *string `json:"indexDocument,omitempty"`
		ErrorDocument *string `json:"errorDocument,omitempty"`
	}{
		Enabled: websiteEnabled,
	}

	if !data.WebsiteIndex.IsNull() {
		indexDoc := data.WebsiteIndex.ValueString()
		updateReq.WebsiteAccess.IndexDocument = &indexDoc
	}

	if !data.WebsiteError.IsNull() {
		errorDoc := data.WebsiteError.ValueString()
		updateReq.WebsiteAccess.ErrorDocument = &errorDoc
	}

	// Configure quotas
	updateReq.Quotas = &client.BucketQuotas{}

	if !data.MaxSize.IsNull() {
		maxSize := data.MaxSize.ValueInt64()
		updateReq.Quotas.MaxSize = &maxSize
	}

	if !data.MaxObjects.IsNull() {
		maxObjects := data.MaxObjects.ValueInt64()
		updateReq.Quotas.MaxObjects = &maxObjects
	}

	_, err := r.client.UpdateBucket(ctx, bucketID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "Updated bucket resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	bucketID := data.ID.ValueString()

	err := r.client.DeleteBucket(ctx, client.DeleteBucketRequest{
		ID: bucketID,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted bucket resource")
}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
