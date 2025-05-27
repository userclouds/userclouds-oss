package internal

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/jsonapi"
	infraoidc "userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/addauthn"
	"userclouds.com/plex/internal/loginapp"
	"userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/reactdev"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

type mfaLoginResponseQueryParam struct {
	key   string
	value string
}
type mfaHandler struct {
	factory provider.Factory
}

func newMFAHandler(factory provider.Factory) (*mfaHandler, http.Handler) {
	h := &mfaHandler{factory: factory}

	hb := builder.NewHandlerBuilder()
	// MFA handlers

	// issue an MFA challenge
	hb.HandleFunc("/challenge", h.mfaIssueChallengeHandler)

	// clear the primary MFA channel
	hb.HandleFunc("/clearprimarychannel", h.mfaClearPrimaryChannelHandler)

	// create an MFA channel
	hb.HandleFunc("/createchannel", h.mfaCreateChannelHandler)

	// delete an MFA channel
	hb.HandleFunc("/deletechannel", h.mfaDeleteChannelHandler)

	// make an MFA channel the primary channel
	hb.HandleFunc("/makeprimarychannel", h.mfaMakePrimaryChannelHandler)

	// get all supported MFA channels
	hb.HandleFunc("/getchannels", h.mfaGetChannelsHandler)

	// get settings for MFA challenge page
	hb.HandleFunc("/getsubmitsettings", h.mfaGetSubmitSettingsHandler)

	// end an MFA configuration session
	hb.HandleFunc("/endconfiguration", h.mfaEndConfigurationHandler)

	// submit an MFA challenge response
	hb.HandleFunc("/submit", h.mfaSubmitHandler)

	// confirm receipt of new recovery code by user
	hb.HandleFunc("/confirmrecoverycode", h.mfaConfirmRecoveryCodeHandler)

	// reissue recovery code for user
	hb.HandleFunc("/reissuerecoverycode", h.mfaReissueRecoveryCodeHandler)

	return h, hb.Build()
}

func getMFALoginResponse(ctx context.Context, sessionID uuid.UUID, subPath string, queryParams ...mfaLoginResponseQueryParam) *plex.LoginResponse {
	redirectTo := reactdev.UIBaseURL(ctx)
	redirectTo.Path = redirectTo.Path + subPath
	query := url.Values{}
	query.Set("session_id", sessionID.String())

	for _, queryParam := range queryParams {
		query.Set(queryParam.key, queryParam.value)
	}

	redirectTo.RawQuery = query.Encode()

	uclog.Debugf(ctx, "redirecting to: %v", redirectTo)
	return &plex.LoginResponse{RedirectTo: redirectTo.String()}
}

func getMFARecoveryCodeResponse(ctx context.Context, sessionID uuid.UUID, recoveryCode string) *plex.LoginResponse {
	queryParam := mfaLoginResponseQueryParam{key: "recovery_code", value: recoveryCode}
	return getMFALoginResponse(ctx, sessionID, paths.MFAShowRecoveryCodeUISubPath, queryParam)
}

func getMFASession(ctx context.Context, s *storage.Storage, sessionID uuid.UUID) (*storage.OIDCLoginSession, *storage.MFAState, error) {
	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	if session.MFAStateID.IsNil() {
		return nil, nil, ucerr.New("no mfa state associated with session")
	}

	mfaState, err := s.GetMFAState(ctx, session.MFAStateID)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	return session, mfaState, nil
}

func saveMFAState(ctx context.Context, s *storage.Storage, mfaState *storage.MFAState, channelID uuid.UUID, challengeState storage.MFAChallengeState) error {
	mfaState.ChannelID = channelID
	mfaState.ChallengeState = challengeState
	return ucerr.Wrap(s.SaveMFAState(ctx, mfaState))
}

func canConfigureMFA(mfaState *storage.MFAState, client iface.Client) error {
	if !client.SupportsMFAConfiguration() {
		return ucerr.Errorf("provider client does not support MFA configuration: '%s'", client.String())
	}

	if !mfaState.Purpose.CanModify() {
		return ucerr.Errorf("cannot configure MFA with MFAState purpose '%v'", mfaState.Purpose)
	}

	return nil
}

