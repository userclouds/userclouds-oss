package saml

import (
	"bytes"
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/xmlenc"
	"github.com/go-http-utils/headers"
	xrv "github.com/mattermost/xml-roundtrip-validator"
	dsig "github.com/russellhaering/goxmldsig"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/tenantplex/samlconfig"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

// IdentityProvider implements the SAML Identity Provider role (IDP).
//
// An identity provider receives SAML assertion requests and responds
// with SAML Assertions.
//
// You must provide a keypair that is used to
// sign assertions.
//
// You must provide an implementation of ServiceProviderProvider which
// returns
//
// You must provide an implementation of the SessionProvider which
// handles the actual authentication (i.e. prompting for a username
// and password).
type IdentityProvider struct {
	Key              crypto.PrivateKey
	Certificate      *x509.Certificate
	Intermediates    []*x509.Certificate
	MetadataURL      url.URL
	SSOURL           url.URL
	LogoutURL        url.URL
	SignatureMethod  string
	ValidDuration    *time.Duration
	ServiceProviders map[string]*samlconfig.EntityDescriptor
}

// Metadata returns the metadata structure for this identity provider.
func (idp *IdentityProvider) Metadata() *samlconfig.EntityDescriptor {
	certStr := base64.StdEncoding.EncodeToString(idp.Certificate.Raw)

	var validDuration time.Duration
	if idp.ValidDuration != nil {
		validDuration = *idp.ValidDuration
	} else {
		validDuration = DefaultValidDuration
	}

	ed := &samlconfig.EntityDescriptor{
		EntityID:      idp.MetadataURL.String(),
		ValidUntil:    TimeNow().Add(validDuration),
		CacheDuration: validDuration,
		IDPSSODescriptors: []samlconfig.IDPSSODescriptor{
			{
				SSODescriptor: samlconfig.SSODescriptor{
					RoleDescriptor: samlconfig.RoleDescriptor{
						ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
						KeyDescriptors: []samlconfig.KeyDescriptor{
							{
								Use: "signing",
								KeyInfo: samlconfig.KeyInfo{
									X509Data: samlconfig.X509Data{
										X509Certificates: []samlconfig.X509Certificate{
											{Data: certStr},
										},
									},
								},
							},
							{
								Use: "encryption",
								KeyInfo: samlconfig.KeyInfo{
									X509Data: samlconfig.X509Data{
										X509Certificates: []samlconfig.X509Certificate{
											{Data: certStr},
										},
									},
								},
								EncryptionMethods: []samlconfig.EncryptionMethod{
									{Algorithm: "http://www.w3.org/2001/04/xmlenc#aes128-cbc"},
									{Algorithm: "http://www.w3.org/2001/04/xmlenc#aes192-cbc"},
									{Algorithm: "http://www.w3.org/2001/04/xmlenc#aes256-cbc"},
									{Algorithm: "http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p"},
								},
							},
						},
					},
					NameIDFormats: []samlconfig.NameIDFormat{samlconfig.NameIDFormat("urn:oasis:names:tc:SAML:2.0:nameid-format:transient")},
				},
				SingleSignOnServices: []samlconfig.Endpoint{
					{
						Binding:  samlconfig.HTTPRedirectBinding,
						Location: idp.SSOURL.String(),
					},
					{
						Binding:  samlconfig.HTTPPostBinding,
						Location: idp.SSOURL.String(),
					},
				},
			},
		},
	}

	if idp.LogoutURL.String() != "" {
		ed.IDPSSODescriptors[0].SSODescriptor.SingleLogoutServices = []samlconfig.Endpoint{
			{
				Binding:  samlconfig.HTTPRedirectBinding,
				Location: idp.LogoutURL.String(),
			},
		}
	}

	return ed
}

// ServeMetadata is an http.HandlerFunc that serves the IDP metadata
func (h *handler) ServeMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idp, _, err := h.getIDP(r)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	buf, err := xml.MarshalIndent(idp.Metadata(), "", "  ")
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set(headers.ContentType, "application/samlmetadata+xml")
	if _, err := w.Write(buf); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
	}
}

// getIDP returns the appropriate IDP structure for this request (to a given app ID etc)
func (h *handler) getIDP(r *http.Request) (*IdentityProvider, *tenantplex.App, error) {
	ctx := r.Context()
	pm := tenantconfig.MustGetPlexMap(ctx)

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		return nil, nil, ucerr.Errorf("invalid path:  %v", r.URL.Path)
	}
	appID := parts[2]

	var app *tenantplex.App
	var sc *tenantplex.SAMLIDP
	for i, a := range pm.Apps {
		if appID == a.ClientID {
			sc = pm.Apps[i].SAMLIDP
			app = &pm.Apps[i]
			break
		}
	}

	if sc == nil {
		return nil, nil, ucerr.Errorf("invalid app id: %v", appID)
	}

	idp, err := getIDPForApp(ctx, app)
	return idp, app, ucerr.Wrap(err)
}

