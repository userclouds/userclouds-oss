package saml

import (
	"bytes"
	"compress/flate"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/testsaml"
	"github.com/crewjam/saml/xmlenc"
	"github.com/gofrs/uuid"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"

	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/tenantplex/samlconfig"
	"userclouds.com/internal/testkeys"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
	"userclouds.com/test/testlogtransport"
)

type IdentityProviderTest struct {
	SPKey         *rsa.PrivateKey
	SPCertificate *x509.Certificate
	SP            ServiceProvider

	Key         crypto.PrivateKey
	Certificate *x509.Certificate
	IDP         IdentityProvider

	h *handler
}

func mustParseURL(s string) url.URL {
	rv, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return *rv
}

func mustParsePrivateKey(pemStr []byte) crypto.PrivateKey {
	b, _ := pem.Decode(pemStr)
	if b == nil {
		panic("cannot parse PEM")
	}
	k, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	if err != nil {
		panic(err)
	}
	return k
}

func mustParseCertificate(pemStr []byte) *x509.Certificate {
	b, _ := pem.Decode(pemStr)
	if b == nil {
		panic("cannot parse PEM")
	}
	cert, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		panic(err)
	}
	return cert
}

// TODO better way to inject TSPs here?
var tsp = []samlconfig.EntityDescriptor{{
	EntityID: "https://sp.example.com/saml2/metadata",
	SPSSODescriptors: []samlconfig.SPSSODescriptor{{
		AssertionConsumerServices: []samlconfig.IndexedEndpoint{{
			Index:    1,
			Binding:  saml.HTTPPostBinding,
			Location: "https://sp.example.com/saml2/acs",
		}},
	}},
}}

var testCert = `-----BEGIN CERTIFICATE-----
MIIB7zCCAVgCCQDFzbKIp7b3MTANBgkqhkiG9w0BAQUFADA8MQswCQYDVQQGEwJV
UzELMAkGA1UECAwCR0ExDDAKBgNVBAoMA2ZvbzESMBAGA1UEAwwJbG9jYWxob3N0
MB4XDTEzMTAwMjAwMDg1MVoXDTE0MTAwMjAwMDg1MVowPDELMAkGA1UEBhMCVVMx
CzAJBgNVBAgMAkdBMQwwCgYDVQQKDANmb28xEjAQBgNVBAMMCWxvY2FsaG9zdDCB
nzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEA1PMHYmhZj308kWLhZVT4vOulqx/9
ibm5B86fPWwUKKQ2i12MYtz07tzukPymisTDhQaqyJ8Kqb/6JjhmeMnEOdTvSPmH
O8m1ZVveJU6NoKRn/mP/BD7FW52WhbrUXLSeHVSKfWkNk6S4hk9MV9TswTvyRIKv
Rsw0X/gfnqkroJcCAwEAATANBgkqhkiG9w0BAQUFAAOBgQCMMlIO+GNcGekevKgk
akpMdAqJfs24maGb90DvTLbRZRD7Xvn1MnVBBS9hzlXiFLYOInXACMW5gcoRFfeT
QLSouMM8o57h0uKjfTmuoWHLQLi6hnF+cvCsEFiJZ4AbF+DgmO6TarJ8O05t8zvn
OwJlNCASPZRH/JmF8tX0hoHuAQ==
-----END CERTIFICATE-----`

var testKey = secret.NewTestString(`-----BEGIN RSA PRIVATE KEY-----
MIICXgIBAAKBgQDU8wdiaFmPfTyRYuFlVPi866WrH/2JubkHzp89bBQopDaLXYxi
3PTu3O6Q/KaKxMOFBqrInwqpv/omOGZ4ycQ51O9I+Yc7ybVlW94lTo2gpGf+Y/8E
PsVbnZaFutRctJ4dVIp9aQ2TpLiGT0xX1OzBO/JEgq9GzDRf+B+eqSuglwIDAQAB
AoGBAMuy1eN6cgFiCOgBsB3gVDdTKpww87Qk5ivjqEt28SmXO13A1KNVPS6oQ8SJ
CT5Azc6X/BIAoJCURVL+LHdqebogKljhH/3yIel1kH19vr4E2kTM/tYH+qj8afUS
JEmArUzsmmK8ccuNqBcllqdwCZjxL4CHDUmyRudFcHVX9oyhAkEA/OV1OkjM3CLU
N3sqELdMmHq5QZCUihBmk3/N5OvGdqAFGBlEeewlepEVxkh7JnaNXAXrKHRVu/f/
fbCQxH+qrwJBANeQERF97b9Sibp9xgolb749UWNlAdqmEpmlvmS202TdcaaT1msU
4rRLiQN3X9O9mq4LZMSVethrQAdX1whawpkCQQDk1yGf7xZpMJ8F4U5sN+F4rLyM
Rq8Sy8p2OBTwzCUXXK+fYeXjybsUUMr6VMYTRP2fQr/LKJIX+E5ZxvcIyFmDAkEA
yfjNVUNVaIbQTzEbRlRvT6MqR+PTCefC072NF9aJWR93JimspGZMR7viY6IM4lrr
vBkm0F5yXKaYtoiiDMzlOQJADqmEwXl0D72ZG/2KDg8b4QZEmC9i5gidpQwJXUc6
hU+IVQoLxRq0fBib/36K9tcrrO5Ba4iEvDcNY+D8yGbUtA==
-----END RSA PRIVATE KEY-----`)

func newRequest(t *testing.T, method, path string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, path, body)
	req = req.WithContext(tenantconfig.TESTONLYSetTenantConfig(&tenantplex.TenantConfig{
		PlexMap: tenantplex.PlexMap{
			Apps: []tenantplex.App{{ClientID: "foo", SAMLIDP: &tenantplex.SAMLIDP{
				Certificate:             testCert,
				PrivateKey:              testKey,
				MetadataURL:             "https://idp.example.com/metadata/foo",
				SSOURL:                  "https://idp.example.com/sso/foo",
				TrustedServiceProviders: tsp,
			}}},
		},
	}))
	assert.NilError(t, err)
	return req
}

