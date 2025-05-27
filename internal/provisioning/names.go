package provisioning

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
)

// TenantBaseResourceNames contains names of per tenant resources required for any provisioned tenant
type TenantBaseResourceNames struct {
	IDPDBName     string
	IDPDBUserName string
}

// NewTenantBaseResourceNames populates resource names for basic resources
func NewTenantBaseResourceNames(tenantID uuid.UUID) TenantBaseResourceNames {
	var t TenantBaseResourceNames
	tenantIDStr := cleanTenantID(tenantID)
	// TODO: rename to "tenantdb_<uuid>"
	t.IDPDBName = fmt.Sprintf("idp_%s", tenantIDStr)
	t.IDPDBUserName = fmt.Sprintf("tenant_%s", tenantIDStr)
	return t
}

func cleanTenantID(tenantID uuid.UUID) string {
	return strings.ReplaceAll(tenantID.String(), "-", "")
}
