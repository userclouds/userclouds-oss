package logtransports

// Transport directing event stream to our server
import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	logServerInterface "userclouds.com/logserver/client"
)

func init() {
	registerDecoder(TransportTypeServer, func(value []byte) (TransportConfig, error) {
		var s LogServerTransportConfig
		// NB: we need to check the type here because the yaml decoder will happily decode an
		// empty struct, since dec.KnownFields(true) gets lost via the yaml.Unmarshaler
		// interface implementation
		if err := json.Unmarshal(value, &s); err == nil && s.Type == TransportTypeServer {
			return &s, nil
		}
		return nil, ucerr.New("Unknown transport type")
	})
}

// TransportTypeServer defines the server transport
const TransportTypeServer TransportType = "server"

// LogServerTransportConfig defines the configuration for transport sending events to our servers
type LogServerTransportConfig struct {
	Type                  TransportType `yaml:"type" json:"type"`
	uclog.TransportConfig `yaml:"transportconfig" json:"transportconfig"`
	TenantID              uuid.UUID `yaml:"tenant_id" json:"tenant_id"`
	LogServiceURL         string    `yaml:"log_service_url" json:"log_service_url"`
	SendRawData           bool      `yaml:"send_raw_data" json:"send_raw_data"`
}

// GetType implements TransportConfig
func (c LogServerTransportConfig) GetType() TransportType {
	return TransportTypeServer
}

// IsSingleton implements TransportConfig
func (c LogServerTransportConfig) IsSingleton() bool {
	return true
}

// GetTransport implements TransportConfig
func (c LogServerTransportConfig) GetTransport(serviceName service.Service, tokenSource jsonclient.Option, machineName string) uclog.Transport {
	return newTransportBackgroundIOWrapper(newLogServerTransport(&c, tokenSource, serviceName, machineName))
}

// Validate implements Validateable
func (c *LogServerTransportConfig) Validate() error {
	if c.Required && c.LogServiceURL == "" {
		return ucerr.New("logging config invalid - missing service url")
	}

	return nil
}

const (
	logServerTransportName = "LogServerTransport"
	// Intervals for sending event data to the server
	queueReadInterval      time.Duration = 100 * time.Millisecond
	maxBackupCacheSize     int           = 10000
	defaultCounterInterval int           = 100
	defaultMessageInterval int           = 100
	// Maximum map sizes
	maxEventMapSize   int = 1000000
	maxMessageMapSize int = 200000
)

type logServerTransport struct {
	// configuration data
	config          LogServerTransportConfig
	service         service.Service
	host            string
	region          region.MachineRegion
	sendRawData     bool
	defaultTenantID uuid.UUID
	instanceID      uuid.UUID

	counterTickCount int
	counterInterval  int
	messageTickCount int
	messageInterval  int

	// counter data
	countersMap     map[uuid.UUID]map[string]int
	countersMapSize int
	startTime       time.Time

	// raw message data
	messageMap     map[uuid.UUID][][]byte
	messageMapSize int

	// connection to log server
	tokenSource       jsonclient.Option
	client            *logServerInterface.Client
	failedServerCalls int64
}

func newLogServerTransport(c *LogServerTransportConfig, tokenSource jsonclient.Option, name service.Service, machineName string) *logServerTransport {
	return &logServerTransport{
		config:      *c,
		tokenSource: tokenSource,
		service:     name,
		host:        machineName,
		region:      region.Current(),
	}
}

func (t *logServerTransport) init(ctx context.Context) (*uclog.TransportConfig, error) {
	c := &uclog.TransportConfig{Required: t.config.Required, MaxLogLevel: t.config.MaxLogLevel}

	// TODO: fix this pattern where we return an error and "valid" object
	if !service.IsValid(t.service) {
		return c, ucerr.New("Invalid service name")
	}

	if t.tokenSource == nil {
		return c, ucerr.New("Invalid token source config")
	}

	var err error

	t.defaultTenantID = t.config.TenantID
	t.instanceID = uuid.Must(uuid.NewV4())
	t.sendRawData = t.config.SendRawData

	// Create a cache for storing events
	t.countersMap = make(map[uuid.UUID]map[string]int)
	t.counterInterval = defaultCounterInterval

	// Create a cache for storing messages
	t.messageMap = make(map[uuid.UUID][][]byte)
	t.messageInterval = defaultMessageInterval

	// Create client for calling log server
	t.client, err = logServerInterface.NewClientForTenantAuth(t.config.LogServiceURL, t.config.TenantID, t.tokenSource, jsonclient.StopLogging(), jsonclient.BypassRouting())
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return c, ucerr.Wrap(err)
}

func (t *logServerTransport) updateConfig(settings logServerInterface.LogTransportSettings) {
	if !settings.Update {
		return
	}
	t.sendRawData = settings.SendRawData
	t.config.MaxLogLevel = settings.LogLevel
	if settings.CountersInterval > 1 {
		t.counterInterval = settings.CountersInterval
	}
	if settings.MessageInterval > 1 {
		t.messageInterval = settings.MessageInterval
	}
}

