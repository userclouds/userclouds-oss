package tokenizer

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp/policy"
)

type accessPolicyTemplateFunc func(ctx context.Context, authZClient *authz.Client, policyContext policy.AccessPolicyContext, params string) bool

type accessPolicyRecord struct {
	fun      accessPolicyTemplateFunc
	template policy.AccessPolicyTemplate
}

// nativeTemplates is a map of access policy templates that have been implemented in native code. Map goes from template ID to version to record
var nativeTemplates = map[uuid.UUID]map[int]accessPolicyRecord{
	policy.AccessPolicyTemplateAllowAll.ID:       {0: {fun: accessPolicyTemplateAllowAll, template: policy.AccessPolicyTemplateAllowAll}},
	policy.AccessPolicyTemplateCheckAttribute.ID: {0: {fun: accessPolicyTemplateCheckAttribute, template: policy.AccessPolicyTemplateCheckAttribute}},
	policy.AccessPolicyTemplateDenyAll.ID:        {0: {fun: accessPolicyTemplateDenyAll, template: policy.AccessPolicyTemplateDenyAll}},
}

func accessPolicyTemplateAllowAll(ctx context.Context, authZClient *authz.Client, policyContext policy.AccessPolicyContext, params string) bool {
	return true
}

func accessPolicyTemplateCheckAttribute(ctx context.Context, authZClient *authz.Client, policyContext policy.AccessPolicyContext, params string) bool {

	parameters := make(map[string]string)
	if err := json.Unmarshal([]byte(params), &parameters); err != nil {
		return false
	}

	var id1, id2 uuid.UUID
	var err error

	userIDUsage, ok := parameters["userIDUsage"]
	if ok {
		if userIDUsage == "id1" {
			id1String, ok := policyContext.User["id"].(string)
			if !ok {
				return false // TODO: figure out how to log or propagate these errors back to the caller
			}
			id1, err = uuid.FromString(id1String)
			if err != nil {
				return false
			}
		} else if userIDUsage == "id2" {
			id2String, ok := policyContext.User["id"].(string)
			if !ok {
				return false
			}
			id2, err = uuid.FromString(id2String)
			if err != nil {
				return false
			}
		}
	}

	if id1.IsNil() {
		if id1String, ok := parameters["id1"]; ok {
			id1, err = uuid.FromString(id1String)
			if err != nil {
				return false
			}
		} else {
			return false
		}
	}

	if id2.IsNil() {
		if id2String, ok := parameters["id2"]; ok {
			id2, err = uuid.FromString(id2String)
			if err != nil {
				return false
			}
		} else {
			return false
		}
	}

	attribute, ok := parameters["attribute"]
	if !ok {
		attribute, ok = policyContext.Client["attribute"].(string)
		if !ok {
			return false
		}
	}

	ret, err := authZClient.CheckAttribute(ctx, id1, id2, attribute)
	if err != nil {
		return false
	}

	return ret.HasAttribute
}

func accessPolicyTemplateDenyAll(ctx context.Context, authZClient *authz.Client, policyContext policy.AccessPolicyContext, params string) bool {
	return false
}
