package resources

import (
	"bytes"
	"context"
	"go/format"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/swaggest/openapi-go/openapi3"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/config"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/schemas"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/stringutils"

	"userclouds.com/infra/ucerr"
)

type data struct {
	Package string
	// Name of resource struct (e.g. UserstoreColumnResource)
	StructName string
	// Name of function that instantiates resource struct (e.g. NewUserstoreColumnResource)
	NewResourceFuncName string
	// Name of terraform model struct (e.g. UserstoreColumnTFModel)
	TFModelName string
	// Name of jsonclient model struct (e.g. UserstoreColumnJSONClientModel)
	JSONClientModelName string
	// Names of model conversion functions
	TFModelToJSONClientFuncName string
	JSONClientModelToTFFuncName string
	// Name suffix of Terraform resource (e.g. "userstore_column" for the userclouds_userstore_column resource)
	TypeNameSuffix              string
	ResourceDescription         string
	ResourceMarkdownDescription string
	AttributesMap               string

	// Path to the REST collection
	CollectionPath string
	// Path to the REST resource
	ResourcePath string
	// Map from path param name to the name of the TF model struct field whose value should be used
	// for that path param. (Model field names should be UpperCamelCase)
	PathParamsToModelFields map[string]string
	// Whether the resource body has a "version" field functioning like an etag
	// (on update, must specify the last known version of the resource)
	IsVersionedResource bool

	// Name of the model for fetch request. Likely the same as JSONClientModelName, but if the GET
	// response doesn't return the model directly (e.g. returns several properties, of which the
	// model is one), then this will differ.
	ReadResponseModel string
	// Name of property within the ReadResponseModel struct that contains the model data,
	// or "" if GetResponseModel = JSONClientModelName (i.e. the GET endpoint returns the model
	// directly)
	ReadResponseDataPropertyName string

	// Name of model struct for creation request (e.g. IdpCreateColumnRequestJSONClientModel)
	CreateRequestModel string
	// Name of property within the CreateRequestModel struct that which contains the object data to
	// create (e.g. "Column" for IdpCreateColumnRequestJSONClientModel)
	CreateRequestDataPropertyName string
	// Whether the CreateRequestDataPropertyName property is an array
	CreateRequestBodyUsesArray bool
	// Name of the model for create response. Likely the same as JSONClientModelName, but if the PUT
	// response doesn't return the model directly (e.g. returns several properties, of which the
	// model is one), then this will differ.
	CreateResponseModel string
	// Name of property within the CreateResponseModel struct that contains the updated
	// model data, or "" if UpdateResponseModel = JSONClientModelName
	CreateResponseDataPropertyName string
	// Whether the CreateResponseDataPropertyName property is an array
	CreateResponseBodyUsesArray bool

	UpdateSupported bool
	// Name of model struct for update request (e.g. IdpUpdateColumnRequestJSONClientModel)
	UpdateRequestModel string
	// Name of property within the UpdateRequestModel struct that which contains the object data to
	// update (e.g. "Column" for IdpUpdateColumnRequestJSONClientModel)
	UpdateRequestDataPropertyName string
	// Name of the model for update response. Likely the same as JSONClientModelName, but if the PUT
	// response doesn't return the model directly (e.g. returns several properties, of which the
	// model is one), then this will differ.
	UpdateResponseModel string
	// Name of property within the UpdateResponseModel struct that contains the updated
	// model data, or "" if UpdateResponseModel = JSONClientModelName
	UpdateResponseDataPropertyName string

	// The name of the query param that should be used to specify the version of
	// the resource to delete, e.g. "policy_version", if required. (empty string
	// if the delete endpoint does not require specifying a version)
	DeleteVersionQueryParam string
}

