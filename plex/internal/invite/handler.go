package invite

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	"userclouds.com/authz"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/tenantconfig"
)

type handler struct {
	emailClient *email.Client
	factory     provider.Factory
}

// NewHandler returns a new invite-user handler for plex
func NewHandler(email *email.Client, factory provider.Factory) http.Handler {
	h := &handler{email, factory}

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)

	return hb.Build()
}

//go:generate genhandler /invite POST,sendHandler,/send

// OpenAPI Summary: Send Invite
// OpenAPI Tags: Invites
// OpenAPI Description: This endpoint sends an invite to a user to join a tenant
func (h *handler) sendHandler(ctx context.Context, req plex.SendInviteRequest) (any, int, []auditlog.Entry, error) {

	subjectType := auth.GetSubjectType(ctx)
	if subjectType != authz.ObjectTypeLoginApp && subjectType != m2m.SubjectTypeM2M {
		return nil, http.StatusForbidden, nil, ucerr.WrapWithName(ucerr.Friendlyf(nil, "User tokens not allowed"), "UserTokensNotAllowed")
	}

	tc := tenantconfig.MustGet(ctx)

	if err := req.Validate(); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	app, _, err := tc.PlexMap.FindAppForClientID(req.ClientID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.WrapWithName(err, "FindApp")
	}

	activeClient, err := provider.NewActiveManagementClient(ctx, h.factory, req.ClientID)
	if err != nil {
		// TODO: differentiate between internal vs. request errors (issue #103).
		return nil, http.StatusBadRequest, nil, ucerr.WrapWithName(err, "ProviderInitErr")
	}

	redirectURL, err := app.ValidateRedirectURI(ctx, req.RedirectURL)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.WrapWithName(err, "BadRedirectURL")
	}

	s := tenantconfig.MustGetStorage(ctx)
	sessionID, otpCode, err := otp.CreateInviteSession(ctx, s, req.ClientID, req.InviteeEmail, req.State, redirectURL, req.Expires)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.WrapWithName(err, "StartFlowFail")
	}
	if h.emailClient != nil {
		emailClient := tc.PlexMap.GetEmailClient(*h.emailClient)
		link := otp.MakeOTPLink(ctx, sessionID, req.InviteeEmail, otpCode)

		data := message.InviteUserTemplateData{
			AppName:        app.Name,
			InviterName:    req.InviterName,
			Link:           template.HTML(link),
			WorkaroundLink: template.HTML(fmt.Sprintf(email.LinkTemplate, link)),
			InviteText:     req.InviteText,
		}

		// determine email type based on whether an account already exists for this email
		emailType := message.EmailInviteNewUser
		if otp.UserWithEmailExists(ctx, activeClient, req.InviteeEmail) {
			emailType = message.EmailInviteExistingUser
		}

		if err := email.SendWithHTMLTemplate(ctx, emailClient, req.InviteeEmail, app.MakeElementGetter(emailType), data); err != nil {
			uclog.Errorf(ctx, "error sending email: %v", err)
			return nil, http.StatusInternalServerError, nil, ucerr.WrapWithName(err, "SendEmail")
		}
	} else {
		uclog.Infof(ctx, "Skipping email send for invite to %s (email disabled)", req.InviteeEmail)
	}

	return nil, http.StatusNoContent, auditlog.NewEntryArray(req.InviterUserID, auditlog.InviteSent,
		auditlog.Payload{"ID": app.ID, "Name": app.Name, "InviteeEmail": req.InviteeEmail, "InviterEmail": req.InviterEmail}), nil
}
