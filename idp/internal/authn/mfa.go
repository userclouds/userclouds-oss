package authn

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	userstoreinternal "userclouds.com/idp/internal/userstore"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/multitenant"
	"userclouds.com/plex/manager"
)

const mfaCodeLength = 6
const mfaMaskedRecoveryCode = "recovery_code"
const mfaRecoveryCodeNumBytes = 32

var mfaCodeExpiration = time.Minute * 15
var mfaEvaluationTimeout = time.Hour * 24 * 30

func generateRecoveryCode() string {
	return crypto.MustRandomBase64(mfaRecoveryCodeNumBytes)
}

func getUserProfile(r *http.Request, userID uuid.UUID) (*userstore.Record, error) {
	ctx := r.Context()

	userData, _, err := userstoreinternal.GetUsers(ctx, true, "", false, userID.String())
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var profile userstore.Record
	if err = json.Unmarshal([]byte(userData[0]), &profile); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &profile, nil
}

// CreateTestMFAEmailChannel creates a email MFA channel for a user - this can only be
// run within unit tests in the test or CI universes.
func (m *Manager) CreateTestMFAEmailChannel(ctx context.Context, userID uuid.UUID, emailAddress string) error {
	if uv := universe.Current(); !uv.IsTestOrCI() {
		return ucerr.Errorf("cannot call CreateTestMFAEmailChannel in %v universe", uv)
	}

	userMFASettings, err := m.configStorage.GetUserMFAConfiguration(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	if userMFASettings == nil {
		userMFASettings = &storage.UserMFAConfiguration{BaseModel: ucdb.NewBaseWithID(userID)}
		userMFASettings.MFAChannels = oidc.NewMFAChannels()
	}

	userMFASettings.LastEvaluated = time.Now().UTC()

	channel, err := userMFASettings.MFAChannels.AddChannel(oidc.MFAEmailChannel, emailAddress, emailAddress, true)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := userMFASettings.MFAChannels.SetPrimary(channel.ID); err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(m.configStorage.SaveUserMFAConfiguration(ctx, userMFASettings))
}

// GetMFASettings returns the set of supported MFA channels for the user based on the specified set of MFA
// channel types, as well as a flag signifying whether the user should re-evaluate their MFA settings
func (m *Manager) GetMFASettings(ctx context.Context,
	channelTypes oidc.MFAChannelTypeSet,
	userID uuid.UUID) (mfaChannels oidc.MFAChannels, evaluateSettings bool, err error) {
	if len(channelTypes) == 0 {
		return mfaChannels, false, nil
	}

	userSettings, err := m.configStorage.GetUserMFAConfiguration(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return mfaChannels, false, ucerr.Wrap(err)
	}

	settingsHaveChanged := false

	if userSettings == nil {
		userSettings = &storage.UserMFAConfiguration{
			BaseModel:     ucdb.NewBaseWithID(userID),
			LastEvaluated: time.Time{},
			MFAChannels:   oidc.NewMFAChannels(),
		}
		settingsHaveChanged = true
	} else {
		// handle the case where a channel type is no longer supported because of a change in
		// tenant settings, making sure that any unsupported channel is not the primary channel
		settingsHaveChanged = userSettings.MFAChannels.ClearUnsupportedPrimary(channelTypes)
	}

	if settingsHaveChanged {
		if err := m.configStorage.SaveUserMFAConfiguration(ctx, userSettings); err != nil {
			return mfaChannels, false, ucerr.Wrap(err)
		}
	}

	// if not all supported channel types have a verified channel, and the user
	// has not evaluated their MFA settings in 30 days, prompt them to do so
	mfaChannels, hasUncoveredChannelTypes := userSettings.MFAChannels.GetVerifiedChannels(channelTypes)
	if err := mfaChannels.Validate(); err != nil {
		return mfaChannels, false, ucerr.Wrap(err)
	}

	if hasUncoveredChannelTypes {
		now := time.Now().UTC()
		if userSettings.LastEvaluated.Add(mfaEvaluationTimeout).Before(now) {
			evaluateSettings = true
		}
	}

	return mfaChannels, evaluateSettings, nil
}

func (h *handler) generateMFACode(ctx context.Context, tenantAuthn *AuthN, mfaReq *storage.MFARequest, channel oidc.MFAChannel) (mfaCode string, err error) {
	switch channel.ChannelType {
	case oidc.MFAEmailChannel, oidc.MFASMSChannel:
		mfaCode = crypto.MustRandomDigits(mfaCodeLength)
		mfaReq.SetCode(channel.ID, mfaCode)
	case oidc.MFAAuthenticatorChannel, oidc.MFARecoveryCodeChannel:
		mfaCode = channel.ChannelName
		mfaReq.SetCode(channel.ID, channel.ChannelTypeID)
	default:
		return mfaCode, ucerr.Errorf("unsupported channel type: '%v'", channel.ChannelType)
	}

	if err := tenantAuthn.ConfigStorage.SaveMFARequest(ctx, mfaReq); err != nil {
		return mfaCode, ucerr.Wrap(err)
	}

	return mfaCode, nil
}

func (h *handler) getMFASettings(ctx context.Context, tenantAuthn *AuthN, clientID string, userID uuid.UUID) (mfaRequired bool, channels oidc.MFAChannels, evaluateChannels bool, err error) {
	ts := multitenant.MustGetTenantState(ctx)
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantAuthn.ID)
	if err != nil {
		return mfaRequired, channels, evaluateChannels, ucerr.Wrap(err)
	}

	mfaRequired, channelTypes, err := tp.PlexConfig.GetMFASettings(clientID)
	if err != nil {
		return mfaRequired, channels, evaluateChannels, ucerr.Wrap(err)
	}

	channels, evaluateChannels, err = tenantAuthn.Manager.GetMFASettings(ctx, channelTypes, userID)
	if err != nil {
		return mfaRequired, channels, evaluateChannels, ucerr.Wrap(err)
	}

	return mfaRequired, channels, evaluateChannels, nil
}