var temp = `// NOTE: automatically generated file -- DO NOT EDIT

package << .Package >>

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
var _ resource.Resource = &<< .StructName >>{}
var _ resource.ResourceWithImportState = &<< .StructName >>{}
<<- if not .UpdateSupported >>
var _ resource.ResourceWithModifyPlan = &<< .StructName >>{}
<<- end >>

// << .NewResourceFuncName >> returns a new instance of the resource.
func << .NewResourceFuncName >>() resource.Resource {
	return &<< .StructName >>{}
}

// << .StructName >> defines the resource implementation.
type << .StructName >> struct {
	client *jsonclient.Client
}

// Metadata describes the resource metadata.
func (r *<< .StructName >>) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_<< .TypeNameSuffix >>"
}

// Schema describes the resource schema.
func (r *<< .StructName >>) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "<< .ResourceDescription >>",
		MarkdownDescription: "<< .ResourceMarkdownDescription >>",
		Attributes: << .AttributesMap >>,
	}
}

// Configure configures the resource.
func (r *<< .StructName >>) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *<< .StructName >>) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *<< .TFModelName >>

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	jsonclientModel, err := << .TFModelToJSONClientFuncName >>(data)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_<< .TypeNameSuffix >> to JSON", err.Error())
		return
	}

	url := "<< .CollectionPath >>"
	// The collection path won't include all path parameters, but it should be fine to ReplaceAll
	// for all the path parameters anyways, since ReplaceAll will just be a no-op if a path
	// parameter doesn't appear in the string.
	<<- range $pathParam, $attr := .PathParamsToModelFields >>
	url = strings.ReplaceAll(url, "{<< $pathParam >>}", data.<< $attr >>.ValueString())
	<<- end >>

	<<- if .CreateRequestBodyUsesArray >>
	modelArray := []<< .JSONClientModelName >>{*jsonclientModel}
	body := << .CreateRequestModel >>{
		<< .CreateRequestDataPropertyName >>: &modelArray,
	}
	<<- else >>
	body := << .CreateRequestModel >>{
		<< .CreateRequestDataPropertyName >>: jsonclientModel,
	}
	<<- end >>

	marshaled, err := json.Marshal(body)
	if err != nil {
		resp.Diagnostics.AddError("Error serializing userclouds_<< .TypeNameSuffix >> JSON request body", err.Error())
		return
	}
	tflog.Trace(ctx, fmt.Sprintf("POST %s: %s", url, string(marshaled)))

	var apiResp << .CreateResponseModel >>
	if err := r.client.Post(ctx, url, body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Error creating userclouds_<< .TypeNameSuffix >>", err.Error())
		return
	}

	<<- if eq .CreateResponseModel .JSONClientModelName >>
	created := apiResp
	<<- else if not .CreateResponseBodyUsesArray >>
	created := *apiResp.<< .CreateResponseDataPropertyName >>
	<<- else >>
	created := (*apiResp.<< .CreateResponseDataPropertyName >>)[0]
	<<- end >>
	createdTF, err := << .JSONClientModelToTFFuncName >>(&created)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_<< .TypeNameSuffix >> response JSON to Terraform state", err.Error())
		return
	}

	tflog.Trace(ctx, "successfully created userclouds_<< .TypeNameSuffix >> with ID "+created.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &createdTF)...)
}

// Read reads the existing resource state.
func (r *<< .StructName >>) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var oldState *<< .TFModelName >>

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &oldState)...)

	if resp.Diagnostics.HasError() {
		return
	}

	url := "<< .ResourcePath >>"
	<<- range $pathParam, $attr := .PathParamsToModelFields >>
	url = strings.ReplaceAll(url, "{<< $pathParam >>}", oldState.<< $attr >>.ValueString())
	<<- end >>
	tflog.Trace(ctx, fmt.Sprintf("GET %s", url))
	var apiResp << .ReadResponseModel >>
	if err := r.client.Get(ctx, url, &apiResp); err != nil {
		var jce jsonclient.Error
		if errors.As(err, &jce) && (jce.StatusCode == http.StatusNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading userclouds_<< .TypeNameSuffix >>", err.Error())
		return
	}
	<<- if eq .ReadResponseModel .JSONClientModelName >>
	current := apiResp
	<<- else >>
	current := *apiResp.<< .ReadResponseDataPropertyName >>
	<<- end >>

	newState, err := << .JSONClientModelToTFFuncName >>(&current)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_<< .TypeNameSuffix >> response JSON to Terraform state", err.Error())
		return
	}

	tflog.Trace(ctx, "successfully read userclouds_<< .TypeNameSuffix >> with ID "+current.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update updates an existing resource.
func (r *<< .StructName >>) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	<<- if .UpdateSupported >>
	var state *<< .TFModelName >>
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var plan *<< .TFModelName >>
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	<<- if .IsVersionedResource >>
	// Must provide the last-known version. (The IncrementOnUpdate plan modifier
	// has already incremented the version in the plan, but we need to provide
	// the old version in our request to the server)
	plan.Version = state.Version
	<<- end >>

	jsonclientModel, err := << .TFModelToJSONClientFuncName >>(plan)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_<< .TypeNameSuffix >> to JSON", err.Error())
		return
	}

	body := << .UpdateRequestModel >>{
		<< .UpdateRequestDataPropertyName >>: jsonclientModel,
	}
	url := "<< .ResourcePath >>"
	<<- range $pathParam, $attr := .PathParamsToModelFields >>
	url = strings.ReplaceAll(url, "{<< $pathParam >>}", state.<< $attr >>.ValueString())
	<<- end >>

	marshaled, err := json.Marshal(body)
	if err != nil {
		resp.Diagnostics.AddError("Error serializing userclouds_<< .TypeNameSuffix >> JSON request body", err.Error())
		return
	}
	tflog.Trace(ctx, fmt.Sprintf("PUT %s: %s", url, string(marshaled)))

	var apiResp << .UpdateResponseModel >>
	if err := r.client.Put(ctx, url, body, &apiResp); err != nil {
		resp.Diagnostics.AddError("Error updating userclouds_<< .TypeNameSuffix >>", err.Error())
		return
	}
	<<- if eq .UpdateResponseModel .JSONClientModelName >>
	updated := apiResp
	<<- else >>
	updated := *apiResp.<< .UpdateResponseDataPropertyName >>
	<<- end >>

	newState, err := << .JSONClientModelToTFFuncName >>(&updated)
	if err != nil {
		resp.Diagnostics.AddError("Error converting userclouds_<< .TypeNameSuffix >> response JSON to Terraform state", err.Error())
		return
	}

	tflog.Trace(ctx, "successfully updated userclouds_<< .TypeNameSuffix >> with ID "+updated.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
	<<- else >>
	resp.Diagnostics.AddError(
		"Updates are not supported for userclouds_<< .TypeNameSuffix >>",
		"Terraform should have suggested destroying and re-creating the resource. Please report this as a provider bug.",
	)
	<<- end >>
}

// Delete deletes an existing resource.
func (r *<< .StructName >>) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *<< .TFModelName >>

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	url := "<< .ResourcePath >>"
	<<- if not (eq .DeleteVersionQueryParam "") >>
	url += "?<< .DeleteVersionQueryParam >>=all"
	<<- end >>
	<<- range $pathParam, $attr := .PathParamsToModelFields >>
	url = strings.ReplaceAll(url, "{<< $pathParam >>}", data.<< $attr >>.ValueString())
	<<- end >>
	tflog.Trace(ctx, fmt.Sprintf("GET %s", url))
	if err := r.client.Delete(ctx, url, nil); err != nil {
		resp.Diagnostics.AddError("Error deleting userclouds_<< .TypeNameSuffix >>", err.Error())
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *<< .StructName >>) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

<<- if not .UpdateSupported >>
// ModifyPlan forces replacement on modification, since updates are not supported for this resource
func (r *<< .StructName >>) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Do not replace on resource creation.
	if req.State.Raw.IsNull() {
		return
	}

	// Do not replace on resource destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	// Do not replace if the plans and states are equal.
	if req.Plan.Raw.Equal(req.State.Raw) {
		return
	}

	// TODO: does this work, or do we need to enumerate all the fields with path.Root()?
	resp.RequiresReplace.Append(path.Empty())
}
<<- end >>
`

