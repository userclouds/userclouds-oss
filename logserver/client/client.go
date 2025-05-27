package client

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/sdkclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	// ServiceQueryArgName is the name of the query parameter
	ServiceQueryArgName = "service"
	// InstanceIDQueryArgName is the name of the query parameter
	InstanceIDQueryArgName = "instance_id"
	// TenantIDQueryArgName is the name of the query parameter
	TenantIDQueryArgName = "tenant_id"
	// BaseTimeQueryArgName is the name of the query parameter
	BaseTimeQueryArgName = "base_time"
	// EventCountQueryArgName is the name of the query parameter
	EventCountQueryArgName = "event_count"
	// FilterQueryArgName is the name of the query parameter
	FilterQueryArgName = "query"
)

const (
	// InstancesHandlerPath is the name of the URL segment
	InstancesHandlerPath = "/logserver/instances"
	// ActivityHandlerPath is the name of the URL segment
	ActivityHandlerPath = "/logserver/activity"
	// ChartQueryHandlerPath is the name of the URL segment
	ChartQueryHandlerPath = "/logserver/chart"
	// CountQueryHandlerPath is the name of the URL segment
	CountQueryHandlerPath = "/logserver/query"
	// RawLogsHandlerPath is the name of the URL segment
	RawLogsHandlerPath = "/logserver/kinesis"
	// CountersHandlerPath is the name of the URL segment
	CountersHandlerPath = "/logserver/counters"
	// EventMetadataPath is the name of the URL segment
	EventMetadataPath = "/logserver/eventmetadata"
)

// Client represents a client to talk to the UserClouds LogServer service
type Client struct {
	client   *sdkclient.Client
	tenantID uuid.UUID
}

// NewClientForTenantAuth constructs a new LogServer client using the provided tenant auth info
func NewClientForTenantAuth(tenantURL string, tenantID uuid.UUID, opts ...jsonclient.Option) (*Client, error) {
	return NewClientForTenant(tenantURL, tenantID, opts...)
}

// NewClientForTenant constructs a new LogServer client using the provided client ID & secret to authenticate
func NewClientForTenant(tenantURL string, tenantID uuid.UUID, opts ...jsonclient.Option) (*Client, error) {
	return NewClient(tenantURL, tenantID, opts...)
}