func NewIdentifyProviderTest(t *testing.T) *IdentityProviderTest {
	test := IdentityProviderTest{}
	TimeNow = func() time.Time {
		rv, err := time.Parse("Mon Jan 2 15:04:05 MST 2006", "Mon Dec 1 01:57:09 UTC 2015")
		assert.NilError(t, err)
		return rv
	}
	ucjwt.TimeFunc = TimeNow
	RandReader = &testRandomReader{}                // TODO(ross): remove this and use the below generator
	xmlenc.RandReader = rand.New(rand.NewSource(0)) //nolint:gosec  // deterministic random numbers for tests

	test.SPKey = mustParsePrivateKey(golden.Get(t, "sp_key.pem")).(*rsa.PrivateKey)
	test.SPCertificate = mustParseCertificate(golden.Get(t, "sp_cert.pem"))
	test.SP = ServiceProvider{
		Key:         test.SPKey,
		Certificate: test.SPCertificate,
		MetadataURL: mustParseURL("https://sp.example.com/saml2/metadata"),
		AcsURL:      mustParseURL("https://sp.example.com/saml2/acs"),
		IDPMetadata: &samlconfig.EntityDescriptor{},
	}

	test.Key = mustParsePrivateKey(golden.Get(t, "idp_key.pem"))
	test.Certificate = mustParseCertificate(golden.Get(t, "idp_cert.pem"))

	test.IDP = IdentityProvider{
		Key:         test.Key,
		Certificate: test.Certificate,
		MetadataURL: mustParseURL("https://idp.example.com/saml/metadata"),
		SSOURL:      mustParseURL("https://idp.example.com/sso/foo"),
		ServiceProviders: map[string]*samlconfig.EntityDescriptor{"https://sp.example.com/saml2/metadata": {
			EntityID: "https://sp.example.com/saml2/metadata",
			SPSSODescriptors: []samlconfig.SPSSODescriptor{{
				AssertionConsumerServices: []samlconfig.IndexedEndpoint{{
					Location: "https://sp.example.com/saml2/acs",
				}},
			}},
		}},
	}

	test.h = &handler{}

	// bind the service provider and the IDP
	test.SP.IDPMetadata = test.IDP.Metadata()
	return &test
}

