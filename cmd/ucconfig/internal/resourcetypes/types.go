package resourcetypes

import (
	"context"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
)

// ResourceType represents a Terraform provider resource type.
type ResourceType struct {
	TerraformTypeSuffix string
	// The returned structs must have an ID field.
	ListResources func(ctx context.Context, client *idp.Client) ([]any, error)
	// References maps attribute paths (e.g. "columns.column") to the terraform type suffix of the
	// resource UUIDs they reference
	References map[string]string
	// WriteAttributesExternally maps attribute paths (e.g.
	// "child_object.attr_with_long_value") to file extensions (e.g. ".js") for
	// attributes whose values should be stored in an external file, rather than
	// inline in a generated manifest
	WriteAttributesExternally map[string]string
	// OmitAttributes is a list of resource attributes that should be omitted from manifest
	// generation, e.g. if there are superfluous details returned in ListWhateverObject API calls
	// that we don't need to include
	OmitAttributes []string
}

func getColumnRetentions(ctx context.Context, client *idp.Client, dt userstore.DataLifeCycleState) ([]any, error) {
	response, err := client.ListColumns(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	out := []any{}
	for _, column := range response.Data {
		retentionResponse, err := client.GetColumnRetentionDurationsForColumn(ctx, dt, column.ID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, retention := range retentionResponse.RetentionDurations {
			if retention.UseDefault {
				// Skip default retentions inherited from elsewhere
				continue
			}
			out = append(out, retention)
		}
	}
	return out, nil
}

var omitRetentionAttributes = []string{
	// The default is computed from tenant/purpose retention settings (i.e. subject to change, would
	// create TF drift) and only used display in the console UI.
	"default_duration",
	// purpose_id and purpose_name should really be a ResourceID instead. In general, we try to
	// exclude names to avoid creating drift if the referenced resources change
	"purpose_name",
}

// ResourceTypes lists the resource types supported by ucconfig.
var ResourceTypes = []ResourceType{
	{
		TerraformTypeSuffix: "userstore_column_data_type",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			response, err := client.ListDataTypes(ctx)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []any{}
			for _, dataType := range response.Data {
				out = append(out, dataType)
			}
			return out, nil
		},
		OmitAttributes: []string{
			// these are fields that are derived in the backend from the provided name
			"camel_case_name",
			"struct_name",
		},
		References: map[string]string{
			"composite_attributes.fields.data_type": "userstore_column_data_type",
		},
	},
	{
		TerraformTypeSuffix: "userstore_column",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			response, err := client.ListColumns(ctx)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []any{}
			for _, column := range response.Data {
				out = append(out, column)
			}
			return out, nil
		},
		OmitAttributes: []string{
			// these are fields that are derived in the backend from the provided name
			"camel_case_name",
			"struct_name",
		},
		References: map[string]string{
			"data_type": "userstore_column_data_type",
		},
	},
	{
		TerraformTypeSuffix: "userstore_column_soft_deleted_retention_duration",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			return getColumnRetentions(ctx, client, userstore.DataLifeCycleStateSoftDeleted)
		},
		References: map[string]string{
			"column_id":  "userstore_column",
			"purpose_id": "userstore_purpose",
		},
		OmitAttributes: omitRetentionAttributes,
	},
	{
		TerraformTypeSuffix: "userstore_accessor",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			response, err := client.ListAccessors(ctx, false)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []any{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
		References: map[string]string{
			"access_policy":       "access_policy",
			"columns.column":      "userstore_column",
			"columns.transformer": "transformer",
			"purposes":            "userstore_purpose",
		},
	},
	{
		TerraformTypeSuffix: "userstore_mutator",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			response, err := client.ListMutators(ctx, false)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []any{}
			for _, d := range response.Data {
				// TODO: this is a temporary workaround to suppress duplicate validator/normalizer fields
				// Only keep normalizer, remove when server no longer returns validator
				for i := range d.Columns {
					d.Columns[i].Validator = userstore.ResourceID{}
				}
				out = append(out, d)
			}
			return out, nil
		},
		References: map[string]string{
			"access_policy":      "access_policy",
			"columns.column":     "userstore_column",
			"columns.normalizer": "transformer",
		},
	},
	{
		TerraformTypeSuffix: "userstore_purpose",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			response, err := client.ListPurposes(ctx)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []any{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
	},
	{
		TerraformTypeSuffix: "access_policy",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			response, err := client.TokenizerClient.ListAccessPolicies(ctx, false)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []any{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
		References: map[string]string{
			"components.policy":   "access_policy",
			"components.template": "access_policy_template",
		},
	},
	{
		TerraformTypeSuffix: "access_policy_template",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			response, err := client.TokenizerClient.ListAccessPolicyTemplates(ctx, false)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []any{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
		WriteAttributesExternally: map[string]string{
			// The JS function should be stored separately to facilitate linting
			// and syntax highlighting
			"function": ".js",
		},
	},
	{
		TerraformTypeSuffix: "transformer",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			response, err := client.TokenizerClient.ListTransformers(ctx)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			out := []any{}
			for _, d := range response.Data {
				out = append(out, d)
			}
			return out, nil
		},
		References: map[string]string{
			"input_data_type":  "userstore_column_data_type",
			"output_data_type": "userstore_column_data_type",
		},
		WriteAttributesExternally: map[string]string{
			// The JS function should be stored separately to facilitate linting
			// and syntax highlighting
			"function": ".js",
		},
	},
}

// GetByTerraformTypeSuffix returns the ResourceType with the given TerraformTypeSuffix.
func GetByTerraformTypeSuffix(s string) *ResourceType {
	for _, rt := range ResourceTypes {
		if rt.TerraformTypeSuffix == s {
			return &rt
		}
	}
	return nil
}

// ValidateTerraformTypeSuffix returns true if s is a supported Terraform type
// suffix.
func ValidateTerraformTypeSuffix(s string) bool {
	return GetByTerraformTypeSuffix(s) != nil
}