func (t *logServerTransport) writeMessages(ctx context.Context, logRecords *logRecord, startTime time.Time, count int) {
	currRecords := logRecords
	for currRecords != nil {
		event := currRecords.event

		if event.TenantID.IsNil() {
			event.TenantID = t.defaultTenantID
			currRecords.event.TenantID = t.defaultTenantID
		}

		// If the record increases a counter add to the counters map
		if event.Code != uclog.EventCodeNone && t.countersMapSize < maxEventMapSize {
			// Set the batch start time if this is the first event being added
			if t.startTime.IsZero() {
				t.startTime = startTime
			}

			offset := int(startTime.Sub(t.startTime).Seconds())
			var key = fmt.Sprintf("%d_%d", event.Code, offset)
			if event.Code == uclog.EventCodeUnknown {
				key = fmt.Sprintf("%s_%s", key, event.Name)
			}
			if event.Payload != "" {
				key = fmt.Sprintf("%s_%s", key, event.Payload)
			}

			if _, ok := t.countersMap[event.TenantID]; !ok {
				t.countersMap[event.TenantID] = make(map[string]int)
			}

			if _, ok := t.countersMap[event.TenantID][key]; !ok {
				t.countersMapSize++
			}
			t.countersMap[event.TenantID][key] = t.countersMap[event.TenantID][key] + event.Count
		}

		currRecords = currRecords.next
	}

	// Check if the record has a qualifying message that should be sent to the server
	if t.sendRawData && t.messageMapSize < maxMessageMapSize {

		recordsReady := EncodeLogForTransfer(logRecords, t.region, t.host, t.service)

		// Store the incoming messages in the buffer to be sent later
		for i, r := range recordsReady {
			if _, ok := t.messageMap[r.TenantID]; !ok {
				t.messageMap[r.TenantID] = make([][]byte, 0, 10)
			}
			enc, _ := json.Marshal(recordsReady[i])

			t.messageMapSize += len(r.Records)
			t.messageMap[r.TenantID] = append(t.messageMap[r.TenantID], []byte(enc))
		}
	}

	// If we have counter updates send them to the server on every timer trigger
	if t.counterTickCount > t.counterInterval {
		t.sendEventToServer(ctx)
		t.counterTickCount = 0
	}

	if t.messageTickCount > t.messageInterval {
		t.sendMessagesToServer(ctx)
		t.messageTickCount = 0
	}

	t.counterTickCount++
	t.messageTickCount++
}

func (t *logServerTransport) sendEventMapToServer(ctx context.Context, baseTime *time.Time, eventMap *map[uuid.UUID]map[string]int) error {
	if t.client != nil && !baseTime.IsZero() {
		settings, err := t.client.PostCounters(ctx, string(t.service), t.instanceID, *baseTime, eventMap)
		// On success update settings if needed and reduce the count of events in the global map
		if err == nil {
			var eventsSent = 0
			for _, tenantEvents := range *eventMap {
				eventsSent += len(tenantEvents)
			}
			t.countersMapSize -= eventsSent

			t.updateConfig(*settings)
		} else {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

func (t *logServerTransport) sendEventToServer(ctx context.Context) {
	if !t.startTime.IsZero() {
		// Try to send out the counters
		if err := t.sendEventMapToServer(ctx, &t.startTime, &t.countersMap); err != nil {
			t.failedServerCalls++
		} else {
			t.startTime = time.Time{}
			t.countersMap = make(map[uuid.UUID]map[string]int)
		}
	}
}

func (t *logServerTransport) sendMessagesToServer(ctx context.Context) {
	if t.sendRawData {

		// Save the current state of the accumulated messages cache
		messagesMap := t.messageMap
		// Reset the cache
		t.messageMap = make(map[uuid.UUID][][]byte)

		// Send out messages if there are any
		if len(messagesMap) > 0 {
			// TODO break up the post if its size is too big
			if t.client != nil {
				if settings, err := t.client.PostRawLogs(ctx, string(t.service), t.instanceID, &messagesMap); err == nil {
					var messageSentCount = 0
					for _, tenantMessages := range messagesMap {
						messageSentCount += len(tenantMessages)
					}
					t.messageMapSize -= messageSentCount
					t.updateConfig(*settings)
				}
			}
		}
	}
}
func (t *logServerTransport) getFailedAPICallsCount() int64 {
	return t.failedServerCalls
}

func (t *logServerTransport) getIOInterval() time.Duration {
	return queueReadInterval
}
func (t *logServerTransport) getMaxLogLevel() uclog.LogLevel {
	return t.config.MaxLogLevel
}

func (t *logServerTransport) supportsCounters() bool {
	return true
}

func (t *logServerTransport) getTransportName() string {
	return logServerTransportName
}

func (t *logServerTransport) flushIOResources() {
	t.sendEventToServer(context.Background())
	t.sendMessagesToServer(context.Background())
}
func (t *logServerTransport) closeIOResources() {
	t.sendEventToServer(context.Background())
	t.sendMessagesToServer(context.Background())
}
