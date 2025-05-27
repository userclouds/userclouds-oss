package api

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"  // required for image.DecodeConfig()
	_ "image/jpeg" // required for image.DecodeConfig()
	_ "image/png"  // required for image.DecodeConfig()
	"io"
	"mime/multipart"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/ucimage"
)

const imageUploadAppIDKey = "app_id"
const imageUploadImageKey = "image"
const imageUploadMaxSize = 2 << 20
const imageUploadSkipUploadKey = "skip_upload"

// ImageMetaData represents metadata about an uploaded image
type ImageMetaData struct {
	Format ucimage.ImageFormat `json:"format"`
	Height int                 `json:"height"`
	Size   int64               `json:"size"`
	Type   ucimage.ImageType   `json:"type"`
	Width  int                 `json:"width"`
}

// Validate implements the Validatable interface
func (imd *ImageMetaData) Validate() error {
	ic, found := imd.Type.GetImageConstraints()
	if !found {
		return ucerr.Friendlyf(nil, "unsupported image type: %v, supported types: %v", imd.Type, ucimage.GetLogoSupportedFormatsMessage())
	}
	if _, found := ic.SupportedFormats[imd.Format]; !found {
		return ucerr.Friendlyf(nil, "unsupported image format %v for image type: %v", imd.Format, imd.Type)
	}
	if imd.Size < ic.MinSize || imd.Size > ic.MaxSize {
		return ucerr.Friendlyf(nil, "Invalid image size: %d, size must be between %d and %d", imd.Size, ic.MinSize, ic.MaxSize)
	}
	if imd.Height < ic.MinHeight || imd.Height > ic.MaxHeight {
		return ucerr.Friendlyf(nil, "Invalid image height: %d, height must be between %d and %d pixels", imd.Height, ic.MinHeight, ic.MaxHeight)
	}
	if imd.Width < ic.MinWidth || imd.Width > ic.MaxWidth {
		return ucerr.Friendlyf(nil, "Invalid image width: %d, width must be between %d and %d pixels", imd.Width, ic.MinWidth, ic.MaxWidth)
	}
	return nil
}

func (imd ImageMetaData) generateS3Key(tenantID uuid.UUID, appID uuid.UUID, imageFileName string) string {
	return ucimage.GenerateS3Key(tenantID, appID, imd.Type, imageFileName, imd.Width, imd.Height, imd.Format)
}

// UploadImageResponse contains the URL of the succesfully uploaded image
type UploadImageResponse struct {
	TenantID      uuid.UUID     `json:"tenant_id"`
	AppID         uuid.UUID     `json:"app_id"`
	ImageURL      string        `json:"image_url"`
	ImageMetaData ImageMetaData `json:"image_meta_data"`
}

func getImageFile(r *http.Request) ([]byte, *multipart.FileHeader, error) {
	imageFile, imageFileHeader, err := r.FormFile(imageUploadImageKey)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	defer imageFile.Close()

	imageFileBuffer, err := io.ReadAll(imageFile)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	return imageFileBuffer, imageFileHeader, nil
}

func getValidS3Service(ctx context.Context, bucket string, skipUpload bool) (*s3.Client, string, error) {
	cfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return nil, "", ucerr.Wrap(err)
	}
	s3Service := s3.NewFromConfig(cfg)

	if !skipUpload {
		// make sure bucket is available
		if _, err := s3Service.GetBucketLocation(ctx,
			&s3.GetBucketLocationInput{Bucket: &bucket}); err != nil {
			return nil, "", ucerr.Errorf("could not find bucket '%s': %v", bucket, err)
		}
	}

	return s3Service, cfg.Region, nil
}