func getIDPForApp(ctx context.Context, app *tenantplex.App) (*IdentityProvider, error) {
	sc := app.SAMLIDP

	mu, err := url.Parse(sc.MetadataURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	su, err := url.Parse(sc.SSOURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	pkPEM, err := sc.PrivateKey.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	key, _ := pem.Decode([]byte(pkPEM))
	if key == nil {
		return nil, ucerr.Errorf("invalid private key")
	}
	pk, err := x509.ParsePKCS1PrivateKey(key.Bytes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	certPEM, _ := pem.Decode([]byte(sc.Certificate))
	if certPEM == nil {
		return nil, ucerr.Errorf("invalid certificate")
	}
	cert, err := x509.ParseCertificate(certPEM.Bytes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var sps = make(map[string]*samlconfig.EntityDescriptor)
	for i, sp := range sc.TrustedServiceProviders {
		sps[sp.EntityID] = &sc.TrustedServiceProviders[i]
	}

	return &IdentityProvider{
		Key:              pk,
		Certificate:      cert,
		MetadataURL:      *mu,
		SSOURL:           *su,
		ServiceProviders: sps,
	}, nil
}

// ServeSSO handles SAML auth requests.
//
// When it gets a request for a user that does not have a valid session,
// then it prompts the user via XXX.
//
// If the session already exists, then it produces a SAML assertion and
// returns an HTTP response according to the specified binding. The
// only supported binding right now is the HTTP-POST binding which returns
// an HTML form in the appropriate format with Javascript to automatically
// submit that form the to service provider's Assertion Customer Service
// endpoint.
//
// If the SAML request is invalid or cannot be verified a simple StatusBadRequest
// response is sent.
//
// If the assertion cannot be created or returned, a StatusInternalServerError
// response is sent.
func (h *handler) ServeSSO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idp, _, err := h.getIDP(r)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	req, err := NewIdpAuthnRequest(idp, r)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	if err := h.ValidateRequest(req); err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	// TODO(ross): we must check that the request ID has not been previously
	//   issued.

	session := h.GetSessionOrRedirect(w, r, req)
	if session == nil {
		return
	}

	h.ContinueSSO(w, r, req, session)
}

// ContinueSSO allows us to continue the SAML sign-in after Plex sign-in completes
func (h *handler) ContinueSSO(w http.ResponseWriter, r *http.Request, req *IdpAuthnRequest, session *storage.SAMLSession) {
	ctx := r.Context()

	if err := h.MakeAssertion(req, session); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
	if err := h.WriteResponse(w, req); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
}

// ServeIDPInitiated handes an IDP-initiated authorization request. Requests of this
// type require us to know a registered service provider and (optionally) the RelayState
// that will be passed to the application.
func (h *handler) ServeIDPInitiated(w http.ResponseWriter, r *http.Request, serviceProviderID string, relayState string) {
	ctx := r.Context()

	idp, _, err := h.getIDP(r)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	req := &IdpAuthnRequest{
		HTTPRequest: r,
		RelayState:  relayState,
		Now:         TimeNow(),
	}

	session := h.GetSessionOrRedirect(w, r, req)
	if session == nil {
		// If GetSession returns nil, it must have written an HTTP response, per the interface
		// (this is probably because it drew a login form or something)
		return
	}

	req.ServiceProviderMetadata, err = idp.GetServiceProvider(serviceProviderID)
	if errors.Is(err, os.ErrNotExist) {
		uchttp.Error(ctx, w, err, http.StatusNotFound)
		return
	} else if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	// find an ACS endpoint that we can use
	for _, spssoDescriptor := range req.ServiceProviderMetadata.SPSSODescriptors {
		for _, endpoint := range spssoDescriptor.AssertionConsumerServices {
			if endpoint.Binding == samlconfig.HTTPPostBinding {
				// explicitly copy loop iterator variables
				//
				// c.f. https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
				//
				// (note that I'm pretty sure this isn't strictly necessary because we break out of the loop immediately,
				// but it certainly doesn't hurt anything and may prevent bugs in the future.)
				endpoint, spssoDescriptor := endpoint, spssoDescriptor

				req.ACSEndpoint = &endpoint
				req.SPSSODescriptor = &spssoDescriptor
				break
			}
		}
		if req.ACSEndpoint != nil {
			break
		}
	}
	if req.ACSEndpoint == nil {
		uchttp.Error(ctx, w, ucerr.New("saml metadata does not contain an Assertion Customer Service url"), http.StatusInternalServerError)
		return
	}

	if err := h.MakeAssertion(req, session); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	if err := h.WriteResponse(w, req); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
}

// IdpAuthnRequest is used by IdentityProvider to handle a single authentication request.
type IdpAuthnRequest struct {
	HTTPRequest             *http.Request
	RelayState              string
	RequestBuffer           []byte
	Request                 saml.AuthnRequest
	ServiceProviderMetadata *samlconfig.EntityDescriptor
	SPSSODescriptor         *samlconfig.SPSSODescriptor
	ACSEndpoint             *samlconfig.IndexedEndpoint
	Assertion               *saml.Assertion
	AssertionEl             *etree.Element
	ResponseEl              *etree.Element
	Now                     time.Time
}

// NewIdpAuthnRequest returns a new IdpAuthnRequest for the given HTTP request to the authorization
// service.
func NewIdpAuthnRequest(idp *IdentityProvider, r *http.Request) (*IdpAuthnRequest, error) {
	req := &IdpAuthnRequest{
		HTTPRequest: r,
		Now:         TimeNow(),
	}

	switch r.Method {
	case "GET":
		compressedRequest, err := base64.StdEncoding.DecodeString(r.URL.Query().Get("SAMLRequest"))
		if err != nil {
			return nil, ucerr.Errorf("cannot decode request: %s", err)
		}
		req.RequestBuffer, err = io.ReadAll(newSaferFlateReader(bytes.NewReader(compressedRequest)))
		if err != nil {
			return nil, ucerr.Errorf("cannot decompress request: %s", err)
		}
		req.RelayState = r.URL.Query().Get("RelayState")
	case "POST":
		if err := r.ParseForm(); err != nil {
			return nil, ucerr.Wrap(err)
		}
		var err error
		req.RequestBuffer, err = base64.StdEncoding.DecodeString(r.PostForm.Get("SAMLRequest"))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		req.RelayState = r.PostForm.Get("RelayState")
	default:
		return nil, ucerr.Errorf("method not allowed")
	}

	return req, nil
}

// Validate checks that the authentication request is valid and assigns
// the AuthnRequest and Metadata properties. Returns a non-nil error if the
// request is not valid.
func (h *handler) ValidateRequest(req *IdpAuthnRequest) error {
	idp, _, err := h.getIDP(req.HTTPRequest)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := xrv.Validate(bytes.NewReader(req.RequestBuffer)); err != nil {
		return ucerr.Wrap(err)
	}

	if err := xml.Unmarshal(req.RequestBuffer, &req.Request); err != nil {
		return ucerr.Wrap(err)
	}

	// We always have exactly one IDP SSO descriptor
	if len(idp.Metadata().IDPSSODescriptors) != 1 {
		panic("expected exactly one IDP SSO descriptor in IDP metadata")
	}
	idpSsoDescriptor := idp.Metadata().IDPSSODescriptors[0]

	// TODO(ross): support signed authn requests
	// For now we do the safe thing and fail in the case where we think
	// requests might be signed.
	if idpSsoDescriptor.WantAuthnRequestsSigned != nil && *idpSsoDescriptor.WantAuthnRequestsSigned {
		return ucerr.Errorf("authn request signature checking is not currently supported")
	}

	// In http://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf §3.4.5.2
	// we get a description of the Destination attribute:
	//
	//   If the message is signed, the Destination XML attribute in the root SAML
	//   element of the protocol message MUST contain the URL to which the sender
	//   has instructed the user agent to deliver the message. The recipient MUST
	//   then verify that the value matches the location at which the message has
	//   been received.
	//
	// We require the destination be correct either (a) if signing is enabled or
	// (b) if it was provided.
	mustHaveDestination := idpSsoDescriptor.WantAuthnRequestsSigned != nil && *idpSsoDescriptor.WantAuthnRequestsSigned
	mustHaveDestination = mustHaveDestination || req.Request.Destination != ""
	if mustHaveDestination {
		if req.Request.Destination != idp.SSOURL.String() {
			return ucerr.Errorf("expected destination to be %q, not %q", idp.SSOURL.String(), req.Request.Destination)
		}
	}

	if req.Request.IssueInstant.Add(MaxIssueDelay).Before(req.Now) {
		return ucerr.Errorf("request expired at %s",
			req.Request.IssueInstant.Add(MaxIssueDelay))
	}
	if req.Request.Version != "2.0" {
		return ucerr.Errorf("expected SAML request version 2.0 got %v", req.Request.Version)
	}

	// find the service provider
	serviceProviderID := req.Request.Issuer.Value
	serviceProvider, err := idp.GetServiceProvider(serviceProviderID)
	if errors.Is(err, os.ErrNotExist) {
		return ucerr.Errorf("cannot handle request from unknown service provider %s", serviceProviderID)
	} else if err != nil {
		return ucerr.Errorf("cannot find service provider %s: %v", serviceProviderID, err)
	}
	req.ServiceProviderMetadata = serviceProvider

	// Check that the ACS URL matches an ACS endpoint in the SP metadata.
	if err := req.getACSEndpoint(); err != nil {
		return ucerr.Errorf("cannot find assertion consumer service: %v", err)
	}

	return nil
}

func (req *IdpAuthnRequest) getACSEndpoint() error {
	if req.Request.AssertionConsumerServiceIndex != "" {
		for _, spssoDescriptor := range req.ServiceProviderMetadata.SPSSODescriptors {
			for _, spAssertionConsumerService := range spssoDescriptor.AssertionConsumerServices {
				if strconv.Itoa(spAssertionConsumerService.Index) == req.Request.AssertionConsumerServiceIndex {
					// explicitly copy loop iterator variables
					//
					// c.f. https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
					//
					// (note that I'm pretty sure this isn't strictly necessary because we break out of the loop immediately,
					// but it certainly doesn't hurt anything and may prevent bugs in the future.)
					spssoDescriptor, spAssertionConsumerService := spssoDescriptor, spAssertionConsumerService

					req.SPSSODescriptor = &spssoDescriptor
					req.ACSEndpoint = &spAssertionConsumerService
					return nil
				}
			}
		}
	}

	if req.Request.AssertionConsumerServiceURL != "" {
		for _, spssoDescriptor := range req.ServiceProviderMetadata.SPSSODescriptors {
			for _, spAssertionConsumerService := range spssoDescriptor.AssertionConsumerServices {
				if spAssertionConsumerService.Location == req.Request.AssertionConsumerServiceURL {
					// explicitly copy loop iterator variables
					//
					// c.f. https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
					//
					// (note that I'm pretty sure this isn't strictly necessary because we break out of the loop immediately,
					// but it certainly doesn't hurt anything and may prevent bugs in the future.)
					spssoDescriptor, spAssertionConsumerService := spssoDescriptor, spAssertionConsumerService

					req.SPSSODescriptor = &spssoDescriptor
					req.ACSEndpoint = &spAssertionConsumerService
					return nil
				}
			}
		}
	}

	// Some service providers, like the Microsoft Azure AD service provider, issue
	// assertion requests that don't specify an ACS url at all.
	if req.Request.AssertionConsumerServiceURL == "" && req.Request.AssertionConsumerServiceIndex == "" {
		// find a default ACS binding in the metadata that we can use
		for _, spssoDescriptor := range req.ServiceProviderMetadata.SPSSODescriptors {
			for _, spAssertionConsumerService := range spssoDescriptor.AssertionConsumerServices {
				if spAssertionConsumerService.IsDefault != nil && *spAssertionConsumerService.IsDefault {
					switch spAssertionConsumerService.Binding {
					case samlconfig.HTTPPostBinding, samlconfig.HTTPRedirectBinding:
						// explicitly copy loop iterator variables
						//
						// c.f. https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
						//
						// (note that I'm pretty sure this isn't strictly necessary because we break out of the loop immediately,
						// but it certainly doesn't hurt anything and may prevent bugs in the future.)
						spssoDescriptor, spAssertionConsumerService := spssoDescriptor, spAssertionConsumerService

						req.SPSSODescriptor = &spssoDescriptor
						req.ACSEndpoint = &spAssertionConsumerService
						return nil
					}
				}
			}
		}

		// if we can't find a default, use *any* ACS binding
		for _, spssoDescriptor := range req.ServiceProviderMetadata.SPSSODescriptors {
			for _, spAssertionConsumerService := range spssoDescriptor.AssertionConsumerServices {
				switch spAssertionConsumerService.Binding {
				case samlconfig.HTTPPostBinding, samlconfig.HTTPRedirectBinding:
					// explicitly copy loop iterator variables
					//
					// c.f. https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
					//
					// (note that I'm pretty sure this isn't strictly necessary because we break out of the loop immediately,
					// but it certainly doesn't hurt anything and may prevent bugs in the future.)
					spssoDescriptor, spAssertionConsumerService := spssoDescriptor, spAssertionConsumerService

					req.SPSSODescriptor = &spssoDescriptor
					req.ACSEndpoint = &spAssertionConsumerService
					return nil
				}
			}
		}
	}

	return ucerr.Wrap(os.ErrNotExist) // no ACS url found or specified
}

// MakeAssertion implements AssertionMaker. It produces a SAML assertion from the
// given request and assigns it to req.Assertion.
func (h *handler) MakeAssertion(req *IdpAuthnRequest, session *storage.SAMLSession) error {
	idp, _, err := h.getIDP(req.HTTPRequest)
	if err != nil {
		return ucerr.Wrap(err)
	}

	attributes := []saml.Attribute{}

	var attributeConsumingService *samlconfig.AttributeConsumingService
	for _, acs := range req.SPSSODescriptor.AttributeConsumingServices {
		if acs.IsDefault != nil && *acs.IsDefault {
			// explicitly copy loop iterator variables
			//
			// c.f. https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
			//
			// (note that I'm pretty sure this isn't strictly necessary because we break out of the loop immediately,
			// but it certainly doesn't hurt anything and may prevent bugs in the future.)
			acs := acs

			attributeConsumingService = &acs
			break
		}
	}
	if attributeConsumingService == nil {
		for _, acs := range req.SPSSODescriptor.AttributeConsumingServices {
			// explicitly copy loop iterator variables
			//
			// c.f. https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
			//
			// (note that I'm pretty sure this isn't strictly necessary because we break out of the loop immediately,
			// but it certainly doesn't hurt anything and may prevent bugs in the future.)
			acs := acs

			attributeConsumingService = &acs
			break
		}
	}
	if attributeConsumingService == nil {
		attributeConsumingService = &samlconfig.AttributeConsumingService{}
	}

	for _, requestedAttribute := range attributeConsumingService.RequestedAttributes {
		if requestedAttribute.NameFormat == "urn:oasis:names:tc:SAML:2.0:attrname-format:basic" || requestedAttribute.NameFormat == "urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified" {
			attrName := requestedAttribute.Name
			attrName = regexp.MustCompile("[^A-Za-z0-9]+").ReplaceAllString(attrName, "")
			switch attrName {
			case "email", "emailaddress":
				attributes = append(attributes, saml.Attribute{
					FriendlyName: requestedAttribute.FriendlyName,
					Name:         requestedAttribute.Name,
					NameFormat:   requestedAttribute.NameFormat,
					Values: []saml.AttributeValue{{
						Type:  "xs:string",
						Value: session.UserEmail,
					}},
				})
			case "name", "fullname", "cn", "commonname":
				attributes = append(attributes, saml.Attribute{
					FriendlyName: requestedAttribute.FriendlyName,
					Name:         requestedAttribute.Name,
					NameFormat:   requestedAttribute.NameFormat,
					Values: []saml.AttributeValue{{
						Type:  "xs:string",
						Value: session.UserCommonName,
					}},
				})
			case "givenname", "firstname":
				attributes = append(attributes, saml.Attribute{
					FriendlyName: requestedAttribute.FriendlyName,
					Name:         requestedAttribute.Name,
					NameFormat:   requestedAttribute.NameFormat,
					Values: []saml.AttributeValue{{
						Type:  "xs:string",
						Value: session.UserGivenName,
					}},
				})
			case "surname", "lastname", "familyname":
				attributes = append(attributes, saml.Attribute{
					FriendlyName: requestedAttribute.FriendlyName,
					Name:         requestedAttribute.Name,
					NameFormat:   requestedAttribute.NameFormat,
					Values: []saml.AttributeValue{{
						Type:  "xs:string",
						Value: session.UserSurname,
					}},
				})
			case "uid", "user", "userid":
				attributes = append(attributes, saml.Attribute{
					FriendlyName: requestedAttribute.FriendlyName,
					Name:         requestedAttribute.Name,
					NameFormat:   requestedAttribute.NameFormat,
					Values: []saml.AttributeValue{{
						Type:  "xs:string",
						Value: session.UserName,
					}},
				})
			}
		}
	}

	if session.UserName != "" {
		attributes = append(attributes, saml.Attribute{
			FriendlyName: "uid",
			Name:         "urn:oid:0.9.2342.19200300.100.1.1",
			NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values: []saml.AttributeValue{{
				Type:  "xs:string",
				Value: session.UserName,
			}},
		})
	}

	if session.UserEmail != "" {
		attributes = append(attributes, saml.Attribute{
			FriendlyName: "eduPersonPrincipalName",
			Name:         "urn:oid:1.3.6.1.4.1.5923.1.1.1.6",
			NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values: []saml.AttributeValue{{
				Type:  "xs:string",
				Value: session.UserEmail,
			}},
		})
	}
	if session.UserSurname != "" {
		attributes = append(attributes, saml.Attribute{
			FriendlyName: "sn",
			Name:         "urn:oid:2.5.4.4",
			NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values: []saml.AttributeValue{{
				Type:  "xs:string",
				Value: session.UserSurname,
			}},
		})
	}
	if session.UserGivenName != "" {
		attributes = append(attributes, saml.Attribute{
			FriendlyName: "givenName",
			Name:         "urn:oid:2.5.4.42",
			NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values: []saml.AttributeValue{{
				Type:  "xs:string",
				Value: session.UserGivenName,
			}},
		})
	}

	if session.UserCommonName != "" {
		attributes = append(attributes, saml.Attribute{
			FriendlyName: "cn",
			Name:         "urn:oid:2.5.4.3",
			NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values: []saml.AttributeValue{{
				Type:  "xs:string",
				Value: session.UserCommonName,
			}},
		})
	}

	if session.UserScopedAffiliation != "" {
		attributes = append(attributes, saml.Attribute{
			FriendlyName: "uid",
			Name:         "urn:oid:1.3.6.1.4.1.5923.1.1.1.9",
			NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values: []saml.AttributeValue{{
				Type:  "xs:string",
				Value: session.UserScopedAffiliation,
			}},
		})
	}

	attributes = append(attributes, session.CustomAttributes...)

	if len(session.Groups) != 0 {
		groupMemberAttributeValues := []saml.AttributeValue{}
		for _, group := range session.Groups {
			groupMemberAttributeValues = append(groupMemberAttributeValues, saml.AttributeValue{
				Type:  "xs:string",
				Value: group,
			})
		}
		attributes = append(attributes, saml.Attribute{
			FriendlyName: "eduPersonAffiliation",
			Name:         "urn:oid:1.3.6.1.4.1.5923.1.1.1.1",
			NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values:       groupMemberAttributeValues,
		})
	}

	if session.SubjectID != "" {
		attributes = append(attributes, saml.Attribute{
			Name:       "urn:oasis:names:tc:SAML:attribute:subject-id",
			NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values: []saml.AttributeValue{
				{
					Type:  "xs:string",
					Value: session.SubjectID,
				},
			},
		})
	}

	// allow for some clock skew in the validity period using the
	// issuer's apparent clock.
	notBefore := req.Now.Add(-1 * MaxClockSkew)
	notOnOrAfterAfter := req.Now.Add(MaxIssueDelay)
	if notBefore.Before(req.Request.IssueInstant) {
		notBefore = req.Request.IssueInstant
		notOnOrAfterAfter = notBefore.Add(MaxIssueDelay)
	}

	nameIDFormat := "urn:oasis:names:tc:SAML:2.0:nameid-format:transient"

	if session.NameIDFormat != "" {
		nameIDFormat = session.NameIDFormat
	}

	req.Assertion = &saml.Assertion{
		ID:           fmt.Sprintf("id-%x", randomBytes(20)),
		IssueInstant: TimeNow(),
		Version:      "2.0",
		Issuer: saml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  idp.Metadata().EntityID,
		},
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Format:          nameIDFormat,
				NameQualifier:   idp.Metadata().EntityID,
				SPNameQualifier: req.ServiceProviderMetadata.EntityID,
				Value:           session.NameID,
			},
			SubjectConfirmations: []saml.SubjectConfirmation{
				{
					Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
					SubjectConfirmationData: &saml.SubjectConfirmationData{
						Address:      req.HTTPRequest.RemoteAddr,
						InResponseTo: req.Request.ID,
						NotOnOrAfter: req.Now.Add(MaxIssueDelay),
						Recipient:    req.ACSEndpoint.Location,
					},
				},
			},
		},
		Conditions: &saml.Conditions{
			NotBefore:    notBefore,
			NotOnOrAfter: notOnOrAfterAfter,
			AudienceRestrictions: []saml.AudienceRestriction{
				{
					Audience: saml.Audience{Value: req.ServiceProviderMetadata.EntityID},
				},
			},
		},
		AuthnStatements: []saml.AuthnStatement{
			{
				AuthnInstant: session.Created,
				SessionIndex: session.Index,
				SubjectLocality: &saml.SubjectLocality{
					Address: req.HTTPRequest.RemoteAddr,
				},
				AuthnContext: saml.AuthnContext{
					AuthnContextClassRef: &saml.AuthnContextClassRef{
						Value: "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
					},
				},
			},
		},
		AttributeStatements: []saml.AttributeStatement{
			{
				Attributes: attributes,
			},
		},
	}

	return nil
}