func (h *handler) getUserMFASettings(ctx context.Context, tenantAuthn *AuthN, mfaToken uuid.UUID) (*storage.MFARequest, *storage.UserMFAConfiguration, error) {
	mfaReq, err := tenantAuthn.ConfigStorage.GetMFARequest(ctx, mfaToken)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	userMFAConfiguration, err := tenantAuthn.ConfigStorage.GetUserMFAConfiguration(ctx, mfaReq.UserID)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	return mfaReq, userMFAConfiguration, nil
}

func (h *handler) HandleMFACodeRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.MFAChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mfaReq, userSettings, err := h.getUserMFASettings(ctx, tenantAuthn, req.MFAToken)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
		return
	}

	channel, err := userSettings.MFAChannels.FindChannel(req.MFAChannel.ID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
		return
	}

	if channel != req.MFAChannel {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("IDP '%v' and request '%v' channels do not match", channel, req.MFAChannel))
		return
	}

	if !mfaReq.SupportedChannelTypes.IncludesType(channel.ChannelType) {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("requested channel type '%v' is unsupported for request", channel.ChannelType))
		return
	}

	mfaCode, err := h.generateMFACode(ctx, tenantAuthn, mfaReq, channel)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
		return
	}

	resp := idp.MFACodeResponse{
		MFAToken:   req.MFAToken,
		MFAChannel: channel,
		MFACode:    mfaCode,
	}
	jsonapi.Marshal(w, resp)
}

func (h *handler) generateAuthenticatorAppKey(r *http.Request, userID uuid.UUID) (key string, err error) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	tenant, err := h.companyConfigStorage.GetTenant(ctx, tenantAuthn.ID)
	if err != nil {
		return key, ucerr.Wrap(err)
	}
	issuer := tenant.Name

	userProfile, err := getUserProfile(r, userID)
	if err != nil {
		return key, ucerr.Wrap(err)
	}

	accountName := userProfile.StringValue("email")
	if accountName == "" {
		return key, ucerr.Errorf("no email address found for user '%v'", userID)
	}

	totpKey, err := totp.Generate(totp.GenerateOpts{Issuer: issuer, AccountName: accountName})
	if err != nil {
		return key, ucerr.Wrap(err)
	}

	return totpKey.String(), nil
}