func TestIDPCanProduceMetadata(t *testing.T) {
	test := NewIdentifyProviderTest(t)
	expected := &samlconfig.EntityDescriptor{
		ValidUntil:    TimeNow().Add(DefaultValidDuration),
		CacheDuration: DefaultValidDuration,
		EntityID:      "https://idp.example.com/saml/metadata",
		IDPSSODescriptors: []samlconfig.IDPSSODescriptor{
			{
				SSODescriptor: samlconfig.SSODescriptor{
					RoleDescriptor: samlconfig.RoleDescriptor{
						ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
						KeyDescriptors: []samlconfig.KeyDescriptor{
							{
								Use: "signing",
								KeyInfo: samlconfig.KeyInfo{
									XMLName: xml.Name{},
									X509Data: samlconfig.X509Data{
										X509Certificates: []samlconfig.X509Certificate{
											{Data: "MIIB7zCCAVgCCQDFzbKIp7b3MTANBgkqhkiG9w0BAQUFADA8MQswCQYDVQQGEwJVUzELMAkGA1UECAwCR0ExDDAKBgNVBAoMA2ZvbzESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTEzMTAwMjAwMDg1MVoXDTE0MTAwMjAwMDg1MVowPDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkdBMQwwCgYDVQQKDANmb28xEjAQBgNVBAMMCWxvY2FsaG9zdDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEA1PMHYmhZj308kWLhZVT4vOulqx/9ibm5B86fPWwUKKQ2i12MYtz07tzukPymisTDhQaqyJ8Kqb/6JjhmeMnEOdTvSPmHO8m1ZVveJU6NoKRn/mP/BD7FW52WhbrUXLSeHVSKfWkNk6S4hk9MV9TswTvyRIKvRsw0X/gfnqkroJcCAwEAATANBgkqhkiG9w0BAQUFAAOBgQCMMlIO+GNcGekevKgkakpMdAqJfs24maGb90DvTLbRZRD7Xvn1MnVBBS9hzlXiFLYOInXACMW5gcoRFfeTQLSouMM8o57h0uKjfTmuoWHLQLi6hnF+cvCsEFiJZ4AbF+DgmO6TarJ8O05t8zvnOwJlNCASPZRH/JmF8tX0hoHuAQ=="},
										},
									},
								},
								EncryptionMethods: nil,
							},
							{
								Use: "encryption",
								KeyInfo: samlconfig.KeyInfo{
									XMLName: xml.Name{},
									X509Data: samlconfig.X509Data{
										X509Certificates: []samlconfig.X509Certificate{
											{Data: "MIIB7zCCAVgCCQDFzbKIp7b3MTANBgkqhkiG9w0BAQUFADA8MQswCQYDVQQGEwJVUzELMAkGA1UECAwCR0ExDDAKBgNVBAoMA2ZvbzESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTEzMTAwMjAwMDg1MVoXDTE0MTAwMjAwMDg1MVowPDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkdBMQwwCgYDVQQKDANmb28xEjAQBgNVBAMMCWxvY2FsaG9zdDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEA1PMHYmhZj308kWLhZVT4vOulqx/9ibm5B86fPWwUKKQ2i12MYtz07tzukPymisTDhQaqyJ8Kqb/6JjhmeMnEOdTvSPmHO8m1ZVveJU6NoKRn/mP/BD7FW52WhbrUXLSeHVSKfWkNk6S4hk9MV9TswTvyRIKvRsw0X/gfnqkroJcCAwEAATANBgkqhkiG9w0BAQUFAAOBgQCMMlIO+GNcGekevKgkakpMdAqJfs24maGb90DvTLbRZRD7Xvn1MnVBBS9hzlXiFLYOInXACMW5gcoRFfeTQLSouMM8o57h0uKjfTmuoWHLQLi6hnF+cvCsEFiJZ4AbF+DgmO6TarJ8O05t8zvnOwJlNCASPZRH/JmF8tX0hoHuAQ=="},
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
						Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
						Location: "https://idp.example.com/sso/foo",
					},
					{
						Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
						Location: "https://idp.example.com/sso/foo",
					},
				},
			},
		},
	}
	assert.Check(t, is.DeepEqual(expected, test.IDP.Metadata()))
}

func TestIDPHTTPCanHandleMetadataRequest(t *testing.T) {
	testlogtransport.InitLoggerAndTransportsForTests(t)
	w := httptest.NewRecorder()
	r := newRequest(t, "GET", "https://idp.example.com/metadata/foo", nil)
	h, err := NewHandler()
	assert.NilError(t, err)
	h.ServeHTTP(w, r)
	assert.Check(t, is.Equal(http.StatusOK, w.Code))
	assert.Check(t, is.Equal("application/samlmetadata+xml", w.Header().Get("Content-type")))
	assert.Check(t, strings.HasPrefix(w.Body.String(), "<EntityDescriptor"),
		w.Body.String())
}

func TestIDPCanHandleRequestWithNewSession(t *testing.T) {
	t.Skip() // TODO
	test := NewIdentifyProviderTest(t)

	w := httptest.NewRecorder()

	requestURL, err := test.SP.MakeRedirectAuthenticationRequest("ThisIsTheRelayState")
	assert.Check(t, err)

	decodedRequest, err := testsaml.ParseRedirectRequest(requestURL)
	assert.Check(t, err)
	golden.Assert(t, string(decodedRequest), "idp_authn_request.xml")
	assert.Check(t, is.Equal("ThisIsTheRelayState", requestURL.Query().Get("RelayState")))

	r := newRequest(t, "GET", requestURL.String(), nil)
	h, err := NewHandler()
	assert.NilError(t, err)
	h.ServeHTTP(w, r)
	assert.Check(t, is.Equal(200, w.Code))
	golden.Assert(t, w.Body.String(), t.Name()+"_http_response_body")
}

func TestIDPCanHandleRequestWithExistingSession(t *testing.T) {
	t.Skip() // TODO
	test := NewIdentifyProviderTest(t)

	w := httptest.NewRecorder()
	requestURL, err := test.SP.MakeRedirectAuthenticationRequest("ThisIsTheRelayState")
	assert.Check(t, err)

	decodedRequest, err := testsaml.ParseRedirectRequest(requestURL)
	assert.Check(t, err)
	golden.Assert(t, string(decodedRequest), t.Name()+"_decodedRequest")

	r := newRequest(t, "GET", requestURL.String(), nil)
	h, err := NewHandler()
	assert.NilError(t, err)
	h.ServeHTTP(w, r)
	assert.Check(t, is.Equal(200, w.Code))
	golden.Assert(t, w.Body.String(), t.Name()+"_http_response_body")
}

func TestIDPCanHandlePostRequestWithExistingSession(t *testing.T) {
	t.Skip() // TODO
	test := NewIdentifyProviderTest(t)

	w := httptest.NewRecorder()

	authRequest, err := test.SP.MakeAuthenticationRequest(test.SP.GetSSOBindingLocation(samlconfig.HTTPRedirectBinding), samlconfig.HTTPRedirectBinding, samlconfig.HTTPPostBinding)
	assert.Check(t, err)
	authRequestBuf, err := xml.Marshal(authRequest)
	assert.Check(t, err)
	q := url.Values{}
	q.Set("SAMLRequest", base64.StdEncoding.EncodeToString(authRequestBuf))
	q.Set("RelayState", "ThisIsTheRelayState")

	r := newRequest(t, "POST", "https://idp.example.com/sso/foo", strings.NewReader(q.Encode()))
	r.Header.Set("Content-type", "application/x-www-form-urlencoded")

	h, err := NewHandler()
	assert.NilError(t, err)
	h.ServeHTTP(w, r)
	assert.Check(t, is.Equal(200, w.Code))
	golden.Assert(t, w.Body.String(), t.Name()+"_http_response_body")
}

func TestIDPRejectsInvalidRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest(t, "GET", "https://idp.example.com/sso/foo?RelayState=ThisIsTheRelayState&SAMLRequest=XXX", nil)
	r = r.WithContext(tenantconfig.TESTONLYSetTenantConfig(&tenantplex.TenantConfig{
		PlexMap: tenantplex.PlexMap{
			Apps: []tenantplex.App{{ClientID: "foo", SAMLIDP: &tenantplex.SAMLIDP{
				Certificate: testkeys.CertificatePEM,
				PrivateKey:  testkeys.CertificatePrivateKeyPEMSecret,
			}}},
		},
	}))
	h, err := NewHandler()
	assert.NilError(t, err)

	h.ServeHTTP(w, r)
	assert.Check(t, is.Equal(http.StatusBadRequest, w.Code))

	w = httptest.NewRecorder()
	r = newRequest(t, "POST", "https://idp.example.com/sso/foo",
		strings.NewReader("RelayState=ThisIsTheRelayState&SAMLRequest=XXX"))
	r = r.WithContext(tenantconfig.TESTONLYSetTenantConfig(&tenantplex.TenantConfig{
		PlexMap: tenantplex.PlexMap{
			Apps: []tenantplex.App{{ClientID: "foo", SAMLIDP: &tenantplex.SAMLIDP{
				Certificate: testkeys.CertificatePEM,
				PrivateKey:  testkeys.CertificatePrivateKeyPEMSecret,
			}}},
		},
	}))
	r.Header.Set("Content-type", "application/x-www-form-urlencoded")
	h.ServeHTTP(w, r)
	assert.Check(t, is.Equal(http.StatusBadRequest, w.Code))
}

func TestIDPCanParse(t *testing.T) {
	test := NewIdentifyProviderTest(t)

	r := newRequest(t, "GET", "https://idp.example.com/sso/foo?RelayState=ThisIsTheRelayState&SAMLRequest=lZJNawIxFEX%2FypC9JhnU2jAzYJWCYIvYj0V3j8yzBiaJzXvT6r%2FvaFsqXYjdhsPNuTcpJi1vwgrfWiTOdr4JVIo2BROBHJkAHsmwNQ%2BTu4XJ%2B8psU%2BRoYyOyCREmdjFMY6DWY3rA9O4sPq0Wpdgwb8lISds%2B7sBvG%2Bzb6CWBb3IJlkQ26y50AQ4Bv7ir%2F%2FAU5TpGkc1npXB1TymVq4EaqbECZRVqpXM90CM91qCtxg4kanEeiCFwKXKlhz2d95R%2BVNoMr4y6fhHZ8rvDjQu1C6%2BlENkzJjqadB1FVRxT0iV7wM8KIruNyQOfxw8nXY%2F1ETUY2PFeVGfX8shQA0Mhv6yq4r4Lmc%2BWsXF2%2F883a5r4MU0IjKXg1KKoLrflBIFc51zIU4OqkKefqPoE", nil)
	req, err := NewIdpAuthnRequest(&test.IDP, r)
	assert.Check(t, err)
	assert.Check(t, test.h.ValidateRequest(req))

	r = newRequest(t, "GET", "https://idp.example.com/sso/foo?RelayState=ThisIsTheRelayState", nil)
	idp, _, err := test.h.getIDP(r)
	assert.NilError(t, err)
	_, err = NewIdpAuthnRequest(idp, r)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "cannot decompress request: unexpected EOF"))

	r = newRequest(t, "GET", "https://idp.example.com/sso/foo?RelayState=ThisIsTheRelayState&SAMLRequest=NotValidBase64", nil)
	_, err = NewIdpAuthnRequest(idp, r)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "cannot decode request: illegal base64 data at input byte 12"))

	r = newRequest(t, "GET", "https://idp.example.com/sso/foo?RelayState=ThisIsTheRelayState&SAMLRequest=bm90IGZsYXRlIGVuY29kZWQ%3D", nil)
	_, err = NewIdpAuthnRequest(idp, r)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "cannot decompress request: flate: corrupt input before offset 1"))

	r = newRequest(t, "FROBNICATE", "https://idp.example.com/sso/foo?RelayState=ThisIsTheRelayState&SAMLRequest=lJJBayoxFIX%2FypC9JhnU5wszAz7lgWCLaNtFd5fMbQ1MkmnunVb%2FfUfbUqEgdhs%2BTr5zkmLW8S5s8KVD4mzvm0Cl6FIwEciRCeCRDFuznd2sTD5Upk2Ro42NyGZEmNjFMI%2BBOo9pi%2BnVWbzfrEqxY27JSEntEPfg2waHNnpJ4JtcgiWRLfoLXYBjwDfu6p%2B8JIoiWy5K4eqBUipXIzVRUwXKKtRK53qkJ3qqQVuNPUjU4TIQQ%2BBS5EqPBzofKH2ntBn%2FMervo8jWnyX%2BuVC78FwKkT1gopNKX1JUxSklXTMIfM0gsv8xeeDL%2BPGk7%2FF0Qg0GdnwQ1cW5PDLUwFDID6uquO1Dlot1bJw9%2FPLRmia%2BzRMCYyk4dSiq6205QSDXOxfy3KAq5Pkvqt4DAAD%2F%2Fw%3D%3D", nil)
	_, err = NewIdpAuthnRequest(idp, r)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "method not allowed"))
}

