package delegation

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	goidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/tenantconfig"
)

const objTypeUser = "_delegation_user"
const edgeTypeImpersonates = "_impersonates"

var objTypeUserID = uuid.Must(uuid.FromString("dc0c03ad-464e-4a41-a15a-1a3f9213dfbd"))
var edgeTypeImpersonatesID = uuid.Must(uuid.FromString("07a95f96-5190-461f-917d-b5c6957dc153"))

// ListAccessibleAccounts takes an IDToken and checks to see what accounts
// the owner has access too. By default (if the owner's object doesn't exist in AuthZ),
// the owner is given access to his/her own account.
func ListAccessibleAccounts(ctx context.Context, idToken *goidc.IDToken, clientID string) ([]string, error) {
	tc := tenantconfig.MustGet(ctx)
	azc, err := newAuthZClient(ctx, tc, clientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	userTypeID, err := azc.FindObjectTypeID(ctx, objTypeUser)
	if err != nil {
		// TODO: we probably should provision these with a UI checkbox or something?
		ot, err := azc.CreateObjectType(ctx, objTypeUserID, objTypeUser)
		if err != nil {
			return nil, ucerr.Errorf("failed to create user type ID: %v", err)
		}
		userTypeID = ot.ID
	}

	impersonateTypeID, err := azc.FindEdgeTypeID(ctx, edgeTypeImpersonates)
	if err != nil {
		// TODO: we probably should provision these with a UI checkbox or something?
		et, err := azc.CreateEdgeType(ctx, edgeTypeImpersonatesID, userTypeID, userTypeID, edgeTypeImpersonates, nil)
		if err != nil {
			return nil, ucerr.Errorf("failed to create impersonate type ID: %v", err)
		}
		impersonateTypeID = et.ID
	}

	user, err := azc.GetObjectForName(ctx, userTypeID, idToken.Subject)
	if err != nil {
		// TODO: is it ok to provision this here? or should we require external provisioning?
		user, err = azc.CreateObject(ctx, uuid.Must(uuid.NewV4()), userTypeID, idToken.Subject)
		if err != nil {
			return nil, ucerr.Errorf("failed to get user for %v: %v", idToken.Subject, err)
		}

		// if we created the object, then create an edge allowing us to impersonate ourselves (by default)
		if _, err := azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), user.ID, user.ID, impersonateTypeID); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	var impersonatableIDs []uuid.UUID
	cursor := pagination.CursorBegin
	for {
		resp, err := azc.ListEdgesOnObject(ctx, user.ID, authz.Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			return nil, ucerr.Errorf("failed to get user %v edges: %v", user.ID, err)
		}
		for _, edge := range resp.Data {
			if edge.EdgeTypeID != impersonateTypeID {
				continue
			}
			if edge.SourceObjectID != user.ID {
				continue
			}
			impersonatableIDs = append(impersonatableIDs, edge.TargetObjectID)
		}
		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}

	var accounts []string
	for _, id := range impersonatableIDs {
		acc, err := azc.GetObject(ctx, id)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if acc.Alias != nil { // TODO this seems like a potential bug as UC user objects don't have aliases
			accounts = append(accounts, *acc.Alias)
		}
	}

	return accounts, nil
}

type account struct {
	Username string
	Name     string
}

// ShowChooser shows a "nice" account chooser UI
func ShowChooser(ctx context.Context, w http.ResponseWriter, factory provider.Factory, clientID string, accounts []string, sessionID uuid.UUID, impersonatorName, impersonatorEmail string) {
	uclog.Debugf(ctx, "show chooser: %v", accounts)

	// get full account metadata from provider
	azmc, err := provider.NewActiveManagementClient(ctx, factory, clientID)
	if err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToAMClient")
		return
	}

	data := make(map[string]any)
	var names []account
	for _, a := range accounts {
		u, err := azmc.GetUser(ctx, a)
		if err != nil {
			uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToGetUser")
			return
		}
		names = append(names, account{Username: a, Name: u.Name})
	}
	data["Accounts"] = names

	data["AccountChooserPath"] = fmt.Sprintf("/delegation%s", paths.AccountChooser)
	data["SessionID"] = sessionID.String()
	data["ImpersonatorName"] = impersonatorName
	data["ImpersonatorEmail"] = impersonatorEmail

	tmp, err := template.New("html").Parse(chooserTemplate)
	if err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToParseTemplate")
		return
	}

	if err := tmp.Execute(w, data); err != nil {
		uclog.Errorf(ctx, "template execution: %v", err)
	}
}

