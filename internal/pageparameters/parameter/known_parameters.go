package parameter

import (
	"userclouds.com/internal/pageparameters/parametertype"
)

// Name constants for supported page parameters - each
// supported parameter must be registered along with its
// parameter type, used for validation, in the init()
// method below
const (
	ActionButtonBorderColor                 Name = "actionButtonBorderColor"
	ActionButtonFillColor                   Name = "actionButtonFillColor"
	ActionButtonTextColor                   Name = "actionButtonTextColor"
	ActionSubmitButtonText                  Name = "actionSubmitButtonText"
	AllowCreation                           Name = "allowCreation"
	AuthenticationMethods                   Name = "authenticationMethods"
	CreateAccountText                       Name = "createAccountText"
	CreateUserDisabledText                  Name = "createUserDisabledText"
	CreateUserFailStatusText                Name = "createUserFailStatusText"
	CreateUserStartStatusText               Name = "createUserStartStatusText"
	CustomCreateUserPagePostFieldsetHTMLCSS Name = "customCreateUserPagePostFieldsetHTMLCSS"
	CustomCreateUserPagePreFieldsetHTMLCSS  Name = "customCreateUserPagePreFieldsetHTMLCSS"
	CustomCreateUserPagePostFormHTMLCSS     Name = "customCreateUserPagePostFormHTMLCSS"
	CustomCreateUserPagePreFormHTMLCSS      Name = "customCreateUserPagePreFormHTMLCSS"
	CustomCreateUserPagePreMainHTMLCSS      Name = "customCreateUserPagePreMainHTMLCSS"
	CustomLoginPagePostFieldsetHTMLCSS      Name = "customLoginPagePostFieldsetHTMLCSS"
	CustomLoginPagePreFieldsetHTMLCSS       Name = "customLoginPagePreFieldsetHTMLCSS"
	CustomLoginPagePostFormHTMLCSS          Name = "customLoginPagePostFormHTMLCSS"
	CustomLoginPagePreFormHTMLCSS           Name = "customLoginPagePreFormHTMLCSS"
	CustomLoginPagePreMainHTMLCSS           Name = "customLoginPagePreMainHTMLCSS"
	DisabledAuthenticationMethods           Name = "disabledAuthenticationMethods"
	DisabledMFAMethods                      Name = "disabledMFAMethods"
	EmailLabel                              Name = "emailLabel"
	EmailExistsCreateAccountText            Name = "emailExistsCreateAccountText"
	EmailExistsLoginText                    Name = "emailExistsLoginText"
	EnabledAuthenticationMethods            Name = "enabledAuthenticationMethods"
	EnabledMFAMethods                       Name = "enabledMFAMethods"
	FooterHTML                              Name = "footerHTML"
	ForgotPasswordText                      Name = "forgotPasswordText"
	HeadingText                             Name = "headingText"
	LoginFailStatusText                     Name = "loginFailStatusText"
	LoginStartStatusText                    Name = "loginStartStatusText"
	LoginSuccessStatusText                  Name = "loginSuccessStatusText"
	LogoImageFile                           Name = "logoImageFile"
	MissingOTPCodeText                      Name = "missingOTPCodeText"
	MFACodeLabel                            Name = "mfaCodeLabel"
	MFAMethods                              Name = "mfaMethods"
	MFARequired                             Name = "mfaRequired"
	NewPasswordLabel                        Name = "newPasswordLabel"
	OIDCAuthenticationSettings              Name = "oidcAuthenticationSettings"
	OtpCodeLabel                            Name = "otpCodeLabel"
	PageBackgroundColor                     Name = "pageBackgroundColor"
	PageOrderSocialFirst                    Name = "pageOrderSocialFirst"
	PageTextColor                           Name = "pageTextColor"
	PasswordLabel                           Name = "passwordLabel"
	PasswordlessLoginText                   Name = "passwordlessLoginText"
	PasswordlessSendEmailStartStatusText    Name = "passwordlessSendEmailStartStatusText"
	PasswordlessSendEmailSuccessStatusText  Name = "passwordlessSendEmailSuccessStatusText"
	PasswordResetEnabled                    Name = "passwordResetEnabled"
	RequireName                             Name = "requireName"
	ResetPasswordFailStatusText             Name = "resetPasswordFailStatusText"
	ResetPasswordStartStatusText            Name = "resetPasswordStartStatusText"
	ResetPasswordSuccessStatusText          Name = "resetPasswordSuccessStatusText"
	SocialRedirectStatusText                Name = "socialRedirectStatusText"
	SubheadingText                          Name = "subheadingText"
	UserNameLabel                           Name = "userNameLabel"
)