func TestIDPCanValidate(t *testing.T) {
	test := NewIdentifyProviderTest(t)
	r := newRequest(t, http.MethodGet, "/sso/foo", nil)
	req := IdpAuthnRequest{
		Now: TimeNow(),
		RequestBuffer: []byte("" +
			"<AuthnRequest xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"  AssertionConsumerServiceURL=\"https://sp.example.com/saml2/acs\" " +
			"  Destination=\"https://idp.example.com/sso/foo\" " +
			"  ID=\"id-00020406080a0c0e10121416181a1c1e\" " +
			"  IssueInstant=\"2015-12-01T01:57:09Z\" ProtocolBinding=\"\" " +
			"  Version=\"2.0\">" +
			"  <Issuer xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\" " +
			"    Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://sp.example.com/saml2/metadata</Issuer>" +
			"  <NameIDPolicy xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"    AllowCreate=\"true\">urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDPolicy>" +
			"</AuthnRequest>"),
		HTTPRequest: r,
	}
	assert.Check(t, test.h.ValidateRequest(&req))
	assert.Check(t, req.ServiceProviderMetadata != nil)
	assert.Check(t, is.DeepEqual(&samlconfig.IndexedEndpoint{
		Binding: "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST", Location: "https://sp.example.com/saml2/acs",
		Index: 1,
	}, req.ACSEndpoint))

	req = IdpAuthnRequest{
		Now:           TimeNow(),
		RequestBuffer: []byte("<AuthnRequest"),
		HTTPRequest:   r,
	}
	err := test.h.ValidateRequest(&req)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "XML syntax error on line 1: unexpected EOF"))

	req = IdpAuthnRequest{
		Now: TimeNow(),
		RequestBuffer: []byte("" +
			"<AuthnRequest xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"  AssertionConsumerServiceURL=\"https://sp.example.com/saml2/acs\" " +
			"  Destination=\"https://idp.wrongDestination.com/saml/sso\" " +
			"  ID=\"id-00020406080a0c0e10121416181a1c1e\" " +
			"  IssueInstant=\"2015-12-01T01:57:09Z\" ProtocolBinding=\"\" " +
			"  Version=\"2.0\">" +
			"  <Issuer xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\" " +
			"    Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://sp.example.com/saml2/metadata</Issuer>" +
			"  <NameIDPolicy xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"    AllowCreate=\"true\">urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDPolicy>" +
			"</AuthnRequest>"),
		HTTPRequest: r,
	}
	err = test.h.ValidateRequest(&req)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "expected destination to be \"https://idp.example.com/saml/sso\", not \"https://idp.wrongDestination.com/saml/sso\""))

	req = IdpAuthnRequest{
		Now: TimeNow(),
		RequestBuffer: []byte("" +
			"<AuthnRequest xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"  AssertionConsumerServiceURL=\"https://sp.example.com/saml2/acs\" " +
			"  Destination=\"https://idp.example.com/saml/sso\" " +
			"  ID=\"id-00020406080a0c0e10121416181a1c1e\" " +
			"  IssueInstant=\"2014-12-01T01:57:09Z\" ProtocolBinding=\"\" " +
			"  Version=\"2.0\">" +
			"  <Issuer xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\" " +
			"    Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://sp.example.com/saml2/metadata</Issuer>" +
			"  <NameIDPolicy xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"    AllowCreate=\"true\">urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDPolicy>" +
			"</AuthnRequest>"),
		HTTPRequest: r,
	}
	err = test.h.ValidateRequest(&req)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "request expired at 2014-12-01 01:58:39 +0000 UTC"))

	req = IdpAuthnRequest{
		Now: TimeNow(),
		RequestBuffer: []byte("" +
			"<AuthnRequest xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"  AssertionConsumerServiceURL=\"https://sp.example.com/saml2/acs\" " +
			"  Destination=\"https://idp.example.com/saml/sso\" " +
			"  ID=\"id-00020406080a0c0e10121416181a1c1e\" " +
			"  IssueInstant=\"2015-12-01T01:57:09Z\" ProtocolBinding=\"\" " +
			"  Version=\"4.2\">" +
			"  <Issuer xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\" " +
			"    Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://sp.example.com/saml2/metadata</Issuer>" +
			"  <NameIDPolicy xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"    AllowCreate=\"true\">urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDPolicy>" +
			"</AuthnRequest>"),
		HTTPRequest: r,
	}
	err = test.h.ValidateRequest(&req)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "expected SAML request version 2.0 got 4.2"))

	req = IdpAuthnRequest{
		Now: TimeNow(),
		RequestBuffer: []byte("" +
			"<AuthnRequest xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"  AssertionConsumerServiceURL=\"https://sp.example.com/saml2/acs\" " +
			"  Destination=\"https://idp.example.com/saml/sso\" " +
			"  ID=\"id-00020406080a0c0e10121416181a1c1e\" " +
			"  IssueInstant=\"2015-12-01T01:57:09Z\" ProtocolBinding=\"\" " +
			"  Version=\"2.0\">" +
			"  <Issuer xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\" " +
			"    Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://unknownSP.example.com/saml2/metadata</Issuer>" +
			"  <NameIDPolicy xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"    AllowCreate=\"true\">urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDPolicy>" +
			"</AuthnRequest>"),
		HTTPRequest: r,
	}
	err = test.h.ValidateRequest(&req)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "cannot handle request from unknown service provider https://unknownSP.example.com/saml2/metadata"))

	req = IdpAuthnRequest{
		Now: TimeNow(),
		RequestBuffer: []byte("" +
			"<AuthnRequest xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"  AssertionConsumerServiceURL=\"https://unknown.example.com/saml2/acs\" " +
			"  Destination=\"https://idp.example.com/saml/sso\" " +
			"  ID=\"id-00020406080a0c0e10121416181a1c1e\" " +
			"  IssueInstant=\"2015-12-01T01:57:09Z\" ProtocolBinding=\"\" " +
			"  Version=\"2.0\">" +
			"  <Issuer xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\" " +
			"    Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://sp.example.com/saml2/metadata</Issuer>" +
			"  <NameIDPolicy xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"    AllowCreate=\"true\">urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDPolicy>" +
			"</AuthnRequest>"),
		HTTPRequest: r,
	}
	err = test.h.ValidateRequest(&req)
	assert.Check(t, err != nil, strings.Contains(err.Error(), "cannot find assertion consumer service: file does not exist"))

}

