package ucimage

import (
	"fmt"
	"regexp"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

var imageTypeConstraints = map[ImageType]ImageConstraints{
	// TODO: finalize constraint values after testing
	Logo: NewImageConstraints(32, 80, 32, 200, 128, 2<<20, GIF, JPEG, PNG),
}

// GetLogoSupportedFormatsMessage returns a human readable messages regarding image constaints.
func GetLogoSupportedFormatsMessage() string {
	return imageTypeConstraints[Logo].SupportedFormatMessage
}

// GetImageConstraints returns the image constraints for an image type
func (it ImageType) GetImageConstraints() (ImageConstraints, bool) {
	ic, found := imageTypeConstraints[it]
	return ic, found
}

// Client is a configured instance of a ucimage client
type Client struct {
	config Config
}

// NewClient returns a new instance of a ucimage client
func NewClient(c Config) (*Client, error) {
	if err := c.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &Client{config: c}, nil
}

// GenerateURL generates a URL for retrieving the image referred to by the S3 bucket, region, and key
func (c *Client) GenerateURL(regionCode string, key string) string {
	return fmt.Sprintf("https://%s/%s", c.config.Host, key)
}

// S3Bucket returns the configured s3 bucket used for image storage
func (c *Client) S3Bucket() string {
	return c.config.S3Bucket
}

// GenerateS3Key generates an S3 key for the associated image information
func GenerateS3Key(tenantID uuid.UUID, appID uuid.UUID, t ImageType, imageFileName string, width int, height int, f ImageFormat) string {
	return fmt.Sprintf("tenants/%v/apps/%v/%v/%s.%dx%d.%v.%s", tenantID, appID, t, imageFileName, width, height, f, crypto.MustRandomHex(4))
}

// MustGenerateURLRegexp generates a regexp pattern string suitable for matching a valid image URL
func MustGenerateURLRegexp(it ImageType) *regexp.Regexp {
	// Image URLs are of the form :
	//

	var hostName string
	if universe.Current().IsDev() {
		// In dev, we don't use cloudfront, so we use the s3 bucket directly (see: config/consoles/dev.yaml image.host)
		hostName = `.+.s3..+.amazonaws\.com`
	} else {
		// https://<cloudfront domain>/tenants/<uuid>/apps/<uuid>/<image type>/<filename>.<width>x<height>.<image format>.<8 random hex chars>
		hostName = ".+[.]cloudfront[.]net"
	}
	dimensionRegexp := "[0-9]{1,4}"
	fileNameRegexp := "\\w.*"
	imageFormatRegexp := imageTypeConstraints[it].SupportedFormatRegex
	imageTypeRegexp := fmt.Sprintf("(%v)", it)
	randomCharsRegexp := "[A-Fa-f0-9]{8}"
	uuidRegexp := "[A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12}"

	return regexp.MustCompile(fmt.Sprintf("^https://%s/tenants/%s/apps/%s/%s/%s[.]%sx%s[.]%s[.]%s$",
		hostName,
		uuidRegexp,
		uuidRegexp,
		imageTypeRegexp,
		fileNameRegexp,
		dimensionRegexp,
		dimensionRegexp,
		imageFormatRegexp,
		randomCharsRegexp))
}
