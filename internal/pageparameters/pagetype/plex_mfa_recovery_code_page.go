package pagetype

import "userclouds.com/internal/pageparameters/parameter"

// PlexMfaRecoveryCodePage is the page type for the plex page for showing an mfa recovery code
const PlexMfaRecoveryCodePage Type = "plex_mfa_recovery_code_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.HeadingText, "New Multifactor Recovery Code")

	if err := registerPageParameters(PlexMfaRecoveryCodePage, b.Build()); err != nil {
		panic(err)
	}
}
