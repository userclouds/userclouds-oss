package uctest

import "net/url"

// MustParseURL is a test helper that allows for global "constant" URLs to be created.
func MustParseURL(u string) *url.URL {
	parsedURL, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	return parsedURL
}