// Get the OpenAPI schema ref for the request body of the given path and method.
func getRequestBodySchemaRef(spec *openapi3.Spec, path string, method string) (*openapi3.SchemaReference, error) {
	endpoint, ok := spec.Paths.MapOfPathItemValues[path]
	if !ok {
		return nil, ucerr.Errorf("could not find path %s in the OpenAPI spec", path)
	}
	op, ok := endpoint.MapOfOperationValues[method]
	if !ok {
		return nil, ucerr.Errorf("path %s does not support %s", path, strings.ToUpper(method))
	}
	body := op.RequestBody
	if body == nil {
		return nil, ucerr.Errorf("path %s %s operation does not have a request body", path, strings.ToUpper(method))
	}
	if body.RequestBody == nil {
		// TODO: use RequestBodyRef if the spec uses that instead of specifying the body inline
		return nil, ucerr.Errorf("path %s %s operation request body uses a ref, but we haven't yet implemented following request body refs", path, strings.ToUpper(method))
	}
	jsonBody, ok := body.RequestBody.Content["application/json"]
	if !ok {
		return nil, ucerr.Errorf("path %s %s operation request body does specify the application/json content type", path, strings.ToUpper(method))
	}
	if jsonBody.Schema == nil {
		return nil, ucerr.Errorf("path %s %s operation request body does not specify a schema", path, strings.ToUpper(method))
	}
	if jsonBody.Schema.SchemaReference == nil {
		return nil, ucerr.Errorf("path %s %s operation request body schema does not specify a ref", path, strings.ToUpper(method))
	}
	return jsonBody.Schema.SchemaReference, nil
}

