package logtransports

// Transport directing event stream to our server
import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	logServerInterface "userclouds.com/logserver/client"
)

const (
	defaultEventMetadataDownloadInterval time.Duration = time.Second
	eventMetadataURL                     string        = "/eventmetadata/default"
)

type logServerMapFetcher struct {
	instanceID             uuid.UUID
	service                string
	client                 *jsonclient.Client
	eventMetadataRequests  []uuid.UUID
	queueLock              sync.Mutex
	updateEventDataHandler func(updatedMap *uclog.EventMetadataMap, tenantID uuid.UUID) error

	fetchMutex      sync.Mutex
	fetchTicker     time.Ticker
	done            chan bool
	runningBGThread bool

	failedServerCalls int
}

func newLogServerMapFetcher(logServiceURL string, service, machineName string) (*logServerMapFetcher, error) {
	if logServiceURL == "" {
		return nil, ucerr.New("logServiceURL is empty")
	}
	ua := fmt.Sprintf("UserClouds LogServerMapFetcher for %s [%s]", service, machineName)
	fetcher := logServerMapFetcher{
		service: service,
		client:  jsonclient.New(logServiceURL, jsonclient.HeaderUserAgent(ua))}

	return &fetcher, nil
}

func (l *logServerMapFetcher) createPath(tenantID uuid.UUID) string {
	return eventMetadataURL + "?" + logServerInterface.InstanceIDQueryArgName + "=" + l.instanceID.String() + "&" +
		logServerInterface.ServiceQueryArgName + "=" + l.service + "&" + logServerInterface.TenantIDQueryArgName + "=" +
		tenantID.String()
}

func (l *logServerMapFetcher) Init(updateHandler func(updatedMap *uclog.EventMetadataMap, tenantID uuid.UUID) error) error {
	l.instanceID = uuid.Must(uuid.NewV4())

	// Setup event metadata state
	l.eventMetadataRequests = make([]uuid.UUID, 0)
	l.updateEventDataHandler = updateHandler

	l.fetchTicker = *time.NewTicker(defaultEventMetadataDownloadInterval)
	l.done = make(chan bool)
	go func() {
		ctx := context.Background()
		l.runningBGThread = true
		for {
			select {
			case <-l.done:
				l.fetchMutex.Lock()
				l.runningBGThread = false
				l.fetchMutex.Unlock()
				return
			case <-l.fetchTicker.C:
				l.fetchMutex.Lock()
				l.updateEventMetadata(ctx)
				l.fetchMutex.Unlock()
			}
		}
	}()

	return nil
}

// FetchEventMetadataForTenant tries to fetch the event metadata map for given tenant
func (l *logServerMapFetcher) FetchEventMetadataForTenant(tenantID uuid.UUID) {
	l.queueLock.Lock()
	l.eventMetadataRequests = append(l.eventMetadataRequests, tenantID)
	l.queueLock.Unlock()
}

func (l *logServerMapFetcher) updateEventMetadata(ctx context.Context) {
	// Check if there are any requests and copy them into a local array
	l.queueLock.Lock()
	q := l.eventMetadataRequests
	l.eventMetadataRequests = make([]uuid.UUID, 0)
	l.queueLock.Unlock()

	for _, tenantID := range q {
		var newEventMetadata uclog.EventMetadataMap
		path := l.createPath(tenantID)
		err := l.client.Get(ctx, path, &newEventMetadata)
		if err == nil {
			if err := l.updateEventDataHandler(&newEventMetadata, tenantID); err == nil {
				continue
			}
		}
		uclog.Warningf(ctx, "Failed to fetch event metadata for tenant %v [%s]: %v", tenantID, path, err)
		// Since provisioning tenants takes a while delay retrying
		if jsonclient.IsHTTPNotFound(err) {
			go func() {
				uclog.Debugf(context.Background(), "No event metadata found for tenant %v which is being provisioned, delaying retry", tenantID)
				time.Sleep(60 * time.Second)
				l.FetchEventMetadataForTenant(tenantID)
			}()
		} else {
			l.FetchEventMetadataForTenant(tenantID)
		}
		l.failedServerCalls++
	}
}

func (l *logServerMapFetcher) Close() {
	if l.runningBGThread {
		// Send signal to background thread to perform final flush
		l.done <- true
	}
}
