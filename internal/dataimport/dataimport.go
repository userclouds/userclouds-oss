package dataimport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/worker"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantmap"
)

// ImportDataFromFile imports data from a file
func ImportDataFromFile(ctx context.Context, filePath string, ts *tenantmap.TenantState) error {
	f, err := os.Open(filePath)
	if err != nil {
		return ucerr.Wrap(err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(dataImportHelper(ctx, f, fi.Size(), ts, func(record string, err error) {
		uclog.Warningf(ctx, "bad record: %s, %s", record, err)
	}, func(processedRecordCount int, processedSize int64, totalSize int64) {
		uclog.Infof(ctx, "processed %d of %d bytes, %d records", processedSize, totalSize, processedRecordCount)
	}))
}

// ImportDataFromS3Bucket imports data from an S3 bucket and deletes the object if the import is successful
func ImportDataFromS3Bucket(ctx context.Context,
	s3Service *s3.Client,
	bucketName,
	objectKey string,
	ts *tenantmap.TenantState,
	badRecordCB BadRecordCallback,
	statusUpdateCB StatusUpdateCallback) error {
	object, err := s3Service.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucketName, Key: &objectKey})
	if err != nil {
		return ucerr.Wrap(err)
	}

	err = dataImportHelper(ctx, object.Body, *object.ContentLength, ts, badRecordCB, statusUpdateCB)
	object.Body.Close()
	if err != nil {
		return ucerr.Wrap(err)
	}

	if _, err := s3Service.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
	}); err != nil {
		return ucerr.Wrap(err)
	}

	return nil

}

// BadRecordCallback is a callback for bad records
type BadRecordCallback func(record string, err error)

// StatusUpdateCallback is a callback for status updates
type StatusUpdateCallback func(processedRecordCount int, processedSize int64, totalSize int64)

const headerSize = 37
const maxBadEntriesBeforeFailure = 100
const statusUpdateInterval = 100

func dataImportHelper(ctx context.Context,
	r io.ReadCloser,
	fileSize int64,
	ts *tenantmap.TenantState,
	badRecordCB BadRecordCallback,
	statusUpdateCB StatusUpdateCallback) error {

	header := make([]byte, headerSize)
	n, err := r.Read(header)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if n != headerSize {
		return ucerr.Friendlyf(nil, "file is not in the expected format")
	}

	mutatorID, err := uuid.FromString(strings.TrimSpace(string(header)))
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to parse tenant ID")
	}

	scanByteCounter := scanByteCounter{BytesRead: headerSize}
	jsonScanner := bufio.NewScanner(r)
	jsonScanner.Split(scanByteCounter.scanRowsDelimitedByChr2)
	if !jsonScanner.Scan() {
		return ucerr.Friendlyf(nil, "file is not in the expected format")
	}
	var clientContext policy.ClientContext
	if err := json.Unmarshal([]byte(jsonScanner.Text()), &clientContext); err != nil {
		return ucerr.Friendlyf(err, "Failed to parse access policy context")
	}

	badRecordCount := 0
	totalRecords := 0
	for jsonScanner.Scan() {
		if badRecordCount >= maxBadEntriesBeforeFailure {
			statusUpdateCB(totalRecords, scanByteCounter.BytesRead, fileSize)
			return ucerr.Friendlyf(nil, "too many bad records (%d) max bad records allowed: %d", badRecordCount, maxBadEntriesBeforeFailure)
		}

		line := jsonScanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		totalRecords++
		if totalRecords%statusUpdateInterval == 0 {
			statusUpdateCB(totalRecords, scanByteCounter.BytesRead, fileSize)
		}

		parts := strings.Split(line, "\x01")
		if len(parts) != 2 {
			badRecordCB(line, ucerr.Friendlyf(nil, "record is not in the expected format"))
			badRecordCount++
			continue
		}
		var selectorValues userstore.UserSelectorValues
		if err := json.Unmarshal([]byte(strings.TrimSpace(parts[0])), &selectorValues); err != nil {
			badRecordCB(line, err)
			badRecordCount++
			continue
		}
		var rowData map[string]idp.ValueAndPurposes
		if err := json.Unmarshal([]byte(strings.TrimSpace(parts[1])), &rowData); err != nil {
			badRecordCB(line, err)
			badRecordCount++
			continue
		}

		if _, _, err := worker.ExecuteMutator(
			ctx,
			idp.ExecuteMutatorRequest{
				MutatorID:      mutatorID,
				Context:        clientContext,
				SelectorValues: selectorValues,
				RowData:        rowData,
			},
			ts,
			nil,
		); err != nil {
			badRecordCB(line, err)
			badRecordCount++
			continue
		}
	}

	statusUpdateCB(totalRecords, scanByteCounter.BytesRead, fileSize)
	return nil
}

type scanByteCounter struct {
	BytesRead int64
}

func (s *scanByteCounter) scanRowsDelimitedByChr2(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\x02'); i >= 0 {
		// We have a full newline-terminated line.
		s.BytesRead += int64(i) + 1
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		s.BytesRead += int64(len(data))
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