// Get the OpenAPI schema ref for the response body of the given path and method.
func getResponseBodySchemaRef(spec *openapi3.Spec, path string, method string) (*openapi3.SchemaReference, error) {
	endpoint, ok := spec.Paths.MapOfPathItemValues[path]
	if !ok {
		return nil, ucerr.Errorf("could not find path %s in the OpenAPI spec", path)
	}
	op, ok := endpoint.MapOfOperationValues[method]
	if !ok {
		return nil, ucerr.Errorf("path %s does not support %s", path, strings.ToUpper(method))
	}
	d, ok := op.Responses.MapOfResponseOrRefValues["200"]
	if !ok {
		d, ok = op.Responses.MapOfResponseOrRefValues["201"]
	}
	if !ok {
		return nil, ucerr.Errorf("path %s %s operation does not have a response defined for status 200 or 201", path, strings.ToUpper(method))
	}
	resp := d.Response
	if resp == nil {
		// TODO: use ResponseRef if the spec uses that instead of specifying the response inline
		return nil, ucerr.Errorf("path %s %s operation default response does not have response info declared inline", path, strings.ToUpper(method))
	}
	schemaOrRef := resp.Content["application/json"].Schema
	if schemaOrRef == nil {
		return nil, ucerr.Errorf("path %s %s operation default response does not specify a schema for the application/json content type", path, strings.ToUpper(method))
	}
	if schemaOrRef.SchemaReference == nil {
		return nil, ucerr.Errorf("path %s %s operation request body schema does not specify a ref", path, strings.ToUpper(method))
	}
	return schemaOrRef.SchemaReference, nil
}

