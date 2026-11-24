// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jkossis/terraform-provider-garage/internal/client"
)

var _ ephemeral.EphemeralResource = &KeyEphemeralResource{}
var _ ephemeral.EphemeralResourceWithConfigure = &KeyEphemeralResource{}

func NewKeyEphemeralResource() ephemeral.EphemeralResource {
	return &KeyEphemeralResource{}
}

// KeyEphemeralResource reads the secret of an access key from Garage, so that
// the state does not keep it.
type KeyEphemeralResource struct {
	client *client.Client
}

type KeyEphemeralResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

func (r *KeyEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = typeNamePrefix + "_key"
}

func (r *KeyEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the secret of a Garage access key.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The access key ID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the access key.",
			},
			"secret_access_key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The secret access key.",
			},
		},
	}
}

func (r *KeyEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *KeyEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data KeyEphemeralResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.GetKeyInfo(ctx, client.GetKeyInfoRequest{
		ID:            data.ID.ValueString(),
		ShowSecretKey: true,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read access key, got error: %s", err))
		return
	}
	if key == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Garage has no access key %q.", data.ID.ValueString()))
		return
	}
	if key.SecretAccessKey == nil {
		resp.Diagnostics.AddError("No Secret", fmt.Sprintf("Garage returned no secret for access key %q.", data.ID.ValueString()))
		return
	}

	data.Name = types.StringValue(key.Name)
	data.SecretAccessKey = types.StringValue(*key.SecretAccessKey)

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
