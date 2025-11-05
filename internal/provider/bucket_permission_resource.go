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
var _ resource.Resource = &BucketPermissionResource{}
var _ resource.ResourceWithImportState = &BucketPermissionResource{}

func NewBucketPermissionResource() resource.Resource {
	return &BucketPermissionResource{}
}

// BucketPermissionResource defines the resource implementation.
type BucketPermissionResource struct {
	client *client.Client
}

// BucketPermissionResourceModel describes the resource data model.
type BucketPermissionResourceModel struct {
	ID          types.String `tfsdk:"id"`
	BucketID    types.String `tfsdk:"bucket_id"`
	AccessKeyID types.String `tfsdk:"access_key_id"`
	Read        types.Bool   `tfsdk:"read"`
	Write       types.Bool   `tfsdk:"write"`
	Owner       types.Bool   `tfsdk:"owner"`
}

func (r *BucketPermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_permission"
}

func (r *BucketPermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages permissions for an access key on a Garage S3 bucket.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the permission (format: bucket_id/access_key_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the bucket.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_key_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the access key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"read": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Grant read permission to the access key.",
			},
			"write": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Grant write permission to the access key.",
			},
			"owner": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Grant owner permission to the access key.",
			},
		},
	}
}

func (r *BucketPermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BucketPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketPermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating bucket permission", map[string]interface{}{
		"bucket_id":     data.BucketID.ValueString(),
		"access_key_id": data.AccessKeyID.ValueString(),
	})

	// Grant permissions using AllowBucketKey
	allowReq := client.BucketKeyPermRequest{
		BucketID:    data.BucketID.ValueString(),
		AccessKeyID: data.AccessKeyID.ValueString(),
		Permissions: client.Permissions{
			Read:  data.Read.ValueBool(),
			Write: data.Write.ValueBool(),
			Owner: data.Owner.ValueBool(),
		},
	}

	bucket, err := r.client.AllowBucketKey(ctx, allowReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket permission, got error: %s", err))
		return
	}

	// Set the ID
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.BucketID.ValueString(), data.AccessKeyID.ValueString()))

	// Update state from bucket info to ensure consistency
	r.updateStateFromBucket(ctx, &data, bucket)

	tflog.Trace(ctx, "Created bucket permission resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketPermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	bucketID := data.BucketID.ValueString()
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

	// Update state from bucket info
	r.updateStateFromBucket(ctx, &data, bucket)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketPermissionResourceModel
	var state BucketPermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating bucket permission", map[string]interface{}{
		"bucket_id":     data.BucketID.ValueString(),
		"access_key_id": data.AccessKeyID.ValueString(),
	})

	// Determine which permissions need to be granted or revoked
	readChanged := data.Read.ValueBool() != state.Read.ValueBool()
	writeChanged := data.Write.ValueBool() != state.Write.ValueBool()
	ownerChanged := data.Owner.ValueBool() != state.Owner.ValueBool()

	bucketID := data.BucketID.ValueString()
	accessKeyID := data.AccessKeyID.ValueString()

	var bucket *client.Bucket
	var err error

	// If any permission is being enabled, use AllowBucketKey
	if (readChanged && data.Read.ValueBool()) ||
		(writeChanged && data.Write.ValueBool()) ||
		(ownerChanged && data.Owner.ValueBool()) {

		allowReq := client.BucketKeyPermRequest{
			BucketID:    bucketID,
			AccessKeyID: accessKeyID,
			Permissions: client.Permissions{
				Read:  readChanged && data.Read.ValueBool(),
				Write: writeChanged && data.Write.ValueBool(),
				Owner: ownerChanged && data.Owner.ValueBool(),
			},
		}

		bucket, err = r.client.AllowBucketKey(ctx, allowReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to grant bucket permissions, got error: %s", err))
			return
		}
	}

	// If any permission is being disabled, use DenyBucketKey
	if (readChanged && !data.Read.ValueBool()) ||
		(writeChanged && !data.Write.ValueBool()) ||
		(ownerChanged && !data.Owner.ValueBool()) {

		denyReq := client.BucketKeyPermRequest{
			BucketID:    bucketID,
			AccessKeyID: accessKeyID,
			Permissions: client.Permissions{
				Read:  readChanged && !data.Read.ValueBool(),
				Write: writeChanged && !data.Write.ValueBool(),
				Owner: ownerChanged && !data.Owner.ValueBool(),
			},
		}

		bucket, err = r.client.DenyBucketKey(ctx, denyReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to revoke bucket permissions, got error: %s", err))
			return
		}
	}

	// Update state from bucket info to ensure consistency
	if bucket != nil {
		r.updateStateFromBucket(ctx, &data, bucket)
	}

	tflog.Trace(ctx, "Updated bucket permission resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketPermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting bucket permission", map[string]interface{}{
		"bucket_id":     data.BucketID.ValueString(),
		"access_key_id": data.AccessKeyID.ValueString(),
	})

	// Revoke all permissions
	denyReq := client.BucketKeyPermRequest{
		BucketID:    data.BucketID.ValueString(),
		AccessKeyID: data.AccessKeyID.ValueString(),
		Permissions: client.Permissions{
			Read:  data.Read.ValueBool(),
			Write: data.Write.ValueBool(),
			Owner: data.Owner.ValueBool(),
		},
	}

	_, err := r.client.DenyBucketKey(ctx, denyReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket permission, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted bucket permission resource")
}

func (r *BucketPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: bucket_id/access_key_id
	// Parse the import ID
	bucketID, accessKeyID, found := parseImportID(req.ID)
	if !found {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: bucket_id/access_key_id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket_id"), bucketID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_key_id"), accessKeyID)...)
}

// updateStateFromBucket updates the resource state from bucket info
func (r *BucketPermissionResource) updateStateFromBucket(ctx context.Context, data *BucketPermissionResourceModel, bucket *client.Bucket) {
	// Find the permissions for this access key in the bucket info
	accessKeyID := data.AccessKeyID.ValueString()
	found := false

	for _, keyInfo := range bucket.Keys {
		if keyInfo.AccessKeyID == accessKeyID {
			data.Read = types.BoolValue(keyInfo.Permissions.Read)
			data.Write = types.BoolValue(keyInfo.Permissions.Write)
			data.Owner = types.BoolValue(keyInfo.Permissions.Owner)
			found = true
			break
		}
	}

	if !found {
		// If the key is not in the bucket's key list, all permissions are false
		data.Read = types.BoolValue(false)
		data.Write = types.BoolValue(false)
		data.Owner = types.BoolValue(false)
	}
}

// parseImportID parses an import ID in the format "bucket_id/access_key_id"
func parseImportID(id string) (bucketID, accessKeyID string, ok bool) {
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			return id[:i], id[i+1:], true
		}
	}
	return "", "", false
}
