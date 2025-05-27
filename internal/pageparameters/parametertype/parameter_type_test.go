package parametertype_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/crypto"
	paramtype "userclouds.com/internal/pageparameters/parametertype"
)

func TestTypeValidity(t *testing.T) {
	for _, pt := range paramtype.Types() {
		assert.True(t, pt.Validate() == nil)
	}
	var badType paramtype.Type = "bad type"
	assert.True(t, badType.Validate() != nil)
}

func TestAuthenticationMethods(t *testing.T) {
	assert.False(t, paramtype.AuthenticationMethods.IsValid("password,password"))
	assert.False(t, paramtype.AuthenticationMethods.IsValid("passwordless,passwordless"))
	assert.False(t, paramtype.AuthenticationMethods.IsValid("rocket,rocket"))
	assert.False(t, paramtype.AuthenticationMethods.IsValid("facebook,passwordless,facebook"))
	assert.False(t, paramtype.AuthenticationMethods.IsValid(","))
	assert.False(t, paramtype.AuthenticationMethods.IsValid("google , google"))
	assert.False(t, paramtype.AuthenticationMethods.IsValid("google,"))
	assert.False(t, paramtype.AuthenticationMethods.IsValid(",google"))

	assert.True(t, paramtype.AuthenticationMethods.IsValid(""))
	assert.True(t, paramtype.AuthenticationMethods.IsValid(" google "))
	assert.True(t, paramtype.AuthenticationMethods.IsValid("password"))
	assert.True(t, paramtype.AuthenticationMethods.IsValid("rocket"))
	assert.True(t, paramtype.AuthenticationMethods.IsValid("passwordless"))
	assert.True(t, paramtype.AuthenticationMethods.IsValid("google"))
	assert.True(t, paramtype.AuthenticationMethods.IsValid("facebook"))
	assert.True(t, paramtype.AuthenticationMethods.IsValid("password,facebook,google,passwordless"))
	assert.True(t, paramtype.AuthenticationMethods.IsValid("password,passwordless"))
	assert.True(t, paramtype.AuthenticationMethods.IsValid("google,facebook"))
}

func TestBool(t *testing.T) {
	assert.True(t, paramtype.Bool.IsValid("true"))
	assert.True(t, paramtype.Bool.IsValid("false"))
	assert.False(t, paramtype.Bool.IsValid("bad value"))
	assert.False(t, paramtype.Bool.IsValid(""))
}

func TestButtonText(t *testing.T) {
	assert.True(t, paramtype.ButtonText.IsValid("Login"))
	assert.False(t, paramtype.ButtonText.IsValid(""))
}

func TestCSSColor(t *testing.T) {
	assert.False(t, paramtype.CSSColor.IsValid(""))
	assert.False(t, paramtype.CSSColor.IsValid("123"))
	assert.False(t, paramtype.CSSColor.IsValid("123456"))
	assert.False(t, paramtype.CSSColor.IsValid("1234567"))
	assert.False(t, paramtype.CSSColor.IsValid("#1"))
	assert.False(t, paramtype.CSSColor.IsValid("#12"))
	assert.False(t, paramtype.CSSColor.IsValid("#1234"))
	assert.False(t, paramtype.CSSColor.IsValid("#12345"))
	assert.False(t, paramtype.CSSColor.IsValid("#1234567"))
	assert.False(t, paramtype.CSSColor.IsValid("#1 23"))
	assert.False(t, paramtype.CSSColor.IsValid("#12 3456"))
	assert.False(t, paramtype.CSSColor.IsValid("#HHH"))
	assert.False(t, paramtype.CSSColor.IsValid("#GGGGGG"))

	assert.True(t, paramtype.CSSColor.IsValid("#123"))
	assert.True(t, paramtype.CSSColor.IsValid("#ABC"))
	assert.True(t, paramtype.CSSColor.IsValid("#DEF"))
	assert.True(t, paramtype.CSSColor.IsValid("#def"))
	assert.True(t, paramtype.CSSColor.IsValid("#123456"))
	assert.True(t, paramtype.CSSColor.IsValid("#ABCDEF"))
	assert.True(t, paramtype.CSSColor.IsValid("#abcdef"))
	assert.True(t, paramtype.CSSColor.IsValid("#123456"))
	assert.True(t, paramtype.CSSColor.IsValid("#7890AB"))
	assert.True(t, paramtype.CSSColor.IsValid("transparent"))
}