// NewClient constructs a new LogServer client
func NewClient(url string, tenantID uuid.UUID, opts ...jsonclient.Option) (*Client, error) {
	c := &Client{
		client:   sdkclient.New(strings.TrimSuffix(url, "/"), "logserver", opts...),
		tenantID: tenantID,
	}
	if err := c.client.ValidateBearerTokenHeader(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return c, nil
}

// CounterRecord provides data for row in the event log
type CounterRecord struct {
	ID        uint64 `json:"id"`
	EventName string `json:"event_name"`
	EventType string `json:"event_type"`
	Service   string `json:"service"`
	Timestamp int64  `json:"timestamp"`
	Count     int    `json:"count"`
}

// SourceRecord contains information about a "running" instance of service that has
// registired with the server and can send events
type SourceRecord struct {
	InstanceID           uuid.UUID      `json:"instance_id"`
	LastTenantID         uuid.UUID      `json:"last_tenant_id"`
	Multitenant          bool           `json:"multitenant"`
	Service              string         `json:"service"`
	Region               string         `json:"region"`
	CodeVersion          string         `json:"code_version"`
	StartupTime          time.Time      `json:"startup_time"`
	EventMetadataVersion int            `json:"event_metadata_version"`
	LastActivity         time.Time      `json:"last_activity"`
	CallCount            int            `json:"call_count"`
	EventCount           int            `json:"event_count"`
	ErrorInternalCount   int            `json:"error_internal_count"`
	ErrorInputCount      int            `json:"error_input_count"`
	SendRawData          bool           `json:"send_raw_data"`
	LogLevel             uclog.LogLevel `json:"log_level"`
	MessageInterval      int            `json:"message_interval"`
	CountersInterval     int            `json:"counters_interval"`
	ProcessedStartup     bool           `json:"processed_startup"`
}

// CounterQuery describes a query to the server which generates a raw chart data
// TODO: this should be shared with logserver/.../countersbackend
type CounterQuery struct {
	TenantID  uuid.UUID     `json:"tenant_id"`
	Service   string        `json:"service"`
	EventType []int         `json:"event_codes"`
	Start     time.Time     `json:"start"`
	End       time.Time     `json:"end"`
	Period    time.Duration `json:"period"`
}

// CountQueryRequest is the request for a event count
type CountQueryRequest struct {
	TenantID     uuid.UUID       `json:"tenant_id"`
	Service      service.Service `json:"service"`
	EventStrings []string        `json:"event_strings"`
	Start        time.Time       `json:"start"`
	End          time.Time       `json:"end"`
}

// CountQueryResponse is the response for a count query
type CountQueryResponse struct {
	EventCode []uclog.EventCode `json:"event_code"`
	Count     int               `json:"count"`
}

// LogTransportSettings is sent by the server to update local transport settings
type LogTransportSettings struct {
	Update           bool
	SendRawData      bool
	LogLevel         uclog.LogLevel
	MessageInterval  int
	CountersInterval int
}

// RechartsData represents data for a single column in a Recharts chart.
type RechartsData struct {
	XAxis  string         `json:"xAxis"`
	Values map[string]int `json:"values"`
}

// RechartsColumn represents data for a single column in a Recharts chart.
type RechartsColumn struct {
	Column []RechartsData `json:"column"`
}

// RechartsChart represents a Recharts chart.
type RechartsChart struct {
	Chart []RechartsColumn `json:"chart"`
}

// RechartsResponse represents a response for a Recharts chart.
type RechartsResponse struct {
	Charts []RechartsChart `json:"charts"`
}

// MetricAttributes contains set of attributes
type MetricAttributes struct {
	// Ignore indicate that the client shouldn't send this event to the server
	Ignore bool `db:"ignore" json:"ignore"`
	// System indicate that this is a system (UC) event and not per object event
	System bool `db:"system" json:"system"`
	// AnyService indicate that this is event can be triggered by any service
	AnyService bool `db:"anyservice" json:"anyservice"`
}

// MetricMetadata describes metadata for an event
type MetricMetadata struct {
	ucdb.BaseModel

	// Service in which this event occurs
	Service service.Service `db:"service" json:"service"`
	// Category is the category of the event
	Category uclog.EventCategory `db:"category" json:"category"`
	// NameString is the name the client will pass if it doesn't have the code
	StringID string `db:"string_id" json:"string_id"`
	// Unique numeric code for the event, stays same across namestring changes
	Code uclog.EventCode `db:"code" json:"code"`
	// URL to object this events relates to if available
	ReferenceURL string `db:"url" json:"url"`
	// Human readable name for the event
	Name string `db:"name" json:"name"`
	// Description for what the event represents
	Description string `db:"description" json:"description"`
	// Attributes for the event
	Attributes MetricAttributes `db:"attributes" json:"attributes"`
}

// PostCounters posts a set of events for a time period (/counters)
func (c *Client) PostCounters(ctx context.Context, service string, instanceID uuid.UUID, baseTime time.Time, counters *map[uuid.UUID]map[string]int) (*LogTransportSettings, error) {
	basetimeBytes, err := time.Now().UTC().MarshalText()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	// TODO clean up the interface so that all parameters are in the body
	requestURL := url.URL{
		Path: CountersHandlerPath,
		RawQuery: url.Values{
			InstanceIDQueryArgName: []string{instanceID.String()},
			BaseTimeQueryArgName:   []string{string(basetimeBytes)},
			ServiceQueryArgName:    []string{service},
		}.Encode(),
	}

	var res LogTransportSettings
	return &res, ucerr.Wrap(c.client.Post(ctx, requestURL.String(), counters, &res))
}

// PostRawLogs posts a set of log messages (/kinesis)
func (c *Client) PostRawLogs(ctx context.Context, service string, instanceID uuid.UUID, messages *map[uuid.UUID][][]byte) (*LogTransportSettings, error) {
	requestURL := url.URL{
		Path: RawLogsHandlerPath,
		RawQuery: url.Values{
			TenantIDQueryArgName:   []string{c.tenantID.String()},
			InstanceIDQueryArgName: []string{instanceID.String()},
			ServiceQueryArgName:    []string{service},
		}.Encode(),
	}

	var res LogTransportSettings
	return &res, ucerr.Wrap(c.client.Post(ctx, requestURL.String(), messages, &res))
}

// GetCharts gets event counts for given event types/timeperiod and returns a formatted list of RechartsData
func (c *Client) GetCharts(ctx context.Context, queries *[]CounterQuery) (*RechartsResponse, error) {
	requestURL := url.URL{
		Path: ChartQueryHandlerPath,
	}

	var res RechartsResponse

	if err := c.client.Post(ctx, requestURL.String(), &queries, &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// GetCounts gets event counts for given event types/timeperiod
func (c *Client) GetCounts(ctx context.Context, query *CountQueryRequest) (*CountQueryResponse, error) {
	requestURL := url.URL{
		Path: CountQueryHandlerPath,
	}

	var res CountQueryResponse

	if err := c.client.Post(ctx, requestURL.String(), &query, &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// ListCounterRecords gets a list of events for a time period/service (/activity)
func (c *Client) ListCounterRecords(ctx context.Context, service string, count int) (*[]CounterRecord, error) {
	return c.ListCounterRecordsForTenant(ctx, service, count, c.tenantID)
}

// ListCounterRecordsForTenant gets a list of events for a time period/service (/activity)
func (c *Client) ListCounterRecordsForTenant(ctx context.Context, service string, count int, tenantID uuid.UUID) (*[]CounterRecord, error) {
	var res []CounterRecord

	requestURL := url.URL{
		Path: ActivityHandlerPath,
		RawQuery: url.Values{
			TenantIDQueryArgName:   []string{tenantID.String()},
			EventCountQueryArgName: []string{strconv.Itoa(count)},
			ServiceQueryArgName:    []string{service},
		}.Encode(),
	}

	if err := c.client.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// GetSources gets a list of event sources generating events /instances
func (c *Client) GetSources(ctx context.Context) (*[]SourceRecord, error) {
	return c.GetSourcesForTenant(ctx, c.tenantID)
}

// GetSourcesForTenant gets a list of event sources generating events /instances
func (c *Client) GetSourcesForTenant(ctx context.Context, tenantID uuid.UUID) (*[]SourceRecord, error) {
	var res []SourceRecord
	requestURL := url.URL{
		Path: InstancesHandlerPath,
		RawQuery: url.Values{
			TenantIDQueryArgName: []string{tenantID.String()},
		}.Encode(),
	}

	if err := c.client.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// CreateEventType create a new event metadata type
func (c *Client) CreateEventType(ctx context.Context, service string, instanceID uuid.UUID, tenantID uuid.UUID, eventDef *[]MetricMetadata) (*[]MetricMetadata, error) {
	return c.CreateEventTypesForTenant(ctx, service, instanceID, c.tenantID, eventDef)
}

// CreateEventTypesForTenant create a new event metadata type
func (c *Client) CreateEventTypesForTenant(ctx context.Context, service string, instanceID uuid.UUID, tenantID uuid.UUID, eventDef *[]MetricMetadata) (*[]MetricMetadata, error) {
	requestURL := url.URL{
		Path: EventMetadataPath,
		RawQuery: url.Values{
			InstanceIDQueryArgName: []string{instanceID.String()},
			ServiceQueryArgName:    []string{service},
			TenantIDQueryArgName:   []string{tenantID.String()},
		}.Encode(),
	}

	var res []MetricMetadata
	return &res, ucerr.Wrap(c.client.Post(ctx, requestURL.String(), eventDef, &res))
}

// GetEventTypes gets a list of event sources generating events /instances
func (c *Client) GetEventTypes(ctx context.Context, referenceURL string) (*[]MetricMetadata, error) {
	return c.GetEventTypesTenant(ctx, referenceURL, c.tenantID)
}

// GetEventTypesTenant gets a list of event sources generating events /instances
func (c *Client) GetEventTypesTenant(ctx context.Context, referenceURL string, tenantID uuid.UUID) (*[]MetricMetadata, error) {
	requestURL := url.URL{
		Path: EventMetadataPath,
		RawQuery: url.Values{
			TenantIDQueryArgName: []string{tenantID.String()},
			FilterQueryArgName:   []string{referenceURL},
		}.Encode(),
	}
	var res []MetricMetadata

	if err := c.client.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// DeleteEventTypesForReferenceURL deletes all the custom events types for the reference object
func (c *Client) DeleteEventTypesForReferenceURL(ctx context.Context, instanceID uuid.UUID, referenceURL string) error {
	return ucerr.Wrap(c.DeleteEventTypeForReferenceURLForTenant(ctx, instanceID, referenceURL, c.tenantID))
}

// DeleteEventTypeForReferenceURLForTenant deletes all the custom events types for the reference object for given tenant
func (c *Client) DeleteEventTypeForReferenceURLForTenant(ctx context.Context, instanceID uuid.UUID, referenceURL string, tenantID uuid.UUID) error {
	requestURL := url.URL{
		Path: EventMetadataPath,
		RawQuery: url.Values{
			InstanceIDQueryArgName: []string{instanceID.String()},
			TenantIDQueryArgName:   []string{tenantID.String()},
			FilterQueryArgName:     []string{referenceURL},
		}.Encode(),
	}

	return ucerr.Wrap(c.client.Delete(ctx, requestURL.String(), nil))
}