const chooserTemplate = `
<html>
<body style="display: flex;">

<div
      style="
        margin: auto;
        border: 1px solid;
        border-radius: 5px;
        padding: 10 25 25 15px;
      "
    >
      <h3>Welcome, {{ .ImpersonatorName }}</h3>
      Please choose the account you'd like to access:
	{{- range $account := .Accounts }}
	<div style="display: block; margins: 10px; ">
		<a href="{{ $.AccountChooserPath }}?a={{ $account.Username }}&session={{ $.SessionID }}&u={{ $.ImpersonatorEmail }}">
			{{ $account.Name }}
		</a>
	</div>
	{{- end }}
</div>

</body>
</html>
`

// ShowBlocked tells you that you can't access your account, and who to ask for help
func ShowBlocked(ctx context.Context, w http.ResponseWriter, factory provider.Factory, clientID string, blockedName, blockedUser string) {
	tc := tenantconfig.MustGet(ctx)
	azc, err := newAuthZClient(ctx, tc, clientID)
	if err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToGetAuthZClient")
		return
	}

	obj, err := azc.GetObjectForName(ctx, objTypeUserID, blockedUser)
	if err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToGetBlockedUser")
		return
	}

	var accounts []string
	cursor := pagination.CursorBegin
	for {
		resp, err := azc.ListEdgesOnObject(ctx, obj.ID, authz.Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToGetEdges")
			return
		}
		for _, edge := range resp.Data {
			if edge.EdgeTypeID != edgeTypeImpersonatesID {
				continue
			}
			if edge.SourceObjectID == obj.ID {
				// should never happen if we're here
				uclog.Errorf(ctx, "ShowBlocked found user access edge ID %v", edge.ID)
				continue
			}
			if edge.TargetObjectID != obj.ID {
				uclog.Errorf(ctx, "ShowBlocked got to a weird place %v", edge.ID)
				continue
			}

			src, err := azc.GetObject(ctx, edge.SourceObjectID)
			if err != nil {
				uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToGetSourceObj")
				return
			}

			if src.Alias != nil { // TODO this seems like a bug for user clouds IDP
				accounts = append(accounts, *src.Alias)
			}
		}
		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}

	// get full account metadata from provider
	azmc, err := provider.NewActiveManagementClient(ctx, factory, clientID)
	if err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToGetAMClient")
		return
	}

	data := make(map[string]any)

	var names []string
	for _, a := range accounts {
		u, err := azmc.GetUser(ctx, a)
		if err != nil {
			uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToGetUser")
			return
		}
		names = append(names, u.Name)
	}

	data["Name"] = blockedName
	data["Accounts"] = strings.Join(names, " or ")

	tmp, err := template.New("html").Parse(blockedTemplate)
	if err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToParseTemplate")
		return
	}

	if err := tmp.Execute(w, data); err != nil {
		uclog.Errorf(ctx, "template execution: %v", err)
	}
}

const blockedTemplate = `
<html>
<body style="display: flex;">

<div
      style="
        margin: auto;
        border: 1px solid;
        border-radius: 5px;
        padding: 10 25 25 15px;
      "
    >
      <h3>Hello, {{ .Name }}</h3>
      Sorry, you don't have login access at the moment.<br />
	  Please contact {{ .Accounts }} for help.
</div>

</body>
</html>
`

func newAuthZClient(ctx context.Context, tc tenantplex.TenantConfig, clientID string) (*authz.Client, error) {
	app, _, err := tc.PlexMap.FindAppForClientID(clientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	azc, err := apiclient.NewAuthzClientFromTenantStateWithClientSecret(ctx, app.ClientID, app.ClientSecret)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return azc, nil
}
