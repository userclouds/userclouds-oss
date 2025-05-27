package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/console/internal/api"
	"userclouds.com/console/testhelpers"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucerr"
	paramtype "userclouds.com/internal/pageparameters/parametertype"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/ucimage"
)

func uploadLogoURL(tenantID uuid.UUID, appID uuid.UUID) string {
	// we skip actually uploading to S3, since we don't want to do that as part of unit testing
	return fmt.Sprintf("/api/tenants/%v/uploadlogo?app_id=%v&skip_upload=true", tenantID, appID)
}

func uploadLogoURLFromTenantConfig(tenantID uuid.UUID, tc tenantplex.TenantConfig) string {
	return uploadLogoURL(tenantID, tc.PlexMap.Apps[0].ID)
}

type imageSender struct {
	t      *testing.T
	tf     *testhelpers.TestFixture
	cookie *http.Cookie
}

func newImageSender(t *testing.T) *imageSender {
	t.Helper()

	tf := testhelpers.NewTestFixture(t)
	assert.NotNil(t, tf, assert.Must())
	_, cookie, _ := tf.MakeUCAdmin(context.Background())
	return &imageSender{
		t:      t,
		tf:     tf,
		cookie: cookie,
	}
}

func (is *imageSender) loadTestImage(imageKey string, imageFileName string) (body *bytes.Buffer, contentType string) {
	is.t.Helper()

	body = &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile(imageKey, imageFileName)
	assert.NoErr(is.t, err)

	file, err := os.Open(imageFileName)
	assert.NoErr(is.t, err)

	_, err = io.Copy(fw, file)
	assert.NoErr(is.t, err)

	contentType = writer.FormDataContentType()
	err = writer.Close()
	assert.NoErr(is.t, err)

	return body, contentType
}

func (is *imageSender) sendImageUploadRequest(logoUploadURL string, imageKey string, imageFileName string, uir *api.UploadImageResponse) error {
	is.t.Helper()

	requestURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(is.tf.ConsoleServerURL, "/"), strings.TrimPrefix(logoUploadURL, "/"))
	body, contentType := is.loadTestImage(imageKey, imageFileName)
	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(body.Bytes()))
	assert.NoErr(is.t, err)
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(is.cookie)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return ucerr.Wrap(err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return ucerr.Wrap(ucerr.Errorf("status code: %v", response.StatusCode))
	}

	if err := json.NewDecoder(response.Body).Decode(uir); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func TestUploadLogoRequests(t *testing.T) {
	t.Parallel()
	is := newImageSender(t)

	tc := loadTenantConfig(t, is.tf)

	t.Run("TestGoodUploadLogoRequests", func(t *testing.T) {
		var uir api.UploadImageResponse

		// gif
		url := uploadLogoURLFromTenantConfig(is.tf.ConsoleTenantID, tc)
		assert.IsNil(t, is.sendImageUploadRequest(url, "image", "internal/ucimage/test/logo.200x80.gif", &uir), assert.Must())
		assert.Equal(t, uir.TenantID, is.tf.ConsoleTenantID)
		assert.Equal(t, uir.AppID, tc.PlexMap.Apps[0].ID)
		assert.True(t, paramtype.ImageURL.IsValid(uir.ImageURL))
		assert.Equal(t, uir.ImageMetaData.Format, ucimage.GIF)
		assert.Equal(t, uir.ImageMetaData.Height, 80)
		assert.Equal(t, uir.ImageMetaData.Size, int64(2540))
		assert.Equal(t, uir.ImageMetaData.Type, ucimage.Logo)
		assert.Equal(t, uir.ImageMetaData.Width, 200)

		// jpeg
		url = uploadLogoURLFromTenantConfig(is.tf.ConsoleTenantID, tc)
		assert.IsNil(t, is.sendImageUploadRequest(url, "image", "internal/ucimage/test/logo.200x80.jpeg", &uir), assert.Must())
		assert.Equal(t, uir.TenantID, is.tf.ConsoleTenantID)
		assert.Equal(t, uir.AppID, tc.PlexMap.Apps[0].ID)
		assert.True(t, paramtype.ImageURL.IsValid(uir.ImageURL))
		assert.Equal(t, uir.ImageMetaData.Format, ucimage.JPEG)
		assert.Equal(t, uir.ImageMetaData.Height, 80)
		assert.Equal(t, uir.ImageMetaData.Size, int64(5441))
		assert.Equal(t, uir.ImageMetaData.Type, ucimage.Logo)
		assert.Equal(t, uir.ImageMetaData.Width, 200)

		// png
		url = uploadLogoURLFromTenantConfig(is.tf.ConsoleTenantID, tc)
		assert.IsNil(t, is.sendImageUploadRequest(url, "image", "internal/ucimage/test/logo.200x80.png", &uir), assert.Must())
		assert.Equal(t, uir.TenantID, is.tf.ConsoleTenantID)
		assert.Equal(t, uir.AppID, tc.PlexMap.Apps[0].ID)
		assert.True(t, paramtype.ImageURL.IsValid(uir.ImageURL))
		assert.Equal(t, uir.ImageMetaData.Format, ucimage.PNG)
		assert.Equal(t, uir.ImageMetaData.Height, 80)
		assert.Equal(t, uir.ImageMetaData.Size, int64(16874))
		assert.Equal(t, uir.ImageMetaData.Type, ucimage.Logo)
		assert.Equal(t, uir.ImageMetaData.Width, 200)

	})

	t.Run("TestBadUploadLogoRequests", func(t *testing.T) {
		var uir api.UploadImageResponse

		// bad tenant id
		url := uploadLogoURL(uuid.Nil, uuid.Nil)
		assert.NotNil(t, is.sendImageUploadRequest(url, "image", "internal/ucimage/test/logo.200x80.gif", &uir))

		// missing app id
		url = fmt.Sprintf("/api/tenants/%v/uploadlogo?skip_upload=true", is.tf.ConsoleTenantID)
		assert.NotNil(t, is.sendImageUploadRequest(url, "image", "internal/ucimage/test/logo.200x80.gif", &uir))

		// bad app id
		url = uploadLogoURL(is.tf.ConsoleTenantID, uuid.Nil)
		assert.NotNil(t, is.sendImageUploadRequest(url, "image", "internal/ucimage/test/logo.200x80.gif", &uir))

		// missing image
		url = uploadLogoURLFromTenantConfig(is.tf.ConsoleTenantID, tc)
		assert.NotNil(t, is.sendImageUploadRequest(url, "bad key", "internal/ucimage/test/logo.200x80.gif", &uir))

		// bad image format
		url = uploadLogoURLFromTenantConfig(is.tf.ConsoleTenantID, tc)
		assert.NotNil(t, is.sendImageUploadRequest(url, "image", "internal/ucimage/test/logo.svg", &uir))

		// bad image dimensions - too small
		url = uploadLogoURLFromTenantConfig(is.tf.ConsoleTenantID, tc)
		assert.NotNil(t, is.sendImageUploadRequest(url, "image", "internal/ucimage/test/logo.30x30.jpeg", &uir))

		// bad image dimensions - too large
		url = uploadLogoURLFromTenantConfig(is.tf.ConsoleTenantID, tc)
		assert.NotNil(t, is.sendImageUploadRequest(url, "image", "internal/ucimage/test/logo.300x200.jpeg", &uir))
	})
}