// The Canonicalizer prefix list MUST be empty. Various implementations
// (maybe ours?) do not appear to support non-empty prefix lists in XML C14N.
const canonicalizerPrefixList = ""

// MakeAssertionEl sets `AssertionEl` to a signed, possibly encrypted, version of `Assertion`.
func (h *handler) MakeAssertionEl(req *IdpAuthnRequest) error {
	idp, _, err := h.getIDP(req.HTTPRequest)
	if err != nil {
		return ucerr.Wrap(err)
	}

	keyPair := tls.Certificate{
		Certificate: [][]byte{idp.Certificate.Raw},
		PrivateKey:  idp.Key,
		Leaf:        idp.Certificate,
	}
	for _, cert := range idp.Intermediates {
		keyPair.Certificate = append(keyPair.Certificate, cert.Raw)
	}
	keyStore := dsig.TLSCertKeyStore(keyPair)

	signatureMethod := idp.SignatureMethod
	if signatureMethod == "" {
		signatureMethod = dsig.RSASHA1SignatureMethod
	}

	signingContext := dsig.NewDefaultSigningContext(keyStore)
	signingContext.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList(canonicalizerPrefixList)
	if err := signingContext.SetSignatureMethod(signatureMethod); err != nil {
		return ucerr.Wrap(err)
	}

	assertionEl := req.Assertion.Element()

	signedAssertionEl, err := signingContext.SignEnveloped(assertionEl)
	if err != nil {
		return ucerr.Wrap(err)
	}

	sigEl := signedAssertionEl.Child[len(signedAssertionEl.Child)-1]
	req.Assertion.Signature = sigEl.(*etree.Element)
	signedAssertionEl = req.Assertion.Element()

	certBuf, err := req.getSPEncryptionCert()
	if errors.Is(err, os.ErrNotExist) {
		req.AssertionEl = signedAssertionEl
		return nil
	} else if err != nil {
		return ucerr.Wrap(err)
	}

	var signedAssertionBuf []byte
	{
		doc := etree.NewDocument()
		doc.SetRoot(signedAssertionEl)
		signedAssertionBuf, err = doc.WriteToBytes()
		if err != nil {
			return ucerr.Wrap(err)
		}
	}

	encryptor := xmlenc.OAEP()
	encryptor.BlockCipher = xmlenc.AES128CBC
	encryptor.DigestMethod = &xmlenc.SHA1
	encryptedDataEl, err := encryptor.Encrypt(certBuf, signedAssertionBuf, nil)
	if err != nil {
		return ucerr.Wrap(err)
	}
	encryptedDataEl.CreateAttr("Type", "http://www.w3.org/2001/04/xmlenc#Element")

	encryptedAssertionEl := etree.NewElement("saml:EncryptedAssertion")
	encryptedAssertionEl.AddChild(encryptedDataEl)
	req.AssertionEl = encryptedAssertionEl

	return nil
}

