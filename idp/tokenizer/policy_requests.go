package tokenizer

import (
	"userclouds.com/idp/policy"
	"userclouds.com/infra/ucerr"
)

// CreateAccessPolicyRequest creates a new AP
type CreateAccessPolicyRequest struct {
	AccessPolicy policy.AccessPolicy `json:"access_policy"`
}

//go:generate genvalidate CreateAccessPolicyRequest

// UpdateAccessPolicyRequest updates an AP by creating a new version
type UpdateAccessPolicyRequest struct {
	AccessPolicy policy.AccessPolicy `json:"access_policy"`
}

//go:generate genvalidate UpdateAccessPolicyRequest

// CreateAccessPolicyTemplateRequest creates a new AP Template
type CreateAccessPolicyTemplateRequest struct {
	AccessPolicyTemplate policy.AccessPolicyTemplate `json:"access_policy_template"`
}

//go:generate genvalidate CreateAccessPolicyTemplateRequest

// UpdateAccessPolicyTemplateRequest updates an AP Template
type UpdateAccessPolicyTemplateRequest struct {
	AccessPolicyTemplate policy.AccessPolicyTemplate `json:"access_policy_template"`
}

//go:generate genvalidate UpdateAccessPolicyTemplateRequest

// CreateTransformerRequest creates a new GP
type CreateTransformerRequest struct {
	Transformer policy.Transformer `json:"transformer"`
}

//go:generate genvalidate CreateTransformerRequest

// UpdateTransformerRequest updates a Transformer by creating a new version
type UpdateTransformerRequest struct {
	Transformer policy.Transformer `json:"transformer"`
}

//go:generate genvalidate UpdateTransformerRequest

// TestTransformerRequest lets you run an unsaved policy for testing
type TestTransformerRequest struct {
	Transformer policy.Transformer `json:"transformer"`
	Data        string             `json:"data"`
}

// Validate implements Validateable
func (o TestTransformerRequest) Validate() error {
	blankName := false
	if o.Transformer.Name == "" {
		blankName = true
		o.Transformer.Name = "temp"
	}
	if err := o.Transformer.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if blankName {
		o.Transformer.Name = ""
	}
	return nil
}

// TestTransformerResponse is the response to a TestTransformer call
type TestTransformerResponse struct {
	Value string         `json:"value"`
	Debug map[string]any `json:"debug,omitempty"`
}

// TestAccessPolicyRequest lets you run an unsaved policy with a given context for testing
type TestAccessPolicyRequest struct {
	AccessPolicy policy.AccessPolicy        `json:"access_policy"`
	Context      policy.AccessPolicyContext `json:"context"`
}

// Validate implements Validateable
func (o TestAccessPolicyRequest) Validate() error {
	blankName := false
	if o.AccessPolicy.Name == "" {
		blankName = true
		o.AccessPolicy.Name = "temp"
	}
	if err := o.AccessPolicy.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if blankName {
		o.AccessPolicy.Name = ""
	}

	return nil
}

// TestAccessPolicyTemplateRequest lets you run an unsaved policy template with a given context for testing
type TestAccessPolicyTemplateRequest struct {
	AccessPolicyTemplate policy.AccessPolicyTemplate `json:"access_policy_template"`
	Context              policy.AccessPolicyContext  `json:"context"`
	Params               string                      `json:"params"`
}

// Validate implements Validateable
func (o TestAccessPolicyTemplateRequest) Validate() error {
	blankName := false
	if o.AccessPolicyTemplate.Name == "" {
		blankName = true
		o.AccessPolicyTemplate.Name = "temp"
	}
	if err := o.AccessPolicyTemplate.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if blankName {
		o.AccessPolicyTemplate.Name = ""
	}

	return nil
}

// TestAccessPolicyResponse is the response to a TestAccessPolicy call
type TestAccessPolicyResponse struct {
	Allowed bool           `json:"allowed"`
	Debug   map[string]any `json:"debug,omitempty"`
}

// CreateSecretRequest is the request to create a secret
type CreateSecretRequest struct {
	Secret policy.Secret `json:"secret"`
}
