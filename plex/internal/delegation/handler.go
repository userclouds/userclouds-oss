package delegation

import (
	"errors"
	"fmt"
	"net/http"
	"text/template"
	"time"

	goidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	infraoidc "userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

type handler struct {
	factory provider.Factory
}

// NewHandler returns an http.Handler for delegation workflows
func NewHandler(factory provider.Factory, jwtVerifier auth.Verifier, consoleTenantID uuid.UUID) http.Handler {
	h := handler{factory}

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, &h)

	// Use handle-safe here because this is just a func handler with middleware wrapping it, not
	// actually a sub-handler that needs a trailing slash and StripPrefix.
	// TODO: this granularity of applying middleware seems error prone, but
	// most Plex endpoints don't use auth. Perhaps we should refactor those
	// that do into their own top-level handler so this can be applied in routes.go?
	hb.Handle("/invite", auth.Middleware(jwtVerifier, consoleTenantID).Apply(http.HandlerFunc(h.sendInvite))) // handle-safe
	hb.HandleFunc("/accept", h.acceptInvite)

	hb.HandleFunc(paths.Auth0RedirectCallbackPath, h.oidcProviderCallback)
	hb.HandleFunc(paths.AccountChooser, h.chooseAccount)

	return hb.Build()
}

//go:generate genhandler /delegation

func (h *handler) oidcProviderCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantURL := tenantconfig.MustGetTenantURLString(ctx)

	// load the session ... note that this also validates the state since it's a random GUID
	// TODO: does overloading state here open us up to any strange replay attacks?
	sessionID, err := uuid.FromString(r.URL.Query().Get("state"))
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	// exchange code for token
	tc := tenantconfig.MustGet(ctx)
	p, err := tc.PlexMap.GetActiveProvider()
	if err != nil {
		uchttp.Error(ctx, w, ucerr.New("missing active provider"), http.StatusInternalServerError)
		return
	}

	if p.Type != tenantplex.ProviderTypeAuth0 || !p.Auth0.Redirect {
		uchttp.Error(ctx, w, ucerr.New("callback not valid with type != Auth0, Redirect true"), http.StatusFailedDependency)
		return
	}

	redirect := fmt.Sprintf("%s%s", tenantURL, paths.Auth0RedirectCallbackPath)

	plexApp, _, err := tc.PlexMap.FindAppForClientID(session.ClientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	auth0App, err := p.Auth0.FindProviderApp(plexApp)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	authn, err := infraoidc.NewAuthenticator(ctx, fmt.Sprintf("https://%s/", p.Auth0.Domain), auth0App.ClientID, auth0App.ClientSecret, redirect)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("failed to create new auth: %v", err), http.StatusInternalServerError)
		return
	}

	token, err := authn.Config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("failed to exchange code: %v", err), http.StatusUnauthorized)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		uchttp.Error(ctx, w, ucerr.New("missing id_token field"), http.StatusInternalServerError)
		return
	}

	oidcConfig := &goidc.Config{
		ClientID: authn.Config.ClientID,
	}

	idToken, err := authn.Provider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("failed to verify: %v", err), http.StatusInternalServerError)
		return
	}

	// if we've gotten this far we know we're valid
	// check to see if we have any special permissions (like impersonation)
	accounts, err := ListAccessibleAccounts(ctx, idToken, plexApp.ClientID)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("delegation: %v", err), http.StatusInternalServerError)
		return
	}

	// start to grab claims here so we can annotate impersonation token
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("claims unmarshalling error: %v", err), http.StatusInternalServerError)
		return
	}

	email, ok := claims["email"].(string)
	if !ok {
		uchttp.Error(ctx, w, ucerr.New("no email claim"), http.StatusInternalServerError)
		return
	}

	name, ok := claims["name"].(string)
	if !ok {
		uchttp.Error(ctx, w, ucerr.New("no name claim"), http.StatusInternalServerError)
		return
	}

	nickname, ok := claims["nickname"].(string)
	if !ok {
		uchttp.Error(ctx, w, ucerr.New("no nickname claim"), http.StatusInternalServerError)
		return
	}

	if len(accounts) == 0 {
		ShowBlocked(ctx, w, h.factory, session.ClientID, name, idToken.Subject)
		return
	}

	// don't show the chooser on invite flows, but we don't care about errors
	// (since we don't expect an invite here)
	otpState, err := otp.HasUnusedInvite(ctx, s, session)
	_ = err // lint: errcheck safe

	if len(accounts) > 1 && otpState == nil {
		ds := &storage.DelegationState{
			BaseModel:           ucdb.NewBase(),
			AuthenticatedUserID: idToken.Subject,
		}
		if err := s.SaveDelegationState(ctx, ds); err != nil {
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}

		session.DelegationStateID = ds.ID
		if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}

		ShowChooser(ctx, w, h.factory, session.ClientID, accounts, sessionID, name, email)
		return
	}

	// start to rebuild our own token
	profile := &iface.UserProfile{
		ID: idToken.Subject,
		UserBaseProfile: idp.UserBaseProfile{
			Email:    email,
			Name:     name,
			Nickname: nickname,
		},
	}

	// TODO: if we continue to support this flow, we should store underlying token here, but will require
	// type refactoring to match them somehow
	if err := storage.GenerateUserPlexToken(ctx, tenantURL, &tc, s, profile, session, nil /* TODO: underlying token */); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "CreateTokenError")
		return
	}

	// If this session has an invite, bind it to this user to mark it used and fail only
	// if the invite was already used.
	if err := otp.BindInviteToUser(ctx, s, session, profile.ID, profile.Email, plexApp); err != nil &&
		!errors.Is(err, otp.ErrNoInviteAssociatedWithSession) {
		if errors.Is(err, otp.ErrInviteBoundToAnotherUser) {
			jsonapi.MarshalErrorL(ctx, w, err, "InviteAlreadyBound", jsonapi.Code(http.StatusBadRequest))
			return
		}
		jsonapi.MarshalErrorL(ctx, w, err, "BindInvite")
		return
	}

	redirectURL, err := oidc.NewLoginResponse(ctx, session)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "NewLoginResponse")
		return
	}

	auditlog.Post(ctx, auditlog.NewEntry(profile.ID, auditlog.LoginSuccess,
		auditlog.Payload{"ID": plexApp.ID, "Name": plexApp.Name, "Actor": profile.Email, "Type": "OIDC"}))

	uchttp.Redirect(w, r, redirectURL.RedirectTo, http.StatusFound)
}