var defaultID = uuid.Must(uuid.NewV4())

func TestIDPMakeAssertion(t *testing.T) {
	t.Skip() // TODO
	test := NewIdentifyProviderTest(t)
	req := IdpAuthnRequest{
		Now: TimeNow(),
		RequestBuffer: []byte("" +
			"<AuthnRequest xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"  AssertionConsumerServiceURL=\"https://sp.example.com/saml2/acs\" " +
			"  Destination=\"https://idp.example.com/saml/sso\" " +
			"  ID=\"id-00020406080a0c0e10121416181a1c1e\" " +
			"  IssueInstant=\"2015-12-01T01:57:09Z\" ProtocolBinding=\"\" " +
			"  Version=\"2.0\">" +
			"  <Issuer xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\" " +
			"    Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://sp.example.com/saml2/metadata</Issuer>" +
			"  <NameIDPolicy xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"    AllowCreate=\"true\">urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDPolicy>" +
			"</AuthnRequest>"),
	}

	req.HTTPRequest = newRequest(t, "POST", "http://idp.example.com/sso/foo", nil)
	assert.Check(t, test.h.ValidateRequest(&req))
	err := test.h.MakeAssertion(&req, &storage.SAMLSession{
		BaseModel: ucdb.NewBaseWithID(defaultID),
		UserName:  "alice",
	})
	assert.NilError(t, err)

	expected := &saml.Assertion{
		ID:           "id-00020406080a0c0e10121416181a1c1e20222426",
		IssueInstant: TimeNow(),
		Version:      "2.0",
		Issuer: saml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  "https://idp.example.com/saml/metadata",
		},
		Signature: nil,
		Subject: &saml.Subject{
			NameID: &saml.NameID{Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient", NameQualifier: "https://idp.example.com/saml/metadata", SPNameQualifier: "https://sp.example.com/saml2/metadata", Value: ""},
			SubjectConfirmations: []saml.SubjectConfirmation{
				{
					Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
					SubjectConfirmationData: &saml.SubjectConfirmationData{
						Address:      "",
						InResponseTo: "id-00020406080a0c0e10121416181a1c1e",
						NotOnOrAfter: TimeNow().Add(MaxIssueDelay),
						Recipient:    "https://sp.example.com/saml2/acs",
					},
				},
			},
		},
		Conditions: &saml.Conditions{
			NotBefore:    TimeNow(),
			NotOnOrAfter: TimeNow().Add(MaxIssueDelay),
			AudienceRestrictions: []saml.AudienceRestriction{
				{
					Audience: saml.Audience{Value: "https://sp.example.com/saml2/metadata"},
				},
			},
		},
		AuthnStatements: []saml.AuthnStatement{
			{
				AuthnInstant:    time.Time{},
				SessionIndex:    "",
				SubjectLocality: &saml.SubjectLocality{},
				AuthnContext: saml.AuthnContext{
					AuthnContextClassRef: &saml.AuthnContextClassRef{Value: "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport"},
				},
			},
		},
		AttributeStatements: []saml.AttributeStatement{
			{
				Attributes: []saml.Attribute{
					{
						FriendlyName: "uid",
						Name:         "urn:oid:0.9.2342.19200300.100.1.1",
						NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
						Values: []saml.AttributeValue{
							{
								Type:  "xs:string",
								Value: "alice",
							},
						},
					},
				},
			},
		},
	}
	assert.Check(t, is.DeepEqual(expected, req.Assertion))

	err = test.h.MakeAssertion(&req, &storage.SAMLSession{
		BaseModel:      ucdb.NewBaseWithID(defaultID),
		ExpireTime:     TimeNow().Add(time.Hour),
		Index:          "9999",
		NameID:         "ba5eba11",
		Groups:         []string{"Users", "Administrators", "♀"},
		UserName:       "alice",
		UserEmail:      "alice@example.com",
		UserCommonName: "Alice Smith",
		UserSurname:    "Smith",
		UserGivenName:  "Alice",
	})
	assert.NilError(t, err)

	expectedAttributes :=
		[]saml.Attribute{
			{
				FriendlyName: "uid",
				Name:         "urn:oid:0.9.2342.19200300.100.1.1",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "alice",
					},
				},
			},
			{
				FriendlyName: "eduPersonPrincipalName",
				Name:         "urn:oid:1.3.6.1.4.1.5923.1.1.1.6",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "alice@example.com",
					},
				},
			},
			{
				FriendlyName: "sn",
				Name:         "urn:oid:2.5.4.4",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Smith",
					},
				},
			},
			{
				FriendlyName: "givenName",
				Name:         "urn:oid:2.5.4.42",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Alice",
					},
				},
			},
			{
				FriendlyName: "cn",
				Name:         "urn:oid:2.5.4.3",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Alice Smith",
					},
				},
			},
			{
				FriendlyName: "eduPersonAffiliation",
				Name:         "urn:oid:1.3.6.1.4.1.5923.1.1.1.1",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Users",
					},
					{
						Type:  "xs:string",
						Value: "Administrators",
					},
					{
						Type:  "xs:string",
						Value: "♀",
					},
				},
			},
		}

	assert.Check(t, is.DeepEqual(expectedAttributes,
		req.Assertion.AttributeStatements[0].Attributes))
}

