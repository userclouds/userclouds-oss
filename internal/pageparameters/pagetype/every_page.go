package pagetype

import (
	"userclouds.com/internal/pageparameters/parameter"
)

// EveryPage is a page type that represents all pages
const EveryPage Type = "every_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.ActionButtonBorderColor, "#1090FF").
		AddParameter(parameter.ActionButtonFillColor, "#1090FF").
		AddParameter(parameter.ActionButtonTextColor, "#FFFFFF").
		AddParameter(parameter.AuthenticationMethods, "password,passwordless").
		AddParameter(parameter.DisabledAuthenticationMethods, "{{.DisabledAuthenticationMethods}}").
		AddParameter(parameter.DisabledMFAMethods, "{{.DisabledMFAMethods}}").
		AddParameter(parameter.EnabledAuthenticationMethods, "{{.EnabledAuthenticationMethods}}").
		AddParameter(parameter.EnabledMFAMethods, "{{.EnabledMFAMethods}}").
		AddParameter(parameter.LogoImageFile, "").
		AddParameter(parameter.MFAMethods, "").
		AddParameter(parameter.MFARequired, "false").
		AddParameter(parameter.OIDCAuthenticationSettings, "{{.OIDCAuthenticationSettings}}").
		AddParameter(parameter.PageBackgroundColor, "#F8F8F8").
		AddParameter(parameter.PageOrderSocialFirst, "false").
		AddParameter(parameter.PageTextColor, "#505060")

	if err := registerPageParameters(EveryPage, b.Build()); err != nil {
		panic(err)
	}
}
