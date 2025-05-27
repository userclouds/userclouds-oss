package defaults

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
)

var defaultAccessPolicyTemplatesByID = map[uuid.UUID]storage.AccessPolicyTemplate{}
var defaultAccessPolicyTemplates = []storage.AccessPolicyTemplate{
	accessPolicyTemplateAllowAll,
	accessPolicyTemplateCheckAttribute,
	accessPolicyTemplateDenyAll,
}

// GetDefaultAccessPolicyTemplates returns the default access policy templates
func GetDefaultAccessPolicyTemplates() []storage.AccessPolicyTemplate {
	var accessPolicyTemplates []storage.AccessPolicyTemplate
	accessPolicyTemplates = append(accessPolicyTemplates, defaultAccessPolicyTemplates...)
	return accessPolicyTemplates
}

// IsDefaultAccessPolicyTemplate returns true if id refers to a default access policy template
func IsDefaultAccessPolicyTemplate(id uuid.UUID) bool {
	if _, found := defaultAccessPolicyTemplatesByID[id]; found {
		return true
	}

	return false
}

func init() {
	for _, dapt := range defaultAccessPolicyTemplates {
		if _, found := defaultAccessPolicyTemplatesByID[dapt.ID]; found {
			panic(fmt.Sprintf("access policy template %s has conflicting id %v", dapt.Name, dapt.ID))
		}
		defaultAccessPolicyTemplatesByID[dapt.ID] = dapt
	}
}
