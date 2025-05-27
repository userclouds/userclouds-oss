package companyconfig

import (
	"regexp"
	"strings"

	"userclouds.com/infra/ucerr"
)

// Valid hostnames can't start or end with a hyphen, and we always lowercase them
// in our code to simplify things. They must also have at least one alphanumeric
// character and not exceed a max length.
var validHostname = regexp.MustCompile(`^([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])$`)
var stripInvalidChars = regexp.MustCompile(`[^a-z0-9-]+`)
var noDoubleHyphens = regexp.MustCompile(`\-\-+`)

// Max label length is 63, but we should reserve some chars in case?
const maxTenantHostnameLength = 48

// GenerateSafeHostname converts an arbitrary string to a safe hostname
func GenerateSafeHostname(name string) (string, error) {
	// Always store & work with lower case internally
	lowerName := strings.ToLower(name)

	// Strip all non alphanumeric and hyphen characters
	noInvalidChars := stripInvalidChars.ReplaceAllString(lowerName, "")

	// Get rid of double hyphens since there are some weird rules about those
	// https://datatracker.ietf.org/doc/html/rfc5891#section-4.2.3.1.
	noInvalidChars = noDoubleHyphens.ReplaceAllString(noInvalidChars, "-")

	// Trim prefix hyphens
	trimmed := noInvalidChars
	for strings.HasPrefix(trimmed, "-") {
		trimmed = strings.TrimPrefix(trimmed, "-")
	}

	// Trim overall length
	if len(trimmed) > maxTenantHostnameLength {
		trimmed = trimmed[0:maxTenantHostnameLength]
	}

	// Trim suffix hyphens (in case trimming the length resulted in a hyphen at the end)
	for strings.HasSuffix(trimmed, "-") {
		trimmed = strings.TrimSuffix(trimmed, "-")
	}

	if len(trimmed) == 0 {
		return "", ucerr.Errorf("unable to construct valid hostname from '%s'", name)
	}

	// Final sanity check
	if validHostname.MatchString(trimmed) {
		return trimmed, nil
	}

	// Not sure how we could get here?
	return "", ucerr.Errorf("error with auto-generated hostname '%s' constructed from '%s'", trimmed, name)
}