func (h *handler) chooseAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantURL := tenantconfig.MustGetTenantURLString(ctx)

	account := r.URL.Query().Get("a")
	sessionID := uuid.Must(uuid.FromString(r.URL.Query().Get("session")))

	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	// check that delegation is part of this session
	if session.DelegationStateID.IsNil() {
		uchttp.Error(ctx, w, ucerr.New("missing delegation state"), http.StatusBadRequest)
		return
	}

	ds, err := s.GetDelegationState(ctx, session.DelegationStateID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	if ds.AuthenticatedUserID == "" {
		uchttp.Error(ctx, w, ucerr.New("empty authed user ID"), http.StatusInternalServerError)
		return
	}

	tc := tenantconfig.MustGet(ctx)

	// check that the auth'd user still has access to the account they chose
	azc, err := newAuthZClient(ctx, tc, session.ClientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	impAzObj, err := azc.GetObjectForName(ctx, objTypeUserID, ds.AuthenticatedUserID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	targetAzObj, err := azc.GetObjectForName(ctx, objTypeUserID, account)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	edges, err := azc.ListEdgesBetweenObjects(ctx, impAzObj.ID, targetAzObj.ID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	if len(edges) != 1 {
		uchttp.Error(ctx, w, ucerr.Errorf("expected 1 edge, got %d", len(edges)), http.StatusInternalServerError)
		return
	}

	if edges[0].EdgeTypeID != edgeTypeImpersonatesID {
		uchttp.Error(ctx, w, ucerr.Errorf("expected delegation edge type, got %v", edges[0].EdgeTypeID), http.StatusInternalServerError)
		return
	}

	// Look up all the data we need to write the impersonated token
	azmc, err := provider.NewActiveManagementClient(ctx, h.factory, session.ClientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	impersonator, err := azmc.GetUser(ctx, ds.AuthenticatedUserID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	profile, err := azmc.GetUser(ctx, account)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	subject := profile.ID
	email := profile.Email
	name := profile.Name
	nickname := profile.Nickname

	newProfile := &iface.UserProfile{
		ID: subject,
		UserBaseProfile: idp.UserBaseProfile{
			Email:    email,
			Name:     name,
			Nickname: nickname,
		},
	}

	impEmail := impersonator.Email
	if email == impersonator.Email {
		impEmail = ""
	}

	// TOOD: if we continue to support this flow, we should store underlying token (maybe? in impersonation it might
	// be riskier?) but will require type refactoring
	plexApp, _, err := tc.PlexMap.FindAppForClientID(session.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "CreateTokenError")
		return
	}
	if err := storage.GenerateImpersonatedUserPlexToken(ctx, tenantURL, &tc, s, newProfile, session, nil /* TODO: underlying token? */, impEmail); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "CreateTokenError")
		return
	}

	redirectURL, err := oidc.NewLoginResponse(ctx, session)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "NewLoginResponse")
		return
	}

	auditlog.Post(ctx, auditlog.NewEntry(profile.ID, auditlog.LoginSuccess,
		auditlog.Payload{"ID": plexApp.ID, "Name": plexApp.Name, "Actor": profile.Email, "Impersonator": ds.AuthenticatedUserID, "Type": "OIDC"}))

	uchttp.Redirect(w, r, redirectURL.RedirectTo, http.StatusFound)
}