func (h *handler) HandleMFAClearPrimaryChannelRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.MFAClearPrimaryChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mfaReq, userSettings, err := h.getUserMFASettings(ctx, tenantAuthn, req.MFAToken)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !userSettings.MFAChannels.HasPrimaryChannel() {
		jsonapi.MarshalError(ctx, w, ucerr.New("There is no primary channel to clear"))
		return
	}

	mfaRequired, _, _, err := h.getMFASettings(ctx, tenantAuthn, req.ClientID, mfaReq.UserID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if mfaRequired {
		jsonapi.MarshalError(ctx, w, ucerr.New("Cannot clear primary MFA channel if MFA is required"))
		return
	}

	userSettings.MFAChannels.ClearPrimary()
	userSettings.MarkEvaluated()

	if err := tenantAuthn.ConfigStorage.SaveUserMFAConfiguration(ctx, userSettings); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	channels, _, err := tenantAuthn.Manager.GetMFASettings(ctx, mfaReq.SupportedChannelTypes.ChannelTypes, mfaReq.UserID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := idp.MFAGetChannelsResponse{
		MFAToken:    mfaReq.ID,
		MFAChannels: channels,
	}
	jsonapi.Marshal(w, resp)
}

func (h *handler) HandleMFACreateChannelRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.MFACreateChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !req.ChannelType.CanConfigure() {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("channel type '%v' is not createable", req.ChannelType))
		return
	}

	mfaReq, userSettings, err := h.getUserMFASettings(ctx, tenantAuthn, req.MFAToken)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !mfaReq.SupportedChannelTypes.IncludesType(req.ChannelType) {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("channel type '%v' is not supported", req.ChannelType))
		return
	}

	var channel oidc.MFAChannel

	switch req.ChannelType {
	case oidc.MFAEmailChannel, oidc.MFASMSChannel:
		channel, err = userSettings.MFAChannels.AddChannel(req.ChannelType, req.ChannelTypeID, req.ChannelTypeID, false)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	case oidc.MFAAuthenticatorChannel:
		key, err := h.generateAuthenticatorAppKey(r, userSettings.ID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		channel, err = userSettings.MFAChannels.AddChannel(req.ChannelType, key, req.ChannelTypeID, false)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	default:
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("channel type '%v' is not supported", req.ChannelType))
		return
	}
	userSettings.MarkEvaluated()

	if err := tenantAuthn.ConfigStorage.SaveUserMFAConfiguration(ctx, userSettings); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mfaCode, err := h.generateMFACode(ctx, tenantAuthn, mfaReq, channel)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
		return
	}

	resp := idp.MFACodeResponse{
		MFAToken:   req.MFAToken,
		MFAChannel: channel,
		MFACode:    mfaCode,
	}
	jsonapi.Marshal(w, resp)
}

func (h *handler) HandleMFADeleteChannelRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.MFAChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !req.MFAChannel.ChannelType.CanConfigure() {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("channel type '%v' is not deleteable", req.MFAChannel.ChannelType))
		return
	}

	mfaReq, userSettings, err := h.getUserMFASettings(ctx, tenantAuthn, req.MFAToken)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	channel, err := userSettings.MFAChannels.FindChannel(req.MFAChannel.ID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if channel != req.MFAChannel {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("IDP '%v' and request '%v' channels do not match", channel, req.MFAChannel))
		return
	}

	if !mfaReq.SupportedChannelTypes.IncludesType(channel.ChannelType) {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("channel type '%v' is not supported", channel.ChannelType))
		return
	}

	if err := userSettings.MFAChannels.DeleteChannel(channel.ID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := tenantAuthn.ConfigStorage.SaveUserMFAConfiguration(ctx, userSettings); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	channels, _, err := tenantAuthn.Manager.GetMFASettings(ctx, mfaReq.SupportedChannelTypes.ChannelTypes, mfaReq.UserID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := idp.MFAGetChannelsResponse{
		MFAToken:    mfaReq.ID,
		MFAChannels: channels,
	}
	jsonapi.Marshal(w, resp)
}

func (h *handler) HandleMFAGetChannelsRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.MFAGetChannelsRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// get supported channel types and verified channels for user and client

	_, channels, _, err := h.getMFASettings(ctx, tenantAuthn, req.ClientID, req.UserID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.NewServerError(err))
		return
	}

	if len(channels.ChannelTypes) == 0 {
		jsonapi.MarshalError(ctx, w, ucerr.New("MFA is not enabled"))
		return
	}

	// create a new MFAReq and update last evaluated time for UserMFAConfiguration

	mfaReq := storage.NewMFARequest(req.UserID, channels.ChannelTypes)
	if err := tenantAuthn.ConfigStorage.SaveMFARequest(ctx, mfaReq); err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.NewServerError(err))
		return
	}

	userSettings, err := tenantAuthn.ConfigStorage.GetUserMFAConfiguration(ctx, req.UserID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		jsonapi.MarshalError(ctx, w, ucerr.NewServerError(err))
		return
	}

	if userSettings == nil {
		userSettings = &storage.UserMFAConfiguration{BaseModel: ucdb.NewBaseWithID(req.UserID)}
	}
	userSettings.MarkEvaluated()

	if err := tenantAuthn.ConfigStorage.SaveUserMFAConfiguration(ctx, userSettings); err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.NewServerError(err))
		return
	}

	resp := idp.MFAGetChannelsResponse{
		MFAToken:    mfaReq.ID,
		MFAChannels: channels,
	}
	jsonapi.Marshal(w, resp)
}

func (h *handler) HandleMFAMakePrimaryChannelRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.MFAChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mfaReq, userSettings, err := h.getUserMFASettings(ctx, tenantAuthn, req.MFAToken)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	channel, err := userSettings.MFAChannels.FindChannel(req.MFAChannel.ID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if channel != req.MFAChannel {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("IDP '%v' and request '%v' channels do not match", channel, req.MFAChannel))
		return
	}

	if !mfaReq.SupportedChannelTypes.IncludesType(channel.ChannelType) {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("channel type '%v' is not supported", channel.ChannelType))
		return
	}

	if err := userSettings.MFAChannels.SetPrimary(channel.ID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	userSettings.MarkEvaluated()

	if err := tenantAuthn.ConfigStorage.SaveUserMFAConfiguration(ctx, userSettings); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	channels, _, err := tenantAuthn.Manager.GetMFASettings(ctx, mfaReq.SupportedChannelTypes.ChannelTypes, mfaReq.UserID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := idp.MFAGetChannelsResponse{
		MFAToken:    mfaReq.ID,
		MFAChannels: channels,
	}
	jsonapi.Marshal(w, resp)
}

func (h *handler) HandleMFAReissueRecoveryCodeRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.MFAChannelRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mfaReq, userSettings, err := h.getUserMFASettings(ctx, tenantAuthn, req.MFAToken)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !mfaReq.SupportedChannelTypes.ChannelTypes[oidc.MFARecoveryCodeChannel] {
		jsonapi.MarshalError(ctx, w, ucerr.New("recovery codes are not supported"))
		return
	}

	channel, err := userSettings.MFAChannels.FindChannel(req.MFAChannel.ID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !channel.ChannelType.IsRecoveryCode() {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("requested channel '%v' is not a recovery code", req.MFAChannel.ID))
		return
	}

	channel.ChannelTypeID = generateRecoveryCode()

	if err := userSettings.MFAChannels.UpdateChannel(channel); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	userSettings.MarkEvaluated()

	if err := tenantAuthn.ConfigStorage.SaveUserMFAConfiguration(ctx, userSettings); err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
		return
	}

	resp := idp.MFAReissueRecoveryCodeResponse{
		MFAToken:   req.MFAToken,
		MFAChannel: channel,
	}
	jsonapi.Marshal(w, resp)
}