func TestIDPMarshalAssertion(t *testing.T) {
	t.Skip() // TODO
	test := NewIdentifyProviderTest(t)
	req := IdpAuthnRequest{
		Now: TimeNow(),
		RequestBuffer: []byte("" +
			"<AuthnRequest xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"  AssertionConsumerServiceURL=\"https://sp.example.com/saml2/acs\" " +
			"  Destination=\"https://idp.example.com/saml/sso\" " +
			"  ID=\"id-00020406080a0c0e10121416181a1c1e\" " +
			"  IssueInstant=\"2015-12-01T01:57:09Z\" ProtocolBinding=\"\" " +
			"  Version=\"2.0\">" +
			"  <Issuer xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\" " +
			"    Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://sp.example.com/saml2/metadata</Issuer>" +
			"  <NameIDPolicy xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\" " +
			"    AllowCreate=\"true\">urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDPolicy>" +
			"</AuthnRequest>"),
	}
	req.HTTPRequest = newRequest(t, "POST", "http://idp.example.com/sso/foo", nil)
	err := test.h.ValidateRequest(&req)
	assert.Check(t, err)
	err = test.h.MakeAssertion(&req, &storage.SAMLSession{
		BaseModel: ucdb.NewBaseWithID(defaultID),
		UserName:  "alice",
	})
	assert.Check(t, err)
	err = test.h.MakeAssertionEl(&req)
	assert.Check(t, err)

	// Compare the plaintext first
	expectedPlaintext := "<saml:Assertion xmlns:saml=\"urn:oasis:names:tc:SAML:2.0:assertion\" ID=\"id-00020406080a0c0e10121416181a1c1e20222426\" IssueInstant=\"2015-12-01T01:57:09Z\" Version=\"2.0\"><saml:Issuer Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:entity\">https://idp.example.com/saml/metadata</saml:Issuer><ds:Signature xmlns:ds=\"http://www.w3.org/2000/09/xmldsig#\"><ds:SignedInfo><ds:CanonicalizationMethod Algorithm=\"http://www.w3.org/2001/10/xml-exc-c14n#\"/><ds:SignatureMethod Algorithm=\"http://www.w3.org/2000/09/xmldsig#rsa-sha1\"/><ds:Reference URI=\"#id-00020406080a0c0e10121416181a1c1e20222426\"><ds:Transforms><ds:Transform Algorithm=\"http://www.w3.org/2000/09/xmldsig#enveloped-signature\"/><ds:Transform Algorithm=\"http://www.w3.org/2001/10/xml-exc-c14n#\"/></ds:Transforms><ds:DigestMethod Algorithm=\"http://www.w3.org/2000/09/xmldsig#sha1\"/><ds:DigestValue>gjE0eLUMVt+kK0rIGYvnzHV/2Ok=</ds:DigestValue></ds:Reference></ds:SignedInfo><ds:SignatureValue>Jm1rrxo2x7SYTnaS97bCdnVLQGeQuCMTjiSUvwzBkWFR+xcPr+n38dXmv0q0R68tO7L2ELhLtBdLm/dWsxruN23TMGVQyHIPMgJExdnYb7fwqx6es/NAdbDUBTbSdMX0vhIlTsHu5F0bJ0Tg0iAo9uRk9VeBdkaxtPa7+4yl1PQ=</ds:SignatureValue><ds:KeyInfo><ds:X509Data><ds:X509Certificate>MIIB7zCCAVgCCQDFzbKIp7b3MTANBgkqhkiG9w0BAQUFADA8MQswCQYDVQQGEwJVUzELMAkGA1UECAwCR0ExDDAKBgNVBAoMA2ZvbzESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTEzMTAwMjAwMDg1MVoXDTE0MTAwMjAwMDg1MVowPDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkdBMQwwCgYDVQQKDANmb28xEjAQBgNVBAMMCWxvY2FsaG9zdDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEA1PMHYmhZj308kWLhZVT4vOulqx/9ibm5B86fPWwUKKQ2i12MYtz07tzukPymisTDhQaqyJ8Kqb/6JjhmeMnEOdTvSPmHO8m1ZVveJU6NoKRn/mP/BD7FW52WhbrUXLSeHVSKfWkNk6S4hk9MV9TswTvyRIKvRsw0X/gfnqkroJcCAwEAATANBgkqhkiG9w0BAQUFAAOBgQCMMlIO+GNcGekevKgkakpMdAqJfs24maGb90DvTLbRZRD7Xvn1MnVBBS9hzlXiFLYOInXACMW5gcoRFfeTQLSouMM8o57h0uKjfTmuoWHLQLi6hnF+cvCsEFiJZ4AbF+DgmO6TarJ8O05t8zvnOwJlNCASPZRH/JmF8tX0hoHuAQ==</ds:X509Certificate></ds:X509Data></ds:KeyInfo></ds:Signature><saml:Subject><saml:NameID Format=\"urn:oasis:names:tc:SAML:2.0:nameid-format:transient\" NameQualifier=\"https://idp.example.com/saml/metadata\" SPNameQualifier=\"https://sp.example.com/saml2/metadata\"/><saml:SubjectConfirmation Method=\"urn:oasis:names:tc:SAML:2.0:cm:bearer\"><saml:SubjectConfirmationData InResponseTo=\"id-00020406080a0c0e10121416181a1c1e\" NotOnOrAfter=\"2015-12-01T01:58:39Z\" Recipient=\"https://sp.example.com/saml2/acs\"/></saml:SubjectConfirmation></saml:Subject><saml:Conditions NotBefore=\"2015-12-01T01:57:09Z\" NotOnOrAfter=\"2015-12-01T01:58:39Z\"><saml:AudienceRestriction><saml:Audience>https://sp.example.com/saml2/metadata</saml:Audience></saml:AudienceRestriction></saml:Conditions><saml:AuthnStatement AuthnInstant=\"0001-01-01T00:00:00Z\"><saml:SubjectLocality/><saml:AuthnContext><saml:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport</saml:AuthnContextClassRef></saml:AuthnContext></saml:AuthnStatement><saml:AttributeStatement><saml:Attribute FriendlyName=\"uid\" Name=\"urn:oid:0.9.2342.19200300.100.1.1\" NameFormat=\"urn:oasis:names:tc:SAML:2.0:attrname-format:uri\"><saml:AttributeValue xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:type=\"xs:string\">alice</saml:AttributeValue></saml:Attribute></saml:AttributeStatement></saml:Assertion>"
	actualPlaintext := ""
	{
		doc := etree.NewDocument()
		doc.SetRoot(req.AssertionEl)
		el := doc.FindElement("//EncryptedAssertion/EncryptedData")
		actualPlaintextBuf, err := xmlenc.Decrypt(test.SPKey, el)
		assert.Check(t, err)
		actualPlaintext = string(actualPlaintextBuf)
	}
	assert.Check(t, is.Equal(expectedPlaintext, actualPlaintext))

	doc := etree.NewDocument()
	doc.SetRoot(req.AssertionEl)
	assertionBuffer, err := doc.WriteToBytes()
	assert.Check(t, err)
	golden.Assert(t, string(assertionBuffer), t.Name()+"_encrypted_assertion")
}

