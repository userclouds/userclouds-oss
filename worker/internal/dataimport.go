package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/dataimport"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/worker"
	"userclouds.com/worker/storage"
)

func dataImport(ctx context.Context, wc workerclient.Client, ts *tenantmap.TenantState, jobID uuid.UUID, objectReady bool) error {
	s := storage.New(ts.TenantDB)
	job, err := s.GetIDPDataImportJob(ctx, jobID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if job.Status != storage.IDPDataImportJobStatusPending {
		return ucerr.Errorf("unexpected status '%v' for data import job %v (expected: '%v')", job.Status, job.ID, storage.IDPDataImportJobStatusPending)
	}

	cfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// check S3 for the object
	s3Service := s3.NewFromConfig(cfg)
	if !objectReady {
		if _, err := s3Service.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &job.S3Bucket, Key: &job.ObjectKey}); err != nil {
			if time.Now().UTC().After(job.Created.Add(time.Duration(job.ExpirationMinutes) * time.Minute)) {
				s.SetDataImportJobStatus(ctx, job, storage.IDPDataImportJobStatusExpired)
				return ucerr.Errorf("data import job %v expired after %v minutes", job.ID, job.ExpirationMinutes)
			}
			// TODO: we can get rid of this re-try code after we integrate with the AWS S3 notification via SQS
			uclog.Infof(ctx, "data import job %v is pending - waiting for upload to complete", job.ID)
			time.Sleep(5 * time.Second)
			if err := wc.Send(ctx, worker.DataImportMessage(ts.ID, job.ID, false)); err != nil {
				return ucerr.Wrap(err)
			}
			return nil
		}
	}
	return ucerr.Wrap(importS3Object(ctx, s, s3Service, job, ts))
}

func importS3Object(ctx context.Context, s *storage.Storage, s3Service *s3.Client, job *storage.IDPDataImportJob, ts *tenantmap.TenantState) error {
	job.Status = storage.IDPDataImportJobStatusInProgress
	job.LastRunTime = time.Now().UTC()
	if err := s.SaveIDPDataImportJob(ctx, job); err != nil {
		return ucerr.Wrap(err)
	}

	if err := dataimport.ImportDataFromS3Bucket(ctx, s3Service, job.S3Bucket, job.ObjectKey, ts,
		func(record string, err error) {
			job.FailedRecordCount++
			job.FailedRecords = append(job.FailedRecords, fmt.Sprintf("record: %s, error: %s", record, err))
			s.SaveDataImportJobNoError(ctx, job)
		}, func(processedRecords int, processedSize int64, totalSize int64) {
			job.ProcessedRecordCount = processedRecords
			job.ProcessedSize = processedSize
			job.FileSize = totalSize
			s.SaveDataImportJobNoError(ctx, job)
		},
	); err != nil {
		// We expose the error message to the user (via /userstore/upload/dataimport/), so we want to have a friendly error that doesn't leak stack traces.
		job.Error = ucerr.UserFriendlyMessage(err)
		job.Status = storage.IDPDataImportJobStatusFailed
		if err := s.SaveIDPDataImportJob(ctx, job); err != nil {
			uclog.Errorf(ctx, "failed to save data import job %v: %v", job.ID, err)
		}
		return ucerr.Wrap(err)
	}

	job.Status = storage.IDPDataImportJobStatusCompleted
	if err := s.SaveIDPDataImportJob(ctx, job); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