func (h *handler) HandleMFAResponse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.MFALoginRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mfaReq, userSettings, err := h.getUserMFASettings(ctx, tenantAuthn, req.MFAToken)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	channelID, code, issued, err := mfaReq.GetCode()
	if err != nil {
		uclog.Debugf(ctx, "MFA auth %v failed because this is not an active MFA request", req.MFAToken)
		jsonapi.MarshalError(ctx, w, ucerr.New("unauthorized"), jsonapi.Code(http.StatusUnauthorized))
		return
	}

	channel, err := userSettings.MFAChannels.FindChannel(channelID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
		return
	}

	resp := idp.LoginResponse{
		Status: idp.LoginStatusSuccess,
		UserID: mfaReq.UserID,
	}

	// validate code based on channel type

	switch channel.ChannelType {
	case oidc.MFAEmailChannel, oidc.MFASMSChannel:
		now := time.Now().UTC()
		if issued.Add(mfaCodeExpiration).Before(now) {
			uclog.Debugf(ctx, "MFA auth %v failed because it took too long", req.MFAToken)
			resp.Status = idp.LoginStatusMFACodeExpired
		} else if req.MFACode != code {
			uclog.Debugf(ctx, "MFA auth failed for req %v", req.MFAToken)
			resp.Status = idp.LoginStatusMFACodeInvalid
		}
	case oidc.MFAAuthenticatorChannel:
		key, err := otp.NewKeyFromURL(channel.ChannelTypeID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		codeValid, err := totp.ValidateCustom(
			req.MFACode,
			key.Secret(),
			time.Now().UTC(),
			totp.ValidateOpts{
				Skew:      1,
				Digits:    key.Digits(),
				Algorithm: key.Algorithm(),
			})
		if err != nil {
			uclog.Debugf(ctx, "MFA authenticator app auth failed for req '%v' with error '%v'", req.MFAToken, err)
			resp.Status = idp.LoginStatusMFACodeInvalid
		} else if !codeValid {
			uclog.Debugf(ctx, "MFA authenticator app auth failed for req '%v'", req.MFAToken)
			resp.Status = idp.LoginStatusMFACodeInvalid
		}
	case oidc.MFARecoveryCodeChannel:
		if req.MFACode != code {
			uclog.Debugf(ctx, "Recovery Code auth failed for req '%v'", req.MFAToken)
			resp.Status = idp.LoginStatusMFACodeInvalid
		}
	default:
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("unsupported channel type: '%v'", channel.ChannelType))
		return
	}

	if resp.Status == idp.LoginStatusSuccess {
		if channel.ChannelType == oidc.MFARecoveryCodeChannel {
			// delete the used recovery code
			if err := userSettings.MFAChannels.DeleteChannel(channelID); err != nil {
				jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
				return
			}
		} else {
			// mark the channel verified, and primary if a primary channel does not already exist
			if err := userSettings.MFAChannels.SetVerifiedAndPrimary(channelID); err != nil {
				jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
				return
			}
		}

		// create a new recovery code if recovery codes are supported and one does not already exist
		if mfaReq.SupportedChannelTypes.IncludesType(oidc.MFARecoveryCodeChannel) {
			if _, isUncovered := userSettings.MFAChannels.GetVerifiedChannels(oidc.MFAChannelTypeSet{oidc.MFARecoveryCodeChannel: true}); isUncovered {
				resp.NewRecoveryCode = generateRecoveryCode()
				if _, err := userSettings.MFAChannels.AddChannel(oidc.MFARecoveryCodeChannel, resp.NewRecoveryCode, mfaMaskedRecoveryCode, true); err != nil {
					jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
					return
				}
			}
		}

		if err := tenantAuthn.ConfigStorage.SaveUserMFAConfiguration(ctx, userSettings); err != nil {
			jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
			return
		}

		// delete the MFA request since it's been successfully challenged
		if err := tenantAuthn.ConfigStorage.DeleteMFARequest(ctx, mfaReq.ID); err != nil {
			jsonapi.MarshalError(ctx, w, ucerr.Wrap(err))
			return
		}
	}

	jsonapi.Marshal(w, resp)
}