// Given a schema ref for a request/response body, get the name of the key within that request body
// object that contains the data schema for the resource we are getting/creating/updating. E.g.
// given a ref for IdpCreateColumnRequest, this will return "column," since that's the key for the
// UserstoreColumn schema that stores the column resource.
func findDataSchemaInAPISchema(spec *openapi3.Spec, apiBodySchemaRef *openapi3.SchemaReference, expectedResourceSchemaName string, assertSingleProperty bool) (key string, isArray bool, err error) {
	bodySchema, err := schemas.ResolveSchema(spec, &openapi3.SchemaOrRef{SchemaReference: apiBodySchemaRef})
	if err != nil {
		return "", false, ucerr.Errorf("could not resolve request/response body schema ref %s: %v", apiBodySchemaRef.Ref, err)
	}
	if len(bodySchema.Properties) != 1 && assertSingleProperty {
		return "", false, ucerr.Errorf("expected exactly 1 property in request body schema %s. If we have added more properties, we need to double check this codegen code to make sure they get set in POST/PUT requests as necessary", apiBodySchemaRef.Ref)
	}
	for key, prop := range bodySchema.Properties {
		isArray = false
		var schemaRef *openapi3.SchemaReference
		if prop.Schema != nil && prop.Schema.Type != nil && *prop.Schema.Type == openapi3.SchemaTypeArray {
			isArray = true
			if prop.Schema.Items == nil {
				return "", false, ucerr.Errorf("request body schema %s property %s is supposedly an array, but is missing the items schema", apiBodySchemaRef.Ref, key)
			}
			schemaRef = prop.Schema.Items.SchemaReference
		} else {
			schemaRef = prop.SchemaReference
		}
		if schemaRef == nil || schemaRef.Ref == "" {
			// This prop isn't an object (or doesn't use a ref whose name we can check), so keep
			// looking
			continue
		}
		if name := schemas.SchemaNameFromRef(schemaRef.Ref); name == expectedResourceSchemaName {
			return key, isArray, nil
		}
	}
	return "", false, ucerr.Errorf("expected request body schema %s to have a property whose schema is %s, but did not find such a property", apiBodySchemaRef.Ref, expectedResourceSchemaName)
}

