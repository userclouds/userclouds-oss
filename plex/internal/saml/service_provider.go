package saml

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/logger"
	"github.com/crewjam/saml/xmlenc"
	"github.com/go-http-utils/headers"
	xrv "github.com/mattermost/xml-roundtrip-validator"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/russellhaering/goxmldsig/etreeutils"

	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantplex/samlconfig"
)

// Name ID formats
const (
	UnspecifiedNameIDFormat  samlconfig.NameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"
	TransientNameIDFormat    samlconfig.NameIDFormat = "urn:oasis:names:tc:SAML:2.0:nameid-format:transient"
	EmailAddressNameIDFormat samlconfig.NameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	PersistentNameIDFormat   samlconfig.NameIDFormat = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
)

// SignatureVerifier verifies a signature
//
// Can be implemented in order to override ServiceProvider's default
// way of verifying signatures.
type SignatureVerifier interface {
	VerifySignature(validationContext *dsig.ValidationContext, el *etree.Element) error
}

// ServiceProvider implements SAML Service provider.
//
// In SAML, service providers delegate responsibility for identifying
// clients to an identity provider. If you are writing an application
// that uses passwords (or whatever) stored somewhere else, then you
// are service provider.
//
// See the example directory for an example of a web application using
// the service provider interface.
type ServiceProvider struct {
	// Entity ID is optional - if not specified then MetadataURL will be used
	EntityID string

	// Key is the RSA private key we use to sign requests.
	Key *rsa.PrivateKey

	// Certificate is the RSA public part of Key.
	Certificate   *x509.Certificate
	Intermediates []*x509.Certificate

	// HTTPClient to use during SAML artifact resolution
	HTTPClient *http.Client

	// MetadataURL is the full URL to the metadata endpoint on this host,
	// i.e. https://example.com/saml/metadata
	MetadataURL url.URL

	// AcsURL is the full URL to the SAML Assertion Customer Service endpoint
	// on this host, i.e. https://example.com/saml/acs
	AcsURL url.URL

	// SloURL is the full URL to the SAML Single Logout endpoint on this host.
	// i.e. https://example.com/saml/slo
	SloURL url.URL

	// IDPMetadata is the metadata from the identity provider.
	IDPMetadata *samlconfig.EntityDescriptor

	// AuthnNameIDFormat is the format used in the NameIDPolicy for
	// authentication requests
	AuthnNameIDFormat samlconfig.NameIDFormat

	// MetadataValidDuration is a duration used to calculate validUntil
	// attribute in the metadata endpoint
	MetadataValidDuration time.Duration

	// ForceAuthn allows you to force re-authentication of users even if the user
	// has a SSO session at the IdP.
	ForceAuthn *bool

	// RequestedAuthnContext allow you to specify the requested authentication
	// context in authentication requests
	RequestedAuthnContext *saml.RequestedAuthnContext

	// AllowIdpInitiated
	AllowIDPInitiated bool

	// DefaultRedirectURI where untracked requests (as of IDPInitiated) are redirected to
	DefaultRedirectURI string

	// SignatureVerifier, if non-nil, allows you to implement an alternative way
	// to verify signatures.
	SignatureVerifier SignatureVerifier

	// SignatureMethod, if non-empty, authentication requests will be signed
	SignatureMethod string

	// LogoutBindings specify the bindings available for SLO endpoint. If empty,
	// HTTP-POST binding is used.
	LogoutBindings []string
}

// MaxIssueDelay is the longest allowed time between when a SAML assertion is
// issued by the IDP and the time it is received by ParseResponse. This is used
// to prevent old responses from being replayed (while allowing for some clock
// drift between the SP and IDP).
var MaxIssueDelay = time.Second * 90

// MaxClockSkew allows for leeway for clock skew between the IDP and SP when
// validating assertions. It defaults to 180 seconds (matches shibboleth).
var MaxClockSkew = time.Second * 180

// DefaultValidDuration is how long we assert that the SP metadata is valid.
const DefaultValidDuration = time.Hour * 24 * 2

// DefaultCacheDuration is how long we ask the IDP to cache the SP metadata.
const DefaultCacheDuration = time.Hour * 24 * 1