// IdpAuthnRequestForm contans HTML form information to be submitted to the
// SAML HTTP POST binding ACS.
type IdpAuthnRequestForm struct {
	URL          string
	SAMLResponse string
	RelayState   string
}

// PostBinding creates the HTTP POST form information for this
// `IdpAuthnRequest`. If `Response` is not already set, it calls MakeResponse
// to produce it.
func (h *handler) PostBinding(req *IdpAuthnRequest) (IdpAuthnRequestForm, error) {
	var form IdpAuthnRequestForm

	if req.ResponseEl == nil {
		if err := h.MakeResponse(req); err != nil {
			return form, ucerr.Wrap(err)
		}
	}

	doc := etree.NewDocument()
	doc.SetRoot(req.ResponseEl)
	responseBuf, err := doc.WriteToBytes()
	if err != nil {
		return form, ucerr.Wrap(err)
	}

	if req.ACSEndpoint.Binding != samlconfig.HTTPPostBinding {
		return form, ucerr.Errorf("%s: unsupported binding %s",
			req.ServiceProviderMetadata.EntityID,
			req.ACSEndpoint.Binding)
	}

	form.URL = req.ACSEndpoint.Location
	form.SAMLResponse = base64.StdEncoding.EncodeToString(responseBuf)
	form.RelayState = req.RelayState

	return form, nil
}