func (h *mfaHandler) createMFAState(
	ctx context.Context,
	session *storage.OIDCLoginSession,
	purpose storage.MFAPurpose,
	mfaToken string,
	mfaProvider uuid.UUID,
	supportedChannels infraoidc.MFAChannels,
	evaluateSupportedChannels bool) (*storage.MFAState, error) {
	mfaState := &storage.MFAState{
		BaseModel:                 ucdb.NewBase(),
		SessionID:                 session.ID,
		Token:                     mfaToken,
		Provider:                  mfaProvider,
		ChannelID:                 uuid.Nil,
		SupportedChannels:         supportedChannels,
		Purpose:                   purpose,
		ChallengeState:            storage.MFAChallengeStateNoChallenge,
		EvaluateSupportedChannels: evaluateSupportedChannels,
	}

	s := tenantconfig.MustGetStorage(ctx)

	if err := s.SaveMFAState(ctx, mfaState); err != nil {
		return nil, ucerr.Wrap(err)
	}

	session.MFAStateID = mfaState.ID
	if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return mfaState, nil
}

func (h *mfaHandler) issueMFAChallenge(ctx context.Context,
	s *storage.Storage,
	session *storage.OIDCLoginSession,
	mfaState *storage.MFAState,
	channelID uuid.UUID) (*plex.LoginResponse, error) {
	channel, err := mfaState.SupportedChannels.FindChannel(channelID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// if the channel is restricted, mark the challenge as expired and redirect
	// to MFA code submit page

	if restricted, _ := session.MFAChannelStates.IsRestricted(channel); restricted {
		if err := saveMFAState(ctx, s, mfaState, channelID, storage.MFAChallengeStateExpired); err != nil {
			return nil, ucerr.Wrap(err)
		}

		return getMFALoginResponse(ctx, session.ID, paths.MFACodeUISubPath), nil
	}

	// issue a challenge for the specified channel

	client, err := provider.NewClientForProviderID(ctx, h.factory, session.ClientID, mfaState.Provider)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	updatedChannel, err := client.MFAChallenge(ctx, mfaState.Token, channel)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if updatedChannel.ID != channelID {
		return nil, ucerr.Errorf("channel id of challenge channel '%v' does not match request id '%v'", updatedChannel.ID, channelID)
	}

	// update the channels, mark the challenge as issued, and redirect to MFA code submit page

	if err := mfaState.SupportedChannels.UpdateChannel(*updatedChannel); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if err := saveMFAState(ctx, s, mfaState, updatedChannel.ID, storage.MFAChallengeStateIssued); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return getMFALoginResponse(ctx, session.ID, paths.MFACodeUISubPath), nil
}

func (h *mfaHandler) startMFAChannelsSession(ctx context.Context, session *storage.OIDCLoginSession, userID string) (*plex.LoginResponse, error) {
	// retrieve MFA channel information from active provider

	activeClient, err := provider.NewActiveClient(ctx, h.factory, session.ClientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	idpResp, err := activeClient.MFAGetChannels(ctx, userID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// create MFA state with this channel information

	if _, err := h.createMFAState(ctx, session, storage.MFAPurposeConfigure, idpResp.MFAToken, idpResp.MFAProvider, idpResp.SupportedMFAChannels, false); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// generate redirect to MFA channels page

	return getMFALoginResponse(ctx, session.ID, paths.MFAChannelUISubPath), nil
}

func (h *mfaHandler) mfaConfirmRecoveryCodeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	query := r.URL.Query()
	sessionID, err := uuid.FromString(query.Get("session_id"))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest")
		return
	}

	session, mfaState, err := getMFASession(ctx, s, sessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	// return to channel configuration page if purpose is configure

	if mfaState.Purpose == storage.MFAPurposeConfigure {
		jsonapi.Marshal(w, getMFALoginResponse(ctx, session.ID, paths.MFAChannelUISubPath))
		return
	}

	// start a configuration session if requested

	if mfaState.EvaluateSupportedChannels {
		plexToken, err := s.GetPlexToken(ctx, session.PlexTokenID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "GetPlexTokenError")
			return
		}

		resp, err := h.startMFAChannelsSession(ctx, session, plexToken.IDPSubject)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "StartMFAChannelsSessionError")
			return
		}

		jsonapi.Marshal(w, resp)
		return
	}

	// login is complete - clear the MFA state; the session being committed as
	// part of generating the login response

	session.MFAStateID = uuid.Nil

	redirectURL, err := oidc.NewLoginResponse(ctx, session)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "NewLoginResponse")
		return
	}

	jsonapi.Marshal(w, redirectURL)
}

