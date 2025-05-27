package secret

import (
	"os"
	"strings"

	"userclouds.com/infra/ucerr"
)

// PrefixEnv tells us this is a secret from the environment variables
const PrefixEnv Prefix = "env://"

func getEnvVariableSecret(s string) (string, error) {
	if !strings.HasPrefix(s, string(PrefixEnv)) {
		return "", ucerr.Errorf("getEnvVariableSecret got secret %s without EnvPrefix %s", s, PrefixEnv)
	}

	varName := strings.TrimPrefix(s, string(PrefixEnv))
	secret, defined := os.LookupEnv(varName)
	if !defined {
		return "", ucerr.Errorf("Can't load secret from environment variable %s", varName)
	}
	if secret == "" {
		return "", ucerr.Errorf("Secret from environment variable %s is empty", varName)
	}
	return secret, nil
}
