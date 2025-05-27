package internal_test

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/test/testlogtransport"
	"userclouds.com/worker"
	"userclouds.com/worker/internal/testhelpers"
)

const (
	s3NotificationRawJsonTemplate = `{"Records":[{"eventVersion":"2.1","eventSource":"{{.EventSource}}","awsRegion":"us-west-2","eventTime":"2024-04-01T17:16:32.850Z","eventName":"{{.EventName}}","userIdentity":{"principalId":"AWS:AROAUWTUBTZ5YQAHCLGWW:example@userclouds.com"},"requestParameters":{"sourceIPAddress":"0.0.1.1"},"responseElements":{"x-amz-request-id":"ARFJQ2WYC43TXZ7G","x-amz-id-2":"GZU6POg4mXrKXi51ryp073P101ouM64RzyYJlrb4NE/h4UcI9uuUl5AeO2MQa82NHwOKv06CmkZvnUwXlzdFFufEZg2BJRUm"},"s3":{"s3SchemaVersion":"1.0","configurationId":"tf-s3-queue-20240329203449319200000001","bucket":{"name":"userclouds-debug-data-import","ownerIdentity":{"principalId":"A1ZPGZHDX8YD2J"},"arn":"arn:aws:s3:::userclouds-debug-data-import"},"object":{"key":"{{.S3Key}}","size":29,"eTag":"174793a114aad8afb86c81e661a0fd6d","sequencer":"00660AEBF0B86FC5B2"}}}]}`
	s3TestEvent                   = `{"Service":"Amazon S3","Event":"s3:TestEvent","Time":"2024-04-25T13:32:43.073Z","Bucket":"userclouds-debug-data-import","RequestId":"1191QJ303456NWF5","HostId":"8eMfjFahYCIa/AIUVm7vFbs3IU+3K+JikvQg/56m/3jw17JOvpxVsuABrfZ7mocGKM4Doifx2fI="}`
)

func getS3NotificationForKey(t *testing.T, s3Key string) string {
	return getS3Notification(t, s3Key, "aws:s3", "ObjectCreated:Put")
}

func getS3Notification(t *testing.T, s3Key, eventSource, eventName string) string {
	var tpl bytes.Buffer
	tmpl := template.Must(template.New("s3Notification").Parse(s3NotificationRawJsonTemplate))
	data := struct {
		S3Key       string
		EventName   string
		EventSource string
	}{
		S3Key:       s3Key,
		EventName:   eventName,
		EventSource: eventSource,
	}
	assert.NoErr(t, tmpl.Execute(&tpl, data), assert.Errorf("error generating s3 notification from template"))
	return tpl.String()
}

func TestS3Notification(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	testClient := workerclient.NewTestClient()
	wh, _, _, _ := testhelpers.SetupWorkerForTest(ctx, t, testClient)
	rr := httptest.NewRecorder()
	s3Key := fmt.Sprintf("%v/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator/v1/faf35f1e-6797-449a-ac78-6feb243203dd", universe.Current())
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(getS3NotificationForKey(t, s3Key)))
	wh.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, http.StatusOK)
	queuedMsg := testClient.WaitForMessages(t, 1, 5*time.Second)[0]
	assert.Equal(t, queuedMsg.Task, worker.TaskDataImport)
	assert.NotNil(t, queuedMsg.DataImportParams)
	assert.Equal(t, queuedMsg.DataImportParams.JobID, uuid.FromStringOrNil("faf35f1e-6797-449a-ac78-6feb243203dd"))
	assert.True(t, queuedMsg.DataImportParams.ObjectReady)
	assert.Equal(t, queuedMsg.TenantID, uuid.FromStringOrNil("726c6277-e77b-43ad-8d55-13c799dbb9ac"))
}

func TestS3TestEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)
	testClient := workerclient.NewTestClient()
	wh, _, _, _ := testhelpers.SetupWorkerForTest(ctx, t, testClient)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(s3TestEvent))
	wh.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, http.StatusOK)
	tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 1)
	tt.AssertLogsContainString("Unknown message")
}
