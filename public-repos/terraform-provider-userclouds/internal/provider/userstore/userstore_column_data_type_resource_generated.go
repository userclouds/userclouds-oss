// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"userclouds.com/infra/jsonclient"
)

// Note: revive is complaining about stuttering in the generated schema names (e.g. an OpenAPI
// schema might be called "UserstoreColumnTFModel", and then we generate it in the "userstore"
// package, so it becomes "userstore.UserstoreColumnTFModel"), but these names are coming from the
// OpenAPI spec and no one is going to be reading this generated code anyways, so we should just
// silence the rule.
//revive:disable:exported

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &UserstoreColumnDataTypeResource{}
var _ resource.ResourceWithImportState = &UserstoreColumnDataTypeResource{}

// NewUserstoreColumnDataTypeResource returns a new instance of the resource.
func NewUserstoreColumnDataTypeResource() resource.Resource {
	return &UserstoreColumnDataTypeResource{}
}

// UserstoreColumnDataTypeResource defines the resource implementation.
type UserstoreColumnDataTypeResource struct {
	client *jsonclient.Client
}

// Metadata describes the resource metadata.
func (r *UserstoreColumnDataTypeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_userstore_column_data_type"
}

// Schema describes the resource schema.
func (r *UserstoreColumnDataTypeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a User Store column data type. For more details, refer to the User Store documentation: https://docs.userclouds.com/docs/introduction",
		MarkdownDescription: "Manages a User Store column data type. For more details, refer to the [User Store documentation](https://docs.userclouds.com/docs/introduction).",
		Attributes:          UserstoreColumnDataTypeAttributes,
	}
}

// Configure configures the resource.
func (r *UserstoreColumnDataTypeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*jsonclient.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *jsonclient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates a new resource.
func (r *UserstoreColumnDataTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *UserstoreColumnDataTypeTFModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	jsonclientModel, err := UserstoreColumnDataTypeTFModelToJSONClient(data)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_userstore_column_data_type to JSON", err.Error())
		return
	}

	url := "/userstore/config/datatypes"
	// The collection path won't include all path parameters, but it should be fine to ReplaceAll
	// for all the path parameters anyways, since ReplaceAll will just be a no-op if a path
	// parameter doesn't appear in the string.
	url = strings.ReplaceAll(url, "{id}", data.ID.ValueString())
	body := IdpCreateDataTypeRequestJSONClientModel{
		DataType: jsonclientModel,
	}

	marshaled, err := json.Marshal(body)
	if err != nil {
		resp.Diagnostics.AddError("Error serializing userclouds_userstore_column_data_type JSON request body", err.Error())
		return
	}
	tflog.Trace(ctx, fmt.Sprintf("POST %s: %s", url, string(marshaled)))

	var apiResp UserstoreColumnDataTypeJSONClientModel
	if err := r.client.Post(ctx, url, body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Error creating userclouds_userstore_column_data_type", err.Error())
		return
	}
	created := apiResp
	createdTF, err := UserstoreColumnDataTypeJSONClientModelToTF(&created)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_userstore_column_data_type response JSON to Terraform state", err.Error())
		return
	}

	tflog.Trace(ctx, "successfully created userclouds_userstore_column_data_type with ID "+created.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &createdTF)...)
}

// Read reads the existing resource state.
func (r *UserstoreColumnDataTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var oldState *UserstoreColumnDataTypeTFModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &oldState)...)

	if resp.Diagnostics.HasError() {
		return
	}

	url := "/userstore/config/datatypes/{id}"
	url = strings.ReplaceAll(url, "{id}", oldState.ID.ValueString())
	tflog.Trace(ctx, fmt.Sprintf("GET %s", url))
	var apiResp UserstoreColumnDataTypeJSONClientModel
	if err := r.client.Get(ctx, url, &apiResp); err != nil {
		var jce jsonclient.Error
		if errors.As(err, &jce) && (jce.StatusCode == http.StatusNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading userclouds_userstore_column_data_type", err.Error())
		return
	}
	current := apiResp

	newState, err := UserstoreColumnDataTypeJSONClientModelToTF(&current)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_userstore_column_data_type response JSON to Terraform state", err.Error())
		return
	}

	tflog.Trace(ctx, "successfully read userclouds_userstore_column_data_type with ID "+current.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update updates an existing resource.
func (r *UserstoreColumnDataTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state *UserstoreColumnDataTypeTFModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var plan *UserstoreColumnDataTypeTFModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	jsonclientModel, err := UserstoreColumnDataTypeTFModelToJSONClient(plan)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_userstore_column_data_type to JSON", err.Error())
		return
	}

	body := UserstoreUpdateDataTypeRequestJSONClientModel{
		DataType: jsonclientModel,
	}
	url := "/userstore/config/datatypes/{id}"
	url = strings.ReplaceAll(url, "{id}", state.ID.ValueString())

	marshaled, err := json.Marshal(body)
	if err != nil {
		resp.Diagnostics.AddError("Error serializing userclouds_userstore_column_data_type JSON request body", err.Error())
		return
	}
	tflog.Trace(ctx, fmt.Sprintf("PUT %s: %s", url, string(marshaled)))

	var apiResp UserstoreColumnDataTypeJSONClientModel
	if err := r.client.Put(ctx, url, body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Error updating userclouds_userstore_column_data_type", err.Error())
		return
	}
	updated := apiResp

	newState, err := UserstoreColumnDataTypeJSONClientModelToTF(&updated)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_userstore_column_data_type response JSON to Terraform state", err.Error())
		return
	}

	tflog.Trace(ctx, "successfully updated userclouds_userstore_column_data_type with ID "+updated.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Delete deletes an existing resource.
func (r *UserstoreColumnDataTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *UserstoreColumnDataTypeTFModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	url := "/userstore/config/datatypes/{id}"
	url = strings.ReplaceAll(url, "{id}", data.ID.ValueString())
	tflog.Trace(ctx, fmt.Sprintf("GET %s", url))
	if err := r.client.Delete(ctx, url, nil); err != nil {
		resp.Diagnostics.AddError("Error deleting userclouds_userstore_column_data_type", err.Error())
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *UserstoreColumnDataTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
