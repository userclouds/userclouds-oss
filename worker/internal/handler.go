package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/getsentry/sentry-go"
	"github.com/gofrs/uuid"

	idpConfig "userclouds.com/idp/config"
	acmeinfra "userclouds.com/infra/acme"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/dnsclient"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucmetrics"
	"userclouds.com/infra/uctrace"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/ucopensearch"
	"userclouds.com/internal/ucsentry"
	"userclouds.com/plex/manager"
	"userclouds.com/plex/worker/appimport"
	"userclouds.com/worker"
	"userclouds.com/worker/config"
	"userclouds.com/worker/internal/acme"
	"userclouds.com/worker/internal/cachetool"
	"userclouds.com/worker/internal/cleanup"
	"userclouds.com/worker/internal/searchindex"
	"userclouds.com/worker/internal/sqlshimingest"
	"userclouds.com/worker/internal/tenant"
	"userclouds.com/worker/internal/usersync"
	"userclouds.com/worker/storage"
)

const (
	workerSubsystem = ucmetrics.Subsystem("worker")
	longPollTimeout = time.Second * 10
)

var (
	tracer                    = uctrace.NewTracer("worker")
	metricTaskCount           = ucmetrics.CreateCounter(workerSubsystem, "tasks_total", "The total number of processed tasks", "task")
	metricTaskDurationSeconds = ucmetrics.CreateHistogram(workerSubsystem, "task_duration_seconds", "time taken to handle task", "task")
)

type provisionDBInfo struct {
	companyDB *ucdb.Config
	logDB     *ucdb.Config
}
type handler struct {
	provisionDBInfo      provisionDBInfo
	openSearchCfg        *ucopensearch.Config
	acmeCfg              *acmeinfra.Config
	cacheCfg             *cache.Config
	dnsClient            dnsclient.Client
	consoleTenantInfo    companyconfig.TenantInfo
	m2mAuth              jsonclient.Option
	companyConfigStorage *companyconfig.Storage
	tm                   *tenantmap.StateMap
	wc                   workerclient.Client
	enableTracing        bool
	runningTasks         *RunningTasks
}

func newHandler(cfg *config.Config, m2mAuth jsonclient.Option, consoleTenantInfo companyconfig.TenantInfo, ccs *companyconfig.Storage, tm *tenantmap.StateMap, wc workerclient.Client, runningTasks *RunningTasks) *handler {
	dnsClient := dnsclient.NewFromConfig(&cfg.DNS)
	return &handler{
		provisionDBInfo:      provisionDBInfo{companyDB: &cfg.CompanyDB, logDB: &cfg.LogDB},
		acmeCfg:              cfg.ACME,
		openSearchCfg:        cfg.OpenSearchConfig,
		cacheCfg:             cfg.CacheConfig,
		dnsClient:            dnsClient,
		consoleTenantInfo:    consoleTenantInfo,
		m2mAuth:              m2mAuth,
		companyConfigStorage: ccs,
		tm:                   tm,
		wc:                   wc,
		enableTracing:        cfg.Tracing != nil,
		runningTasks:         runningTasks,
	}
}

// NewHTTPHandler returns a new HTTP message handler
func NewHTTPHandler(cfg *config.Config, ccs *companyconfig.Storage, m2mAuth jsonclient.Option, consoleTenantInfo companyconfig.TenantInfo, tm *tenantmap.StateMap, wc workerclient.Client, runningTasks *RunningTasks) http.Handler {
	hb := builder.NewHandlerBuilder()
	h := newHandler(cfg, m2mAuth, consoleTenantInfo, ccs, tm, wc, runningTasks)
	// handler could really just implement ServeHTTP since there's only ever one route
	// here (per the SQS / EB worker spec), but sticking to our common pattern for
	// tooling compat, etc.
	hb.HandleFunc("/", h.handleMessageEndpoint)
	hb.HandleFunc("/scheduled", h.handleScheduled)

	return hb.Build()
}

func (h *handler) queueDataImportTasks(ctx context.Context, dataImportInfos []storage.DataImportInfo) {
	uclog.Infof(ctx, "queueing %d data import tasks", len(dataImportInfos))
	for _, di := range dataImportInfos {
		uclog.Infof(ctx, "queueing data import task for job %+v", di)
		if err := h.wc.Send(ctx, worker.DataImportMessage(di.TenantID, di.JobID, true)); err != nil {
			uclog.Errorf(ctx, "failed to send data import message: %v", err)
		}
	}
}

func isWorkerMessage(jsonPayload map[string]any) bool {
	_, ok := jsonPayload["task"]
	return ok
}

