package secret

import (
	"encoding/base64"
	"fmt"
	"strings"

	"userclouds.com/infra/ucerr"
)

// PrefixDev tells us this is a dev-only Base64 encoded secret
const PrefixDev Prefix = "dev://"

// PrefixDevLiteral tells us this is dev-only and not obfuscated
// This exists separate from DevPrefix because sometimes it's useful
// to be able to read the secret in plaintext (eg. in ci.yaml files)
// and sometimes we want a slightly more interesting test of resolvers
const PrefixDevLiteral Prefix = "dev-literal://"

func getDevSecret(s string) (string, error) {
	if !strings.HasPrefix(s, string(PrefixDev)) {
		return "", ucerr.Errorf("getDevSecret got secret %s without DevPrefix %s", s, PrefixDev)
	}

	data := strings.TrimPrefix(s, string(PrefixDev))
	bs, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	return string(bs), nil
}

func newDevString(s string) String {
	return String{location: fmt.Sprintf("%s%s", PrefixDev, base64.StdEncoding.EncodeToString([]byte(s)))}
}
