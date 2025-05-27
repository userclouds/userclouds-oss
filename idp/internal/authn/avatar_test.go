package authn_test

import (
	"fmt"
	"net/url"
	"testing"

	"userclouds.com/idp/internal/authn"
	"userclouds.com/infra/assert"
)

func TestGravatarHash(t *testing.T) {
	// Examples taken from https://en.gravatar.com/site/implement/hash/
	assert.Equal(t, authn.GenerateGravatarHash(" \tMyEmailAddress@contoso.com \n "), "d194c37c380e6f9ea9fa19a0d8abe7f9")
	assert.Equal(t, authn.GenerateGravatarHash("MyEmailAddress@contoso.com"), "d194c37c380e6f9ea9fa19a0d8abe7f9")
}

func TestDefaultAvatarURL(t *testing.T) {
	suffixString := url.QueryEscape(fmt.Sprintf("/%d/%s/%s", authn.DefaultSize, authn.DefaultBackground, authn.DefaultForeground))
	assert.Equal(t, authn.GenerateDefaultAvatarURL("test@userclouds.com", "display@name.com").String(),
		"https://www.gravatar.com/avatar/5c42705eda69dd7660ac012361eefb81?d=https%3A%2F%2Fui-avatars.com%2Fapi%2Fdisplay%40name.com"+suffixString)
	assert.Equal(t, authn.GenerateDefaultAvatarURL("test@userclouds.com", "Bob").String(),
		"https://www.gravatar.com/avatar/5c42705eda69dd7660ac012361eefb81?d=https%3A%2F%2Fui-avatars.com%2Fapi%2FBob"+suffixString)
	assert.Equal(t, authn.GenerateDefaultAvatarURL("test@userclouds.com", "foo Bar").String(),
		"https://www.gravatar.com/avatar/5c42705eda69dd7660ac012361eefb81?d=https%3A%2F%2Fui-avatars.com%2Fapi%2Ffoo%2520Bar"+suffixString)
}
