package appimport

import (
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantplex"
)

// useful reference here:
// https://auth0.com/docs/get-started/applications/application-grant-types#auth0-extension-grants
func mapAuth0GrantTypesToUC(grantTypes []string) (tenantplex.GrantTypes, error) {
	var gts tenantplex.GrantTypes
	for _, gt := range grantTypes {
		switch gt {
		case "implicit":
			fallthrough
		case "authorization_code":
			fallthrough
		case "client_credentials":
			fallthrough
		case "password":
			fallthrough
		case "refresh_token":
			fallthrough
		case "urn:ietf:params:oauth:grant-type:device_code":
			gts = append(gts, tenantplex.GrantType(gt))

		case "http://auth0.com/oauth/grant-type/password-realm":
			gts = append(gts, tenantplex.GrantTypePassword)

		case "http://auth0.com/oauth/grant-type/mfa-oob":
			fallthrough
		case "http://auth0.com/oauth/grant-type/mfa-otp":
			fallthrough
		case "http://auth0.com/oauth/grant-type/mfa-recovery-code":
			fallthrough
		case "http://auth0.com/oauth/grant-type/passwordless/otp":
			gts = append(gts, tenantplex.GrantTypeMFA)

		default:
			return nil, ucerr.Errorf("unknown auth0 grant type %v", gt)
		}
	}
	return gts, nil
}
