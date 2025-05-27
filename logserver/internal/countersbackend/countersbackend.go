package countersbackend

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/logserver/internal/storage"
)

// GC interval
const (
	writeInterval       time.Duration = 60 * time.Second
	aggregationInterval time.Duration = 60 * time.Second
)

type logEventRecord struct {
	eventCode uclog.EventCode
	count     int
	timestamp time.Time
}

type logEventRecords struct {
	logEvents []*logEventRecord
	tenantID  uuid.UUID
	service   service.Service
}

// CountersStore exposed the backend for aggregating and writing counter values to the DB
type CountersStore struct {
	counters    map[uuid.UUID][]*logEventRecords
	tenantCache *storage.TenantCache
	mutex       sync.Mutex
	writeTicker time.Ticker
	done        chan bool
}

// CounterQuery describes a query for metrics from a single tenant
// TODO: unify with logserver client models
type CounterQuery struct {
	TenantID  uuid.UUID         `json:"tenant_id" yaml:"tenant_id"`
	Service   service.Service   `json:"service" yaml:"service"`
	EventCode []uclog.EventCode `json:"event_codes" yaml:"event_codes"`
	Start     time.Time         `json:"start" yaml:"start"`
	End       time.Time         `json:"end" yaml:"end"`
	Period    time.Duration     `json:"period" yaml:"period"`
}

// CounterData provides data for a single counter including 0 for periods where there is no data
type CounterData struct {
	EventCode uclog.EventCode `json:"event_code" yaml:"event_code"`
	Start     time.Time       `json:"start" yaml:"start"`
	Period    time.Duration   `json:"period" yaml:"period"`
	Counts    []int           `json:"counts" yaml:"counts"`
}

// CounterQueryResponse contains the response to a single query
type CounterQueryResponse struct {
	TenantID uuid.UUID     `json:"tenant_id" yaml:"tenant_id"`
	Rows     []CounterData `json:"rows" yaml:"rows"`
}

// CountQuery is the request for a event count
type CountQuery struct {
	ID        uuid.UUID         `json:"id" yaml:"id"`
	TenantID  uuid.UUID         `json:"tenant_id" yaml:"tenant_id"`
	Service   service.Service   `json:"service" yaml:"service"`
	EventCode []uclog.EventCode `json:"event_codes" yaml:"event_codes"`
	Start     time.Time         `json:"start" yaml:"start"`
	End       time.Time         `json:"end" yaml:"end"`
}

// CountQueryResponse is the response for a count query
type CountQueryResponse struct {
	EventCode []uclog.EventCode `json:"event_code" yaml:"event_code"`
	Start     time.Time         `json:"start" yaml:"start"`
	End       time.Time         `json:"end" yaml:"end"`
	Count     int               `json:"count" yaml:"count"`
}

// NewCounterStore gets an instance of CountersStore
func NewCounterStore(ctx context.Context, s *storage.TenantCache) (*CountersStore, error) {
	var c CountersStore
	// Create a cache for storing counters. TODO this emulates a global in memory key/value pair store
	// (like memcache) that should be same across multiple instance of LogServer
	c.counters = make(map[uuid.UUID][]*logEventRecords)
	// Create mutex to protect read and write access to the map since it is not thread safe
	c.mutex = sync.Mutex{}

	c.tenantCache = s

	// Initialize a timer to write counts pooled over multiple plex instances (potentially for same tenant) to the DB
	c.writeTicker = *time.NewTicker(writeInterval)
	c.done = make(chan bool)
	go func() {
		for {
			select {
			case <-c.done:
				return
			case <-c.writeTicker.C:
				c.writeCounters(ctx)
			}
		}
	}()

	return &c, nil
}

func (c *CountersStore) writeCounters(ctx context.Context) {
	// Swap the table out so the writes are not blocked while DB writes to the backend are happening
	c.mutex.Lock()
	countersMap := c.counters
	c.counters = make(map[uuid.UUID][]*logEventRecords)
	c.mutex.Unlock()
	for _, companyCounters := range countersMap {
		// All the writes for a single companyCounter go to the same DB
		for _, envCounters := range companyCounters {
			// All the writes for each envCounter go to the same table
			var queryBuffer bytes.Buffer
			tableName := "metrics_" + string(envCounters.service)
			queryBuffer.WriteString("/* lint-sql-ok */ INSERT INTO " + tableName + "(id, type, timestamp, count) VALUES ")
			var first = true
			for _, counter := range envCounters.logEvents {
				if !first {
					queryBuffer.WriteString(", ")
				}
				queryBuffer.WriteString(fmt.Sprintf("(0, %d, %d, %d)", counter.eventCode, counter.timestamp.Unix(), counter.count))
				first = false
			}
			queryBuffer.WriteString(";")

			s, err := c.tenantCache.GetStorageForTenant(ctx, envCounters.tenantID)
			if err != nil {
				uclog.Warningf(ctx, "Failed to connect to DB for tenant %v with %v", envCounters.tenantID, err)
				continue
			}
			if err := s.WriteCounters(ctx, queryBuffer.String()); err != nil {
				// TODO errors are silently ignored, which is really safe from contention/overload perspective
				// but leads to data loss due intermittent failures. Consider retrying a minute later with back off
				uclog.Warningf(ctx, "Failed writing counters to DB %s with %v", queryBuffer.String(), err)
			}
		}
	}
}

