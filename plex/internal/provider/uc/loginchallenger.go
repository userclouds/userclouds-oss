package uc

import (
	"context"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/infra/uctypes/messaging/telephony"
	phone "userclouds.com/infra/uctypes/messaging/telephony/phonenumber"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/tenantconfig"
)

type challengeType int

const (
	authChallengeType   challengeType = 1
	verifyChallengeType challengeType = 2
)

// loginChallenger wraps state for issuing an MFA challenge
type loginChallenger struct {
	clientID    string
	emailClient email.Client
}

// issueChallenge issues a challenge over the specified channel with a given challengeCode for the specified challengeType
func (lc *loginChallenger) issueChallenge(ctx context.Context, code string, c oidc.MFAChannel, ct challengeType) error {
	tc := tenantconfig.MustGet(ctx)
	app, _, err := tc.PlexMap.FindAppForClientID(lc.clientID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	switch c.ChannelType {
	case oidc.MFAAuthenticatorChannel:
	case oidc.MFARecoveryCodeChannel:
	case oidc.MFAEmailChannel:
		return ucerr.Wrap(lc.issueEmailChallenge(ctx, &tc, app, code, c, ct))
	case oidc.MFASMSChannel:
		return ucerr.Wrap(lc.issueSMSChallenge(ctx, &tc, app, code, c, ct))
	default:
		return ucerr.Errorf("channel type '%v' is unsupported", c.ChannelType)
	}

	return nil
}

func (lc *loginChallenger) issueEmailChallenge(ctx context.Context, tenant *tenantplex.TenantConfig, app *tenantplex.App, code string, c oidc.MFAChannel, ct challengeType) error {
	var messageType message.MessageType
	switch ct {
	case authChallengeType:
		messageType = message.EmailMFAChallenge
	case verifyChallengeType:
		messageType = message.EmailMFAVerify
	default:
		return ucerr.Errorf("challenge type '%v' is unsupported", ct)
	}

	data := message.MFATemplateData{
		AppName: app.Name,
		Code:    code,
	}

	if err := email.SendWithHTMLTemplate(
		ctx,
		tenant.PlexMap.GetEmailClient(lc.emailClient),
		c.ChannelTypeID,
		app.MakeElementGetter(messageType), data); err != nil {
		uclog.Debugf(ctx, "error sending email: %v", err)
		return ucerr.Wrap(err)
	}

	return nil
}

func (lc *loginChallenger) issueSMSChallenge(ctx context.Context, tenant *tenantplex.TenantConfig, app *tenantplex.App, code string, c oidc.MFAChannel, ct challengeType) error {
	var messageType message.MessageType
	switch ct {
	case authChallengeType:
		messageType = message.SMSMFAChallenge
	case verifyChallengeType:
		messageType = message.SMSMFAVerify
	default:
		return ucerr.Errorf("challenge type '%v' is unsupported", ct)
	}

	client, err := telephony.CreateClient(ctx, &tenant.PlexMap.TelephonyProvider)
	if err != nil {
		return ucerr.Wrap(err)
	}

	data := message.MFATemplateData{
		AppName: app.Name,
		Code:    code,
	}

	if err := telephony.SendSMSWithTemplate(
		ctx,
		client,
		phone.PhoneNumber(c.ChannelTypeID),
		app.MakeElementGetter(messageType),
		data); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
