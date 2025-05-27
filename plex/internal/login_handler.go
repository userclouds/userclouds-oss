package internal

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/authz/ucauthz"
	"userclouds.com/idp"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	infraOidc "userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/addauthn"
	"userclouds.com/plex/internal/loginapp"
	"userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

// loginHandler actually does the real work of taking the user creds and logging them in (or denying it)
// based on plex config, primary & follower config, etc
func (h *handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req plex.LoginRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest")
		return
	}

	uclog.Infof(ctx, "login session for %s", req.Username)

	// Check if username is blocked
	if h.checker.IsCallBlocked(ctx, req.Username) {
		// TODO better API between security events and general errors
		uchttp.ErrorL(ctx, w, ucerr.New("call volume exceeded"), http.StatusForbidden, "UsernameBlocked")
		return
	}

	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetOIDCLoginSession(ctx, req.SessionID)
	if err != nil {
		uclog.Debugf(ctx, "invalid session ID specified: %s", req.SessionID)
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("invalid session ID specified"), "InvalidID", jsonapi.Code(http.StatusBadRequest))
		return
	}

	activeClient, err := provider.NewActiveClient(ctx, h.factory, session.ClientID)
	if err != nil {
		// TODO: differentiate error types (issue #103).
		jsonapi.MarshalErrorL(ctx, w, err, "ProviderInitErr")
		return
	}

	uclog.Debugf(ctx, "trying login to provider %s", activeClient)

	idpResp, err := activeClient.UsernamePasswordLogin(ctx, req.Username, req.Password)
	if err != nil {
		// TODO: translate to better human readable errors.
		if errors.Is(err, jsonclient.ErrIncorrectUsernamePassword) {
			jsonapi.MarshalErrorL(
				ctx,
				w,
				ucerr.Friendlyf(nil, "Incorrect username or password"),
				"UsernamePasswordError",
				jsonapi.Code(http.StatusBadRequest),
			)
		} else {
			jsonapi.MarshalErrorL(ctx, w, err, "UsernamePasswordError")
		}
		return
	}

	if idpResp.Status == idp.LoginStatusMFARequired {
		uclog.Debugf(ctx, "mfa required: %+v", idpResp)

		primaryChannel, err := idpResp.SupportedMFAChannels.FindPrimaryChannel()
		if err != nil {
			// The user does not have a supported primary channel. Create an
			// MFAState with purpose MFAPurposeLoginSetup, and redirect user
			// to UI page for setting up an MFA channel. There is no need
			// to evaluate supported MFA channels after login, since they
			// will be evaluated as part of this step.

			if _, err := h.mfaHandler.createMFAState(ctx, session, storage.MFAPurposeLoginSetup, idpResp.MFAToken, idpResp.MFAProvider, idpResp.SupportedMFAChannels, false); err != nil {
				jsonapi.MarshalErrorL(ctx, w, err, "CreateMFAStateError")
				return
			}

			jsonapi.Marshal(w, getMFALoginResponse(ctx, session.ID, paths.MFAChannelUISubPath))
			return
		}

		// The user has a supported primary channel. Create an MFAState
		// with purpose MFAPurposeLogin, issue a challenge on the primary
		// channel, and redirect user to UI page for answering the challenge.

		mfaState, err := h.mfaHandler.createMFAState(ctx, session, storage.MFAPurposeLogin, idpResp.MFAToken, idpResp.MFAProvider, idpResp.SupportedMFAChannels, idpResp.EvaluateSupportedMFAChannels)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "CreateMFAStateError")
			return
		}

		resp, err := h.mfaHandler.issueMFAChallenge(ctx, s, session, mfaState, primaryChannel.ID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "FailedMFAChallenge")
			return
		}

		jsonapi.Marshal(w, resp)
		return
	}

	if idpResp.Status != idp.LoginStatusSuccess {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("unexpected LoginStatus '%s'", idpResp.Status), "InvalidLoginStatus")
		return
	}

	// username/password login was successful and MFA was not required

	// extract claims

	claims, err := infraOidc.ExtractClaims(idpResp.Claims)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "ExtractClaimsError")
		return
	}
	profile := iface.NewUserProfileFromClaims(*claims)

	tc := tenantconfig.MustGet(ctx)
	app, _, err := tc.PlexMap.FindAppForClientID(session.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FindApp")
		return
	}

	// confirm user can log in

	hasAccess, err := loginapp.CheckLoginAccessForUser(ctx, tc, app, profile.ID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "RestrictedAccessError")
		return
	}
	if !hasAccess {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Friendlyf(nil, "You are not permitted to login to this app"), "RestrictedAccessDenied", jsonapi.Code(http.StatusForbidden))
		return
	}

	// update providers

	// TODO: we should actually update secondary IDP with u/p if MFA is required
	// TODO: we should only update the secondary if a flag is set.
	followerClients, err := provider.NewFollowerManagementClients(ctx, h.factory, session.ClientID)
	if err != nil {
		// TODO (sgarrity 10/23): this shouldn't fail the whole login request (high risk), but this
		// bypass means that we'll potentially get the followers out of sync
		uclog.Errorf(ctx, "failed to get follower clients: %v", err)
	}

	for _, client := range followerClients {
		uclog.Debugf(ctx, "primary login success, writing to secondary(s)")
		err = client.UpdateUsernamePassword(ctx, req.Username, req.Password)
		if err != nil {
			if errors.Is(err, iface.ErrUserNotFound) {
				// User doesn't exist yet, create it
				_, err = client.CreateUserWithPassword(ctx, req.Username, req.Password, *profile)
			}

			if err != nil {
				// Not fatal, ignore.
				uclog.IncrementEvent(ctx, "UsernamePasswordError")
				uclog.Debugf(ctx, "update secondary IDP [%s] username/password failed: %v", client, err)
			}
		}
		// TODO: (#105) update user profile data if user already existed
	}

	// generate plex token and write to audit log
	tu := tenantconfig.MustGetTenantURLString(ctx)
	if err := storage.GenerateUserPlexToken(ctx, tu, &tc, s, profile, session, nil /* no underlying token for u/p login */); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "CreateTokenError")
		return
	}

	auditlog.Post(ctx, auditlog.NewEntry(profile.ID, auditlog.LoginSuccess,
		auditlog.Payload{"ID": app.ID, "Name": app.Name, "Actor": req.Username, "Type": "UC"}))

	// If this session has an invite, bind it to this user to mark it used and fail only
	// if the invite was already used.
	if err := otp.BindInviteToUser(ctx, s, session, profile.ID, req.Username, app); err != nil &&
		!errors.Is(err, otp.ErrNoInviteAssociatedWithSession) {
		if errors.Is(err, otp.ErrInviteBoundToAnotherUser) {
			jsonapi.MarshalErrorL(ctx, w, err, "InviteAlreadyBound", jsonapi.Code(http.StatusBadRequest))
			return
		}

		jsonapi.MarshalErrorL(ctx, w, err, "BindInvite")
		return
	}

	// Check the session to see if we need to add a new authn provider
	prov, err := provider.NewActiveManagementClient(ctx, h.factory, session.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetManagementClient", jsonapi.Code(http.StatusBadRequest))
		return
	}
	addauthn.CheckAndAddAuthnToUser(ctx, session, profile.ID, profile.Email, prov)

	// redirect to MFA channels page if user should reevaluate their settings

	if idpResp.EvaluateSupportedMFAChannels {
		resp, err := h.mfaHandler.startMFAChannelsSession(ctx, session, profile.ID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "StartMFAChannelsSessionError")
			return
		}

		jsonapi.Marshal(w, resp)
		return
	}

	redirectURL, err := oidc.NewLoginResponse(ctx, session)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "NewLoginResponse")
		return
	}

	jsonapi.Marshal(w, redirectURL)
}

