package pagetype

import "userclouds.com/internal/pageparameters/parameter"

// PlexMfaChannelPage is the page type for the plex page for selecting an mfa channel
const PlexMfaChannelPage Type = "plex_mfa_channel_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.HeadingText, "Choose a Multifactor authentication method")

	if err := registerPageParameters(PlexMfaChannelPage, b.Build()); err != nil {
		panic(err)
	}
}