// MFAClearPrimaryChannelRequest represents the request for clearing the primary MFA channel
type MFAClearPrimaryChannelRequest struct {
	SessionID uuid.UUID `json:"session_id"`
}

func (h *mfaHandler) mfaClearPrimaryChannelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	var req MFAClearPrimaryChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "BadRequest")
		return
	}

	// look up MFA session and provider client

	session, mfaState, err := getMFASession(ctx, s, req.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	client, err := provider.NewClientForProviderID(ctx, h.factory, session.ClientID, mfaState.Provider)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetClient")
		return
	}

	// make sure we are in right state to do this

	if err := canConfigureMFA(mfaState, client); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "MFAConfigureStateError")
		return
	}

	// clear the primary channel

	channels, err := client.MFAClearPrimaryChannel(ctx, mfaState.Token)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedMFAClearPrimaryChannel")
		return
	}

	// update the channels, save MFA state, and redirect to MFA channels page

	mfaState.SupportedChannels = *channels

	if err := saveMFAState(ctx, s, mfaState, uuid.Nil, storage.MFAChallengeStateNoChallenge); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "SaveMFASessionError")
		return
	}

	jsonapi.Marshal(w, getMFALoginResponse(ctx, session.ID, paths.MFAChannelUISubPath))
}

// MFACreateChannelRequest represents the request for creating an MFA channel
type MFACreateChannelRequest struct {
	SessionID     uuid.UUID                `json:"session_id"`
	ChannelType   infraoidc.MFAChannelType `json:"mfa_channel_type"`
	ChannelTypeID string                   `json:"mfa_channel_type_id"`
}

func (h *mfaHandler) mfaCreateChannelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	var req MFACreateChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "BadRequest")
		return
	}

	// look up MFA session and provider client

	session, mfaState, err := getMFASession(ctx, s, req.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	client, err := provider.NewClientForProviderID(ctx, h.factory, session.ClientID, mfaState.Provider)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetClient")
		return
	}

	// make sure we are in right state to do this

	if err := canConfigureMFA(mfaState, client); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "MFAConfigureStateError")
		return
	}

	// create channel

	channel, err := client.MFACreateChannel(ctx, mfaState.Token, req.ChannelType, req.ChannelTypeID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedMFACreateChannel")
		return
	}

	// update the channels, mark the challenge as issued, and redirect to MFA code submit page

	mfaState.SupportedChannels.Channels[channel.ID] = *channel

	if err := saveMFAState(ctx, s, mfaState, channel.ID, storage.MFAChallengeStateIssued); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "SaveMFASessionError")
		return
	}

	jsonapi.Marshal(w, getMFALoginResponse(ctx, session.ID, paths.MFACodeUISubPath))
}

// MFAChannelRequest represents the request when deleting an MFA channel or making it primary, or when
// issuing a challenge for the channel.
type MFAChannelRequest struct {
	SessionID uuid.UUID `json:"session_id"`
	ChannelID uuid.UUID `json:"mfa_channel_id"`
}

