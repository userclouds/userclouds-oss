package cognito

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"maps"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/token"
)

type authClient struct {
	iface.BaseClient

	id   uuid.UUID
	name string

	baseURL string

	cp     *tenantplex.CognitoProvider
	app    *tenantplex.CognitoApp
	awsCfg aws.Config
}

// NewClient creates a Cognito provider client that implements iface.Client.
func NewClient(ctx context.Context,
	id uuid.UUID,
	name string,
	cp *tenantplex.CognitoProvider,
	providerAppID uuid.UUID,
	plexClientID string,
	emailClient email.Client) (iface.Client, error) {
	cfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	ac := &authClient{
		baseURL: fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", cp.AWSConfig.Region, cp.UserPoolID),
		cp:      cp,
		awsCfg:  cfg,
	}

	for i, app := range cp.Apps {
		if app.ID == providerAppID {
			ac.app = &(cp.Apps[i])
		}
	}

	if ac.app == nil {
		return nil, ucerr.Errorf("app with ID %v not found in CognitoProvider %v", providerAppID, id)
	}

	return ac, nil
}

// Secret hash is not a client secret itself, but a base64 encoded hmac-sha256
// hash.
// The actual AWS documentation on how to compute this hash is here:
// https://docs.aws.amazon.com/cognito/latest/developerguide/signing-up-users-in-your-app.html#cognito-user-pools-computing-secret-hash
func computeSecretHash(clientSecret string, username string, clientID string) string {
	mac := hmac.New(sha256.New, []byte(clientSecret))
	mac.Write([]byte(username + clientID))
	hash := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hash
}

func (c *authClient) UsernamePasswordLogin(ctx context.Context, username, password string) (*iface.LoginResponseWithClaims, error) {
	// TODO (sgarrity 2/24): do we care about caching this? is it safe? not immediately clear from docs, although code looks safe
	cc := cognitoidentityprovider.NewFromConfig(c.awsCfg)

	cs, err := c.app.ClientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	params := map[string]string{
		"USERNAME":    username,
		"PASSWORD":    password,
		"SECRET_HASH": computeSecretHash(cs, username, c.app.ClientID),
	}

	authTry := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       types.AuthFlowTypeUserPasswordAuth,
		AuthParameters: params,
		ClientId:       aws.String(c.app.ClientID),
	}

	res, err := cc.InitiateAuth(ctx, authTry)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if res.AuthenticationResult != nil {
		return c.parseLoginResponse(ctx, res.AuthenticationResult)
	}

	switch res.ChallengeName {
	case types.ChallengeNameTypeNewPasswordRequired:
		// TODO (sgarrity 2/24): we're just overriding this for now, but maybe need support in the future?
		crs := map[string]string{}
		maps.Copy(crs, res.ChallengeParameters)
		crs["USERNAME"] = username
		crs["SECRET_HASH"] = computeSecretHash(cs, username, c.app.ClientID)
		crs["NEW_PASSWORD"] = password

		res, err := cc.RespondToAuthChallenge(ctx, &cognitoidentityprovider.RespondToAuthChallengeInput{
			ChallengeName:      res.ChallengeName,
			ChallengeResponses: crs,
			ClientId:           aws.String(c.app.ClientID),
			Session:            res.Session,
		})
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if res.AuthenticationResult != nil {
			return c.parseLoginResponse(ctx, res.AuthenticationResult)
		}

		return nil, ucerr.Errorf("unexpected response from Cognito: %+v", res)

	case types.ChallengeNameTypeSmsMfa:
		return nil, ucerr.Errorf("SMS MFA not yet supported for Cognito provider")
	default:
		return nil, ucerr.Errorf("unexpected challenge from Cognito: %v", res.ChallengeName)
	}
}

func (c *authClient) parseLoginResponse(ctx context.Context, ar *types.AuthenticationResultType) (*iface.LoginResponseWithClaims, error) {
	if ar.IdToken == nil {
		return nil, ucerr.New("no access token returned")
	}

	claims, err := token.ExtractClaimsFromJWT(ctx, c.baseURL, c.app.ClientID, *ar.IdToken)
	if err != nil {
		return nil, ucerr.Errorf("error extracting claims from Auth0 login JWT: %w", err)
	}

	lr := &iface.LoginResponseWithClaims{
		Status: idp.LoginStatusSuccess,
		Claims: claims,
	}

	return lr, nil
}

func (c *authClient) LoginURL(ctx context.Context, sessionID uuid.UUID, app *tenantplex.App) (*url.URL, error) {
	// if we want to implement MITM roles (eg. Auth0.Redirect setting that we built for OM), we can do that here
	return paths.LoginURL(ctx, sessionID)
}

func (c *authClient) Logout(ctx context.Context, redirectURL string) (string, error) {
	// No-op for now. UC IDP & Plex don't yet set cookies so there's nothing to clear.
	// TODO: validate redirect URI is allowed for this app.
	return redirectURL, nil
}

func (c authClient) String() string {
	// NOTE: non-pointer receiver required for this to work on both pointer & non-pointer types
	return fmt.Sprintf("type '%s', name: '%s', id: '%v'", tenantplex.ProviderTypeCognito, c.name, c.id)
}
