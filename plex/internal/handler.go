package internal

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/service"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/security"
	"userclouds.com/plex/internal/create"
	"userclouds.com/plex/internal/delegation"
	"userclouds.com/plex/internal/invite"
	"userclouds.com/plex/internal/loginapp"
	"userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/resetpassword"
	"userclouds.com/plex/internal/saml"
	"userclouds.com/plex/internal/social"
	"userclouds.com/plex/internal/tenantconfig"
)

// Option defines an interface to pass optional arguments to NewHandler
type Option interface {
	apply(*handler)
}

type optFunc func(h *handler)

func (o optFunc) apply(h *handler) {
	o(h)
}

// Factory returns an Option that overrides the factory for this handler
func Factory(f provider.Factory) Option {
	return optFunc(func(h *handler) { h.factory = f })
}

type handler struct {
	checker              security.ReqValidator
	mfaHandler           *mfaHandler
	companyConfigStorage *companyconfig.Storage
	tcCache              *tenantconfig.Cache
	factory              provider.Factory
	consoleTenantInfo    companyconfig.TenantInfo
	m2mAuth              jsonclient.Option
}

func (h *handler) applyOptions(opts []Option) {
	for _, o := range opts {
		o.apply(h)
	}
}

// NewHandler returns a Plex handler.
func NewHandler(
	s *companyconfig.Storage,
	jwtVerifier auth.Verifier,
	validator security.ReqValidator,
	emailClient *email.Client,
	tcCache *tenantconfig.Cache,
	qc workerclient.Client,
	m2mAuth jsonclient.Option,
	consoleTenantInfo companyconfig.TenantInfo,
	consoleEP *service.Endpoint,
	consolePK *rsa.PublicKey,
	opts ...Option,
) (http.Handler, error) {
	h := &handler{
		checker:              validator,
		companyConfigStorage: s,
		tcCache:              tcCache,
		consoleTenantInfo:    consoleTenantInfo,
		m2mAuth:              m2mAuth,
		factory:              provider.ProdFactory{EmailClient: emailClient, ConsoleEP: consoleEP},
	}
	h.applyOptions(opts)
	hb := builder.NewHandlerBuilder()

	// API to handle login requests from our login UI or from an embedded
	// login page hosted by the application.
	hb.HandleFunc("/login", h.loginHandler)

	hb.HandleFunc("/impersonateuser", h.impersonateHandler)

	// API to handle logout requests & redirect user agent from application.
	hb.HandleFunc(paths.LogoutPath, h.logoutHandler)
	mfaHandler, mfaHTTPHandler := newMFAHandler(h.factory)
	h.mfaHandler = mfaHandler
	hb.Handle("/mfa/", mfaHTTPHandler)

	// API to trigger starting passwordless login.
	if emailClient != nil {
		plHandler := newPasswordlessHandler(validator, *emailClient, h.factory)
		hb.Handle("/passwordless", plHandler)
	}

	// API to retrieve page parameters for a login session.
	hb.MethodHandler("/login/pageparameters").Post(h.pageParametersHandler)

	otpHandler := otp.NewHandler(h.factory)
	hb.Handle(otp.RootPath, otpHandler)

	createHandler := create.NewHandler(emailClient, h.factory)
	hb.Handle("/create/", createHandler)

	inviteHandler := invite.NewHandler(emailClient, h.factory)
	// TODO: this granularity of applying middleware seems error prone, but
	// most Plex endpoints don't use auth. Perhaps we should refactor those
	// that do into their own top-level handler so this can be applied in routes.go?

	hb.Handle("/invite/", auth.Middleware(jwtVerifier, consoleTenantInfo.TenantID).Apply(inviteHandler))

	loginAppHandler := loginapp.NewHandler()
	hb.Handle("/loginapp/", auth.Middleware(jwtVerifier, consoleTenantInfo.TenantID).Apply(loginAppHandler))

	delegationHandler := delegation.NewHandler(h.factory, jwtVerifier, consoleTenantInfo.TenantID)
	hb.Handle("/delegation", delegationHandler)
	if emailClient != nil {
		resetPasswordHandler := resetpassword.NewHandler(*emailClient, h.factory)
		hb.Handle(paths.ResetPasswordRootPath, resetPasswordHandler)
	}

	oidcHandler, authorize, userinfo := oidc.NewHandler(h.factory, consolePK)
	hb.Handle("/oidc/", oidcHandler)

	// we also map the same three URLS (/authorize, /token, /userinfo) to these paths
	// so that we are drop-in replacement for Auth0 even for libraries *cough* Auth0.js *cough*
	// that do not correctly use OIDC discovery
	hb.Handle("/oauth/", oidcHandler)
	hb.HandleFunc("/authorize", authorize)
	hb.HandleFunc("/userinfo", userinfo)

	socialHandler := social.NewHandler(h.factory)
	hb.Handle(paths.SocialRootPath, socialHandler)

	// Auth0 compatibility/shim APIs - TODO: where do these live longer term?
	hb.HandleFunc("/v2/logout", h.auth0LogoutHandler)

	// employee auth callback
	hb.HandleFunc("/employee/authcallback", h.handleEmployeeAuthCallback)

	// SAML handling
	samlHandler, err := saml.NewHandler()
	if err != nil {
		uclog.Fatalf(context.Background(), "failed to create SAML handler: %v", err)
	}
	hb.Handle("/saml/", samlHandler)

	// don't 404 on the URL, at least say hi
	hb.HandleFunc("/", h.baseHandler)

	return hb.Build(), nil
}

// this is a very basic friendly root handler
func (h *handler) baseHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tu := tenantconfig.MustGetTenantURLString(ctx)

	fmt.Fprintf(w, "Welcome to %s", tu)
}