func init() {
	register(ActionButtonBorderColor, parametertype.CSSColor)
	register(ActionButtonFillColor, parametertype.CSSColor)
	register(ActionButtonTextColor, parametertype.CSSColor)
	register(ActionSubmitButtonText, parametertype.ButtonText)
	register(AllowCreation, parametertype.Bool)
	register(AuthenticationMethods, parametertype.SelectedAuthenticationMethods)
	register(CreateAccountText, parametertype.Text)
	register(CreateUserDisabledText, parametertype.Text)
	register(CreateUserFailStatusText, parametertype.StatusText)
	register(CreateUserStartStatusText, parametertype.StatusText)
	register(DisabledAuthenticationMethods, parametertype.AuthenticationMethods)
	register(DisabledMFAMethods, parametertype.MFAMethods)
	register(EmailLabel, parametertype.LabelText)
	register(EmailExistsCreateAccountText, parametertype.Text)
	register(EmailExistsLoginText, parametertype.Text)
	register(EnabledAuthenticationMethods, parametertype.AuthenticationMethods)
	register(EnabledMFAMethods, parametertype.MFAMethods)
	register(FooterHTML, parametertype.HTMLSnippet)
	register(ForgotPasswordText, parametertype.Text)
	register(HeadingText, parametertype.HeadingText)
	register(LoginFailStatusText, parametertype.StatusText)
	register(LoginStartStatusText, parametertype.StatusText)
	register(LoginSuccessStatusText, parametertype.StatusText)
	register(LogoImageFile, parametertype.ImageURL)
	register(MissingOTPCodeText, parametertype.StatusText)
	register(MFACodeLabel, parametertype.LabelText)
	register(MFAMethods, parametertype.MFAMethods)
	register(MFARequired, parametertype.Bool)
	register(NewPasswordLabel, parametertype.LabelText)
	register(OIDCAuthenticationSettings, parametertype.OIDCAuthenticationSettings)
	register(OtpCodeLabel, parametertype.LabelText)
	register(PageBackgroundColor, parametertype.CSSColor)
	register(PageOrderSocialFirst, parametertype.Bool)
	register(PageTextColor, parametertype.CSSColor)
	register(PasswordLabel, parametertype.LabelText)
	register(PasswordlessLoginText, parametertype.Text)
	register(PasswordlessSendEmailStartStatusText, parametertype.StatusText)
	register(PasswordlessSendEmailSuccessStatusText, parametertype.StatusText)
	register(PasswordResetEnabled, parametertype.Bool)
	register(RequireName, parametertype.Bool)
	register(ResetPasswordFailStatusText, parametertype.StatusText)
	register(ResetPasswordStartStatusText, parametertype.StatusText)
	register(ResetPasswordSuccessStatusText, parametertype.StatusText)
	register(SocialRedirectStatusText, parametertype.StatusText)
	register(SubheadingText, parametertype.SubheadingText)
	register(UserNameLabel, parametertype.LabelText)

	registerWhitelisted(CustomCreateUserPagePostFieldsetHTMLCSS, parametertype.HTMLCSSSnippet)
	registerWhitelisted(CustomCreateUserPagePreFieldsetHTMLCSS, parametertype.HTMLCSSSnippet)
	registerWhitelisted(CustomCreateUserPagePostFormHTMLCSS, parametertype.HTMLCSSSnippet)
	registerWhitelisted(CustomCreateUserPagePreFormHTMLCSS, parametertype.HTMLCSSSnippet)
	registerWhitelisted(CustomCreateUserPagePreMainHTMLCSS, parametertype.HTMLCSSSnippet)
	registerWhitelisted(CustomLoginPagePostFieldsetHTMLCSS, parametertype.HTMLCSSSnippet)
	registerWhitelisted(CustomLoginPagePreFieldsetHTMLCSS, parametertype.HTMLCSSSnippet)
	registerWhitelisted(CustomLoginPagePostFormHTMLCSS, parametertype.HTMLCSSSnippet)
	registerWhitelisted(CustomLoginPagePreFormHTMLCSS, parametertype.HTMLCSSSnippet)
	registerWhitelisted(CustomLoginPagePreMainHTMLCSS, parametertype.HTMLCSSSnippet)
}
