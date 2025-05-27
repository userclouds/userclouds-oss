package uctest

import (
	"context"
	"net/url"
	"strings"

	"userclouds.com/infra/ucerr"
)

// EmailClient is a test-friendly email client.
type EmailClient struct {
	Bodies     []string
	HTMLBodies []string
}

// ExtractURL parses the first URL from a given string
func ExtractURL(emailBody string) (*url.URL, error) {
	urlIndex := strings.Index(emailBody, "http://")
	if urlIndex == -1 {
		urlIndex = strings.Index(emailBody, "https://")
	}

	if urlIndex == -1 {
		return nil, ucerr.Errorf("did not find url scheme in string '%s'", emailBody)
	}

	trimmed := emailBody[urlIndex:]
	if endIndex := strings.Index(trimmed, `"`); endIndex != -1 {
		trimmed = trimmed[:endIndex]
	}
	stringParts := strings.Fields(trimmed)
	if len(stringParts) > 0 {
		return url.Parse(stringParts[0])
	}

	// this can't actually happen because we should have had a non-empty trimmed string
	return nil, ucerr.New("unknown parse error in ExtractURL")
}

// Clear resets the mock email client for testing
func (e *EmailClient) Clear() {
	e.Bodies = nil
	e.HTMLBodies = nil
}

// Send implements email.Client and captures emails, URLs, etc
func (e *EmailClient) Send(ctx context.Context, to, from, subject, body string) error {
	e.Bodies = append(e.Bodies, body)
	return nil
}

// SendWithHTML implements email.Client and captures emails, URLs, etc
func (e *EmailClient) SendWithHTML(ctx context.Context, to, from, subject, body string, htmlBody string) error {
	e.Bodies = append(e.Bodies, body)
	e.HTMLBodies = append(e.HTMLBodies, htmlBody)
	return nil
}
