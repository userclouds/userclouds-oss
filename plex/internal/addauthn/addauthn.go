package addauthn

import (
	"context"

	"userclouds.com/idp"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

// CheckAndAddAuthnToUser checks the session for an authn to add to the user's account, and if found, adds it
func CheckAndAddAuthnToUser(ctx context.Context, session *storage.OIDCLoginSession, userID string, email string, prov iface.ManagementClient) {

	// Check first whether user has granted permission to add this authn to their account
	if session.AddAuthnProviderData.PermissionToAdd && !session.AddAuthnProviderData.DoNotAdd {
		if email == session.AddAuthnProviderData.NewProviderEmail {

			// Since the emails match and ther user has given permission, add the new authn provider to the user
			if session.AddAuthnProviderData.NewProviderAuthnType == idp.AuthnTypePassword {
				if err := prov.AddPasswordAuthnToUser(ctx,
					userID,
					session.AddAuthnProviderData.NewProviderEmail,
					session.AddAuthnProviderData.NewProviderPassword); err != nil {

					// Log the error but allow login to continue; if we don't, then
					// the user will be stuck until they refresh the session
					uclog.Errorf(ctx, "Failed to add password authn to user: %v", err)
				}

			} else if session.AddAuthnProviderData.NewProviderAuthnType == idp.AuthnTypeOIDC {
				if err := prov.AddOIDCAuthnToUser(ctx,
					userID,
					session.AddAuthnProviderData.NewOIDCProvider,
					session.AddAuthnProviderData.NewProviderOIDCIssuerURL,
					session.AddAuthnProviderData.NewProviderOIDCSubject); err != nil {

					// Same as above
					uclog.Errorf(ctx, "Failed to add OIDC authn to user: %v", err)
				}
			}

		} else {
			uclog.Errorf(ctx, "User tried adding OIDC authn to password account then logged in with different email")
		}
	}
}

// CheckForExistingAccounts checks whether there are any existing accounts with the same email address and if so, returns a list of the authn providers for those accounts
func CheckForExistingAccounts(ctx context.Context, session *storage.OIDCLoginSession, email string, passwordOrSubject string, s *storage.Storage, prov iface.ManagementClient) ([]string, error) {

	// Check first whether user has specified that they want to create a new account rather than add this authn to an existing account
	if session != nil && !session.AddAuthnProviderData.DoNotAdd {

		// Look for any users with the same email address
		var authnTypeToLookFor idp.AuthnType
		if session.OIDCProvider == oidc.ProviderTypeNone {
			// Only look for social-based accounts during password-based account creation
			authnTypeToLookFor = idp.AuthnTypeOIDC
		} else {
			// Look for both password- and social-based accounts during OIDC-based account creation
			authnTypeToLookFor = idp.AuthnTypeAll
		}
		users, err := prov.ListUsersForEmail(ctx, email, authnTypeToLookFor)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if len(users) > 0 {
			tc := tenantconfig.MustGet(ctx)

			authns := []string{}
			for _, user := range users {
				for _, authn := range user.Authns {
					// For each existing account, add its authn type to the list
					if authn.AuthnType == idp.AuthnTypePassword {
						authns = append(authns, string(idp.AuthnTypePassword))

					} else if authn.AuthnType == idp.AuthnTypeOIDC {
						p, err := tc.OIDCProviders.GetProviderForIssuerURL(authn.OIDCProvider, authn.OIDCIssuerURL)
						if err != nil {
							return nil, ucerr.Wrap(err)
						}

						authns = append(authns, p.GetName())
					}
				}
			}

			// Save the prospective new authn provider to the session
			if session.OIDCProvider == oidc.ProviderTypeNone {
				session.AddAuthnProviderData = storage.AddAuthnProviderData{
					NewProviderAuthnType: idp.AuthnTypePassword,
					NewProviderEmail:     email,
					NewProviderPassword:  passwordOrSubject,
				}

			} else {
				session.AddAuthnProviderData = storage.AddAuthnProviderData{
					NewProviderAuthnType:     idp.AuthnTypeOIDC,
					NewOIDCProvider:          session.OIDCProvider,
					NewProviderEmail:         email,
					NewProviderOIDCIssuerURL: session.OIDCIssuerURL,
					NewProviderOIDCSubject:   passwordOrSubject,
				}
			}
			if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
				return nil, ucerr.Wrap(err)
			}

			return authns, nil
		}
	}
	return nil, nil
}