func dumpHeaders(r *http.Request) string {
	sb := strings.Builder{}
	for name, values := range r.Header {
		sb.WriteString(fmt.Sprintf("%s: %s\n", name, strings.Join(values, ", ")))
	}
	return sb.String()
}

func (h *handler) handleScheduled(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	msgBody, err := io.ReadAll(r.Body)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	uclog.Infof(ctx, "got schedule message: %v\nheaders:\n%s", string(msgBody), dumpHeaders(r))
	jsonapi.Marshal(w, nil, jsonapi.Code(http.StatusOK))
}

func (h *handler) handleMessageEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	msgBody, err := io.ReadAll(r.Body)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	err = h.handleMessage(ctx, msgBody, request.GetRequestID(ctx))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	// be explicit about HTTP 200 OK - done with this SQS message
	jsonapi.Marshal(w, nil, jsonapi.Code(http.StatusOK))
}

// StartLongPollSQS starts a long running go routine to poll an SQS queue for messages and process them.
func StartLongPollSQS(ctx context.Context, cfg *config.Config, ccs *companyconfig.Storage, m2mAuth jsonclient.Option, consoleTenantInfo companyconfig.TenantInfo, tm *tenantmap.StateMap, wc workerclient.Client, runningTasks *RunningTasks) error {
	h := newHandler(cfg, m2mAuth, consoleTenantInfo, ccs, tm, wc, runningTasks)
	uv := universe.Current()
	if !uv.IsCloud() {
		return ucerr.Errorf("This tool can only be run in the cloud universe, not %v", uv)
	}
	queueURL := cfg.WorkerClient.URL
	uclog.Infof(ctx, "starting long poll for queue %v  timeout: %v", queueURL, longPollTimeout)
	client, err := workerclient.GetSQSClient(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "failed get SQS client: %v", err)
	}

	go func() {
		for {
			if err := h.pollAndProcessMessages(ctx, client, queueURL, longPollTimeout); err != nil {
				uclog.Fatalf(ctx, "failed to long poll SQS: %v", err)
			}
		}
	}()
	return nil
}

func (h *handler) pollAndProcessMessages(ctx context.Context, client *sqs.Client, queueURL string, longPollTimeout time.Duration) error {
	input := &sqs.ReceiveMessageInput{QueueUrl: &queueURL, MaxNumberOfMessages: 3, WaitTimeSeconds: int32(longPollTimeout.Seconds())}
	resp, err := client.ReceiveMessage(ctx, input)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if len(resp.Messages) > 0 {
		uclog.Infof(ctx, "got %d messages", len(resp.Messages))
	}
	for _, msg := range resp.Messages {
		requestID := uuid.Must(uuid.NewV4())
		ctx = request.SetRequestID(ctx, requestID)
		if err := h.handleMessage(ctx, []byte(*msg.Body), requestID); err != nil {
			uclog.Errorf(ctx, "failed to handle message: %v", err)
		}
		if _, err := client.DeleteMessage(ctx, &sqs.DeleteMessageInput{QueueUrl: &queueURL, ReceiptHandle: msg.ReceiptHandle}); err != nil {
			uclog.Errorf(ctx, "failed to delete message %s from sqs queue: %v", *msg.ReceiptHandle, err)

		}
	}
	return nil
}

func (h *handler) handleMessage(ctx context.Context, msgBody []byte, requestID uuid.UUID) error {
	var jsonPayload map[string]any
	if err := json.Unmarshal(msgBody, &jsonPayload); err != nil {
		return ucerr.Wrap(err)
	}
	if isS3Notification(jsonPayload) {
		if isDataImportS3Notification(msgBody) {
			dataImportInfos, err := h.getDataImportS3Notifications(ctx, msgBody)
			if err != nil {
				return ucerr.Wrap(err)
			}
			h.queueDataImportTasks(ctx, dataImportInfos)
		} else {
			uclog.Warningf(ctx, "Unknown S3 notification: %v", string(msgBody))
		}
	} else if isWorkerMessage(jsonPayload) {
		var msg worker.Message
		if err := json.Unmarshal(msgBody, &msg); err != nil {
			return ucerr.Wrap(err)
		}
		if err := msg.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
		uclog.Debugf(ctx, "Got message: %v", msg)
		go h.handleTask(msg, requestID)
	} else {
		uclog.Warningf(ctx, "Unknown message: %v", string(msgBody))
	}
	return nil
}