func (h *mfaHandler) mfaDeleteChannelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	var req MFAChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "BadRequest")
		return
	}

	// look up MFA session and provider client

	session, mfaState, err := getMFASession(ctx, s, req.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	client, err := provider.NewClientForProviderID(ctx, h.factory, session.ClientID, mfaState.Provider)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetClient")
		return
	}

	// make sure we are in right state to do this

	if err := canConfigureMFA(mfaState, client); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "MFAConfigureStateError")
		return
	}

	// look up channel and delete

	channel, err := mfaState.SupportedChannels.FindChannel(req.ChannelID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFAChannelError")
		return
	}

	channels, err := client.MFADeleteChannel(ctx, mfaState.Token, channel)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedMFADeleteChannel")
		return
	}

	// update the channels, save MFA state, and redirect to MFA channels page

	mfaState.SupportedChannels = *channels

	if err := saveMFAState(ctx, s, mfaState, uuid.Nil, storage.MFAChallengeStateNoChallenge); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "SaveMFASessionError")
		return
	}

	jsonapi.Marshal(w, getMFALoginResponse(ctx, session.ID, paths.MFAChannelUISubPath))
}

// UIMFAChannelType is a UI-centric representation of an MFA channel type
type UIMFAChannelType struct {
	ChannelType infraoidc.MFAChannelType `json:"mfa_channel_type"`
	CanCreate   bool                     `json:"can_create"`
}

// UIMFAChannel is a UI-centric representation of an MFA channel
type UIMFAChannel struct {
	ChannelID                uuid.UUID                `json:"mfa_channel_id"`
	ChannelType              infraoidc.MFAChannelType `json:"mfa_channel_type"`
	ChannelDescription       string                   `json:"mfa_channel_description"`
	Primary                  bool                     `json:"primary"`
	CanChallenge             bool                     `json:"can_challenge"`
	ChallengeBlockExpiration string                   `json:"challenge_block_expiration"`
	CanDelete                bool                     `json:"can_delete"`
	CanMakePrimary           bool                     `json:"can_make_primary"`
	CanReissue               bool                     `json:"can_reissue"`
}

// MFAChannelsResponse is returned to the UI for interacting with the set of supported MFA channels for a user
type MFAChannelsResponse struct {
	Purpose            storage.MFAPurpose                       `json:"mfa_purpose"`
	ChannelTypes       []UIMFAChannelType                       `json:"mfa_channel_types"`
	Channels           []UIMFAChannel                           `json:"mfa_channels"`
	AuthenticatorTypes []infraoidc.AuthenticatorTypeDescription `json:"mfa_authenticator_types"`
	CanDisable         bool                                     `json:"can_disable"`
	CanDismiss         bool                                     `json:"can_dismiss"`
	CanEnable          bool                                     `json:"can_enable"`
	MaxChannels        int                                      `json:"max_mfa_channels"`
	Description        string                                   `json:"description"`
}

func makeMFAChannelsResponse(session *storage.OIDCLoginSession, mfaState *storage.MFAState, mfaRequired bool) MFAChannelsResponse {
	resp := MFAChannelsResponse{
		Purpose:     mfaState.Purpose,
		CanDismiss:  mfaState.Purpose.IsConfiguration(),
		MaxChannels: infraoidc.MaxMFAChannels,
	}

	switch mfaState.Purpose {
	case storage.MFAPurposeLogin:
		resp.Description = "Pick a channel for MFA challenge"
	case storage.MFAPurposeLoginSetup:
		resp.Description = "MFA is required - configure an MFA channel"
	case storage.MFAPurposeConfigure:
		resp.Description = "Configure your MFA channel(s)"
	}

	canChallenge := mfaState.Purpose.CanChallenge()
	canModify := mfaState.Purpose.CanModify()
	shouldMask := mfaState.Purpose.ShouldMask()
	hasPrimaryChannel := mfaState.SupportedChannels.HasPrimaryChannel()
	resp.CanDisable = canModify && !mfaRequired && hasPrimaryChannel

	totalNonRecoveryCodeChannels := 0

	for _, c := range mfaState.SupportedChannels.Channels {
		isRecoveryCode := c.ChannelType.IsRecoveryCode()
		if !isRecoveryCode {
			totalNonRecoveryCodeChannels++
		}

		channel := UIMFAChannel{
			ChannelID:          c.ID,
			ChannelType:        c.ChannelType,
			ChannelDescription: c.GetChannelDescription(shouldMask),
			Primary:            c.Primary,
			CanChallenge:       canChallenge,
			CanDelete:          canModify && !c.Primary && !isRecoveryCode,
			CanMakePrimary:     canModify && !isRecoveryCode,
			CanReissue:         canModify && isRecoveryCode && hasPrimaryChannel,
		}

		restricted, expiration := session.MFAChannelStates.IsRestricted(c)
		if restricted {
			channel.ChallengeBlockExpiration = expiration.Format(time.UnixDate)
		}

		resp.Channels = append(resp.Channels, channel)
	}

	resp.CanEnable = canModify && !hasPrimaryChannel && totalNonRecoveryCodeChannels > 0

	canCreate := canModify && totalNonRecoveryCodeChannels < infraoidc.MaxMFAChannels
	for ct := range mfaState.SupportedChannels.ChannelTypes {
		if ct == infraoidc.MFAAuthenticatorChannel {
			resp.AuthenticatorTypes = infraoidc.GetAuthenticatorTypes()
		}
		channelType := UIMFAChannelType{
			ChannelType: ct,
			CanCreate:   canCreate && !ct.IsRecoveryCode(),
		}
		resp.ChannelTypes = append(resp.ChannelTypes, channelType)
	}

	return resp
}

