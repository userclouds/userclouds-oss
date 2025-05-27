package internal

import (
	"context"
	"encoding/json"
	"strings"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/worker/storage"
)

// S3Notification represents the structure of an S3 notification message. see https: //docs.aws.amazon.com/AmazonS3/latest/userguide/notification-content-structure.html
type S3Notification struct {
	Records []struct {
		EventVersion string `json:"eventVersion" yaml:"eventVersion"`
		EventSource  string `json:"eventSource" yaml:"eventSource"`
		AwsRegion    string `json:"awsRegion" yaml:"awsRegion"`
		EventTime    string `json:"eventTime" yaml:"eventTime"`
		EventName    string `json:"eventName" yaml:"eventName"`
		S3           struct {
			S3SchemaVersion string `json:"s3SchemaVersion" yaml:"s3SchemaVersion"`
			Bucket          struct {
				Name string `json:"name" yaml:"name"`
			} `json:"bucket" yaml:"bucket"`
			Object struct {
				Key  string `json:"key" yaml:"key"`
				Size int    `json:"size" yaml:"size"`
			} `json:"object" yaml:"object"`
		} `json:"s3" yaml:"s3"`
	} `json:"Records" yaml:"Records"`
}

func isS3Notification(jsonPayload map[string]any) bool {
	_, ok := jsonPayload["Records"]
	return ok
}

func isDataImportS3Notification(msgBody []byte) bool {
	var notification S3Notification
	if err := json.Unmarshal(msgBody, &notification); err != nil {
		return false
	}
	for _, record := range notification.Records {
		if strings.Contains(record.S3.Bucket.Name, "data-import") {
			return true
		}
	}
	return false
}

func (h *handler) getDataImportS3Notifications(ctx context.Context, msgBody []byte) ([]storage.DataImportInfo, error) {
	var notification S3Notification
	if err := json.Unmarshal(msgBody, &notification); err != nil {
		return nil, ucerr.Wrap(err)
	}
	dataImports := make([]storage.DataImportInfo, 0, len(notification.Records))
	uclog.Infof(ctx, "received S3 notification for %d records", len(notification.Records))
	for _, record := range notification.Records {
		if record.EventSource != "aws:s3" {
			uclog.Errorf(ctx, "unexpected event source: %s", record.EventSource)
			continue
		}
		if record.EventName != "ObjectCreated:Put" {
			uclog.Errorf(ctx, "unexpected event name: %s", record.EventName)
			continue
		}
		di, err := storage.ParseDataImportPath(record.S3.Object.Key)
		if err != nil {
			uclog.Errorf(ctx, "error parsing data import path: %v", err)
			continue
		}
		dataImports = append(dataImports, *di)
	}
	return dataImports, nil
}