// Metadata returns the service provider metadata
func (sp *ServiceProvider) Metadata() *samlconfig.EntityDescriptor {
	validDuration := DefaultValidDuration
	if sp.MetadataValidDuration > 0 {
		validDuration = sp.MetadataValidDuration
	}

	authnRequestsSigned := len(sp.SignatureMethod) > 0
	wantAssertionsSigned := true
	validUntil := TimeNow().Add(validDuration)

	var keyDescriptors []samlconfig.KeyDescriptor
	if sp.Certificate != nil {
		certBytes := sp.Certificate.Raw
		for _, intermediate := range sp.Intermediates {
			certBytes = append(certBytes, intermediate.Raw...)
		}
		keyDescriptors = []samlconfig.KeyDescriptor{
			{
				Use: "encryption",
				KeyInfo: samlconfig.KeyInfo{
					X509Data: samlconfig.X509Data{
						X509Certificates: []samlconfig.X509Certificate{
							{Data: base64.StdEncoding.EncodeToString(certBytes)},
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
		}
		if len(sp.SignatureMethod) > 0 {
			keyDescriptors = append(keyDescriptors, samlconfig.KeyDescriptor{
				Use: "signing",
				KeyInfo: samlconfig.KeyInfo{
					X509Data: samlconfig.X509Data{
						X509Certificates: []samlconfig.X509Certificate{
							{Data: base64.StdEncoding.EncodeToString(certBytes)},
						},
					},
				},
			})
		}
	}

	sloEndpoints := make([]samlconfig.Endpoint, len(sp.LogoutBindings))
	for i, binding := range sp.LogoutBindings {
		sloEndpoints[i] = samlconfig.Endpoint{
			Binding:          binding,
			Location:         sp.SloURL.String(),
			ResponseLocation: sp.SloURL.String(),
		}
	}

	return &samlconfig.EntityDescriptor{
		EntityID:   firstSet(sp.EntityID, sp.MetadataURL.String()),
		ValidUntil: validUntil,

		SPSSODescriptors: []samlconfig.SPSSODescriptor{
			{
				SSODescriptor: samlconfig.SSODescriptor{
					RoleDescriptor: samlconfig.RoleDescriptor{
						ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
						KeyDescriptors:             keyDescriptors,
						ValidUntil:                 &validUntil,
					},
					SingleLogoutServices: sloEndpoints,
					NameIDFormats:        []samlconfig.NameIDFormat{sp.AuthnNameIDFormat},
				},
				AuthnRequestsSigned:  &authnRequestsSigned,
				WantAssertionsSigned: &wantAssertionsSigned,

				AssertionConsumerServices: []samlconfig.IndexedEndpoint{
					{
						Binding:  samlconfig.HTTPPostBinding,
						Location: sp.AcsURL.String(),
						Index:    1,
					},
					{
						Binding:  samlconfig.HTTPArtifactBinding,
						Location: sp.AcsURL.String(),
						Index:    2,
					},
				},
			},
		},
	}
}

// MakeRedirectAuthenticationRequest creates a SAML authentication request using
// the HTTP-Redirect binding. It returns a URL that we will redirect the user to
// in order to start the auth process.
func (sp *ServiceProvider) MakeRedirectAuthenticationRequest(relayState string) (*url.URL, error) {
	req, err := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(samlconfig.HTTPRedirectBinding), samlconfig.HTTPRedirectBinding, samlconfig.HTTPPostBinding)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return sp.MakeRedirect(relayState, req)
}

// MakeRedirect returns a URL suitable for using the redirect binding with the request
func (sp *ServiceProvider) MakeRedirect(relayState string, req *saml.AuthnRequest) (*url.URL, error) {
	w := &bytes.Buffer{}
	w1 := base64.NewEncoder(base64.StdEncoding, w)
	w2, err := flate.NewWriter(w1, 9)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	doc := etree.NewDocument()
	doc.SetRoot(req.Element())
	if _, err := doc.WriteTo(w2); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if err := w2.Close(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	if err := w1.Close(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	rv, err := url.Parse(req.Destination)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// We can't depend on Query().set() as order matters for signing
	query := rv.RawQuery
	if len(query) > 0 {
		query += "&SAMLRequest=" + url.QueryEscape(w.String())
	} else {
		query += "SAMLRequest=" + url.QueryEscape(w.String())
	}

	if relayState != "" {
		query += "&RelayState=" + relayState
	}
	if len(sp.SignatureMethod) > 0 {
		query += "&SigAlg=" + url.QueryEscape(sp.SignatureMethod)
		signingContext, err := GetSigningContext(sp)

		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		sig, err := signingContext.SignString(query)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		query += "&Signature=" + url.QueryEscape(base64.StdEncoding.EncodeToString(sig))
	}

	rv.RawQuery = query

	return rv, nil
}

// GetSSOBindingLocation returns URL for the IDP's Single Sign On Service binding
// of the specified type (HTTPRedirectBinding or HTTPPostBinding)
func (sp *ServiceProvider) GetSSOBindingLocation(binding string) string {
	for _, idpSSODescriptor := range sp.IDPMetadata.IDPSSODescriptors {
		for _, singleSignOnService := range idpSSODescriptor.SingleSignOnServices {
			if singleSignOnService.Binding == binding {
				return singleSignOnService.Location
			}
		}
	}
	return ""
}

// GetArtifactBindingLocation returns URL for the IDP's Artifact binding of the
// specified type
func (sp *ServiceProvider) GetArtifactBindingLocation(binding string) string {
	for _, idpSSODescriptor := range sp.IDPMetadata.IDPSSODescriptors {
		for _, artifactResolutionService := range idpSSODescriptor.ArtifactResolutionServices {
			if artifactResolutionService.Binding == binding {
				return artifactResolutionService.Location
			}
		}
	}
	return ""
}

// GetSLOBindingLocation returns URL for the IDP's Single Log Out Service binding
// of the specified type (HTTPRedirectBinding or HTTPPostBinding)
func (sp *ServiceProvider) GetSLOBindingLocation(binding string) string {
	for _, idpSSODescriptor := range sp.IDPMetadata.IDPSSODescriptors {
		for _, singleLogoutService := range idpSSODescriptor.SingleLogoutServices {
			if singleLogoutService.Binding == binding {
				return singleLogoutService.Location
			}
		}
	}
	return ""
}

// getIDPSigningCerts returns the certificates which we can use to verify things
// signed by the IDP in PEM format, or nil if no such certificate is found.
func (sp *ServiceProvider) getIDPSigningCerts() ([]*x509.Certificate, error) {
	var certStrs []string

	// We need to include non-empty certs where the "use" attribute is
	// either set to "signing" or is missing
	for _, idpSSODescriptor := range sp.IDPMetadata.IDPSSODescriptors {
		for _, keyDescriptor := range idpSSODescriptor.KeyDescriptors {
			if len(keyDescriptor.KeyInfo.X509Data.X509Certificates) != 0 {
				switch keyDescriptor.Use {
				case "", "signing":
					for _, certificate := range keyDescriptor.KeyInfo.X509Data.X509Certificates {
						certStrs = append(certStrs, certificate.Data)
					}
				}
			}
		}
	}

	if len(certStrs) == 0 {
		return nil, ucerr.New("cannot find any signing certificate in the IDP SSO descriptor")
	}

	certs := make([]*x509.Certificate, len(certStrs))

	// cleanup whitespace
	regex := regexp.MustCompile(`\s+`)
	for i, certStr := range certStrs {
		certStr = regex.ReplaceAllString(certStr, "")
		certBytes, err := base64.StdEncoding.DecodeString(certStr)
		if err != nil {
			return nil, ucerr.Errorf("cannot parse certificate: %s", err)
		}

		parsedCert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		certs[i] = parsedCert
	}

	return certs, nil
}

// MakeArtifactResolveRequest produces a new ArtifactResolve object to send to the idp's Artifact resolver
func (sp *ServiceProvider) MakeArtifactResolveRequest(artifactID string) (*saml.ArtifactResolve, error) {
	req := saml.ArtifactResolve{
		ID:           fmt.Sprintf("id-%x", randomBytes(20)),
		IssueInstant: TimeNow(),
		Version:      "2.0",
		Issuer: &saml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  firstSet(sp.EntityID, sp.MetadataURL.String()),
		},
		Artifact: artifactID,
	}

	if len(sp.SignatureMethod) > 0 {
		if err := sp.SignArtifactResolve(&req); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &req, nil
}

// MakeAuthenticationRequest produces a new AuthnRequest object to send to the idpURL
// that uses the specified binding (HTTPRedirectBinding or HTTPPostBinding)
func (sp *ServiceProvider) MakeAuthenticationRequest(idpURL string, binding string, resultBinding string) (*saml.AuthnRequest, error) {

	allowCreate := true
	nameIDFormat := sp.nameIDFormat()
	req := saml.AuthnRequest{
		AssertionConsumerServiceURL: sp.AcsURL.String(),
		Destination:                 idpURL,
		ProtocolBinding:             resultBinding, // default binding for the response
		ID:                          fmt.Sprintf("id-%x", randomBytes(20)),
		IssueInstant:                TimeNow(),
		Version:                     "2.0",
		Issuer: &saml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  firstSet(sp.EntityID, sp.MetadataURL.String()),
		},
		NameIDPolicy: &saml.NameIDPolicy{
			AllowCreate: &allowCreate,
			// TODO(ross): figure out exactly policy we need
			// urn:mace:shibboleth:1.0:nameIdentifier
			// urn:oasis:names:tc:SAML:2.0:nameid-format:transient
			Format: &nameIDFormat,
		},
		ForceAuthn:            sp.ForceAuthn,
		RequestedAuthnContext: sp.RequestedAuthnContext,
	}
	// We don't need to sign the XML document if the IDP uses HTTP-Redirect binding
	if len(sp.SignatureMethod) > 0 && binding == samlconfig.HTTPPostBinding {
		if err := sp.SignAuthnRequest(&req); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}
	return &req, nil
}

// GetSigningContext returns a dsig.SigningContext initialized based on the Service Provider's configuration
func GetSigningContext(sp *ServiceProvider) (*dsig.SigningContext, error) {
	keyPair := tls.Certificate{
		Certificate: [][]byte{sp.Certificate.Raw},
		PrivateKey:  sp.Key,
		Leaf:        sp.Certificate,
	}
	// TODO: add intermediates for SP
	// for _, cert := range sp.Intermediates {
	// 	keyPair.Certificate = append(keyPair.Certificate, cert.Raw)
	// }
	keyStore := dsig.TLSCertKeyStore(keyPair)

	if sp.SignatureMethod != dsig.RSASHA1SignatureMethod &&
		sp.SignatureMethod != dsig.RSASHA256SignatureMethod &&
		sp.SignatureMethod != dsig.RSASHA512SignatureMethod {
		return nil, ucerr.Errorf("invalid signing method %s", sp.SignatureMethod)
	}
	signatureMethod := sp.SignatureMethod
	signingContext := dsig.NewDefaultSigningContext(keyStore)
	signingContext.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList(canonicalizerPrefixList)
	if err := signingContext.SetSignatureMethod(signatureMethod); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return signingContext, nil
}

// SignArtifactResolve adds the `Signature` element to the `ArtifactResolve`.
func (sp *ServiceProvider) SignArtifactResolve(req *saml.ArtifactResolve) error {
	signingContext, err := GetSigningContext(sp)
	if err != nil {
		return ucerr.Wrap(err)
	}
	assertionEl := req.Element()

	signedRequestEl, err := signingContext.SignEnveloped(assertionEl)
	if err != nil {
		return ucerr.Wrap(err)
	}

	sigEl := signedRequestEl.Child[len(signedRequestEl.Child)-1]
	req.Signature = sigEl.(*etree.Element)
	return nil
}

// SignAuthnRequest adds the `Signature` element to the `AuthnRequest`.
func (sp *ServiceProvider) SignAuthnRequest(req *saml.AuthnRequest) error {

	signingContext, err := GetSigningContext(sp)
	if err != nil {
		return ucerr.Wrap(err)
	}
	assertionEl := req.Element()

	signedRequestEl, err := signingContext.SignEnveloped(assertionEl)
	if err != nil {
		return ucerr.Wrap(err)
	}

	sigEl := signedRequestEl.Child[len(signedRequestEl.Child)-1]
	req.Signature = sigEl.(*etree.Element)
	return nil
}

// MakePostAuthenticationRequest creates a SAML authentication request using
// the HTTP-POST binding. It returns HTML text representing an HTML form that
// can be sent presented to a browser to initiate the login process.
func (sp *ServiceProvider) MakePostAuthenticationRequest(relayState string) ([]byte, error) {
	req, err := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(samlconfig.HTTPPostBinding), samlconfig.HTTPPostBinding, samlconfig.HTTPPostBinding)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return req.Post(relayState), nil
}

// AssertionAttributes is a list of AssertionAttribute
type AssertionAttributes []AssertionAttribute

// Get returns the assertion attribute whose Name or FriendlyName
// matches name, or nil if no matching attribute is found.
func (aa AssertionAttributes) Get(name string) *AssertionAttribute {
	for _, attr := range aa {
		if attr.Name == name {
			return &attr
		}
		if attr.FriendlyName == name {
			return &attr
		}
	}
	return nil
}

// AssertionAttribute represents an attribute of the user extracted from
// a SAML Assertion.
type AssertionAttribute struct {
	FriendlyName string
	Name         string
	Value        string
}

// InvalidResponseError is the error produced by ParseResponse when it fails.
// The underlying error is in PrivateErr. Response is the response as it was
// known at the time validation failed. Now is the time that was used to validate
// time-dependent parts of the assertion.
type InvalidResponseError struct {
	PrivateErr error
	Response   string
	Now        time.Time
}

func (ivr *InvalidResponseError) Error() string {
	return "Authentication failed"
}

// ErrBadStatus is returned when the assertion provided is valid but the
// status code is not "urn:oasis:names:tc:SAML:2.0:status:Success".
type ErrBadStatus struct {
	Status string
}

func (e ErrBadStatus) Error() string {
	return e.Status
}

// ParseResponse extracts the SAML IDP response received in req, resolves
// artifacts when necessary, validates it, and returns the verified assertion.
func (sp *ServiceProvider) ParseResponse(req *http.Request, possibleRequestIDs []string) (*saml.Assertion, error) {
	if artifactID := req.Form.Get("SAMLart"); artifactID != "" {
		return sp.handleArtifactRequest(req.Context(), artifactID, possibleRequestIDs)
	}
	return sp.parseResponseHTTP(req, possibleRequestIDs)
}

func (sp *ServiceProvider) handleArtifactRequest(ctx context.Context, artifactID string, possibleRequestIDs []string) (*saml.Assertion, error) {
	retErr := &InvalidResponseError{Now: TimeNow()}

	artifactResolveRequest, err := sp.MakeArtifactResolveRequest(artifactID)
	if err != nil {
		retErr.PrivateErr = ucerr.Errorf("cannot generate artifact resolution request: %s", err)
		return nil, ucerr.Wrap(retErr)
	}

	requestBody, err := elementToBytes(artifactResolveRequest.SoapRequest())
	if err != nil {
		retErr.PrivateErr = err
		return nil, ucerr.Wrap(retErr)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sp.GetArtifactBindingLocation(samlconfig.SOAPBinding),
		bytes.NewReader(requestBody))
	if err != nil {
		retErr.PrivateErr = err
		return nil, ucerr.Wrap(retErr)
	}
	req.Header.Set(headers.ContentType, "text/xml")

	httpClient := sp.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	response, err := httpClient.Do(req)
	if err != nil {
		retErr.PrivateErr = ucerr.Errorf("cannot resolve artifact: %s", err)
		return nil, ucerr.Wrap(retErr)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			logger.DefaultLogger.Printf("Error while closing response body during artifact resolution: %v", err)
		}
	}()
	if response.StatusCode != 200 {
		retErr.PrivateErr = ucerr.Errorf("Error during artifact resolution: HTTP status %d (%s)", response.StatusCode, response.Status)
		return nil, ucerr.Wrap(retErr)
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		retErr.PrivateErr = ucerr.Errorf("Error during artifact resolution: %s", err)
		return nil, ucerr.Wrap(retErr)
	}
	assertion, err := sp.ParseXMLArtifactResponse(responseBody, possibleRequestIDs, artifactResolveRequest.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return assertion, nil
}

func (sp *ServiceProvider) parseResponseHTTP(req *http.Request, possibleRequestIDs []string) (*saml.Assertion, error) {
	retErr := &InvalidResponseError{
		Now: TimeNow(),
	}

	rawResponseBuf, err := base64.StdEncoding.DecodeString(req.PostForm.Get("SAMLResponse"))
	if err != nil {
		retErr.PrivateErr = ucerr.Errorf("cannot parse base64: %s", err)
		return nil, ucerr.Wrap(retErr)
	}

	assertion, err := sp.ParseXMLResponse(rawResponseBuf, possibleRequestIDs)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return assertion, nil
}

// ParseXMLArtifactResponse validates the SAML Artifact resolver response
// and returns the verified assertion.
//
// This function handles verifying the digital signature, and verifying
// that the specified conditions and properties are met.
//
// If the function fails it will return an InvalidResponseError whose
// properties are useful in describing which part of the parsing process
// failed. However, to discourage inadvertent disclosure the diagnostic
// information, the Error() method returns a static string.
func (sp *ServiceProvider) ParseXMLArtifactResponse(soapResponseXML []byte, possibleRequestIDs []string, artifactRequestID string) (*saml.Assertion, error) {
	now := TimeNow()
	retErr := &InvalidResponseError{
		Response: string(soapResponseXML),
		Now:      now,
	}

	// ensure that the response XML is well-formed before we parse it
	if err := xrv.Validate(bytes.NewReader(soapResponseXML)); err != nil {
		retErr.PrivateErr = ucerr.Errorf("invalid xml: %s", err)
		return nil, ucerr.Wrap(retErr)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(soapResponseXML); err != nil {
		retErr.PrivateErr = ucerr.Errorf("cannot unmarshal response: %s", err)
		return nil, ucerr.Wrap(retErr)
	}
	if doc.Root() == nil {
		retErr.PrivateErr = ucerr.New("invalid xml: no root")
		return nil, ucerr.Wrap(retErr)
	}
	if doc.Root().NamespaceURI() != "http://schemas.xmlsoap.org/soap/envelope/" ||
		doc.Root().Tag != "Envelope" {
		retErr.PrivateErr = ucerr.Errorf("expected a SOAP Envelope")
		return nil, ucerr.Wrap(retErr)
	}

	soapBodyEl, err := findOneChild(doc.Root(), "http://schemas.xmlsoap.org/soap/envelope/", "Body")
	if err != nil {
		retErr.PrivateErr = err
		return nil, ucerr.Wrap(retErr)
	}

	artifactResponseEl, err := findOneChild(soapBodyEl, "urn:oasis:names:tc:SAML:2.0:protocol", "ArtifactResponse")
	if err != nil {
		retErr.PrivateErr = err
		return nil, ucerr.Wrap(retErr)
	}

	return sp.parseArtifactResponse(artifactResponseEl, possibleRequestIDs, artifactRequestID, now)
}

func (sp *ServiceProvider) parseArtifactResponse(artifactResponseEl *etree.Element, possibleRequestIDs []string, artifactRequestID string, now time.Time) (*saml.Assertion, error) {
	retErr := &InvalidResponseError{
		Now:      now,
		Response: elementToString(artifactResponseEl),
	}

	{
		var artifactResponse saml.ArtifactResponse
		if err := unmarshalElement(artifactResponseEl, &artifactResponse); err != nil {
			retErr.PrivateErr = err
			return nil, ucerr.Wrap(retErr)
		}
		if artifactResponse.InResponseTo != artifactRequestID {
			retErr.PrivateErr = ucerr.Errorf("`InResponseTo` does not match the artifact request ID (expected %s)", artifactRequestID)
			return nil, ucerr.Wrap(retErr)
		}
		if artifactResponse.IssueInstant.Add(MaxIssueDelay).Before(now) {
			retErr.PrivateErr = ucerr.Errorf("response IssueInstant expired at %s", artifactResponse.IssueInstant.Add(MaxIssueDelay))
			return nil, ucerr.Wrap(retErr)
		}
		if artifactResponse.Issuer != nil && artifactResponse.Issuer.Value != sp.IDPMetadata.EntityID {
			retErr.PrivateErr = ucerr.Errorf("response Issuer does not match the IDP metadata (expected %q)", sp.IDPMetadata.EntityID)
			return nil, ucerr.Wrap(retErr)
		}
		if artifactResponse.Status.StatusCode.Value != saml.StatusSuccess {
			retErr.PrivateErr = ErrBadStatus{Status: artifactResponse.Status.StatusCode.Value}
			return nil, ucerr.Wrap(retErr)
		}
	}

	var signatureRequirement signatureRequirement
	sigErr := sp.validateSignature(artifactResponseEl)
	switch sigErr {
	case nil:
		signatureRequirement = signatureNotRequired
	case errSignatureElementNotPresent:
		signatureRequirement = signatureRequired
	default:
		retErr.PrivateErr = sigErr
		return nil, ucerr.Wrap(retErr)
	}

	responseEl, err := findOneChild(artifactResponseEl, "urn:oasis:names:tc:SAML:2.0:protocol", "Response")
	if err != nil {
		retErr.PrivateErr = err
		return nil, ucerr.Wrap(retErr)
	}

	assertion, err := sp.parseResponse(responseEl, possibleRequestIDs, now, signatureRequirement)
	if err != nil {
		retErr.PrivateErr = err
		return nil, ucerr.Wrap(retErr)
	}

	return assertion, nil
}

// ParseXMLResponse parses and validates the SAML IDP response and
// returns the verified assertion.
//
// This function handles decrypting the message, verifying the digital
// signature on the assertion, and verifying that the specified conditions
// and properties are met.
//
// If the function fails it will return an InvalidResponseError whose
// properties are useful in describing which part of the parsing process
// failed. However, to discourage inadvertent disclosure the diagnostic
// information, the Error() method returns a static string.
func (sp *ServiceProvider) ParseXMLResponse(decodedResponseXML []byte, possibleRequestIDs []string) (*saml.Assertion, error) {
	now := TimeNow()
	var err error
	retErr := &InvalidResponseError{
		Now:      now,
		Response: string(decodedResponseXML),
	}

	// ensure that the response XML is well-formed before we parse it
	if err := xrv.Validate(bytes.NewReader(decodedResponseXML)); err != nil {
		retErr.PrivateErr = ucerr.Errorf("invalid xml: %s", err)
		return nil, ucerr.Wrap(retErr)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(decodedResponseXML); err != nil {
		retErr.PrivateErr = err
		return nil, ucerr.Wrap(retErr)
	}
	if doc.Root() == nil {
		retErr.PrivateErr = ucerr.New("invalid xml: no root")
		return nil, ucerr.Wrap(retErr)
	}

	assertion, err := sp.parseResponse(doc.Root(), possibleRequestIDs, now, signatureRequired)
	if err != nil {
		retErr.PrivateErr = err
		return nil, ucerr.Wrap(retErr)
	}

	return assertion, nil
}

type signatureRequirement int

const (
	signatureRequired signatureRequirement = iota
	signatureNotRequired
)

// validateXMLResponse validates the SAML IDP response and returns
// the verified assertion.
//
// This function handles decrypting the message, verifying the digital
// signature on the assertion, and verifying that the specified conditions
// and properties are met.
func (sp *ServiceProvider) parseResponse(responseEl *etree.Element, possibleRequestIDs []string, now time.Time, signatureRequirement signatureRequirement) (*saml.Assertion, error) {
	var responseSignatureErr error
	var responseHasSignature bool
	if signatureRequirement == signatureRequired {
		responseSignatureErr = sp.validateSignature(responseEl)
		if !errors.Is(responseSignatureErr, errSignatureElementNotPresent) {
			responseHasSignature = true
		}

		// Note: we're deferring taking action on the signature validation until after we've
		// processed the request attributes, because certain test cases seem to require this mis-feature.
		// TODO(ross): adjust the test cases so that we can abort here if the Response signature is invalid.
	}

	// validate request attributes
	{
		var response saml.Response
		if err := unmarshalElement(responseEl, &response); err != nil {
			return nil, ucerr.Errorf("cannot unmarshal response: %v", err)
		}

		// If the response is *not* signed, the Destination may be omitted.
		if responseHasSignature || response.Destination != "" {
			if response.Destination != sp.AcsURL.String() {
				return nil, ucerr.Errorf("`Destination` does not match AcsURL (expected %q, actual %q)", sp.AcsURL.String(), response.Destination)
			}
		}

		requestIDvalid := false
		if sp.AllowIDPInitiated {
			requestIDvalid = true
		} else {
			for _, possibleRequestID := range possibleRequestIDs {
				if response.InResponseTo == possibleRequestID {
					requestIDvalid = true
				}
			}
		}
		if !requestIDvalid {
			return nil, ucerr.Errorf("`InResponseTo` does not match any of the possible request IDs (expected %v)", possibleRequestIDs)
		}

		if response.IssueInstant.Add(MaxIssueDelay).Before(now) {
			return nil, ucerr.Errorf("response IssueInstant expired at %s", response.IssueInstant.Add(MaxIssueDelay))
		}
		if response.Issuer != nil && response.Issuer.Value != sp.IDPMetadata.EntityID {
			return nil, ucerr.Errorf("response Issuer does not match the IDP metadata (expected %q)", sp.IDPMetadata.EntityID)
		}
		if response.Status.StatusCode.Value != saml.StatusSuccess {
			return nil, ucerr.Wrap(ErrBadStatus{Status: response.Status.StatusCode.Value})
		}
	}

	if signatureRequirement == signatureRequired {
		switch responseSignatureErr {
		case nil:
			// since the request has a signature, none of the Assertions need one
			signatureRequirement = signatureNotRequired
		case errSignatureElementNotPresent:
			// the request has no signature, so assertions must be signed
			signatureRequirement = signatureRequired // nop
		default:
			return nil, ucerr.Wrap(responseSignatureErr)
		}
	}

	var errs []error
	var assertions []saml.Assertion

	// look for encrypted assertions
	{
		encryptedAssertionEls, err := findChildren(responseEl, "urn:oasis:names:tc:SAML:2.0:assertion", "EncryptedAssertion")
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, encryptedAssertionEl := range encryptedAssertionEls {
			assertion, err := sp.parseEncryptedAssertion(encryptedAssertionEl, possibleRequestIDs, now, signatureRequirement)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			assertions = append(assertions, *assertion)
		}
	}

	// look for plaintext assertions
	{
		assertionEls, err := findChildren(responseEl, "urn:oasis:names:tc:SAML:2.0:assertion", "Assertion")
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, assertionEl := range assertionEls {
			assertion, err := sp.parseAssertion(assertionEl, possibleRequestIDs, now, signatureRequirement)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			assertions = append(assertions, *assertion)
		}
	}

	if len(assertions) == 0 {
		if len(errs) > 0 {
			return nil, ucerr.Wrap(errs[0])
		}
		return nil, ucerr.Errorf("expected at least one valid Assertion, none found")
	}

	// if we have at least one assertion, return the first one. It is almost universally true that valid responses
	// contain only one assertion. This is less that fully correct, but we didn't realize that there could be more
	// than one assertion at the time of establishing the public interface of ParseXMLResponse(), so for compatibility
	// we return the first one.
	return &assertions[0], nil
}

func (sp *ServiceProvider) parseEncryptedAssertion(encryptedAssertionEl *etree.Element, possibleRequestIDs []string, now time.Time, signatureRequirement signatureRequirement) (*saml.Assertion, error) {
	assertionEl, err := sp.decryptElement(encryptedAssertionEl)
	if err != nil {
		return nil, ucerr.Errorf("failed to decrypt EncryptedAssertion: %v", err)
	}
	return sp.parseAssertion(assertionEl, possibleRequestIDs, now, signatureRequirement)
}

func (sp *ServiceProvider) decryptElement(encryptedEl *etree.Element) (*etree.Element, error) {
	encryptedDataEl, err := findOneChild(encryptedEl, "http://www.w3.org/2001/04/xmlenc#", "EncryptedData")
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var key any = sp.Key
	if keyEl := encryptedEl.FindElement("./EncryptedKey"); keyEl != nil {
		var err error
		key, err = xmlenc.Decrypt(sp.Key, keyEl)
		if err != nil {
			return nil, ucerr.Errorf("failed to decrypt key from response: %s", err)
		}
	}

	plaintextEl, err := xmlenc.Decrypt(key, encryptedDataEl)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if err := xrv.Validate(bytes.NewReader(plaintextEl)); err != nil {
		return nil, ucerr.Errorf("plaintext response contains invalid XML: %s", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(plaintextEl); err != nil {
		return nil, ucerr.Errorf("cannot parse plaintext response %v", err)
	}
	return doc.Root(), nil
}

func (sp *ServiceProvider) parseAssertion(assertionEl *etree.Element, possibleRequestIDs []string, now time.Time, signatureRequirement signatureRequirement) (*saml.Assertion, error) {
	if signatureRequirement == signatureRequired {
		if sigErr := sp.validateSignature(assertionEl); sigErr != nil {
			return nil, ucerr.Wrap(sigErr)
		}
	}

	// parse the assertion we just validated
	var assertion saml.Assertion
	if err := unmarshalElement(assertionEl, &assertion); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if err := sp.validateAssertion(&assertion, possibleRequestIDs, now); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &assertion, nil
}

// validateAssertion checks that the conditions specified in assertion match
// the requirements to accept. If validation fails, it returns an error describing
// the failure. (The digital signature on the assertion is not checked -- this
// should be done before calling this function).
func (sp *ServiceProvider) validateAssertion(assertion *saml.Assertion, possibleRequestIDs []string, now time.Time) error {
	if assertion.IssueInstant.Add(MaxIssueDelay).Before(now) {
		return ucerr.Errorf("expired on %s", assertion.IssueInstant.Add(MaxIssueDelay))
	}
	if assertion.Issuer.Value != sp.IDPMetadata.EntityID {
		return ucerr.Errorf("issuer is not %q", sp.IDPMetadata.EntityID)
	}
	for _, subjectConfirmation := range assertion.Subject.SubjectConfirmations {
		// We *DO NOT* validate InResponseTo when AllowIDPInitiated is set. Here's why:
		//
		// The SAML specification does not provide clear guidance for handling InResponseTo for IDP-initiated
		// requests where there is no request to be in response to. The specification says:
		//
		//   InResponseTo [Optional]
		//       The ID of a SAML protocol message in response to which an attesting entity can present the
		//       assertion. For example, this attribute might be used to correlate the assertion to a SAML
		//       request that resulted in its presentation.
		//
		// The initial thought was that we should specify a single empty string in possibleRequestIDs for IDP-initiated
		// requests so that we would ensure that an InResponseTo was *not* provided in those cases where it wasn't
		// expected. Even that turns out to be frustrating for users. And in practice some IDPs (e.g. Rippling)
		// set a specific non-empty value for InResponseTo in IDP-initiated requests.
		//
		// Finally, it is unclear that there is significant security value in checking InResponseTo when we allow
		// IDP initiated assertions.
		if !sp.AllowIDPInitiated && !slices.Contains(possibleRequestIDs, subjectConfirmation.SubjectConfirmationData.InResponseTo) {
			return ucerr.Errorf("assertion SubjectConfirmation one of the possible request IDs (%v)", possibleRequestIDs)
		}
		if subjectConfirmation.SubjectConfirmationData.Recipient != sp.AcsURL.String() {
			return ucerr.Errorf("assertion SubjectConfirmation Recipient is not %s", sp.AcsURL.String())
		}
		if subjectConfirmation.SubjectConfirmationData.NotOnOrAfter.Add(MaxClockSkew).Before(now) {
			return ucerr.Errorf("assertion SubjectConfirmationData is expired")
		}
	}
	if assertion.Conditions.NotBefore.Add(-MaxClockSkew).After(now) {
		return ucerr.Errorf("assertion Conditions is not yet valid")
	}
	if assertion.Conditions.NotOnOrAfter.Add(MaxClockSkew).Before(now) {
		return ucerr.Errorf("assertion Conditions is expired")
	}

	audienceRestrictionsValid := len(assertion.Conditions.AudienceRestrictions) == 0
	audience := firstSet(sp.EntityID, sp.MetadataURL.String())
	for _, audienceRestriction := range assertion.Conditions.AudienceRestrictions {
		if audienceRestriction.Audience.Value == audience {
			audienceRestrictionsValid = true
		}
	}
	if !audienceRestrictionsValid {
		return ucerr.Errorf("assertion Conditions AudienceRestriction does not contain %q", audience)
	}
	return nil
}

var errSignatureElementNotPresent = ucerr.New("signature element not present")

// validateSignature returns nil iff the Signature embedded in the element is valid
func (sp *ServiceProvider) validateSignature(el *etree.Element) error {
	sigEl, err := findChild(el, "http://www.w3.org/2000/09/xmldsig#", "Signature")
	if err != nil {
		return ucerr.Wrap(err)
	}
	if sigEl == nil {
		return errSignatureElementNotPresent
	}

	certs, err := sp.getIDPSigningCerts()
	if err != nil {
		return ucerr.Errorf("cannot validate signature on %s: %v", el.Tag, err)
	}

	certificateStore := dsig.MemoryX509CertificateStore{
		Roots: certs,
	}

	validationContext := dsig.NewDefaultValidationContext(&certificateStore)
	validationContext.IdAttribute = "ID"
	if Clock != nil {
		validationContext.Clock = Clock
	}

	// Some SAML responses contain a RSAKeyValue element. One of two things is happening here:
	//
	// (1) We're getting something signed by a key we already know about -- the public key
	//     of the signing cert provided in the metadata.
	// (2) We're getting something signed by a key we *don't* know about, and which we have
	//     no ability to verify.
	//
	// The best course of action is to just remove the KeyInfo so that dsig falls back to
	// verifying against the public key provided in the metadata.
	if el.FindElement("./Signature/KeyInfo/X509Data/X509Certificate") == nil {
		if sigEl := el.FindElement("./Signature"); sigEl != nil {
			if keyInfo := sigEl.FindElement("KeyInfo"); keyInfo != nil {
				sigEl.RemoveChild(keyInfo)
			}
		}
	}

	ctx, err := etreeutils.NSBuildParentContext(el)
	if err != nil {
		return ucerr.Errorf("cannot validate signature on %s: %v", el.Tag, err)
	}
	ctx, err = ctx.SubContext(el)
	if err != nil {
		return ucerr.Errorf("cannot validate signature on %s: %v", el.Tag, err)
	}
	el, err = etreeutils.NSDetatch(ctx, el)
	if err != nil {
		return ucerr.Errorf("cannot validate signature on %s: %v", el.Tag, err)
	}

	if sp.SignatureVerifier != nil {
		return ucerr.Wrap(sp.SignatureVerifier.VerifySignature(validationContext, el))
	}

	if _, err := validationContext.Validate(el); err != nil {
		return ucerr.Errorf("cannot validate signature on %s: %v", el.Tag, err)
	}

	return nil
}

// SignLogoutRequest adds the `Signature` element to the `LogoutRequest`.
func (sp *ServiceProvider) SignLogoutRequest(req *saml.LogoutRequest) error {
	keyPair := tls.Certificate{
		Certificate: [][]byte{sp.Certificate.Raw},
		PrivateKey:  sp.Key,
		Leaf:        sp.Certificate,
	}
	// TODO: add intermediates for SP
	// for _, cert := range sp.Intermediates {
	// 	keyPair.Certificate = append(keyPair.Certificate, cert.Raw)
	// }
	keyStore := dsig.TLSCertKeyStore(keyPair)

	if sp.SignatureMethod != dsig.RSASHA1SignatureMethod &&
		sp.SignatureMethod != dsig.RSASHA256SignatureMethod &&
		sp.SignatureMethod != dsig.RSASHA512SignatureMethod {
		return ucerr.Errorf("invalid signing method %s", sp.SignatureMethod)
	}
	signatureMethod := sp.SignatureMethod
	signingContext := dsig.NewDefaultSigningContext(keyStore)
	signingContext.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList(canonicalizerPrefixList)
	if err := signingContext.SetSignatureMethod(signatureMethod); err != nil {
		return ucerr.Wrap(err)
	}

	assertionEl := req.Element()

	signedRequestEl, err := signingContext.SignEnveloped(assertionEl)
	if err != nil {
		return ucerr.Wrap(err)
	}

	sigEl := signedRequestEl.Child[len(signedRequestEl.Child)-1]
	req.Signature = sigEl.(*etree.Element)
	return nil
}

// MakeLogoutRequest produces a new LogoutRequest object for idpURL.
func (sp *ServiceProvider) MakeLogoutRequest(idpURL, nameID string) (*saml.LogoutRequest, error) {

	req := saml.LogoutRequest{
		ID:           fmt.Sprintf("id-%x", randomBytes(20)),
		IssueInstant: TimeNow(),
		Version:      "2.0",
		Destination:  idpURL,
		Issuer: &saml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  firstSet(sp.EntityID, sp.MetadataURL.String()),
		},
		NameID: &saml.NameID{
			Format:          sp.nameIDFormat(),
			Value:           nameID,
			NameQualifier:   sp.IDPMetadata.EntityID,
			SPNameQualifier: sp.Metadata().EntityID,
		},
	}
	if len(sp.SignatureMethod) > 0 {
		if err := sp.SignLogoutRequest(&req); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}
	return &req, nil
}

// MakeRedirectLogoutRequest creates a SAML authentication request using
// the HTTP-Redirect binding. It returns a URL that we will redirect the user to
// in order to start the auth process.
func (sp *ServiceProvider) MakeRedirectLogoutRequest(nameID, relayState string) (*url.URL, error) {
	req, err := sp.MakeLogoutRequest(sp.GetSLOBindingLocation(samlconfig.HTTPRedirectBinding), nameID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return req.Redirect(relayState), nil
}

// MakePostLogoutRequest creates a SAML authentication request using
// the HTTP-POST binding. It returns HTML text representing an HTML form that
// can be sent presented to a browser to initiate the logout process.
func (sp *ServiceProvider) MakePostLogoutRequest(nameID, relayState string) ([]byte, error) {
	req, err := sp.MakeLogoutRequest(sp.GetSLOBindingLocation(samlconfig.HTTPPostBinding), nameID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return req.Post(relayState), nil
}

// MakeLogoutResponse produces a new LogoutResponse object for idpURL and logoutRequestID.
func (sp *ServiceProvider) MakeLogoutResponse(idpURL, logoutRequestID string) (*saml.LogoutResponse, error) {
	response := saml.LogoutResponse{
		ID:           fmt.Sprintf("id-%x", randomBytes(20)),
		InResponseTo: logoutRequestID,
		Version:      "2.0",
		IssueInstant: TimeNow(),
		Destination:  idpURL,
		Issuer: &saml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  firstSet(sp.EntityID, sp.MetadataURL.String()),
		},
		Status: saml.Status{
			StatusCode: saml.StatusCode{
				Value: saml.StatusSuccess,
			},
		},
	}

	if len(sp.SignatureMethod) > 0 {
		if err := sp.SignLogoutResponse(&response); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}
	return &response, nil
}

// MakeRedirectLogoutResponse creates a SAML LogoutResponse using
// the HTTP-Redirect binding. It returns a URL that we will redirect the user to
// for LogoutResponse.
func (sp *ServiceProvider) MakeRedirectLogoutResponse(logoutRequestID, relayState string) (*url.URL, error) {
	resp, err := sp.MakeLogoutResponse(sp.GetSLOBindingLocation(samlconfig.HTTPRedirectBinding), logoutRequestID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return resp.Redirect(relayState), nil
}

// MakePostLogoutResponse creates a SAML LogoutResponse using
// the HTTP-POST binding. It returns HTML text representing an HTML form that
// can be sent presented to a browser for LogoutResponse.
func (sp *ServiceProvider) MakePostLogoutResponse(logoutRequestID, relayState string) ([]byte, error) {
	resp, err := sp.MakeLogoutResponse(sp.GetSLOBindingLocation(samlconfig.HTTPPostBinding), logoutRequestID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return resp.Post(relayState), nil
}

// SignLogoutResponse adds the `Signature` element to the `LogoutResponse`.
func (sp *ServiceProvider) SignLogoutResponse(resp *saml.LogoutResponse) error {
	keyPair := tls.Certificate{
		Certificate: [][]byte{sp.Certificate.Raw},
		PrivateKey:  sp.Key,
		Leaf:        sp.Certificate,
	}
	// TODO: add intermediates for SP
	// for _, cert := range sp.Intermediates {
	// 	keyPair.Certificate = append(keyPair.Certificate, cert.Raw)
	// }
	keyStore := dsig.TLSCertKeyStore(keyPair)

	if sp.SignatureMethod != dsig.RSASHA1SignatureMethod &&
		sp.SignatureMethod != dsig.RSASHA256SignatureMethod &&
		sp.SignatureMethod != dsig.RSASHA512SignatureMethod {
		return ucerr.Errorf("invalid signing method %s", sp.SignatureMethod)
	}
	signatureMethod := sp.SignatureMethod
	signingContext := dsig.NewDefaultSigningContext(keyStore)
	signingContext.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList(canonicalizerPrefixList)
	if err := signingContext.SetSignatureMethod(signatureMethod); err != nil {
		return ucerr.Wrap(err)
	}

	assertionEl := resp.Element()

	signedRequestEl, err := signingContext.SignEnveloped(assertionEl)
	if err != nil {
		return ucerr.Wrap(err)
	}

	sigEl := signedRequestEl.Child[len(signedRequestEl.Child)-1]
	resp.Signature = sigEl.(*etree.Element)
	return nil
}

func (sp *ServiceProvider) nameIDFormat() string {
	var nameIDFormat string
	switch sp.AuthnNameIDFormat {
	case "":
		// To maintain library back-compat, use "transient" if unset.
		nameIDFormat = string(TransientNameIDFormat)
	case UnspecifiedNameIDFormat:
		// Spec defines an empty value as "unspecified" so don't set one.
	default:
		nameIDFormat = string(sp.AuthnNameIDFormat)
	}
	return nameIDFormat
}

// ValidateLogoutResponseRequest validates the LogoutResponse content from the request
func (sp *ServiceProvider) ValidateLogoutResponseRequest(req *http.Request) error {
	if data := req.URL.Query().Get("SAMLResponse"); data != "" {
		return ucerr.Wrap(sp.ValidateLogoutResponseRedirect(data))
	}

	if err := req.ParseForm(); err != nil {
		return ucerr.Errorf("unable to parse form: %v", err)
	}

	return ucerr.Wrap(sp.ValidateLogoutResponseForm(req.PostForm.Get("SAMLResponse")))
}

// ValidateLogoutResponseForm returns a nil error if the logout response is valid.
func (sp *ServiceProvider) ValidateLogoutResponseForm(postFormData string) error {
	retErr := &InvalidResponseError{
		Now: TimeNow(),
	}

	rawResponseBuf, err := base64.StdEncoding.DecodeString(postFormData)
	if err != nil {
		retErr.PrivateErr = ucerr.Errorf("unable to parse base64: %s", err)
		return ucerr.Wrap(retErr)
	}
	retErr.Response = string(rawResponseBuf)

	// TODO(ross): add test case for this (SLO does not have tests right now)
	if err := xrv.Validate(bytes.NewReader(rawResponseBuf)); err != nil {
		return ucerr.Errorf("response contains invalid XML: %s", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(rawResponseBuf); err != nil {
		retErr.PrivateErr = err
		return ucerr.Wrap(retErr)
	}

	if err := sp.validateSignature(doc.Root()); err != nil {
		retErr.PrivateErr = err
		return ucerr.Wrap(retErr)
	}

	var resp saml.LogoutResponse
	if err := unmarshalElement(doc.Root(), &resp); err != nil {
		retErr.PrivateErr = err
		return ucerr.Wrap(retErr)
	}
	return ucerr.Wrap(sp.validateLogoutResponse(&resp))
}

// ValidateLogoutResponseRedirect returns a nil error if the logout response is valid.
//
// URL Binding appears to be gzip / flate encoded
// See https://www.oasis-open.org/committees/download.php/20645/sstc-saml-tech-overview-2%200-draft-10.pdf  6.6
func (sp *ServiceProvider) ValidateLogoutResponseRedirect(queryParameterData string) error {
	retErr := &InvalidResponseError{
		Now: TimeNow(),
	}

	rawResponseBuf, err := base64.StdEncoding.DecodeString(queryParameterData)
	if err != nil {
		retErr.PrivateErr = ucerr.Errorf("unable to parse base64: %s", err)
		return ucerr.Wrap(retErr)
	}
	retErr.Response = string(rawResponseBuf)

	gr, err := io.ReadAll(newSaferFlateReader(bytes.NewBuffer(rawResponseBuf)))
	if err != nil {
		retErr.PrivateErr = err
		return ucerr.Wrap(retErr)
	}

	if err := xrv.Validate(bytes.NewReader(gr)); err != nil {
		return ucerr.Wrap(err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(rawResponseBuf); err != nil {
		retErr.PrivateErr = err
		return ucerr.Wrap(retErr)
	}

	if err := sp.validateSignature(doc.Root()); err != nil {
		retErr.PrivateErr = err
		return ucerr.Wrap(retErr)
	}

	var resp saml.LogoutResponse
	if err := unmarshalElement(doc.Root(), &resp); err != nil {
		retErr.PrivateErr = err
		return ucerr.Wrap(retErr)
	}
	return ucerr.Wrap(sp.validateLogoutResponse(&resp))
}

// validateLogoutResponse validates the LogoutResponse fields. Returns a nil error if the LogoutResponse is valid.
func (sp *ServiceProvider) validateLogoutResponse(resp *saml.LogoutResponse) error {
	if resp.Destination != sp.SloURL.String() {
		return ucerr.Errorf("`Destination` does not match SloURL (expected %q)", sp.SloURL.String())
	}

	now := time.Now().UTC()
	if resp.IssueInstant.Add(MaxIssueDelay).Before(now) {
		return ucerr.Errorf("issueInstant expired at %s", resp.IssueInstant.Add(MaxIssueDelay))
	}
	if resp.Issuer.Value != sp.IDPMetadata.EntityID {
		return ucerr.Errorf("issuer does not match the IDP metadata (expected %q)", sp.IDPMetadata.EntityID)
	}
	if resp.Status.StatusCode.Value != saml.StatusSuccess {
		return ucerr.Errorf("status code was not %s", saml.StatusSuccess)
	}

	return nil
}

func firstSet(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

// findChildren returns all the elements matching childNS/childTag that are direct children of parentEl.
func findChildren(parentEl *etree.Element, childNS string, childTag string) ([]*etree.Element, error) {
	//nolint:prealloc // We don't know how many child elements we'll actually put into this array.
	var rv []*etree.Element
	for _, childEl := range parentEl.ChildElements() {
		if childEl.Tag != childTag {
			continue
		}

		ctx, err := etreeutils.NSBuildParentContext(childEl)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		ctx, err = ctx.SubContext(childEl)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		ns, err := ctx.LookupPrefix(childEl.Space)
		if err != nil {
			return nil, ucerr.Errorf("[%s]:%s cannot find prefix %s: %v", childNS, childTag, childEl.Space, err)
		}
		if ns != childNS {
			continue
		}

		rv = append(rv, childEl)
	}

	return rv, nil
}

// findOneChild finds the specified child element. Returns an error if the element doesn't exist.
func findOneChild(parentEl *etree.Element, childNS string, childTag string) (*etree.Element, error) {
	children, err := findChildren(parentEl, childNS, childTag)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	switch len(children) {
	case 0:
		return nil, ucerr.Errorf("cannot find %s:%s element", childNS, childTag)
	case 1:
		return children[0], nil
	default:
		return nil, ucerr.Errorf("expected exactly one %s:%s element", childNS, childTag)
	}
}

// findChild finds the specified child element. Returns (nil, nil) of the element doesn't exist.
func findChild(parentEl *etree.Element, childNS string, childTag string) (*etree.Element, error) {
	children, err := findChildren(parentEl, childNS, childTag)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	switch len(children) {
	case 0:
		return nil, nil
	case 1:
		return children[0], nil
	default:
		return nil, ucerr.Errorf("expected at most one %s:%s element", childNS, childTag)
	}
}

func elementToBytes(el *etree.Element) ([]byte, error) {
	namespaces := map[string]string{}
	for _, childEl := range el.FindElements("//*") {
		if ns := childEl.NamespaceURI(); ns != "" {
			namespaces[childEl.Space] = ns
		}
	}

	doc := etree.NewDocument()
	doc.SetRoot(el.Copy())
	for space, uri := range namespaces {
		doc.Root().CreateAttr("xmlns:"+space, uri)
	}

	return doc.WriteToBytes()
}

// unmarshalElement serializes el into v by serializing el and then parsing it with xml.Unmarshal.
func unmarshalElement(el *etree.Element, v any) error {
	buf, err := elementToBytes(el)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(xml.Unmarshal(buf, v))
}

func elementToString(el *etree.Element) string {
	buf, err := elementToBytes(el)
	if err != nil {
		return ""
	}
	return string(buf)
}