// GenResource generates the code to implement a Terraform resource.
func GenResource(ctx context.Context, outDir string, outFilePackage string, spec *openapi3.Spec, resource *config.ResourceConfig) {
	if resource.TypeNameSuffix == "" {
		log.Fatalf("Resource is missing type_name_suffix: %+v", resource)
	}

	if resource.OpenapiSchema == "" {
		log.Fatalf("Resource %s is missing openapi_schema", resource.TypeNameSuffix)
	}

	// Make sure we can find the specified OpenAPI schema name
	schemaOrRef, ok := spec.Components.Schemas.MapOfSchemaOrRefValues[resource.OpenapiSchema]
	if !ok {
		log.Fatalf("Could not find schema %s for resource %s", resource.OpenapiSchema, resource.TypeNameSuffix)
	}
	schema, err := schemas.ResolveSchema(spec, &schemaOrRef)
	if err != nil {
		log.Fatalf("Could not resolve schema %s for resource %s: %v", resource.OpenapiSchema, resource.TypeNameSuffix, err)
	}

	if resource.RestCollectionPath == "" {
		log.Fatalf("Resource %s is missing rest_collection_path", resource.TypeNameSuffix)
	}
	if resource.RestResourcePath == "" {
		log.Fatalf("Resource %s is missing rest_resource_path", resource.TypeNameSuffix)
	}

	isVersionedResource := false
	if _, ok := schema.Properties["version"]; ok {
		isVersionedResource = true
	}

	readResponseBodyRef, err := getResponseBodySchemaRef(spec, resource.RestResourcePath, "get")
	if err != nil {
		log.Fatalf("Could not get GET response body schema ref for resource %s: %v", resource.TypeNameSuffix, err)
	}
	var readResponseDataPropertyName string
	if schemas.SchemaNameFromRef(readResponseBodyRef.Ref) != resource.OpenapiSchema {
		var isArray bool
		readResponseDataPropertyName, isArray, err = findDataSchemaInAPISchema(spec, readResponseBodyRef, resource.OpenapiSchema, false)
		if err != nil {
			log.Fatalf("Could not get GET response body schema key for resource %s: %v", resource.TypeNameSuffix, err)
		}
		if isArray {
			log.Fatalf("Expected GET response property %s for resource %s to be a single object, not an array", readResponseDataPropertyName, resource.TypeNameSuffix)
		}
	}

	createRequestBodySchemaRef, err := getRequestBodySchemaRef(spec, resource.RestCollectionPath, "post")
	if err != nil {
		log.Fatalf("Could not get POST request body schema ref for resource %s: %v", resource.TypeNameSuffix, err)
	}
	createRequestBodySchemaKey, createRequestBodyUsesArray, err := findDataSchemaInAPISchema(spec, createRequestBodySchemaRef, resource.OpenapiSchema, true)
	if err != nil {
		log.Fatalf("Could not get POST request body schema key for resource %s: %v", resource.TypeNameSuffix, err)
	}
	createResponseBodyRef, err := getResponseBodySchemaRef(spec, resource.RestCollectionPath, "post")
	if err != nil {
		log.Fatalf("Could not get POST response body schema ref for resource %s: %v", resource.TypeNameSuffix, err)
	}
	var createResponseBodySchemaKey string
	var createResponseBodyUsesArray bool
	if schemas.SchemaNameFromRef(createResponseBodyRef.Ref) != resource.OpenapiSchema {
		createResponseBodySchemaKey, createResponseBodyUsesArray, err = findDataSchemaInAPISchema(spec, createResponseBodyRef, resource.OpenapiSchema, false)
		if err != nil {
			log.Fatalf("Could not get POST response body schema key for resource %s: %v", resource.TypeNameSuffix, err)
		}
	}

	updateSupported := true
	var updateRequestModel string
	var updateRequestDataPropertyName string
	var updateResponseModel string
	var updateResponseDataPropertyName string
	updateRequestBodySchemaRef, err := getRequestBodySchemaRef(spec, resource.RestResourcePath, "put")
	if err != nil {
		updateSupported = false
	} else {
		updateRequestModel = schemas.GetJSONClientModelStructName(schemas.SchemaNameFromRef(updateRequestBodySchemaRef.Ref))
		var isArray bool
		updateRequestDataPropertyName, isArray, err = findDataSchemaInAPISchema(spec, updateRequestBodySchemaRef, resource.OpenapiSchema, true)
		if err != nil {
			log.Fatalf("Could not get PUT request body schema key for resource %s: %v", resource.TypeNameSuffix, err)
		}
		if isArray {
			log.Fatalf("Resource %s PUT request body takes an array for key %s, which we don't support right now", resource.TypeNameSuffix, updateRequestDataPropertyName)
		}
		updateResponseBodySchemaRef, err := getResponseBodySchemaRef(spec, resource.RestResourcePath, "get")
		if err != nil {
			log.Fatalf("Could not get PUT response body schema ref for resource %s: %v", resource.TypeNameSuffix, err)
		}
		updateResponseModel = schemas.GetJSONClientModelStructName(schemas.SchemaNameFromRef((updateResponseBodySchemaRef.Ref)))
		if schemas.SchemaNameFromRef(updateResponseBodySchemaRef.Ref) != resource.OpenapiSchema {
			var isArray bool
			updateResponseDataPropertyName, isArray, err = findDataSchemaInAPISchema(spec, updateResponseBodySchemaRef, resource.OpenapiSchema, false)
			if err != nil {
				log.Fatalf("Could not get PUT response body schema key for resource %s: %v", resource.TypeNameSuffix, err)
			}
			if isArray {
				log.Fatalf("Expected PUT response property %s for resource %s to be a single object, not an array", updateResponseDataPropertyName, resource.TypeNameSuffix)
			}
		}
	}

	deleteVersionQueryParam := ""
	for _, param := range spec.Paths.MapOfPathItemValues[resource.RestResourcePath].MapOfOperationValues["delete"].Parameters {
		if strings.HasSuffix(param.Parameter.Name, "_version") {
			deleteVersionQueryParam = param.Parameter.Name
			break
		}
	}

	pathParamsToModelFields := map[string]string{}
	if resource.PathParamsToSchemaProperty != nil {
		for pathParam, schemaProperty := range resource.PathParamsToSchemaProperty {
			pathParamsToModelFields[pathParam] = stringutils.ToUpperCamel(schemaProperty)
		}
	} else {
		pathParamsToModelFields["id"] = "ID"
	}

	data := data{
		Package:                     outFilePackage,
		StructName:                  TFTypeNameSuffixToResourceName(resource.TypeNameSuffix),
		NewResourceFuncName:         TFTypeNameSuffixToNewResourceFuncName(resource.TypeNameSuffix),
		TFModelName:                 schemas.GetTFModelStructName(resource.OpenapiSchema),
		JSONClientModelName:         schemas.GetJSONClientModelStructName(resource.OpenapiSchema),
		TFModelToJSONClientFuncName: schemas.GetTFModelToJSONClientFuncName(resource.OpenapiSchema),
		JSONClientModelToTFFuncName: schemas.GetJSONClientModelToTFFuncName(resource.OpenapiSchema),
		TypeNameSuffix:              resource.TypeNameSuffix,
		ResourceDescription:         resource.Description,
		ResourceMarkdownDescription: resource.MarkdownDescription,
		AttributesMap:               schemas.GetAttributesMapName(resource.OpenapiSchema),

		CollectionPath:          resource.RestCollectionPath,
		ResourcePath:            resource.RestResourcePath,
		PathParamsToModelFields: pathParamsToModelFields,
		IsVersionedResource:     isVersionedResource,

		ReadResponseModel:            schemas.GetJSONClientModelStructName(schemas.SchemaNameFromRef(readResponseBodyRef.Ref)),
		ReadResponseDataPropertyName: stringutils.ToUpperCamel(readResponseDataPropertyName),

		CreateRequestModel:             schemas.GetJSONClientModelStructName(schemas.SchemaNameFromRef(createRequestBodySchemaRef.Ref)),
		CreateRequestDataPropertyName:  stringutils.ToUpperCamel(createRequestBodySchemaKey),
		CreateRequestBodyUsesArray:     createRequestBodyUsesArray,
		CreateResponseModel:            schemas.GetJSONClientModelStructName(schemas.SchemaNameFromRef(createResponseBodyRef.Ref)),
		CreateResponseDataPropertyName: stringutils.ToUpperCamel(createResponseBodySchemaKey),
		CreateResponseBodyUsesArray:    createResponseBodyUsesArray,

		UpdateSupported:                updateSupported,
		UpdateRequestModel:             updateRequestModel,
		UpdateRequestDataPropertyName:  stringutils.ToUpperCamel(updateRequestDataPropertyName),
		UpdateResponseModel:            updateResponseModel,
		UpdateResponseDataPropertyName: stringutils.ToUpperCamel(updateResponseDataPropertyName),

		DeleteVersionQueryParam: deleteVersionQueryParam,
	}

	temp := template.Must(template.New("resourceTemplate").Delims("<<", ">>").Parse(temp))
	buf := bytes.NewBuffer([]byte{})
	if err := temp.Execute(buf, data); err != nil {
		log.Fatalf("error executing template: %v", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("error formatting source: %v", err)
	}
	fh, err := os.Create(outDir + "/" + outFilePackage + "/" + resource.TypeNameSuffix + "_resource_generated.go")
	if err != nil {
		log.Fatalf("error opening output file: %v", err)
	}
	if _, err := fh.Write(formatted); err != nil {
		log.Fatalf("error writing output file: %v", err)
	}
	if err := fh.Close(); err != nil {
		log.Fatalf("error closing output file: %v", err)
	}
}
