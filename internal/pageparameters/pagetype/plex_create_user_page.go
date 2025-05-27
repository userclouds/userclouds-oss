package pagetype

import "userclouds.com/internal/pageparameters/parameter"

// PlexCreateUserPage is the page type for the plex create user page
const PlexCreateUserPage Type = "plex_create_user_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.ActionSubmitButtonText, "Create").
		AddParameter(parameter.AllowCreation, "{{.AllowCreation}}").
		AddParameter(parameter.CreateUserDisabledText, "Account creation not allowed").
		AddParameter(parameter.CreateUserFailStatusText, "Create user failed").
		AddParameter(parameter.CreateUserStartStatusText, "Creating user...").
		AddParameter(parameter.CustomCreateUserPagePostFieldsetHTMLCSS, "").
		AddParameter(parameter.CustomCreateUserPagePreFieldsetHTMLCSS, "").
		AddParameter(parameter.CustomCreateUserPagePostFormHTMLCSS, "").
		AddParameter(parameter.CustomCreateUserPagePreFormHTMLCSS, "").
		AddParameter(parameter.CustomCreateUserPagePreMainHTMLCSS, "").
		AddParameter(parameter.EmailLabel, "Email").
		AddParameter(parameter.FooterHTML, "").
		AddParameter(parameter.HeadingText, "Create User").
		AddParameter(parameter.LoginFailStatusText, "Login failed").
		AddParameter(parameter.LoginSuccessStatusText, "Redirecting...").
		AddParameter(parameter.PasswordLabel, "Password").
		AddParameter(parameter.RequireName, "true").
		AddParameter(parameter.SubheadingText, "")

	if err := registerPageParameters(PlexCreateUserPage, b.Build()); err != nil {
		panic(err)
	}
}