func (h *mfaHandler) mfaGetChannelsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)
	tc := tenantconfig.MustGet(ctx)

	// look up the MFA state for the passed in session

	query := r.URL.Query()
	sessionID, err := uuid.FromString(query.Get("session_id"))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest")
		return
	}

	session, mfaState, err := getMFASession(ctx, s, sessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	if anyUnverified := mfaState.SupportedChannels.ClearUnverified(); anyUnverified {
		if err := saveMFAState(ctx, s, mfaState, uuid.Nil, storage.MFAChallengeStateNoChallenge); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "SaveMFASessionError")
			return
		}
	}

	mfaRequired, _, err := tc.GetMFASettings(session.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASettingsError")
		return
	}

	resp := makeMFAChannelsResponse(session, mfaState, mfaRequired)
	jsonapi.Marshal(w, resp)
}

func (h *mfaHandler) mfaEndConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	// look up the session

	query := r.URL.Query()
	sessionID, err := uuid.FromString(query.Get("session_id"))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest")
		return
	}

	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetOIDCLoginSessionError")
		return
	}

	// clear the MFA state - this will be committed as part of generating
	// the login response if this was part of a login session, or explicitly
	// otherwise

	session.MFAStateID = uuid.Nil

	if session.PlexTokenID != uuid.Nil {
		// the configuration session was started as part of a login flow

		redirectURL, err := oidc.NewLoginResponse(ctx, session)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "NewLoginResponse")
			return
		}
		jsonapi.Marshal(w, redirectURL)
		return
	}

	// the configuration session was not started as part of a login flow

	if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "SaveOIDCLoginSessionError")
		return
	}

	resp := plex.LoginResponse{RedirectTo: session.RedirectURI}
	jsonapi.Marshal(w, &resp)
}

func (h *mfaHandler) mfaIssueChallengeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	var req MFAChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "BadRequest")
		return
	}

	// look up MFA session

	session, mfaState, err := getMFASession(ctx, s, req.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	// issue an MFA challenge with the associated challenge code via the selected channel,
	// and redirect to code submit page.

	resp, err := h.issueMFAChallenge(ctx, s, session, mfaState, req.ChannelID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedMFAChallenge")
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *mfaHandler) mfaMakePrimaryChannelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	var req MFAChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "BadRequest")
		return
	}

	// look up MFA session and provider client

	session, mfaState, err := getMFASession(ctx, s, req.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	client, err := provider.NewClientForProviderID(ctx, h.factory, session.ClientID, mfaState.Provider)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetClient")
		return
	}

	// make sure we are in right state to do this

	if err := canConfigureMFA(mfaState, client); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "MFAConfigureStateError")
		return
	}

	// look up channel and make it primary

	channel, err := mfaState.SupportedChannels.FindChannel(req.ChannelID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFAChannelError")
		return
	}

	channels, err := client.MFAMakePrimaryChannel(ctx, mfaState.Token, channel)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedMFAMakePrimaryChannel")
		return
	}

	// update the channels, save MFA state, and redirect to MFA channels page

	mfaState.SupportedChannels = *channels

	if err := saveMFAState(ctx, s, mfaState, uuid.Nil, storage.MFAChallengeStateNoChallenge); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "SaveMFASessionError")
		return
	}

	jsonapi.Marshal(w, getMFALoginResponse(ctx, session.ID, paths.MFAChannelUISubPath))
}