func (h *handler) impersonateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tc := tenantconfig.MustGet(ctx)

	var req plex.ImpersonateUserRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest")
		return
	}

	// Check that the incoming request has a valid access token.
	pk, err := ucjwt.LoadRSAPublicKey([]byte(tc.Keys.PublicKey))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToLoadPublicKey", jsonapi.Code(http.StatusBadRequest))
		return
	}
	claims, err := ucjwt.ParseUCClaimsVerified(req.AccessToken, pk)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToParseClaims", jsonapi.Code(http.StatusBadRequest))
		return
	}

	if claims.ImpersonatedBy != "" {
		// Can't impersonate another account using an impersonated token
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest", jsonapi.Code(http.StatusBadRequest))
		return
	}

	// Retrieve the original plex token from the database.
	s := tenantconfig.MustGetStorage(ctx)

	tokenID, err := uuid.FromString(claims.StandardClaims.ID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToParseClaims", jsonapi.Code(http.StatusBadRequest))
		return
	}

	plexToken, err := s.GetPlexToken(ctx, tokenID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetPlexToken", jsonapi.Code(http.StatusBadRequest))
		return
	}

	// Retrieve the target user's profile from the IDP
	activeClient, err := provider.NewActiveManagementClient(ctx, h.factory, plexToken.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetManagementClient", jsonapi.Code(http.StatusBadRequest))
		return
	}

	profile, err := activeClient.GetUser(ctx, req.TargetUserID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetUser", jsonapi.Code(http.StatusBadRequest))
		return
	}

	// Configurable impersonation options
	app, _, err := tc.PlexMap.FindAppForClientID(plexToken.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FindApp")
		return
	}

	ap, err := tc.PlexMap.GetActiveProvider()
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetActiveProvider")
		return
	}

	// Perform authz checks if configured in the PlexApp and we are using the UC provider
	if ap.Type == tenantplex.ProviderTypeUC && (app.ImpersonateUserConfig.CheckAttribute != "" || !app.ImpersonateUserConfig.BypassCompanyAdminCheck) {
		authzClient, err := apiclient.NewAuthzClientFromTenantStateWithClientSecret(ctx, app.ClientID, app.ClientSecret)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "FailedToCreateAuthzClient")
			return
		}

		subjectID, err := uuid.FromString(claims.StandardClaims.Subject)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "InvalidSubjectInClaims", jsonapi.Code(http.StatusBadRequest))
			return
		}

		targetUserID, err := uuid.FromString(profile.ID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "InvalidTargetUserID", jsonapi.Code(http.StatusBadRequest))
			return
		}

		if app.ImpersonateUserConfig.CheckAttribute != "" {

			// Look for the desired attribute between the subject and the target
			resp, err := authzClient.CheckAttribute(ctx, subjectID, targetUserID, app.ImpersonateUserConfig.CheckAttribute)
			if err != nil {
				jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetAttribute")
				return
			}

			if !resp.HasAttribute {
				jsonapi.MarshalErrorL(ctx, w, err, "FailedAuthorizationCheck", jsonapi.Code(http.StatusBadRequest))
				return
			}

		} else {

			// Verify that the subject is an admin of the company
			tenantState := multitenant.MustGetTenantState(ctx)
			if _, err = authzClient.FindEdge(ctx, subjectID, tenantState.CompanyID, ucauthz.AdminEdgeTypeID); err != nil {
				jsonapi.MarshalErrorL(ctx, w, err, "FailedAuthorizationCheck")
				return
			}
		}

	}

	auditlog.Post(ctx, auditlog.NewEntry(claims.StandardClaims.Subject, auditlog.AccountImpersonation,
		auditlog.Payload{"ID": app.ID, "Name": app.Name, "Impersonator ID": claims.StandardClaims.Subject, "Target ID": profile.ID}))

	// Retrieve the original login session and create a new one with the same state
	origSession, err := s.GetOIDCLoginSession(ctx, plexToken.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetSession", jsonapi.Code(http.StatusBadRequest))
		return
	}

	var responseTypes []storage.ResponseType
	for _, rt := range strings.Fields(origSession.ResponseTypes) {
		responseTypes = append(responseTypes, storage.ResponseType(rt))
	}

	redirectURI, err := url.Parse(origSession.RedirectURI)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToParseRedirectURI", jsonapi.Code(http.StatusBadRequest))
		return
	}

	newSessionID, err := storage.CreateOIDCLoginSession(ctx, s, origSession.ClientID, responseTypes, redirectURI, origSession.State, origSession.Scopes)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToCreateSession", jsonapi.Code(http.StatusBadRequest))
		return
	}

	newSession, err := s.GetOIDCLoginSession(ctx, newSessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetSession", jsonapi.Code(http.StatusBadRequest))
		return
	}

	// Generate a new plex token for the impersonation and return a new login response
	underlyingToken := infraOidc.TokenInfo{
		RawIDToken:   plexToken.IDToken,
		AccessToken:  plexToken.AccessToken,
		RefreshToken: plexToken.RefreshToken,
		Claims:       *claims,
	}

	tu := tenantconfig.MustGetTenantURLString(ctx)
	if err := storage.GenerateImpersonatedUserPlexToken(ctx, tu, &tc, s, profile, newSession, &underlyingToken, plexToken.IDPSubject); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGenerateToken", jsonapi.Code(http.StatusBadRequest))
		return
	}

	resp, err := oidc.NewLoginResponse(ctx, newSession)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGenerateResponse", jsonapi.Code(http.StatusBadRequest))
		return
	}

	jsonapi.Marshal(w, resp)
}