func TestHeadingText(t *testing.T) {
	assert.False(t, paramtype.HeadingText.IsValid(""))
	assert.True(t, paramtype.HeadingText.IsValid("Reset Password"))
}

func TestHTMLSnippet(t *testing.T) {
	assert.False(t, paramtype.HTMLSnippet.IsValid("<b>hello"))
	assert.False(t, paramtype.HTMLSnippet.IsValid("<h1>Unsupported tags</h1><p>blah blah"))
	assert.False(t, paramtype.HTMLSnippet.IsValid("<script>alert('you have been pwned')</script>"))
	assert.False(t, paramtype.HTMLSnippet.IsValid("Text with bad link </a>"))
	assert.False(t, paramtype.HTMLSnippet.IsValid("Text with javascript link <a href=\"javascript:foo()\">link</a>"))
	assert.False(t, paramtype.HTMLSnippet.IsValid("Text with bad <a>link"))
	assert.False(t, paramtype.HTMLSnippet.IsValid("Text with bad <a href=\"gttp://badlink.com\">link</a>"))
	assert.False(t, paramtype.HTMLSnippet.IsValid("Text with bad <a href=\"http://nowhere.org\" referrerpolicy=\"origin\">link</a>"))
	assert.False(t, paramtype.HTMLSnippet.IsValid("Bad text <a>with two links <a href=\"http://nowhere.org/foo\">link 1</a> and <a href=\"http://nowhere.org/bar\">link 2</a>"))

	assert.True(t, paramtype.HTMLSnippet.IsValid(""))
	assert.True(t, paramtype.HTMLSnippet.IsValid("Text with no html elements"))
	assert.True(t, paramtype.HTMLSnippet.IsValid("Text with a link <a href=\"http://nowhere.org\" rel=\"external\" target=\"_self\" title=\"terms of service\">link</a>"))
	assert.True(t, paramtype.HTMLSnippet.IsValid("Text with two links <a href=\"http://nowhere.org/foo\">link 1</a> and <a href=\"http://nowhere.org/bar\">link 2</a>"))
}

func generateRandomImageURL(domainPrefix string, fileType string, fileName string, width string, height string, imageFormat string) string {
	return generateImageURL(
		domainPrefix,
		uuid.Must(uuid.NewV4()).String(),
		uuid.Must(uuid.NewV4()).String(),
		fileType,
		fileName,
		width,
		height,
		imageFormat,
		crypto.MustRandomHex(4))
}

func generateImageURL(domainPrefix string, tenantID string, appID string, fileType string, fileName string, width string, height string, imageFormat string, randomSuffix string) string {
	return fmt.Sprintf("https://%s.cloudfront.net/tenants/%s/apps/%s/%s/%s.%sx%s.%s.%s",
		domainPrefix,
		tenantID,
		appID,
		fileType,
		fileName,
		width,
		height,
		imageFormat,
		randomSuffix)
}

