package saml

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"

	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/tenantplex/samlconfig"
)

type ServerTest struct {
	SPKey         *rsa.PrivateKey
	SPCertificate *x509.Certificate
	SP            ServiceProvider
	IDP           IdentityProvider
	Key           crypto.PrivateKey
	Certificate   *x509.Certificate

	handler http.Handler
}

func NewServerTest(t *testing.T) *ServerTest {
	test := ServerTest{}
	TimeNow = func() time.Time {
		rv, err := time.Parse("Mon Jan 2 15:04:05 MST 2006", "Mon Dec 1 01:57:09 UTC 2015")
		assert.NilError(t, err)
		return rv
	}
	ucjwt.TimeFunc = TimeNow
	RandReader = &testRandomReader{}

	test.SPKey = mustParsePrivateKey(golden.Get(t, "sp_key.pem")).(*rsa.PrivateKey)
	test.SPCertificate = mustParseCertificate(golden.Get(t, "sp_cert.pem"))
	test.SP = ServiceProvider{
		Key:         test.SPKey,
		Certificate: test.SPCertificate,
		MetadataURL: mustParseURL("https://sp.example.com/saml2/metadata"),
		AcsURL:      mustParseURL("https://sp.example.com/saml2/acs"),
		IDPMetadata: &samlconfig.EntityDescriptor{},
	}
	test.Key = mustParsePrivateKey(golden.Get(t, "idp_key.pem")).(*rsa.PrivateKey)
	test.Certificate = mustParseCertificate(golden.Get(t, "idp_cert.pem"))

	test.SP.IDPMetadata = test.IDP.Metadata()
	test.IDP.ServiceProviders["https://sp.example.com/saml2/metadata"] = test.SP.Metadata()
	var err error
	test.handler, err = NewHandler()
	assert.NilError(t, err)
	return &test
}

func TestHTTPCanHandleMetadataRequest(t *testing.T) {
	t.Skip() // TODO
	test := NewServerTest(t)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "https://idp.example.com/metadata", nil)
	assert.NilError(t, err)
	test.handler.ServeHTTP(w, r)
	assert.Check(t, is.Equal(http.StatusOK, w.Code))
	assert.Check(t,
		strings.HasPrefix(w.Body.String(), "<EntityDescriptor"),
		w.Body.String())
	golden.Assert(t, w.Body.String(), "http_metadata_response.html")
}

func TestHTTPCanSSORequest(t *testing.T) {
	t.Skip() // TODO
	test := NewServerTest(t)
	u, err := test.SP.MakeRedirectAuthenticationRequest("frob")
	assert.Check(t, err)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", u.String(), nil)
	assert.NilError(t, err)

	test.handler.ServeHTTP(w, r)
	assert.Check(t, is.Equal(http.StatusOK, w.Code))
	assert.Check(t,
		strings.HasPrefix(w.Body.String(), "<html><p></p><form method=\"post\" action=\"https://idp.example.com/sso\">"),
		w.Body.String())
	golden.Assert(t, w.Body.String(), "http_sso_response.html")
}
