package oidc

// TokenSource describes a source of JWTs for jsonclient etc
type TokenSource interface {
	GetToken() (string, error)
}
