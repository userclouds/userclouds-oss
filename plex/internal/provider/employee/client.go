package employee

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/tenantconfig"
)

type authClient struct {
	iface.BaseClient
	id              uuid.UUID
	name            string
	consoleEndpoint service.Endpoint
}

// NewClient creates an Employee provider client that implements iface.Client.
func NewClient(ctx context.Context, id uuid.UUID, name string, consoleEP service.Endpoint) (iface.Client, error) {
	return &authClient{
		id:              id,
		name:            name,
		consoleEndpoint: consoleEP,
	}, nil
}

func (c *authClient) LoginURL(ctx context.Context, sessionID uuid.UUID, employeeApp *tenantplex.App) (*url.URL, error) {
	// We redirect to UC console /employee/login endpoint, passing in the OIDC login session ID, the state
	// stored in the OIDC login session, and the tenant ID. The latter is used to derive the redirect URL
	// in the console /employee/login handler, after the employee has successfully logged into UC console plex.

	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	ts := multitenant.MustGetTenantState(ctx)

	query := url.Values{}
	query.Add("request_session_id", sessionID.String())
	query.Add("request_state", session.State)
	query.Add("request_tenant_id", ts.ID.String())

	consoleURL := c.consoleEndpoint.URL()
	consoleURL.Path = "/auth/employee/login"
	consoleURL.RawQuery = query.Encode()
	return consoleURL, nil
}

func (c *authClient) Logout(ctx context.Context, redirectURL string) (string, error) {
	// redirectURL is validated in plex logout handler
	return redirectURL, nil
}

func (c authClient) String() string {
	// NOTE: non-pointer receiver required for this to work on both pointer & non-pointer types
	return fmt.Sprintf("type '%s', name: '%s', id: '%v', console endpoint: '%s'",
		tenantplex.ProviderTypeEmployee, c.name, c.id, c.consoleEndpoint.BaseURL())
}