func TestIDPMakeResponse(t *testing.T) {
	test := NewIdentifyProviderTest(t)
	req := IdpAuthnRequest{
		Now:           TimeNow(),
		RequestBuffer: golden.Get(t, "TestIDPMakeResponse_request_buffer"),
	}

	req.HTTPRequest = newRequest(t, "POST", "http://idp.example.com/sso/foo", nil)
	assert.Check(t, test.h.ValidateRequest(&req))
	err := test.h.MakeAssertion(&req, &storage.SAMLSession{
		BaseModel: ucdb.NewBaseWithID(defaultID),
		UserName:  "alice",
	})
	assert.Check(t, err)
	assert.Check(t, test.h.MakeAssertionEl(&req))

	req.AssertionEl = etree.NewElement("this-is-an-encrypted-assertion")
	err = test.h.MakeResponse(&req)
	assert.Check(t, err)

	response := saml.Response{}
	err = unmarshalEtreeHack(req.ResponseEl, &response)
	assert.Check(t, err)

	doc := etree.NewDocument()
	doc.SetRoot(req.ResponseEl)
	doc.Indent(2)
	responseStr, err := doc.WriteToString()
	assert.Check(t, err)
	golden.Assert(t, responseStr, t.Name()+"_response.xml")
}

func TestIDPWriteResponse(t *testing.T) {
	test := NewIdentifyProviderTest(t)
	req := IdpAuthnRequest{
		Now:           TimeNow(),
		RelayState:    "THIS_IS_THE_RELAY_STATE",
		RequestBuffer: golden.Get(t, "TestIDPWriteResponse_RequestBuffer.xml"),
		ResponseEl:    etree.NewElement("THIS_IS_THE_SAML_RESPONSE"),
	}

	req.HTTPRequest = newRequest(t, "POST", "http://idp.example.com/sso/foo", nil)
	assert.Check(t, test.h.ValidateRequest(&req))

	w := httptest.NewRecorder()
	assert.Check(t, test.h.WriteResponse(w, &req))
	assert.Check(t, is.Equal(200, w.Code))
	golden.Assert(t, w.Body.String(), t.Name()+"response.html")
}

