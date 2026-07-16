// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/jkossis/terraform-provider-garage/internal/client"
)

func TestKeyResourceSchema_nameRequiresReplacement(t *testing.T) {
	resourceInstance := &KeyResource{}
	response := &resource.SchemaResponse{}
	resourceInstance.Schema(context.Background(), resource.SchemaRequest{}, response)

	name, ok := response.Schema.Attributes["name"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("name attribute type = %T, want resource schema.StringAttribute", response.Schema.Attributes["name"])
	}
	if len(name.PlanModifiers) != 1 || !strings.Contains(name.PlanModifiers[0].Description(context.Background()), "destroy and recreate") {
		t.Fatalf("name plan modifiers = %#v, want RequiresReplace", name.PlanModifiers)
	}
}

func TestBucketResourceCreate_persistsStateWhenPostCreateUpdateFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/CreateBucket":
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{"id":"bucket-id","globalAliases":["bucket"]}`))
		case "/v2/UpdateBucket":
			http.Error(response, "update failed", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request path %q", request.URL.Path)
		}
	}))
	defer server.Close()

	schema := bucketResourceSchema(t)
	bucket := &BucketResource{client: client.NewClient(server.URL, "token")}
	response := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
	bucket.Create(context.Background(), resource.CreateRequest{Plan: bucketPlan(t, schema, BucketResourceModel{
		ID:             types.StringNull(),
		GlobalAlias:    types.StringValue("bucket"),
		WebsiteEnabled: types.BoolValue(true),
		WebsiteIndex:   types.StringNull(),
		WebsiteError:   types.StringNull(),
		MaxSize:        types.Int64Null(),
		MaxObjects:     types.Int64Null(),
	})}, response)

	if !response.Diagnostics.HasError() {
		t.Fatal("Create returned no diagnostic after update failure")
	}
	var state BucketResourceModel
	if diags := response.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("reading partial state: %v", diags)
	}
	if state.ID.ValueString() != "bucket-id" {
		t.Fatalf("partial state ID = %q, want bucket-id", state.ID.ValueString())
	}
}

func TestBucketResourceRead_clearsGlobalAliasWhenRemoteHasNone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v2/GetBucketInfo" {
			t.Fatalf("unexpected request path %q", request.URL.Path)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"id":"bucket-id","globalAliases":[],"websiteAccess":false}`))
	}))
	defer server.Close()

	schema := bucketResourceSchema(t)
	initial := bucketState(t, schema, BucketResourceModel{
		ID:             types.StringValue("bucket-id"),
		GlobalAlias:    types.StringValue("old-alias"),
		WebsiteEnabled: types.BoolValue(false),
		WebsiteIndex:   types.StringNull(),
		WebsiteError:   types.StringNull(),
		MaxSize:        types.Int64Null(),
		MaxObjects:     types.Int64Null(),
	})
	bucket := &BucketResource{client: client.NewClient(server.URL, "token")}
	response := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
	bucket.Read(context.Background(), resource.ReadRequest{State: initial}, response)

	if response.Diagnostics.HasError() {
		t.Fatalf("Read returned diagnostics: %v", response.Diagnostics)
	}
	var state BucketResourceModel
	if diags := response.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("reading state: %v", diags)
	}
	if !state.GlobalAlias.IsNull() {
		t.Fatalf("global_alias = %q, want null", state.GlobalAlias.ValueString())
	}
}

func TestParseImportID(t *testing.T) {
	testCases := map[string]struct {
		id              string
		wantBucketID    string
		wantAccessKeyID string
		wantValid       bool
	}{
		"valid":             {id: "bucket/key", wantBucketID: "bucket", wantAccessKeyID: "key", wantValid: true},
		"missing separator": {id: "bucket"},
		"empty bucket":      {id: "/key"},
		"empty access key":  {id: "bucket/"},
		"extra separator":   {id: "bucket/key/extra"},
		"only separators":   {id: "//"},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			bucketID, accessKeyID, valid := parseImportID(testCase.id)
			if bucketID != testCase.wantBucketID || accessKeyID != testCase.wantAccessKeyID || valid != testCase.wantValid {
				t.Fatalf("parseImportID(%q) = (%q, %q, %t), want (%q, %q, %t)", testCase.id, bucketID, accessKeyID, valid, testCase.wantBucketID, testCase.wantAccessKeyID, testCase.wantValid)
			}
		})
	}
}

func TestValidateBucketDataSourceSelector(t *testing.T) {
	testCases := map[string]struct {
		id             types.String
		alias          types.String
		wantError      bool
		diagnosticPath string
	}{
		"id":                 {id: types.StringValue("bucket-id"), alias: types.StringNull()},
		"global alias":       {id: types.StringNull(), alias: types.StringValue("bucket")},
		"neither":            {id: types.StringNull(), alias: types.StringNull(), wantError: true, diagnosticPath: "id"},
		"both":               {id: types.StringValue("bucket-id"), alias: types.StringValue("bucket"), wantError: true, diagnosticPath: "id"},
		"empty id":           {id: types.StringValue(""), alias: types.StringNull(), wantError: true, diagnosticPath: "id"},
		"empty global alias": {id: types.StringNull(), alias: types.StringValue(""), wantError: true, diagnosticPath: "global_alias"},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			diags := validateBucketDataSourceSelector(BucketDataSourceModel{ID: testCase.id, GlobalAlias: testCase.alias})
			if diags.HasError() != testCase.wantError {
				t.Fatalf("validation error = %t, want %t: %v", diags.HasError(), testCase.wantError, diags)
			}
			if testCase.wantError {
				assertDiagnosticPath(t, diags, path.Root(testCase.diagnosticPath))
			}
		})
	}
}

func bucketResourceSchema(t *testing.T) resourceschema.Schema {
	t.Helper()
	response := &resource.SchemaResponse{}
	(&BucketResource{}).Schema(context.Background(), resource.SchemaRequest{}, response)
	return response.Schema
}

func bucketPlan(t *testing.T, schema resourceschema.Schema, data BucketResourceModel) tfsdk.Plan {
	t.Helper()
	return tfsdk.Plan{Raw: bucketTerraformValue(t, schema, data), Schema: schema}
}

func bucketState(t *testing.T, schema resourceschema.Schema, data BucketResourceModel) tfsdk.State {
	t.Helper()
	return tfsdk.State{Raw: bucketTerraformValue(t, schema, data), Schema: schema}
}

func bucketTerraformValue(t *testing.T, schema resourceschema.Schema, data BucketResourceModel) tftypes.Value {
	t.Helper()
	return tftypes.NewValue(schema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, stringValue(data.ID)),
		"global_alias":           tftypes.NewValue(tftypes.String, stringValue(data.GlobalAlias)),
		"website_enabled":        tftypes.NewValue(tftypes.Bool, boolValue(data.WebsiteEnabled)),
		"website_index_document": tftypes.NewValue(tftypes.String, stringValue(data.WebsiteIndex)),
		"website_error_document": tftypes.NewValue(tftypes.String, stringValue(data.WebsiteError)),
		"max_size":               tftypes.NewValue(tftypes.Number, int64Value(data.MaxSize)),
		"max_objects":            tftypes.NewValue(tftypes.Number, int64Value(data.MaxObjects)),
	})
}

func stringValue(value types.String) interface{} {
	if value.IsNull() {
		return nil
	}
	return value.ValueString()
}

func boolValue(value types.Bool) interface{} {
	if value.IsNull() {
		return nil
	}
	return value.ValueBool()
}

func int64Value(value types.Int64) interface{} {
	if value.IsNull() {
		return nil
	}
	return value.ValueInt64()
}