func TestImageURL(t *testing.T) {
	cfdp := "test_cloudfront_domain"
	assert.False(t, paramtype.ImageURL.IsValid("invalid"))
	assert.False(t, paramtype.ImageURL.IsValid(generateRandomImageURL("", "logo", "test_file_name", "120", "80", "gif")))
	assert.False(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "bad_type", "test_file_name", "120", "80", "gif")))
	assert.False(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "", "120", "80", "gif")))
	assert.False(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "*", "120", "80", "gif")))
	assert.False(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test_file_name", "12000", "80", "gif")))
	assert.False(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test_file_name", "bad_dimension", "80", "gif")))
	assert.False(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test_file_name", "120", "bad_dimension", "gif")))
	assert.False(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test_file_name", "120", "80000", "gif")))
	assert.False(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test_file_name", "120", "80", "bad_image_format")))
	assert.False(t, paramtype.ImageURL.IsValid(generateImageURL(cfdp, "bad_tenant_id", uuid.Must(uuid.NewV4()).String(), "logo", "test_file_name", "120", "80", "gif", crypto.MustRandomHex(4))))
	assert.False(t, paramtype.ImageURL.IsValid(generateImageURL(cfdp, uuid.Must(uuid.NewV4()).String(), "bad_app_id", "logo", "test_file_name", "120", "80", "gif", crypto.MustRandomHex(4))))
	assert.False(t, paramtype.ImageURL.IsValid(generateImageURL(cfdp, uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String(), "logo", "test_file_name", "120", "80", "gif", "bad_random_suffix")))

	assert.True(t, paramtype.ImageURL.IsValid(""))
	assert.True(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test file name", "120", "80", "gif")))
	assert.True(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test file name", "1", "80", "gif")))
	assert.True(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test file name", "12", "80", "gif")))
	assert.True(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test file name", "1200", "80", "gif")))
	assert.True(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test file name", "120", "8", "gif")))
	assert.True(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test file name", "120", "100", "gif")))
	assert.True(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test file name", "120", "1000", "gif")))
	assert.True(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test file name", "120", "80", "jpeg")))
	assert.True(t, paramtype.ImageURL.IsValid(generateRandomImageURL(cfdp, "logo", "test file name", "120", "80", "png")))
}

func TestLabelText(t *testing.T) {
	assert.False(t, paramtype.LabelText.IsValid(""))
	assert.True(t, paramtype.LabelText.IsValid("Password"))
}

func TestMFAMethods(t *testing.T) {
	assert.False(t, paramtype.MFAMethods.IsValid(","))
	assert.False(t, paramtype.MFAMethods.IsValid("foo"))
	assert.False(t, paramtype.MFAMethods.IsValid("email,email"))
	assert.False(t, paramtype.MFAMethods.IsValid("email,"))
	assert.False(t, paramtype.MFAMethods.IsValid(",email"))

	assert.True(t, paramtype.MFAMethods.IsValid(""))
	assert.True(t, paramtype.MFAMethods.IsValid("email"))
}

func TestOIDCAuthenticationSettings(t *testing.T) {
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo:"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo:foo:foo"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo:,bar:bar:bar:bar"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo:foo,bar:bar:bar"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo:foo,foo:foo:foo:foo"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo:foo,"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo:foo,,bar:bar:bar:bar"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid(",foo:foo:foo:foo"))
	assert.False(t, paramtype.OIDCAuthenticationSettings.IsValid(","))

	assert.True(t, paramtype.OIDCAuthenticationSettings.IsValid(""))
	assert.True(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo:foo"))
	assert.True(t, paramtype.OIDCAuthenticationSettings.IsValid("foo:foo:foo:foo,bar:bar:bar:bar"))
}

func TestSelectedAuthenticationMethods(t *testing.T) {
	assert.False(t, paramtype.SelectedAuthenticationMethods.IsValid(""))
	assert.False(t, paramtype.SelectedAuthenticationMethods.IsValid("password,password"))
	assert.False(t, paramtype.SelectedAuthenticationMethods.IsValid("passwordless,passwordless"))
	assert.False(t, paramtype.SelectedAuthenticationMethods.IsValid("rocket,rocket"))
	assert.False(t, paramtype.SelectedAuthenticationMethods.IsValid("facebook,passwordless,facebook"))
	assert.False(t, paramtype.SelectedAuthenticationMethods.IsValid(","))
	assert.False(t, paramtype.SelectedAuthenticationMethods.IsValid("google , google"))
	assert.False(t, paramtype.SelectedAuthenticationMethods.IsValid("google,"))
	assert.False(t, paramtype.SelectedAuthenticationMethods.IsValid(",google"))

	assert.True(t, paramtype.SelectedAuthenticationMethods.IsValid(" google "))
	assert.True(t, paramtype.SelectedAuthenticationMethods.IsValid("password"))
	assert.True(t, paramtype.SelectedAuthenticationMethods.IsValid("passwordless"))
	assert.True(t, paramtype.SelectedAuthenticationMethods.IsValid("rocket"))
	assert.True(t, paramtype.SelectedAuthenticationMethods.IsValid("google"))
	assert.True(t, paramtype.SelectedAuthenticationMethods.IsValid("facebook"))
	assert.True(t, paramtype.SelectedAuthenticationMethods.IsValid("password,facebook,google,passwordless"))
	assert.True(t, paramtype.SelectedAuthenticationMethods.IsValid("password,passwordless"))
	assert.True(t, paramtype.SelectedAuthenticationMethods.IsValid("google,facebook"))
}

func TestStatusText(t *testing.T) {
	assert.False(t, paramtype.StatusText.IsValid(""))
	assert.True(t, paramtype.StatusText.IsValid("Redirecting..."))
}

func TestSubheadingText(t *testing.T) {
	assert.True(t, paramtype.SubheadingText.IsValid(""))
	assert.True(t, paramtype.SubheadingText.IsValid("sample subheading"))
}

func TestText(t *testing.T) {
	assert.False(t, paramtype.Text.IsValid(""))
	assert.True(t, paramtype.Text.IsValid("sample text"))
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")
	os.Exit(m.Run())
}