func TestIDPCanHandleUnencryptedResponse(t *testing.T) {
	t.Skip() // TODO
	test := NewIdentifyProviderTest(t)

	metadata := samlconfig.EntityDescriptor{}
	err := xml.Unmarshal(
		golden.Get(t, "TestIDPCanHandleUnencryptedResponse_idp_metadata.xml"),
		&metadata)
	assert.Check(t, err)
	tsp = append(tsp, metadata)

	req := IdpAuthnRequest{
		Now:           TimeNow(),
		RequestBuffer: golden.Get(t, "TestIDPCanHandleUnencryptedResponse_request"),
	}
	req.HTTPRequest = newRequest(t, "POST", "http://idp.example.com/sso/foo", nil)
	err = test.h.ValidateRequest(&req)
	assert.Check(t, err)
	err = test.h.MakeAssertion(&req, &storage.SAMLSession{
		BaseModel: ucdb.NewBaseWithID(defaultID),
		UserName:  "alice",
	})
	assert.Check(t, err)
	err = test.h.MakeAssertionEl(&req)
	assert.Check(t, err)

	err = test.h.MakeResponse(&req)
	assert.Check(t, err)

	doc := etree.NewDocument()
	doc.SetRoot(req.ResponseEl)
	doc.Indent(2)
	responseStr, err := doc.WriteToString()
	assert.NilError(t, err)
	golden.Assert(t, responseStr, t.Name()+"_response")
}

func TestIDPRequestedAttributes(t *testing.T) {
	t.Skip() // TODO
	test := NewIdentifyProviderTest(t)
	metadata := samlconfig.EntityDescriptor{}
	err := xml.Unmarshal(golden.Get(t, "TestIDPRequestedAttributes_idp_metadata.xml"), &metadata)
	assert.Check(t, err)
	tsp = append(tsp, metadata)

	requestURL, err := test.SP.MakeRedirectAuthenticationRequest("ThisIsTheRelayState")
	assert.Check(t, err)

	r := newRequest(t, "GET", requestURL.String(), nil)
	req, err := NewIdpAuthnRequest(&test.IDP, r)
	assert.NilError(t, err)
	req.ServiceProviderMetadata = &metadata
	req.ACSEndpoint = &metadata.SPSSODescriptors[0].AssertionConsumerServices[0]
	req.SPSSODescriptor = &metadata.SPSSODescriptors[0]
	assert.Check(t, err)
	err = test.h.MakeAssertion(req, &storage.SAMLSession{
		BaseModel:      ucdb.NewBaseWithID(defaultID),
		UserName:       "alice",
		UserEmail:      "alice@example.com",
		UserGivenName:  "Alice",
		UserSurname:    "Smith",
		UserCommonName: "Alice Smith",
	})
	assert.Check(t, err)
	assert.Check(t, req.Assertion != nil)

	expectedAttributes := []saml.AttributeStatement{{
		Attributes: []saml.Attribute{
			{
				FriendlyName: "Email address",
				Name:         "email",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "alice@example.com",
					},
				},
			},
			{
				FriendlyName: "Full name",
				Name:         "name",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Alice Smith",
					},
				},
			},
			{
				FriendlyName: "Given name",
				Name:         "first_name",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Alice",
					},
				},
			},
			{
				FriendlyName: "Family name",
				Name:         "last_name",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Smith",
					},
				},
			},
			{
				FriendlyName: "uid",
				Name:         "urn:oid:0.9.2342.19200300.100.1.1",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "alice",
					},
				},
			},
			{
				FriendlyName: "eduPersonPrincipalName",
				Name:         "urn:oid:1.3.6.1.4.1.5923.1.1.1.6",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "alice@example.com",
					},
				},
			},
			{
				FriendlyName: "sn",
				Name:         "urn:oid:2.5.4.4",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Smith",
					},
				},
			},
			{
				FriendlyName: "givenName",
				Name:         "urn:oid:2.5.4.42",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Alice",
					},
				},
			},
			{
				FriendlyName: "cn",
				Name:         "urn:oid:2.5.4.3",
				NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
				Values: []saml.AttributeValue{
					{
						Type:  "xs:string",
						Value: "Alice Smith",
					},
				},
			},
		}}}
	assert.Check(t, is.DeepEqual(expectedAttributes, req.Assertion.AttributeStatements))
}

func TestIDPNoDestination(t *testing.T) {
	t.Skip() // TODO
	test := NewIdentifyProviderTest(t)

	metadata := samlconfig.EntityDescriptor{}
	assert.Check(t, xml.Unmarshal(golden.Get(t, "TestIDPNoDestination_idp_metadata.xml"), &metadata))

	req := IdpAuthnRequest{
		Now:           TimeNow(),
		RequestBuffer: golden.Get(t, "TestIDPNoDestination_request"),
	}
	req.HTTPRequest = newRequest(t, "POST", "http://idp.example.com/sso/foo", nil)
	assert.Check(t, test.h.ValidateRequest(&req))
	err := test.h.MakeAssertion(&req, &storage.SAMLSession{
		BaseModel: ucdb.NewBaseWithID(defaultID),
		UserName:  "alice",
	})
	assert.Check(t, err)
	assert.Check(t, test.h.MakeAssertionEl(&req))
	assert.Check(t, test.h.MakeResponse(&req))
}

func TestIDPRejectDecompressionBomb(t *testing.T) {
	t.Skip() // TODO
	test := NewIdentifyProviderTest(t)

	data := bytes.Repeat([]byte("a"), 768*1024*1024)
	var compressed bytes.Buffer
	w, err := flate.NewWriter(&compressed, flate.BestCompression)
	assert.NilError(t, err)
	_, err = w.Write(data)
	assert.Check(t, err)
	err = w.Close()
	assert.Check(t, err)
	encoded := base64.StdEncoding.EncodeToString(compressed.Bytes())

	r := newRequest(t, "GET", "/dontcare?"+url.Values{"SAMLRequest": {encoded}}.Encode(), nil)
	_, err = NewIdpAuthnRequest(&test.IDP, r)
	assert.Error(t, err, "cannot decompress request: flate: uncompress limit exceeded (10485760 bytes)")
}
