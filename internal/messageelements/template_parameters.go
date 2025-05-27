package messageelements

import "html/template"

// InviteUserTemplateData defines the template variables  that can
// be used for the EmailInviteExistingUser and EmailInviteNewUser
// message types
type InviteUserTemplateData struct {
	AppName        string
	InviterName    string
	Link           template.HTML // required so we don't HTML-encode this
	WorkaroundLink template.HTML // see SendHTMLTemplateEmail for why this is needed
	InviteText     string
}

// MFATemplateData defines the template variables that can be
// used for an MFA message
type MFATemplateData struct {
	AppName string
	Code    string
}

// OTPEmailTemplateData defines the template variables that can be
// used for the EmailPasswordlessLogin, EmailResetPassword, and
// EmailVerifyEmail message types
type OTPEmailTemplateData struct {
	AppName        string
	Code           string
	Link           template.HTML // required so we don't HTML-encode this
	WorkaroundLink template.HTML // see SendHTMLTemplateEmail for why this is needed
}
