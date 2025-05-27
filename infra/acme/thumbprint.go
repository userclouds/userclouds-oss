package acme

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"

	"github.com/lestrrat-go/jwx/jwk"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// ComputeThumbprint computes the JWK thumbprint of the given private key
func ComputeThumbprint(ctx context.Context, pemKey string) (string, error) {
	b, _ := pem.Decode([]byte(pemKey))
	if b == nil {
		return "", ucerr.New("failed to decode PEM")
	}

	pk, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	j, err := jwk.New(pk)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create JWK: %v", err)
	}

	// JWS thumbprint defined here https://datatracker.ietf.org/doc/html/rfc7638
	// and using SHA256 as defined here: https://www.rfc-editor.org/rfc/rfc8555.html#section-8.1
	bs, err := j.Thumbprint(crypto.SHA256)
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	s := base64.RawURLEncoding.EncodeToString(bs)

	// NB this is sort of wacky, but ACME requires Base64 URL encoding, here:
	// https://www.rfc-editor.org/rfc/rfc8555.html#section-8.1,
	// but with any padding stripped off, as specified here:
	// https://www.rfc-editor.org/rfc/rfc8555.html#section-8.1
	// which references these:
	// https://www.rfc-editor.org/rfc/rfc7515#section-2
	// https://www.rfc-editor.org/rfc/rfc7515#appendix-C

	// TODO: these should never happen with RawURLEncoding
	if strings.Contains(s, "=") {
		uclog.Warningf(ctx, "unexpected = in thumbprint: %s", s)
	}
	s = strings.TrimSuffix(s, "=")

	// i've never seen this occur, so logging a warning for now for more testing
	if strings.Contains(s, "+") || strings.Contains(s, "/") {
		uclog.Warningf(ctx, "untested thumbprint: %s", s)
	}

	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")

	return s, nil
}
