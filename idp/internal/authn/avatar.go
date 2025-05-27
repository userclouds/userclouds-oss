package authn

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// Default avatar constants (120x120px, white text on light blue background)
const (
	DefaultSize       = 120
	DefaultBackground = "70a0ff"
	DefaultForeground = "f0f0f0"
)

// GenerateGravatarHash generates the md5 hash for a user's email so we can
// look up their Gravatar image. Details from
func GenerateGravatarHash(email string) string {
	// From https://en.gravatar.com/site/implement/hash/
	md5Bytes := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(md5Bytes[:])
}

// GenerateDefaultAvatarURL uses Gravatar as an initial "guess" for a user's avatar
// and falls back to https://ui-avatars.com/ ("UI Avatars has a simple-to-use API with
// no limiting or login. No usage tracking and no information is stored. The final images
// are cached, but nothing else. Just write name or surname, or both.")
func GenerateDefaultAvatarURL(email, displayName string) *url.URL {
	// See https://ui-avatars.com/ for documentation of all of the parameters
	uiAvatarsURL := &url.URL{
		Scheme: "https",
		Host:   "ui-avatars.com",
		Path:   fmt.Sprintf("/api/%s/%d/%s/%s", displayName, DefaultSize, DefaultBackground, DefaultForeground),
	}
	// See https://en.gravatar.com/site/implement/images/ for documentation on
	// various parameters, such as ?d=... which specifies a default
	return &url.URL{
		Scheme: "https",
		Host:   "www.gravatar.com",
		Path:   fmt.Sprintf("/avatar/%s", GenerateGravatarHash(email)),
		RawQuery: url.Values{
			"d": []string{uiAvatarsURL.String()},
		}.Encode(),
	}
}
