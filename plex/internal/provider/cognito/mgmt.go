package cognito

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
)

type mgmtClient struct {
	iface.BaseManagementClient

	id   uuid.UUID
	name string

	cp *tenantplex.CognitoProvider

	awsConfig aws.Config
}

// NewManagementClient returns a new client that is configured to only perform management tasks
// Note that because we don't actually have different IDP auth or anything yet, we just use the same
// object with less config (and client implements both iface.Client & iface.ManagementClient), but
// that will likely change as we get more mature.
func NewManagementClient(ctx context.Context, tc *tenantplex.TenantConfig, id uuid.UUID, name string, cp *tenantplex.CognitoProvider, appID uuid.UUID, appOrgID uuid.UUID) (iface.ManagementClient, error) {
	sesh, err := ucaws.NewFromConfig(ctx, cp.AWSConfig)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &mgmtClient{id: id, name: name, cp: cp, awsConfig: sesh}, nil
}

func (c *mgmtClient) CreateUserWithPassword(ctx context.Context, username, password string, profile iface.UserProfile) (string, error) {
	cc := cognitoidentityprovider.NewFromConfig(c.awsConfig)
	res, err := cc.AdminCreateUser(ctx, &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId: &c.cp.UserPoolID,
		Username:   &username,
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(profile.Email),
			},
		},
		MessageAction: types.MessageActionTypeSuppress,
	})
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	if _, err := cc.AdminSetUserPassword(ctx, &cognitoidentityprovider.AdminSetUserPasswordInput{
		UserPoolId: &c.cp.UserPoolID,
		Username:   &username,
		Password:   &password,
		Permanent:  true,
	}); err != nil {
		return "", ucerr.Wrap(err)
	}

	return *res.User.Username, nil
}

func (c *mgmtClient) CreateUserWithOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, profile iface.UserProfile) (string, error) {
	// TODO (sgarrity 2/24): it's not clear to me that AWS Cognito supports this, but I don't think we need it today?
	return "", nil
}

func (c *mgmtClient) GetUserForOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, email string) (*iface.UserProfile, error) {
	uclog.Infof(ctx, "GetUserForOIDC: %s, %s, %s", provider, issuerURL, oidcSubject)
	cc := cognitoidentityprovider.NewFromConfig(c.awsConfig)

	f := fmt.Sprintf(`sub = "%s"`, oidcSubject)
	res, err := cc.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		Filter:     &f,
		UserPoolId: &c.cp.UserPoolID,
	})
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if len(res.Users) == 0 {
		// TODO (sgarrity 2/24): I haven't yet found a way to look up O365 users from Cognito viaOIDC subject
		users, err := c.ListUsersForEmail(ctx, email, idp.AuthnTypeOIDC)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if len(users) == 0 {
			return nil, ucerr.Wrap(iface.ErrUserNotFound)
		}
		if len(users) > 1 {
			return nil, ucerr.New("multiple users found for email: " + email)
		}
		return &users[0], nil
	} else if len(res.Users) > 1 {
		return nil, ucerr.New("multiple users found for subject: " + oidcSubject)
	}

	u := res.Users[0]
	profile := &iface.UserProfile{
		ID: *u.Username,
	}
	for _, a := range u.Attributes {
		if *a.Name == "email" {
			profile.Email = *a.Value
		}
	}

	return profile, nil
}

func (c *mgmtClient) GetUser(ctx context.Context, userID string) (*iface.UserProfile, error) {
	cc := cognitoidentityprovider.NewFromConfig(c.awsConfig)

	f := fmt.Sprintf(`username = "%s"`, userID)
	res, err := cc.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		Filter:     &f,
		UserPoolId: &c.cp.UserPoolID,
	})
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if len(res.Users) == 0 {
		return nil, ucerr.Wrap(iface.ErrUserNotFound)
	} else if len(res.Users) > 1 {
		return nil, ucerr.New("multiple users found for ID: " + userID)
	}

	u := res.Users[0]
	profile := iface.UserProfile{
		ID: *u.Username,
	}

	for _, a := range u.Attributes {
		if *a.Name == "email" {
			profile.Email = *a.Value
		}
	}

	return &profile, nil
}

func (c *mgmtClient) ListUsersForEmail(ctx context.Context, email string, authnType idp.AuthnType) ([]iface.UserProfile, error) {
	cc := cognitoidentityprovider.NewFromConfig(c.awsConfig)

	f := fmt.Sprintf(`email = "%s"`, email)
	res, err := cc.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		Filter:     &f,
		UserPoolId: &c.cp.UserPoolID,
	})
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	profiles := make([]iface.UserProfile, len(res.Users))
	for i, u := range res.Users {
		profiles[i] = iface.UserProfile{
			ID: *u.Username,
		}
		for _, a := range u.Attributes {
			if *a.Name == "email" {
				profiles[i].Email = *a.Value
			}
		}
	}

	return profiles, nil
}

