package oidc

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"

	"userclouds.com/infra/ucerr"
)

// FacebookProvider defines configuration for a facebook OIDC client and implements the Provider interface
type FacebookProvider struct {
	baseProvider
}

var defaultFacebookProviderConfig = ProviderConfig{
	Type:                    ProviderTypeFacebook,
	Name:                    ProviderTypeFacebook.String(),
	Description:             "Facebook",
	IssuerURL:               fbIssuerURL,
	CanUseLocalHostRedirect: true,
	DefaultScopes:           fbDefaultScopes,
	IsNative:                true,
}

const fbAlternateSubjectClaimKey = "id"
const fbAPIVersion = "v15.0"
const fbIssuerURL = "https://www.facebook.com"
const fbJWKSURL = "https://www.facebook.com/.well-known/oauth/openid/jwks"
const fbUserInfoFields = "id,first_name,last_name,name,name_format,picture,short_name,email"

var fbDefaultScopes = fmt.Sprintf("%s %s %s", oidc.ScopeOpenID, "public_profile", "email")

func (FacebookProvider) getVersionedURLs() (authURL string, tokenURL string, userInfoURL string) {
	return fmt.Sprintf("https://www.facebook.com/%s/dialog/oauth", fbAPIVersion),
		fmt.Sprintf("https://graph.facebook.com/%s/oauth/access_token", fbAPIVersion),
		fmt.Sprintf("https://graph.facebook.com/%s/me?fields=%s", fbAPIVersion, fbUserInfoFields)
}

// CreateAuthenticator is part of the Provider interface and creates an authenticator for facebook
func (p FacebookProvider) CreateAuthenticator(redirectURL string) (*Authenticator, error) {
	authURL, tokenURL, userInfoURL := p.getVersionedURLs()
	scopes := p.getCombinedScopes()
	return newAuthenticatorViaConfiguration(
		context.Background(),
		p.GetIssuerURL(),
		authURL,
		tokenURL,
		userInfoURL,
		fbJWKSURL,
		fbAlternateSubjectClaimKey,
		p.config.ClientID,
		p.config.ClientSecret,
		redirectURL,
		scopes)
}

// GetDefaultSettings is part of the Provider interface and returns the default configuration for a facebook provider
func (FacebookProvider) GetDefaultSettings() ProviderConfig {
	return defaultFacebookProviderConfig
}

// ValidateAdditionalScopes is part of the Provider interface and validates the additional scopes
func (p FacebookProvider) ValidateAdditionalScopes() error {
	for _, scope := range SplitTokens(p.GetAdditionalScopes()) {
		if _, exists := validFBScopes[scope]; !exists {
			return ucerr.Friendlyf(nil, "\"%v\" is not a valid Facebook oauth scope", scope)
		}
	}

	return nil
}

var validFBScopes = map[string]bool{
	"ads_management":                  true,
	"ads_read":                        true,
	"attribution_read":                true,
	"catalog_management":              true,
	"business_management":             true,
	"email":                           true,
	"gaming_user_locale":              true,
	"groups_access_member_info":       true,
	"instagram_basic":                 true,
	"instagram_content_publish":       true,
	"instagram_manage_comments":       true,
	"instagram_manage_insights":       true,
	"instagram_manage_messages":       true,
	"instagram_shopping_tag_products": true,
	"leads_retrieval":                 true,
	"manage_pages":                    true,
	"openid":                          true,
	"page_events":                     true,
	"pages_manage_ads":                true,
	"pages_manage_cta":                true,
	"pages_manage_engagement":         true,
	"pages_manage_instant_articles":   true,
	"pages_manage_metadata":           true,
	"pages_manage_posts":              true,
	"pages_messaging":                 true,
	"pages_read_engagement":           true,
	"pages_read_user_content":         true,
	"pages_show_list":                 true,
	"pages_user_gender":               true,
	"pages_user_locale":               true,
	"pages_user_timezone":             true,
	"publish_pages":                   true,
	"public_profile":                  true,
	"publish_to_groups":               true,
	"publish_video":                   true,
	"read_insights":                   true,
	"user_age_range":                  true,
	"user_birthday":                   true,
	"user_friends":                    true,
	"user_gender":                     true,
	"user_hometown":                   true,
	"user_likes":                      true,
	"user_link":                       true,
	"user_location":                   true,
	"user_messenger_contact":          true,
	"user_photos":                     true,
	"user_posts":                      true,
	"user_videos":                     true,
	"whatsapp_business_management":    true,
	"whatsapp_business_messaging":     true,
}
