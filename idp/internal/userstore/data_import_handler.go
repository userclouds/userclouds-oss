package userstore

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/multitenant"
	"userclouds.com/worker"
	"userclouds.com/worker/storage"
)

type dataImportInitializeResponse struct {
	PresignedURL string    `json:"presigned_url"`
	Expiration   time.Time `json:"expiration"`
	ImportID     uuid.UUID `json:"import_id"`
}

func (h *handler) dataImportInitialize(w http.ResponseWriter, r *http.Request) {

	if h.workerClient == nil || h.dataImportConfig == nil {
		jsonapi.MarshalError(r.Context(), w, ucerr.Friendlyf(nil, "data import not available in this environment"), jsonapi.Code(http.StatusServiceUnavailable))
		return
	}

	ctx := r.Context()
	ts := multitenant.MustGetTenantState(ctx)
	cfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusInternalServerError))
		return
	}

	s := storage.New(ts.TenantDB)
	job := &storage.IDPDataImportJob{
		BaseModel:         ucdb.NewBase(),
		ExpirationMinutes: h.dataImportConfig.PresignedURLExpirationMinutes,
		ImportType:        storage.ExecuteMutatorsImportType,
		Status:            storage.IDPDataImportJobStatusPending,
		S3Bucket:          h.dataImportConfig.DataImportS3Bucket,
	}
	job.ObjectKey = storage.GenerateDataImportPath(ts.ID, job.ImportType, job.ID)
	if err := s.SaveIDPDataImportJob(ctx, job); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusInternalServerError))
		return
	}

	s3Service := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(s3Service)
	bucketName := h.dataImportConfig.DataImportS3Bucket
	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &job.ObjectKey,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(h.dataImportConfig.PresignedURLExpirationMinutes) * time.Minute
	})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusInternalServerError))
		return
	}

	if err := h.workerClient.Send(ctx, worker.DataImportMessage(ts.ID, job.ID, false)); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, dataImportInitializeResponse{
		PresignedURL: request.URL,
		Expiration:   time.Now().UTC().Add(time.Duration(h.dataImportConfig.PresignedURLExpirationMinutes) * time.Minute),
		ImportID:     job.ID,
	})
}

type dataimportStatusResponse struct {
	Status  string `json:"status"`
	Details string `json:"details"`
}

func (h *handler) dataImportStatus(w http.ResponseWriter, r *http.Request, importID uuid.UUID) {
	ctx := r.Context()
	ts := multitenant.MustGetTenantState(ctx)
	s := storage.New(ts.TenantDB)
	job, err := s.GetIDPDataImportJob(ctx, importID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "import job not found"), jsonapi.Code(http.StatusNotFound))
			return
		}
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusInternalServerError))
		return
	}
	details := fmt.Sprintf("processed %d of %d bytes, %d records", job.ProcessedSize, job.FileSize, job.ProcessedRecordCount)
	if job.Error != "" {
		details += fmt.Sprintf("\nfailed: %s)", job.Error)
	}
	if len(job.FailedRecords) > 0 {
		details += fmt.Sprintf("failed entries:\n%s", strings.Join(job.FailedRecords, "\n"))
	}
	jsonapi.Marshal(w, dataimportStatusResponse{Status: string(job.Status), Details: details})
}