func (h *handler) sendInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subjectType := auth.GetSubjectType(ctx)
	if subjectType != authz.ObjectTypeLoginApp && subjectType != m2m.SubjectTypeM2M {
		jsonapi.MarshalErrorL(ctx, w, nil, "UserTokensNotAllowed", jsonapi.Code(http.StatusForbidden))
		return
	}

	tenantURL := tenantconfig.MustGetTenantURL(ctx)

	jwt := auth.GetRawJWT(ctx)
	inviteToEmail := r.URL.Query().Get("to")
	inviteFromAccount := r.URL.Query().Get("from")
	inviteFromName := r.URL.Query().Get("from_name")
	inviteFromEmail := r.URL.Query().Get("from_email")
	inviteForAccount := r.URL.Query().Get("for")
	clientID := r.URL.Query().Get("client_id")

	azmc, err := provider.NewActiveManagementClient(ctx, h.factory, clientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	target, err := azmc.GetUser(ctx, inviteForAccount)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	callbackURL := *tenantURL
	callbackURL.Path = "/delegation/accept"

	// save the delegation invite so we're not trusted URL data
	s := tenantconfig.MustGetStorage(ctx)
	di := &storage.DelegationInvite{
		BaseModel:          ucdb.NewBase(),
		ClientID:           clientID,
		InvitedToAccountID: inviteForAccount,
	}
	if err := s.SaveDelegationInvite(ctx, di); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToSaveInvite")
		return
	}

	plexClient := plex.NewClient(tenantURL.String(), jsonclient.HeaderAuthBearer(jwt), security.PassXForwardedFor())
	inviteReq := plex.SendInviteRequest{
		InviteeEmail:  inviteToEmail,
		InviterUserID: inviteFromAccount,
		InviterName:   inviteFromName,
		InviterEmail:  inviteFromEmail,
		ClientID:      clientID,
		State:         di.ID.String(),
		RedirectURL:   callbackURL.String(),
		InviteText:    fmt.Sprintf("Access %s's TwoMedical account.", target.Name),
		Expires:       time.Now().UTC().Add(time.Hour * 24),
	}

	if err := plexClient.SendInvite(ctx, inviteReq); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToSentInvite")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantURL := tenantconfig.MustGetTenantURLString(ctx)

	state := r.URL.Query().Get("state")
	diID, err := uuid.FromString(state)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	s := tenantconfig.MustGetStorage(ctx)
	di, err := s.GetDelegationInvite(ctx, diID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	tc := tenantconfig.MustGet(ctx)

	redirect := fmt.Sprintf("%s%s", tenantURL, "/delegation/accept")

	plexApp, _, err := tc.PlexMap.FindAppForClientID(di.ClientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	authn, err := infraoidc.NewAuthenticator(ctx, tenantURL, plexApp.ClientID, plexApp.ClientSecret, redirect)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("failed to create new auth: %v", err), http.StatusInternalServerError)
		return
	}

	token, err := authn.Config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("failed to exchange code: %v", err), http.StatusUnauthorized)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		uchttp.Error(ctx, w, ucerr.New("missing id_token field"), http.StatusInternalServerError)
		return
	}

	oidcConfig := &goidc.Config{
		ClientID: authn.Config.ClientID,
	}

	idToken, err := authn.Provider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("failed to verify: %v", err), http.StatusInternalServerError)
		return
	}

	azc, err := newAuthZClient(ctx, tc, di.ClientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	targetObj, err := azc.GetObjectForName(ctx, objTypeUserID, di.InvitedToAccountID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	sourceObj, err := azc.GetObjectForName(ctx, objTypeUserID, idToken.Subject)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	_, err = azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), sourceObj.ID, targetObj.ID, edgeTypeImpersonatesID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	azmc, err := provider.NewActiveManagementClient(ctx, h.factory, di.ClientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	claims := make(map[string]any)
	if err := idToken.Claims(&claims); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	name, ok := claims["name"]
	if !ok {
		uchttp.Error(ctx, w, ucerr.New("missing name claim"), http.StatusInternalServerError)
		return
	}

	target, err := azmc.GetUser(ctx, di.InvitedToAccountID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	data := make(map[string]any)
	data["Name"] = name
	data["TargetName"] = target.Name

	tmp, err := template.New("html").Parse(acceptedTemplate)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	if err := tmp.Execute(w, data); err != nil {
		uclog.Errorf(ctx, "template execution: %v", err)
	}
}

const acceptedTemplate = `
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
      You now have access to {{ .TargetName }}'s account.<br />
	  Please <a href="http://localhost:3000/login">click here</a> to continue.
</div>

</body>
</html>
`
