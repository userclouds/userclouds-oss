package saml

import (
	"net/http"

	"userclouds.com/infra/uchttp/builder"
)

type handler struct {
}

// NewHandler returns a new plex SAML handler
func NewHandler() (http.Handler, error) {

	h := &handler{}

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)

	hb.HandleFunc("/callback/", h.LoginCallback)
	hb.HandleFunc("/metadata/", h.ServeMetadata)
	hb.HandleFunc("/sso/", h.ServeSSO)

	return hb.Build(), nil
}

//go:generate genhandler /saml
