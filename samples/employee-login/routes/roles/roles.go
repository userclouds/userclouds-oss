package roles

import (
	"net/http"
	"net/url"
	"os"

	"github.com/userclouds/userclouds/samples/employee-login/routes/templates"
	"github.com/userclouds/userclouds/samples/employee-login/session"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/uchttp"
)

func getRBACClient(accessToken string) (*authz.RBACClient, error) {
	tenantURL := os.Getenv("UC_TENANT_BASE_URL")
	tokenEndpointURL, err := url.Parse(tenantURL)
	if err != nil {
		return nil, err
	}
	tokenEndpointURL.Path = "/oidc/token"

	tokenSource := jsonclient.TokenSource(oidc.ClientCredentialsTokenSource{
		TokenURL:        tokenEndpointURL.String(),
		ClientID:        os.Getenv("PLEX_CLIENT_ID"),
		ClientSecret:    os.Getenv("PLEX_CLIENT_SECRET"),
		CustomAudiences: []string{},
		SubjectJWT:      accessToken,
	})

	authzClient, err := authz.NewClient(tenantURL, authz.JSONClient(tokenSource))
	if err != nil {
		return nil, err
	}
	rbacClient := authz.NewRBACClient(authzClient)
	return rbacClient, nil
}

type role struct {
	GroupID  string
	RoleType string
	Role     string
}

type templateData struct {
	Name    string
	Picture string
	Roles   []role
}

// Handler renders UI for viewing employee roles
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rbacClient, err := getRBACClient(session.GetAccessToken(r))
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	employee, err := rbacClient.GetUser(ctx, session.GetUserID(r))
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	memberships, err := employee.GetMemberships(ctx)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	p := session.GetProfile(r)
	data := templateData{
		Name:    p["name"].(string),
		Picture: p["picture"].(string),
		Roles:   []role{},
	}

	for _, m := range memberships {
		var r role
		r.GroupID = m.Group.ID.String()
		switch m.Role {
		case ucauthz.AdminRole:
			r.RoleType = "Organization"
			r.Role = "Admin"
		case ucauthz.MemberRole:
			r.RoleType = "Organization"
			r.Role = "Member"
		case idpAuthz.EdgeTypeNameUserGroupPolicyFullAccess:
			r.RoleType = "Policy"
			r.Role = "Full Access"
		case idpAuthz.EdgeTypeNameUserGroupPolicyReadAccess:
			r.RoleType = "Policy"
			r.Role = "Read Access"
		default:
			continue
		}
		data.Roles = append(data.Roles, r)
	}
	templates.RenderTemplate(ctx, w, "roles", data)
}
