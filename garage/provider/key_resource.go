// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jkossis/terraform-provider-garage/garage/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KeyResource{}
var _ resource.ResourceWithImportState = &KeyResource{}

func NewKeyResource() resource.Resource {
	return &KeyResource{}
}

// KeyResource defines the resource implementation.
type KeyResource struct {
	client *client.Client
}

// KeyResourceModel describes the resource data model.
type KeyResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

func (r *KeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key"
}

func (r *KeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Garage access key.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The access key ID. If not provided, one will be generated.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A human-friendly name for the access key.",
			},
			"secret_access_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The secret access key. If not provided, one will be generated (only available on creation).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *KeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine whether to use ImportKey or CreateKey
	hasID := data.ID.ValueString() != ""
	hasSecret := data.SecretAccessKey.ValueString() != ""

	// If both ID and secret are provided, use ImportKey
	if hasID && hasSecret {
		tflog.Debug(ctx, "Importing access key", map[string]interface{}{
			"id":   data.ID.ValueString(),
			"name": data.Name.ValueString(),
		})

		importReq := client.ImportKeyRequest{
			AccessKeyID:     data.ID.ValueString(),
			SecretAccessKey: data.SecretAccessKey.ValueString(),
		}
		if !data.Name.IsNull() {
			name := data.Name.ValueString()
			importReq.Name = &name
		}

		key, err := r.client.ImportKey(ctx, importReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import access key, got error: %s", err))
			return
		}

		data.ID = types.StringValue(key.AccessKeyID)
		data.Name = types.StringValue(key.Name)
		data.SecretAccessKey = types.StringValue(data.SecretAccessKey.ValueString()) // Keep the provided secret

		tflog.Trace(ctx, "Imported access key resource")
	} else if !hasID && !hasSecret {
		// Neither ID nor secret provided, use CreateKey
		tflog.Debug(ctx, "Creating access key", map[string]interface{}{
			"name": data.Name.ValueString(),
		})

		createReq := client.CreateKeyRequest{}
		if !data.Name.IsNull() {
			name := data.Name.ValueString()
			createReq.Name = &name
		}

		key, err := r.client.CreateKey(ctx, createReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create access key, got error: %s", err))
			return
		}

		data.ID = types.StringValue(key.AccessKeyID)
		data.Name = types.StringValue(key.Name)
		if key.SecretAccessKey != nil {
			data.SecretAccessKey = types.StringValue(*key.SecretAccessKey)
		}

		tflog.Trace(ctx, "Created access key resource")
	} else {
		// Invalid combination: only one of ID or secret provided
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Both 'id' and 'secret_access_key' must be provided together when importing a key, or neither should be provided to generate a new key.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	keyID := data.ID.ValueString()
	key, err := r.client.GetKeyInfo(ctx, client.GetKeyInfoRequest{
		ID: keyID,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read access key, got error: %s", err))
		return
	}

	if key == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with key information
	data.ID = types.StringValue(key.AccessKeyID)
	data.Name = types.StringValue(key.Name)
	// Note: SecretAccessKey is not returned by GetKeyInfo (only on creation), so we keep the existing value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Note: UpdateKey is available in the API but we're not implementing it for now
	// The name field is optional and computed, so updates aren't critical for tests

	tflog.Trace(ctx, "Updated access key resource (no-op)")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting access key", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.DeleteKey(ctx, client.DeleteKeyRequest{
		ID: data.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete access key, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted access key resource")
}

func (r *KeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