func (h *handler) uploadImage(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, imageType ucimage.ImageType) {
	ctx := r.Context()
	if h.consoleImageClient == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "upload image is not available in this environment"), jsonapi.Code(http.StatusServiceUnavailable))
		return
	}

	// extract and validate request parameters

	queryParameters := r.URL.Query()
	appID, err := uuid.FromString(queryParameters.Get(imageUploadAppIDKey))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "AppIDInvalid", jsonapi.Code(http.StatusBadRequest))
		return
	}
	// TODO: factor out the storage provider so we can use dependency injection
	// to select the appropriate provider for our service (e.g., prod vs. test)
	skipUpload := (queryParameters.Get(imageUploadSkipUploadKey) == "true")

	if err := r.ParseMultipartForm(imageUploadMaxSize); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "MultipartFormInvalid", jsonapi.Code(http.StatusBadRequest))
		return
	}

	imageFileBuffer, imageFileHeader, err := getImageFile(r)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "ImageFileInvalid", jsonapi.Code(http.StatusBadRequest))
		return
	}

	if len(imageFileHeader.Filename) == 0 {
		jsonapi.MarshalErrorL(ctx, w, err, "ImageFileNameInvalid", jsonapi.Code(http.StatusBadRequest))
		return
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(imageFileBuffer))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w,
			ucerr.Friendlyf(err, "Invalid image file, supported formats: %v", ucimage.GetLogoSupportedFormatsMessage()),
			"ImageFileCannotBeDecoded", jsonapi.Code(http.StatusBadRequest))
		return
	}
	imageMetaData := ImageMetaData{
		Format: ucimage.ImageFormat(format),
		Height: config.Height,
		Size:   imageFileHeader.Size,
		Type:   imageType,
		Width:  config.Width,
	}
	if err := imageMetaData.Validate(); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "ImageMetaDataInvalid", jsonapi.Code(http.StatusBadRequest))
		return
	}

	// verify tenant and app are valid and that user has proper permissions

	tam := newTenantAppManager(h, r)
	if err := tam.loadTenantApp(tenantID, appID); err != nil {
		switch {
		case errors.As(err, &tam.errors.badAppID):
			jsonapi.MarshalErrorL(ctx, w, err, "AppIDUnknown", jsonapi.Code(http.StatusNotFound))
		case errors.As(err, &tam.errors.badTenantID):
			jsonapi.MarshalErrorL(ctx, w, err, "TenantIDUnknown", jsonapi.Code(http.StatusNotFound))
		case errors.As(err, &tam.errors.forbidden):
			jsonapi.MarshalErrorL(ctx, w, err, "OperationForbidden", jsonapi.Code(http.StatusForbidden))
		case errors.As(err, &tam.errors.sql):
			jsonapi.MarshalErrorL(ctx, w, err, "CouldNotLoadTenant", jsonapi.Code(http.StatusInternalServerError))
		default:
			jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("unexpected error: '%w'", err), "UnexpectedLoadError", jsonapi.Code(http.StatusInternalServerError))
		}
		return
	}

	// connect to S3 service

	s3Bucket := h.consoleImageClient.S3Bucket()
	s3Service, region, err := getValidS3Service(ctx, s3Bucket, skipUpload)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "CouldNotFindS3Bucket", jsonapi.Code(http.StatusInternalServerError))
		return
	}

	// upload image

	imageS3Key := imageMetaData.generateS3Key(tenantID, appID, imageFileHeader.Filename)

	if !skipUpload {
		// since image files should be comfortably under 100MB, multipart upload is not necessary
		_, err = s3Service.PutObject(ctx, &s3.PutObjectInput{
			Body:   bytes.NewReader(imageFileBuffer),
			Bucket: &s3Bucket,
			Key:    &imageS3Key,
		})
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "CouldNotUploadToS3", jsonapi.Code(http.StatusInternalServerError))
			return
		}
	}

	// generate and return image URL

	jsonapi.Marshal(w, UploadImageResponse{
		TenantID:      tenantID,
		AppID:         appID,
		ImageURL:      h.consoleImageClient.GenerateURL(region, imageS3Key),
		ImageMetaData: imageMetaData})
}

func (h *handler) uploadLogo(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	h.uploadImage(w, r, tenantID, ucimage.Logo)
}