func (h *handler) isTaskDisabled(ctx context.Context, msg worker.Message) bool {
	// We use tenant scope
	//to disable a task that is not scoped to a tenant use 'v1_tenant_00000000-0000-0000-0000-000000000000' as the user id in  the statsig console
	lst := featureflags.GetStringsListForTenant(ctx, featureflags.DisabledWorkerTasks, msg.GetTenantID())
	return slices.Contains(lst, string(msg.Task))
}

func (h *handler) handleTask(msg worker.Message, requestID uuid.UUID) {
	taskName := string(msg.Task)
	handlerName := fmt.Sprintf("handleTask.%s", taskName)
	ctx := uclog.SetHandlerName(request.SetRequestID(context.Background(), requestID), handlerName)
	h.runningTasks.addTask(ctx, msg, requestID)
	defer h.runningTasks.removeTask(requestID)
	uclog.Infof(ctx, "Handle message: %v", msg)
	var ts *tenantmap.TenantState
	var err error
	endSpan := func() {}
	var span uctrace.Span
	if h.enableTracing {
		ctx, span = tracer.StartSpan(ctx, handlerName, false)
		endSpan = span.End
	}
	defer endSpan()
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("task", string(msg.Task))
		if h.isTaskDisabled(ctx, msg) {
			uclog.Infof(ctx, "Task '%v' is disabled for tenant %v", taskName, msg.GetTenantID())
			return // don't run the task
		}

		// TODO (sgarrity 6/23): unify the loading of things like tenant plex, tenantdb, etc here
		// but only for messages that actually need them (eg. not create tenant)
		if !msg.TenantID.IsNil() {
			ts, err = h.tm.GetTenantStateForID(ctx, msg.TenantID)
			if err != nil {
				uclog.Errorf(ctx, "failed to get tenant state for task %v: %v", msg, err)
				sentry.CaptureException(err)
				return
			}
		} else if msg.CreateTenant != nil { // worker.TaskCreateTenant - creating a 'fake' TenantState object we can use for tracing, etc
			ctp := msg.CreateTenant
			ts, err = h.getTenantState(ctx, ctp.Tenant)
			if err != nil {
				uclog.Errorf(ctx, "failed to get tenant state for task %v: %v", msg, err)
				sentry.CaptureException(err)
				return
			}
		} // it is a clear cache message without tenant specified

		if ts != nil {
			ctx = uclog.SetTenantID(ctx, ts.ID)
			if h.enableTracing {
				multitenant.SetTenantAttributes(span, ts)
			}
			ucsentry.SetTenant(scope, ts)
		}
		metricTaskCount.WithLabelValues(taskName).Inc()
		startTime := time.Now().UTC()
		if err := h.runTask(ctx, ts, &msg); err != nil {
			uclog.Errorf(ctx, "Task %v execution failed: %v", msg, err)
			sentry.CaptureException(err)
		}
		took := time.Now().UTC().Sub(startTime).Seconds()
		uclog.Infof(ctx, "Task %s took %v seconds", taskName, took)
		metricTaskDurationSeconds.WithLabelValues(taskName).Observe(took)
	})
}