// WriteResponse writes the `Response` to the http.ResponseWriter. If
// `Response` is not already set, it calls MakeResponse to produce it.
func (h *handler) WriteResponse(w http.ResponseWriter, req *IdpAuthnRequest) error {
	form, err := h.PostBinding(req)
	if err != nil {
		return ucerr.Wrap(err)
	}

	tmpl := template.Must(template.New("saml-post-form").Parse(`<html>` +
		`<form method="post" action="{{.URL}}" id="SAMLResponseForm">` +
		`<input type="hidden" name="SAMLResponse" value="{{.SAMLResponse}}" />` +
		`<input type="hidden" name="RelayState" value="{{.RelayState}}" />` +
		`<input id="SAMLSubmitButton" type="submit" value="Continue" />` +
		`</form>` +
		`<script>document.getElementById('SAMLSubmitButton').style.visibility='hidden';</script>` +
		`<script>document.getElementById('SAMLResponseForm').submit();</script>` +
		`</html>`))

	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, form); err != nil {
		return ucerr.Wrap(err)
	}
	if _, err := io.Copy(w, buf); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// getSPEncryptionCert returns the certificate which we can use to encrypt things
// to the SP in PEM format, or nil if no such certificate is found.
func (req *IdpAuthnRequest) getSPEncryptionCert() (*x509.Certificate, error) {
	certStr := ""
	for _, keyDescriptor := range req.SPSSODescriptor.KeyDescriptors {
		if keyDescriptor.Use == "encryption" {
			certStr = keyDescriptor.KeyInfo.X509Data.X509Certificates[0].Data
			break
		}
	}

	// If there are no certs explicitly labeled for encryption, return the first
	// non-empty cert we find.
	if certStr == "" {
		for _, keyDescriptor := range req.SPSSODescriptor.KeyDescriptors {
			if keyDescriptor.Use == "" && len(keyDescriptor.KeyInfo.X509Data.X509Certificates) != 0 && keyDescriptor.KeyInfo.X509Data.X509Certificates[0].Data != "" {
				certStr = keyDescriptor.KeyInfo.X509Data.X509Certificates[0].Data
				break
			}
		}
	}

	if certStr == "" {
		return nil, ucerr.Wrap(os.ErrNotExist)
	}

	// cleanup whitespace and re-encode a PEM
	certStr = regexp.MustCompile(`\s+`).ReplaceAllString(certStr, "")
	certBytes, err := base64.StdEncoding.DecodeString(certStr)
	if err != nil {
		return nil, ucerr.Errorf("cannot decode certificate base64: %v", err)
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, ucerr.Errorf("cannot parse certificate: %v", err)
	}
	return cert, nil
}

