package samlconfig

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/universe"
)

// KeySecretName generates a specific AWS SM name for a given SAML IDP key
func KeySecretName(ctx context.Context, tenantID, appID uuid.UUID) string {
	return fmt.Sprintf("%s/plex/saml_idp_key/%v/%v", universe.Current(), tenantID, appID)
}
