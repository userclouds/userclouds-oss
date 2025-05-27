package otp

import (
	"context"
	"fmt"
	"html/template"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/plex/internal/tenantconfig"
)

// MakeOTPLink makes a link which can be redeemed at the Passwordless Submit handler
// TODO: change handler name/path to be generic OTP Submit, not just passwordless auth.
func MakeOTPLink(ctx context.Context, sessionID uuid.UUID, email, otpCode string) string {
	u := tenantconfig.MustGetTenantURL(ctx)
	u.Path = fmt.Sprintf("%s%s", RootPath, SubmitSubPath)
	// TODO: one-way hash this code in the URL and validate with same hash func? In fact, maybe hash the whole thing
	// so the link doesn't contain any useful data?
	u.RawQuery = url.Values{
		"session_id": []string{sessionID.String()},
		"email":      []string{email},
		"otp_code":   []string{otpCode},
	}.Encode()
	return u.String()
}

// SendOTPEmail sends an email to a user with a one-time password and magic link.
// The text & html templates will be supplied with {{.AppName}}, {{.Code}}, and {{.Link}}.
func SendOTPEmail(ctx context.Context, emailClient email.Client, sessionID uuid.UUID, appName, emailAddress string, getter message.ElementGetter, otpCode string) error {
	link := MakeOTPLink(ctx, sessionID, emailAddress, otpCode)

	// Initialze a struct storing page data and todo data
	data := message.OTPEmailTemplateData{
		AppName:        appName,
		Code:           otpCode,
		Link:           template.HTML(link),
		WorkaroundLink: template.HTML(fmt.Sprintf(email.LinkTemplate, link)),
	}

	if emailClient == nil {
		return ucerr.New("unable to send email, no client initialized")
	}

	// Got a valid user, send email & wait for user to click on magic link OR enter code.
	if err := email.SendWithHTMLTemplate(ctx, emailClient, emailAddress, getter, data); err != nil {
		uclog.Debugf(ctx, "error sending email: %v", err)
		return ucerr.Wrap(err)
	}
	return nil
}

// MakeOTPLinkUI makes a link which can be redeemed at the Passwordless Submit handler
// TODO: change handler name/path to be generic OTP Submit, not just passwordless auth.
func MakeOTPLinkUI(ctx context.Context, sessionID uuid.UUID, email, otpCode string) string {
	u := tenantconfig.MustGetTenantURL(ctx)
	u.Path = fmt.Sprintf("%s%s", "/plexui"+RootPath, SubmitSubPath)
	// TODO: one-way hash this code in the URL and validate with same hash func? In fact, maybe hash the whole thing
	// so the link doesn't contain any useful data?
	u.RawQuery = url.Values{
		"session_id": []string{sessionID.String()},
		"email":      []string{email},
		"otp_code":   []string{otpCode},
	}.Encode()
	return u.String()
}

// SendOTPEmailUI sends an email to a user with a one-time password and magic link.
// The text & html templates will be supplied with {{.AppName}}, {{.Code}}, and {{.Link}}.
func SendOTPEmailUI(ctx context.Context, emailClient email.Client, sessionID uuid.UUID, appName, emailAddress string, getter message.ElementGetter, otpCode string) error {
	link := MakeOTPLinkUI(ctx, sessionID, emailAddress, otpCode)

	// Initialze a struct storing page data and todo data
	data := message.OTPEmailTemplateData{
		AppName:        appName,
		Code:           otpCode,
		Link:           template.HTML(link),
		WorkaroundLink: template.HTML(fmt.Sprintf(email.LinkTemplate, link)),
	}

	// Got a valid user, send email & wait for user to click on magic link OR enter code.
	if err := email.SendWithHTMLTemplate(ctx, emailClient, emailAddress, getter, data); err != nil {
		uclog.Debugf(ctx, "error sending email: %v", err)
		return ucerr.Wrap(err)
	}
	return nil
}