func (h *handler) getTenantState(ctx context.Context, tenant companyconfig.Tenant) (*tenantmap.TenantState, error) {
	// It is not worth it to fail tenant creation if we can't parse the URL or can't get the company from the DB since we only
	// use the info for logging and tracing and not to do actual work.
	tenantURL, err := url.Parse(tenant.TenantURL)
	if err != nil {
		uclog.Errorf(ctx, "failed to parse tenant URL %v: %v", tenant.TenantURL, err)
	}
	company, err := h.companyConfigStorage.GetCompany(ctx, tenant.CompanyID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return tenantmap.NewTenantState(&tenant, company, tenantURL, nil, nil, nil, "", h.companyConfigStorage, false, nil, h.cacheCfg), nil
}

func (h *handler) runTask(ctx context.Context, ts *tenantmap.TenantState, msg *worker.Message) error {
	uclog.Infof(ctx, "running task: %v", msg)
	switch msg.Task {
	case worker.TaskNoOp:
		if msg.NoOpParams == nil {
			return ucerr.New("missing NoOpParams")
		}
		duration := msg.NoOpParams.Duration
		if duration > time.Minute*5 {
			return ucerr.Errorf("Duration %v is too long", duration)
		}
		uclog.Infof(ctx, "no-op task received. duration: %v", duration)
		timer := time.NewTimer(duration)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ucerr.Wrap(ctx.Err())
		case <-timer.C:
		}
		return nil
	case worker.TaskSyncAllUsers:
		return ucerr.Wrap(usersync.SyncTenantUsers(ctx, msg.TenantID, ts.TenantDB, h.cacheCfg))
	case worker.TaskImportAuth0Apps:
		importAuth0Apps := msg.ImportAuth0Apps
		if importAuth0Apps == nil {
			return ucerr.New("missing ImportAuth0Apps params")
		}
		return ucerr.Wrap(h.importAuth0Apps(ctx, ts.TenantDB, *importAuth0Apps))
	case worker.TaskNewTenantCNAME:
		td := msg.TenantDNS
		if td == nil {
			return ucerr.New("missing TenantDNS params for TaskNewTenantCNAME")
		}
		return ucerr.Wrap(acme.SetupNewTenantURL(ctx, h.dnsClient, h.acmeCfg, ts.TenantDB, td.TenantID, td.URL, h.companyConfigStorage))
	case worker.TaskValidateDNS:
		td := msg.TenantDNS
		if td == nil {
			return ucerr.New("missing TenantDNS params forTaskValidateDNS")
		}
		return ucerr.Wrap(acme.ValidateTenantURL(ctx, h.dnsClient, h.acmeCfg, h.companyConfigStorage, td.TenantID, td.URL, ts.TenantDB, h.wc))

	case worker.TaskFinalizeTenantCNAME:
		p := msg.FinalizeTenantCNAME
		if p == nil {
			return ucerr.New("missing FinalizeTenantCNAME params")
		}
		return ucerr.Wrap(acme.FinalizeNewTenantURL(ctx, h.acmeCfg, h.companyConfigStorage, p.TenantID, ts.TenantDB, p.UCOrderID))
	case worker.TaskCheckTenantCNAME:
		p := msg.CheckTenantCNAME
		if p == nil {
			return ucerr.New("missing CheckTenantCNAME params")
		}
		return ucerr.Wrap(acme.CheckTenantURL(ctx, h.dnsClient, h.companyConfigStorage, p.TenantID, p.TenantURLID))
	case worker.TaskCreateTenant:
		if msg.SourceRegion == region.Current() {
			return ucerr.Wrap(h.createTenant(ctx, msg.CreateTenant))
		}
		uclog.Infof(ctx, "Requeue create tenant message from region %v (need it to run in that region not in %v)", msg.SourceRegion, region.Current())
		return ucerr.Wrap(h.wc.Send(ctx, *msg))
	case worker.TaskProvisionTenantURLs:
		p := msg.TenantURLProvisioningParams
		if p == nil {
			return ucerr.New("missing TenantURLProvisioningParams")
		}
		return ucerr.Wrap(tenant.ProvisionTenantURLs(ctx, h.companyConfigStorage, msg.TenantID, p.AddEKSURLs, p.DeleteURLs, p.DryRun))
	case worker.TaskClearCache:
		if msg.ClearCache == nil {
			return ucerr.New("missing clear cache params")
		}
		return ucerr.Wrap(cachetool.ClearCache(ctx, h.cacheCfg, h.companyConfigStorage, *msg.ClearCache))
	case worker.TaskLogCache:
		if msg.LogCache == nil {
			return ucerr.Errorf("missing log cache params")
		}
		return ucerr.Wrap(cachetool.LogCache(ctx, h.cacheCfg, h.companyConfigStorage, *msg.LogCache))
	case worker.TaskDataImport:
		return ucerr.Wrap(dataImport(ctx, h.wc, ts, msg.DataImportParams.JobID, msg.DataImportParams.ObjectReady))
	case worker.TaskPlexTokenDataCleanup:
		if msg.PlexTokenDataCleanup == nil {
			return ucerr.Errorf("missing plex token data cleanup params")
		}
		return ucerr.Wrap(cleanup.CleanPlexTokensForTenant(ctx, msg.TenantID, ts.TenantDB, h.cacheCfg, *msg.PlexTokenDataCleanup))
	case worker.TaskUserStoreDataCleanup:
		if msg.UserStoreDataCleanup == nil {
			return ucerr.Errorf("missing user store data cleanup params")
		}
		if msg.SourceRegion == region.Current() {
			return ucerr.Wrap(cleanup.CleanUserStoreForTenant(ctx, ts, *msg.UserStoreDataCleanup))
		}
		uclog.Infof(ctx, "Requeue %s message from region %v (need it to run in that region not in %v)", msg.Task, msg.SourceRegion, region.Current())
		return ucerr.Wrap(h.wc.Send(ctx, *msg))
	case worker.TaskIngestSqlshimDatabaseSchema:
		if msg.IngestSqlshimDatabaseSchemasParams == nil {
			return ucerr.Errorf("missing ingest sqlshim database schema params")
		}
		return ucerr.Wrap(sqlshimingest.IngestSqlshimDatabaseSchemas(ctx, ts, msg.IngestSqlshimDatabaseSchemasParams.DatabaseID))
	case worker.TaskProvisionTenantOpenSearchIndex:
		if h.openSearchCfg == nil {
			return ucerr.New("Missing OpenSearch configuration; cannot provision OpenSearch index.")
		}
		provParams := msg.ProvisionTenantOpenSearchIndexParams
		if provParams == nil {
			return ucerr.Errorf("missing ProvisionTenantOpenSearchIndexParams")
		}
		return ucerr.Wrap(searchindex.Provision(ctx, h.openSearchCfg, h.wc, ts, provParams.IndexID))
	case worker.TaskBootstrapTenantOpenSearchIndex:
		if h.openSearchCfg == nil {
			return ucerr.New("Missing OpenSearch configuration; cannot bootstrap OpenSearch index.")
		}
		bsParams := msg.BootstrapTenantOpenSearchIndexParams
		if bsParams == nil {
			return ucerr.Errorf("missing BootstrapTenantOpenSearchIndexParams")
		}
		return ucerr.Wrap(
			searchindex.Bootstrap(
				ctx,
				&idpConfig.SearchUpdateConfig{SearchCfg: h.openSearchCfg},
				h.wc,
				ts,
				bsParams.IndexID,
				bsParams.LastRegionalBootstrappedValueID,
				bsParams.Region,
				bsParams.BatchSize,
			),
		)
	case worker.TaskUpdateTenantOpenSearchIndex:
		updateParams := msg.UpdateTenantOpenSearchIndexParams
		if updateParams == nil {
			return ucerr.Errorf("missing UpdateTenantOpenSearchIndexParams")
		}
		return ucerr.Wrap(searchindex.Update(ctx, msg.TenantID, h.openSearchCfg, updateParams.Data, h.wc, updateParams.Attempt+1))
	case worker.TaskSaveTenantInternal:
		if msg.SaveTenantInternalParams == nil {
			return ucerr.Errorf("missing SaveTenantInternalParams")
		}
		return ucerr.Wrap(h.companyConfigStorage.SaveTenantInternal(ctx, msg.SaveTenantInternalParams.TenantInternal))
	default:
		return ucerr.Errorf("unknown task: %v", msg.Task)
	}
}