// MFASubmitSettings defines the json response to an MFA submit settings request
type MFASubmitSettings struct {
	ChannelType              infraoidc.MFAChannelType `json:"channel_type"`
	ChannelID                uuid.UUID                `json:"channel_id"`
	ChallengeStatus          string                   `json:"challenge_status"`
	ChallengeDescription     string                   `json:"challenge_description"`
	ChallengeBlockExpiration string                   `json:"challenge_block_expiration"`
	CanChangeChannel         bool                     `json:"can_change_channel"`
	CanReissueChallenge      bool                     `json:"can_reissue_challenge"`
	CanSubmitCode            bool                     `json:"can_submit_code"`
	RegistrationLink         string                   `json:"registration_link"`
	RegistrationQRCode       string                   `json:"registration_qr_code"`
	CustomerServiceLink      string                   `json:"customer_service_link"`
	Purpose                  storage.MFAPurpose       `json:"mfa_purpose"`
}

func (h *mfaHandler) mfaGetSubmitSettingsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	// look up the MFA state for the passed in session

	query := r.URL.Query()
	sessionID, err := uuid.FromString(query.Get("session_id"))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest")
		return
	}

	session, mfaState, err := getMFASession(ctx, s, sessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	// make sure a challenge has been issued

	if !mfaState.ChallengeState.HasBeenIssued() {
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("No challenge has been issued"), "InvalidRequest")
		return
	}

	// generate response based on MFAChannelState, MFAState, and channel

	channel, err := mfaState.SupportedChannels.FindChannel(mfaState.ChannelID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFAChannelError")
		return
	}

	canChangeChannel := false
	initialCanChangeChannel := false
	switch mfaState.Purpose {
	case storage.MFAPurposeLogin:
		// allow the user to try another channel if there is more
		// than one available
		if len(mfaState.SupportedChannels.Channels) > 1 {
			canChangeChannel = true
		}
	case storage.MFAPurposeLoginSetup, storage.MFAPurposeConfigure:
		// allow the user to configure a different channel
		canChangeChannel = true
		initialCanChangeChannel = true
	}

	// TODO: get customer service link from configuration and show if there has been a failure

	resp := MFASubmitSettings{
		CanChangeChannel:     initialCanChangeChannel,
		CanSubmitCode:        true,
		CanReissueChallenge:  false,
		ChallengeDescription: channel.GetChallengeDescription(mfaState.Purpose.ShouldMask(), mfaState.ChallengeState.IsFirstChallenge()),
		ChannelType:          channel.ChannelType,
		ChannelID:            channel.ID,
		Purpose:              mfaState.Purpose,
	}

	if restricted, expiration := session.MFAChannelStates.IsRestricted(channel); restricted {
		resp.CanChangeChannel = canChangeChannel
		resp.CanSubmitCode = false
		resp.CanReissueChallenge = true
		resp.ChallengeBlockExpiration = expiration.Format(time.UnixDate)
		resp.ChallengeStatus = "Max number of attempts has been reached"
	} else if mfaState.ChallengeState == storage.MFAChallengeStateExpired {
		resp.CanChangeChannel = canChangeChannel
		resp.CanSubmitCode = false
		resp.CanReissueChallenge = channel.ChannelType.CanReissueChallenge()
		resp.ChallengeStatus = "Code is no longer valid"
	} else if mfaState.ChallengeState == storage.MFAChallengeStateBadChallenge {
		resp.CanChangeChannel = canChangeChannel
		resp.CanReissueChallenge = channel.ChannelType.CanReissueChallenge()
		resp.ChallengeStatus = "Code does not match"
	}

	if registrationLink, registrationQRCode, ok := channel.GetRegistrationInfo(); ok {
		resp.RegistrationLink = registrationLink
		resp.RegistrationQRCode = registrationQRCode
	}

	jsonapi.Marshal(w, resp)
}