// unmarshalEtreeHack parses `el` and sets values in the structure `v`.
//
// This is a hack -- it first serializes the element, then uses xml.Unmarshal.
func unmarshalEtreeHack(el *etree.Element, v any) error {
	doc := etree.NewDocument()
	doc.SetRoot(el)
	buf, err := doc.WriteToBytes()
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(xml.Unmarshal(buf, v))
}

// MakeResponse creates and assigns a new SAML response in ResponseEl. `Assertion` must
// be non-nil. If MakeAssertionEl() has not been called, this function calls it for
// you.
func (h *handler) MakeResponse(req *IdpAuthnRequest) error {
	idp, _, err := h.getIDP(req.HTTPRequest)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if req.AssertionEl == nil {
		if err := h.MakeAssertionEl(req); err != nil {
			return ucerr.Wrap(err)
		}
	}

	response := &saml.Response{
		Destination:  req.ACSEndpoint.Location,
		ID:           fmt.Sprintf("id-%x", randomBytes(20)),
		InResponseTo: req.Request.ID,
		IssueInstant: req.Now,
		Version:      "2.0",
		Issuer: &saml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  idp.MetadataURL.String(),
		},
		Status: saml.Status{
			StatusCode: saml.StatusCode{
				Value: saml.StatusSuccess,
			},
		},
	}

	responseEl := response.Element()
	responseEl.AddChild(req.AssertionEl) // AssertionEl either an EncryptedAssertion or Assertion element

	// Sign the response element (we've already signed the Assertion element)
	{
		keyPair := tls.Certificate{
			Certificate: [][]byte{idp.Certificate.Raw},
			PrivateKey:  idp.Key,
			Leaf:        idp.Certificate,
		}
		for _, cert := range idp.Intermediates {
			keyPair.Certificate = append(keyPair.Certificate, cert.Raw)
		}
		keyStore := dsig.TLSCertKeyStore(keyPair)

		signatureMethod := idp.SignatureMethod
		if signatureMethod == "" {
			signatureMethod = dsig.RSASHA1SignatureMethod
		}

		signingContext := dsig.NewDefaultSigningContext(keyStore)
		signingContext.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList(canonicalizerPrefixList)
		if err := signingContext.SetSignatureMethod(signatureMethod); err != nil {
			return ucerr.Wrap(err)
		}

		signedResponseEl, err := signingContext.SignEnveloped(responseEl)
		if err != nil {
			return ucerr.Wrap(err)
		}

		sigEl := signedResponseEl.ChildElements()[len(signedResponseEl.ChildElements())-1]
		response.Signature = sigEl
		responseEl = response.Element()
		responseEl.AddChild(req.AssertionEl)
	}

	req.ResponseEl = responseEl
	return nil
}

// GetServiceProvider returns the Service Provider metadata for the
// service provider ID, which is typically the service provider's
// metadata URL. If an appropriate service provider cannot be found then
// the returned error must be os.ErrNotExist.
func (idp *IdentityProvider) GetServiceProvider(serviceProviderID string) (*samlconfig.EntityDescriptor, error) {
	rv, ok := idp.ServiceProviders[serviceProviderID]
	if !ok {
		return nil, ucerr.Wrap(os.ErrNotExist)
	}
	return rv, nil
}