func (h *handler) importAuth0Apps(ctx context.Context, tenantDB *ucdb.DB, importAuth0Apps worker.ImportAuth0AppsParams) error {
	if importAuth0Apps.ProviderID.IsNil() {
		return ucerr.New("missing provider ID for TaskImportAuth0Apps")
	}

	// TODO: can we just look up the tenant URL from the TenantID in this message?
	ts, err := m2m.GetM2MTokenSource(ctx, importAuth0Apps.TenantID)
	if err != nil {
		return ucerr.Errorf("failed to get m2m secret for tenant %v: %v", importAuth0Apps.TenantID, err)
	}
	authzClient, err := apiclient.NewAuthzClientWithTokenSource(ctx, h.companyConfigStorage, importAuth0Apps.TenantID, importAuth0Apps.TenantURL, ts)
	if err != nil {
		return ucerr.Errorf("failed to create authz client: %W", err)
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheCfg)
	tp, err := mgr.GetTenantPlex(ctx, importAuth0Apps.TenantID)
	if err != nil {
		return ucerr.Errorf("failed to get tenant plex config: %w", err)
	}

	s := storage.New(tenantDB)
	return ucerr.Wrap(appimport.RunImport(ctx, importAuth0Apps.TenantID, &tp.PlexConfig.PlexMap, importAuth0Apps.ProviderID, h.companyConfigStorage, h.tm, authzClient, s))
}

func (h *handler) createTenant(ctx context.Context, ctp *worker.CreateTenantParams) error {
	if ctp == nil {
		return ucerr.New("missing create tenant params")
	}
	cac, err := apiclient.NewAuthzClientWithTokenSource(ctx, h.companyConfigStorage, h.consoleTenantInfo.TenantID, h.consoleTenantInfo.TenantURL, h.m2mAuth)
	if err != nil {
		return ucerr.Errorf("failed to create console authz client: %v", err)
	}
	return ucerr.Wrap(tenant.CreateTenant(ctx, h.companyConfigStorage, h.tm, h.provisionDBInfo.companyDB, h.provisionDBInfo.logDB, ctp.Tenant, ctp.UserID, cac, h.consoleTenantInfo, h.cacheCfg))

}