func (c *mgmtClient) SetEmailVerified(ctx context.Context, userID string, verified bool) error {
	cc := cognitoidentityprovider.NewFromConfig(c.awsConfig)

	if _, err := cc.AdminUpdateUserAttributes(ctx, &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: &c.cp.UserPoolID,
		Username:   &userID,
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email_verified"),
				Value: aws.String(fmt.Sprintf("%t", verified)),
			},
		},
	}); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (c *mgmtClient) UpdateUsernamePassword(ctx context.Context, username, password string) error {
	cc := cognitoidentityprovider.NewFromConfig(c.awsConfig)

	if _, err := cc.AdminSetUserPassword(ctx, &cognitoidentityprovider.AdminSetUserPasswordInput{
		UserPoolId: &c.cp.UserPoolID,
		Username:   &username,
		Password:   &password,
		Permanent:  true,
	}); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// TODO (sgarrity 2/24): unclear that I've captured all attributes here :)
func (c *mgmtClient) UpdateUser(ctx context.Context, userID string, profile iface.UserProfile) error {
	cc := cognitoidentityprovider.NewFromConfig(c.awsConfig)

	if _, err := cc.AdminUpdateUserAttributes(ctx, &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: &c.cp.UserPoolID,
		Username:   &userID,
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(profile.Email),
			},
		},
	}); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// AWS doesn't seem to have a world where there isn't a u/p authn associated, so I believe just
// setting the password is sufficient to enable this
func (c *mgmtClient) AddPasswordAuthnToUser(ctx context.Context, userID, username, password string) error {
	return ucerr.Wrap(c.UpdateUsernamePassword(ctx, userID, password))
}

func (c *mgmtClient) AddOIDCAuthnToUser(ctx context.Context, userID string, provider oidc.ProviderType, issuerURL string, oidcSubject string) error {
	// TODO (sgarrity 2/24): it's not clear to me that AWS Cognito supports this, but I don't think we need it today?
	return nil
}

func (c mgmtClient) String() string {
	// NOTE: non-pointer receiver required for this to work on both pointer & non-pointer types
	return fmt.Sprintf("type '%s', name: '%s', id: '%v'", tenantplex.ProviderTypeUC, c.name, c.id)
}

type cognitoIdentity struct {
	UserID       string `json:"userId"`
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
	Issuer       string `json:"issuer"` // NB: this appears to usually / always be null?
	Primary      bool   `json:"primary"`
	DateCreated  int64  `json:"dateCreated"`
}

func (c mgmtClient) ListUsers(ctx context.Context) ([]iface.UserProfile, error) {
	cc := cognitoidentityprovider.NewFromConfig(c.awsConfig)

	// TODO (sgarrity 3/24): solve the excessive reallocation problem when we fix pagination below
	var profiles []iface.UserProfile

	// TODO (sgarrity 3/24): plumb pagination farther up the stack
	var paginationToken *string
	for {
		res, err := cc.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
			UserPoolId:      &c.cp.UserPoolID,
			PaginationToken: paginationToken,
		})
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		for _, u := range res.Users {
			profile := iface.UserProfile{
				ID: *u.Username,
			}
			for _, a := range u.Attributes {
				if *a.Name == "email" {
					profile.Email = *a.Value
				}
				if *a.Name == "identities" {
					// because of course AWS packs a JSON blob into this string
					ids := []cognitoIdentity{}
					if err := json.Unmarshal([]byte(*a.Value), &ids); err != nil {
						return nil, ucerr.Wrap(err)
					}

					for _, id := range ids {
						if id.ProviderType == "OIDC" && id.ProviderName == "Microsoft" {
							profile.Authns = append(profile.Authns, idp.UserAuthn{
								AuthnType:     idp.AuthnTypeOIDC,
								OIDCProvider:  oidc.ProviderTypeMicrosoft,
								OIDCIssuerURL: oidc.MicrosoftIssuerURL, // because this is null from cognito :shrug:
								OIDCSubject:   id.UserID,
							})
						}
					}
				}
			}

			// if no OIDC profile, create a username password one
			// TODO (sgarrity 3/24): this is an assumption from the cognito API, but leaves open the case
			// where a user should have both u/p authn and OIDC authn ... so we'll rely on our merge flow then
			if len(profile.Authns) == 0 {
				profile.Authns = append(profile.Authns, idp.UserAuthn{
					AuthnType: idp.AuthnTypePassword,
					Username:  profile.ID,
					// generate a random one since we can't get it from cognito
					Password: fmt.Sprintf("placeholder-%s", crypto.MustRandomHex(16)),
				})
			}

			profiles = append(profiles, profile)
		}

		if res.PaginationToken == nil {
			break
		}
		paginationToken = res.PaginationToken
	}

	return profiles, nil
}