func (h *mfaHandler) mfaReissueRecoveryCodeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	var req MFAChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "BadRequest")
		return
	}

	// look up MFA session and provider client

	session, mfaState, err := getMFASession(ctx, s, req.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	client, err := provider.NewClientForProviderID(ctx, h.factory, session.ClientID, mfaState.Provider)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetClient")
		return
	}

	// make sure we are in right state to do this

	if err := canConfigureMFA(mfaState, client); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "MFAConfigureStateError")
		return
	}

	// find channel and reissue recovery code

	channel, err := mfaState.SupportedChannels.FindChannel(req.ChannelID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFAChannelError")
		return
	}

	updatedChannel, err := client.MFAReissueRecoveryCode(ctx, mfaState.Token, channel)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedMFAReissueRecoveryCode")
		return
	}

	// update the channel, save MFA state, and redirect to show new recovery code

	if err := mfaState.SupportedChannels.UpdateChannel(*updatedChannel); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "UpdateMFAChannelError")
		return
	}

	if err := saveMFAState(ctx, s, mfaState, uuid.Nil, storage.MFAChallengeStateNoChallenge); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "SaveMFASessionError")
		return
	}

	jsonapi.Marshal(w, getMFARecoveryCodeResponse(ctx, session.ID, updatedChannel.ChannelTypeID))
}

// MFASubmitRequest defines the JSON request to the MFA submit handler.
type MFASubmitRequest struct {
	SessionID                 uuid.UUID `json:"session_id"`
	MFACode                   string    `json:"mfa_code"`
	EvaluateSupportedChannels bool      `json:"evaluate_supported_channels"`
}