// UpdateCounters records one counter value to be written to the backend
func (c *CountersStore) UpdateCounters(ctx context.Context,
	tenantID uuid.UUID,
	service service.Service,
	eventCode uclog.EventCode,
	eventTime time.Time, count int) error {

	var tenantCounters []*logEventRecords
	var serviceCounter *logEventRecords
	var counters []*logEventRecord
	var counterFound bool
	var ok bool

	c.mutex.Lock()
	defer c.mutex.Unlock()

	tenantCounters, ok = c.counters[tenantID]
	if ok {
		for _, serviceCounter = range tenantCounters {
			if serviceCounter.service == service {
				counters = serviceCounter.logEvents
				break
			}
		}
	} else {
		// We haven't seen any counters for this company in the last collection period so create new entry
		c.counters[tenantID] = make([]*logEventRecords, 0, 3)
	}
	// If we haven't seen any counters for this tenant_id/service combination create an entry for it
	if counters == nil {
		serviceCounter = &logEventRecords{service: service, tenantID: tenantID, logEvents: make([]*logEventRecord, 0, 5)}
		c.counters[tenantID] = append(c.counters[tenantID], serviceCounter)
		counters = serviceCounter.logEvents
	}
	// If we have seen this event - increment the count for it
	for i := range counters {
		if counters[i].eventCode == eventCode && counters[i].timestamp.Sub(eventTime) < aggregationInterval {
			counters[i].count = counters[i].count + count
			counterFound = true
		}
	}
	// If we haven't seen this particular event type - add an entry for it
	if !counterFound {
		counter := &logEventRecord{eventCode: eventCode, count: count, timestamp: eventTime}
		serviceCounter.logEvents = append(counters, counter)
	}

	return nil
}

func (c *CountersStore) getTableNameFromService(s service.Service) (string, error) {
	// This is a concat of two constants
	return fmt.Sprintf("metrics_%s", s), nil
}

// QueryCount reads the aggregated count from events over a given window
func (c *CountersStore) QueryCount(ctx context.Context, q CountQuery) (*CountQueryResponse, error) {
	var rC CountQueryResponse
	tableName, err := c.getTableNameFromService(q.Service)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// All the reads for metrics against [tenant_id, service] go to the same table, so construct a single query
	var queryBuffer bytes.Buffer
	queryBuffer.WriteString("/* lint-deleted */ SELECT COALESCE(sum(count), 0) FROM " + tableName + " WHERE timestamp > ") // TODO: these queries should be codegen'd
	queryBuffer.WriteString(fmt.Sprintf("%d AND timestamp < %d AND type IN ( ", q.Start.Unix(), q.End.Unix()))

	var first = true
	for _, eventType := range q.EventCode {
		if !first {
			queryBuffer.WriteString(", ")
		}
		queryBuffer.WriteString(fmt.Sprintf("%d", eventType))
		first = false
	}
	queryBuffer.WriteString(");")

	store, err := c.tenantCache.GetStorageForTenant(ctx, q.TenantID)
	if err != nil {
		uclog.Warningf(ctx, "Failed to connect to log DB reading raw log for tenant %v with %v", q.TenantID, err)
	}

	// Execute the query getting the rows from the metrics table
	var r *int
	if r, err = store.ReadCount(ctx, queryBuffer.String()); err != nil {
		return nil, ucerr.Wrap(err)
	}

	rC = CountQueryResponse{
		EventCode: q.EventCode,
		Start:     q.Start,
		End:       q.End,
		Count:     *r,
	}

	return &rC, nil
}

// QueryCounters reads the counters from the table in the DB and fills in data for given window
func (c *CountersStore) QueryCounters(ctx context.Context, qA []CounterQuery) (*[]CounterQueryResponse, error) {
	var mR []CounterQueryResponse
	for rI, q := range qA {
		tableName, err := c.getTableNameFromService(q.Service)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		// All the reads for metrics against [tenant_id, service] go to the same table, so construct a single query
		var queryBuffer bytes.Buffer
		queryBuffer.WriteString("/* lint-deleted */ SELECT * FROM " + tableName + " WHERE timestamp > ") // TODO: these queries should be codegen'd
		queryBuffer.WriteString(fmt.Sprintf("%d AND timestamp < %d AND type IN ( ", q.Start.Unix(), q.End.Unix()))
		var first = true
		for _, eventType := range q.EventCode {
			if !first {
				queryBuffer.WriteString(", ")
			}
			queryBuffer.WriteString(fmt.Sprintf("%d", eventType))
			first = false
		}
		queryBuffer.WriteString(");")

		s, err := c.tenantCache.GetStorageForTenant(ctx, q.TenantID)
		if err != nil {
			uclog.Warningf(ctx, "Failed to connect to log DB to write events for tenant %v with %v", q.TenantID, err)
			continue
		}

		// Execute the query getting the rows from the metrics table
		r, err := s.ReadCounters(ctx, queryBuffer.String())
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		// On success aggregate the data according to the input
		mR = append(mR, CounterQueryResponse{TenantID: q.TenantID, Rows: make([]CounterData, len(q.EventCode))})

		// Calculate the number of periods in the return data
		endTime := q.End.Unix()
		startTime := q.Start.Unix()
		fPeriodCount := float64(endTime-startTime) / float64(q.Period.Seconds())
		periodCount := int(fPeriodCount)
		if math.Trunc(fPeriodCount) != fPeriodCount {
			periodCount++
		}

		// While periodCount is validated in the handler - add extra validation since this API is exported
		if periodCount > 60 || periodCount < 1 {
			return nil, ucerr.Errorf("Too many or too few time periods specified for counter read - %d", periodCount)
		}

		// Fill in 0 counts for all counters and periods
		var counts = make(map[uclog.EventCode]*[]int, len(q.EventCode))
		for i := range mR[rI].Rows {
			mR[rI].Rows[i].EventCode = q.EventCode[i]
			mR[rI].Rows[i].Period = q.Period
			mR[rI].Rows[i].Start = q.Start
			mR[rI].Rows[i].Counts = make([]int, periodCount)
			counts[q.EventCode[i]] = &(mR[rI].Rows[i].Counts)
		}

		// Fill in periods for which we had activity combining rows within a single time period
		for _, e := range *r {
			// Calculate the period the event falls into
			pC := int(float64(e.Timestamp-startTime) / float64(q.Period.Seconds()))
			// Look up the counter array by event type and increment the count
			c := counts[e.EventCode]
			(*c)[pC] = (*c)[pC] + e.Count
		}
	}
	// Return a set of responses
	return &mR, nil
}

// QueryCountersLog reads the counters from the table in the DB and fills in a raw log
func (c *CountersStore) QueryCountersLog(ctx context.Context, tenantID uuid.UUID, services []service.Service, eventsToFetch int) (*[]storage.MetricsRow, error) {
	var rC []storage.MetricsRow
	for _, s := range services {
		tableName, err := c.getTableNameFromService(s)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		// All the reads for metrics against [tenant_id, service] go to the same table, so construct a single query
		var queryBuffer bytes.Buffer
		// order by timestamp desc limit 50;
		queryBuffer.WriteString("/* lint-deleted */ SELECT * FROM " + tableName + " ORDER BY timestamp DESC ") // TODO: these queries should be codegen'd
		queryBuffer.WriteString(fmt.Sprintf("LIMIT %d;", eventsToFetch))

		s, err := c.tenantCache.GetStorageForTenant(ctx, tenantID)
		if err != nil {
			uclog.Warningf(ctx, "Failed to connect to log DB reading raw log for tenant %v with %v", tenantID, err)
			continue
		}

		// Execute the query getting the rows from the metrics table
		var r *[]storage.MetricsRow
		if r, err = s.ReadCounters(ctx, queryBuffer.String()); err != nil {
			return nil, ucerr.Wrap(err)
		}
		rC = append(rC, *r...)
	}
	// Return a set of events
	return &rC, nil
}

// Validate ensures that the counter query is valid
func (q *CounterQuery) Validate() error {
	// TODO Add tests
	if !service.IsValid(q.Service) {
		return ucerr.Errorf("Invalid service type specified")
	}
	if len(q.EventCode) == 0 {
		return ucerr.Errorf("Too few event types for counter read - %d", len(q.EventCode))
	}
	if len(q.EventCode) > 100 {
		return ucerr.Errorf("Too many event types for counter read - %d", len(q.EventCode))
	}

	if q.Period.Seconds() < 1 {
		return ucerr.Errorf("Period is too small for counter read - %d", q.Period.Seconds())
	}

	// Validate the number of periods is less than max and start and end time are in order
	endTime := q.End.Unix()
	startTime := q.Start.Unix()

	if endTime <= startTime {
		return ucerr.Errorf("End time has to be greate than start time counter read")
	}

	fPeriodCount := float64(endTime-startTime) / float64(q.Period.Seconds())
	periodCount := int(fPeriodCount)
	if math.Trunc(fPeriodCount) != fPeriodCount {
		periodCount++
	}
	if periodCount > 60 {
		return ucerr.Errorf("Too many time periods specified for counter read - %d", periodCount)
	}

	return nil
}

// Validate ensures that the count query is valid
func (q *CountQuery) Validate() error {
	if !service.IsValid(q.Service) {
		return ucerr.Errorf("Invalid service type specified")
	}
	if len(q.EventCode) == 0 {
		return ucerr.Errorf("Too few event types for counter read - %d", len(q.EventCode))
	}
	if len(q.EventCode) > 100 {
		return ucerr.Errorf("Too many event types for counter read - %d", len(q.EventCode))
	}

	return nil
}