// mfaSubmitHandler accepts the code and passes it on to the IDP (Auth0 or UC or ...)
func (h *mfaHandler) mfaSubmitHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := tenantconfig.MustGetStorage(ctx)

	var req MFASubmitRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "BadRequest")
		return
	}

	// look up MFA session

	session, mfaState, err := getMFASession(ctx, s, req.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFASessionError")
		return
	}

	// if a challenge cannot be successfully responded to, redirect to code submit page

	if mfaState.ChallengeState != storage.MFAChallengeStateIssued && mfaState.ChallengeState != storage.MFAChallengeStateBadChallenge {
		jsonapi.Marshal(w, getMFALoginResponse(ctx, session.ID, paths.MFACodeUISubPath))
		return
	}

	// look up channel and try logging in with the challenge code

	channel, err := mfaState.SupportedChannels.FindChannel(mfaState.ChannelID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetMFAChannelError")
		return
	}

	client, err := provider.NewClientForProviderID(ctx, h.factory, session.ClientID, mfaState.Provider)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetClient")
		return
	}

	idpResp, err := client.MFALogin(ctx, mfaState.Token, req.MFACode, channel)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedMFALogin")
		return
	}

	newChallengeState := storage.MFAChallengeStateNoChallenge
	if idpResp.Status == idp.LoginStatusMFACodeInvalid {
		newChallengeState = storage.MFAChallengeStateBadChallenge
	} else if idpResp.Status == idp.LoginStatusMFACodeExpired {
		newChallengeState = storage.MFAChallengeStateExpired
	} else if idpResp.Status != idp.LoginStatusSuccess {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("unexpected login status '%s'", idpResp.Status), "UnexpectedLoginStatus")
		return
	}

	if newChallengeState != storage.MFAChallengeStateNoChallenge {
		session.MFAChannelStates.RecordFailure(channel)
		if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "SaveOIDCLoginSessionError")
			return
		}

		if err := saveMFAState(ctx, s, mfaState, mfaState.ChannelID, newChallengeState); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "SaveMFAStateError")
			return
		}

		jsonapi.Marshal(w, getMFALoginResponse(ctx, session.ID, paths.MFACodeUISubPath))
		return
	}

	// challenge was successful

	// reset channel statistics

	session.MFAChannelStates.Reset(channel)

	// extract claims from response

	claims, err := infraoidc.ExtractClaims(idpResp.Claims)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "ExtractClaimsError")
		return
	}
	profile := iface.NewUserProfileFromClaims(*claims)

	if mfaState.Purpose == storage.MFAPurposeConfigure {
		// get updated supported channels from provider and create a new MFA state

		resp, err := client.MFAGetChannels(ctx, profile.ID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "MFAGetChannelsError")
			return
		}
		if _, err = h.createMFAState(ctx, session, storage.MFAPurposeConfigure, resp.MFAToken, resp.MFAProvider, resp.SupportedMFAChannels, false); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "CreateMFAStateError")
			return
		}

		if idpResp.NewRecoveryCode != "" {
			jsonapi.Marshal(w, getMFARecoveryCodeResponse(ctx, session.ID, idpResp.NewRecoveryCode))
			return
		}

		//   redirect to MFA channels page

		jsonapi.Marshal(w, getMFALoginResponse(ctx, session.ID, paths.MFAChannelUISubPath))
		return
	}

	// verify user has access

	tc := tenantconfig.MustGet(ctx)
	app, _, err := tc.PlexMap.FindAppForClientID(session.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FindApp")
		return
	}

	hasAccess, err := loginapp.CheckLoginAccessForUser(ctx, tc, app, profile.ID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "RestrictedAccessError")
		return
	}
	if !hasAccess {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Friendlyf(nil, "You are not permitted to login to this app"), "RestrictedAccessDenied", jsonapi.Code(http.StatusForbidden))
		return
	}

	// generate plex token and write to audit log
	tu := tenantconfig.MustGetTenantURLString(ctx)
	if err := storage.GenerateUserPlexToken(ctx, tu, &tc, s, profile, session, nil /* TODO: this is redirected MFA, we should support underlying token for eg auth0? */); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "CreateTokenError")
		return
	}

	auditlog.Post(ctx, auditlog.NewEntry(profile.ID, auditlog.LoginSuccess,
		auditlog.Payload{"ID": app.ID, "Name": app.Name, "Actor": channel.ChannelTypeID, "Type": channel.ChannelType.GetAuditLogType()}))

	// update providers

	// TODO: Update password for other providers. We should (probably) only update followers if session.MFAProvider == active provider, because
	// if/when we (eventually) support failover to a follower, if the follower's password is out-of-date with the active, we shouldn't overwrite
	// the active provider's credential with a potentially stale one. This could happen if we failover to a follower and the active comes back up
	// before the MFA process completes.

	// If this session has an invite, bind it to this user to mark it used and fail only
	// if the invite was already used.
	if err := otp.BindInviteToUser(ctx, s, session, profile.ID, profile.Email, app); err != nil &&
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

	// make sure we eventually evaluate MFA settings if requested to

	if req.EvaluateSupportedChannels && !mfaState.EvaluateSupportedChannels {
		mfaState.EvaluateSupportedChannels = true
		if err := saveMFAState(ctx, s, mfaState, mfaState.ChannelID, mfaState.ChallengeState); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "SaveMFAStateError")
			return
		}
	}

	// redirect to show recovery code if a new one was generated

	if idpResp.NewRecoveryCode != "" {
		jsonapi.Marshal(w, getMFARecoveryCodeResponse(ctx, session.ID, idpResp.NewRecoveryCode))
		return
	}

	// redirect to MFA channels page if user should reevaluate their settings

	if mfaState.EvaluateSupportedChannels {
		resp, err := h.startMFAChannelsSession(ctx, session, profile.ID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "StartMFAChannelsSessionError")
			return
		}

		jsonapi.Marshal(w, resp)
		return
	}

	// login is complete, and MFA state is no longer needed, so clear; session will be
	// committed as part of  generating the login response

	session.MFAStateID = uuid.Nil

	redirectURL, err := oidc.NewLoginResponse(ctx, session)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "NewLoginResponse")
		return
	}
	jsonapi.Marshal(w, redirectURL)
}
