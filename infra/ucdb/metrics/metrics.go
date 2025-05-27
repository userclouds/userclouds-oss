package metrics

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"userclouds.com/infra/ucerr"
)

type contextKey int

const (
	ctxDBMetrics contextKey = 1
)

var inFlightCount atomic.Int32

// IncrementInFlightCount increments the number of DB calls in flight
func IncrementInFlightCount() int {
	return int(inFlightCount.Add(1))
}

// DecrementInFlightCount decrements the number of DB calls in flight
func DecrementInFlightCount() int {
	return int(inFlightCount.Add(-1))
}

// DBMetrics keeps track of DB calls during a request
type DBMetrics struct {
	metricsLock     sync.RWMutex
	GetCount        int
	SelectCount     int
	WriteCount      int
	MaxInFlight     int
	GetLatencies    time.Duration
	SelectLatencies time.Duration
	WriteLatencies  time.Duration
}

// GetTotalDuration returns the sum of the time spend calling the DB
func (m *DBMetrics) GetTotalDuration() time.Duration {
	m.metricsLock.RLock()
	defer m.metricsLock.RUnlock()
	return m.GetLatencies + m.SelectLatencies + m.WriteLatencies
}

// GetTotalCalls returns to total number of DB calls
func (m *DBMetrics) GetTotalCalls() int {
	m.metricsLock.RLock()
	defer m.metricsLock.RUnlock()
	return m.GetCount + m.SelectCount + m.WriteCount
}

// GetMaxInFlight returns the maximum number of DB calls in flight during a db operation
func (m *DBMetrics) GetMaxInFlight() int {
	m.metricsLock.RLock()
	defer m.metricsLock.RUnlock()
	return m.MaxInFlight
}

// HadCalls returns true if there were any DB calls
func (m *DBMetrics) HadCalls() bool {
	m.metricsLock.RLock()
	defer m.metricsLock.RUnlock()
	return m.GetTotalCalls() > 0
}

// GetCopy returns a copy of the DB metrics struct
func (m *DBMetrics) GetCopy() DBMetrics {
	m.metricsLock.RLock()
	defer m.metricsLock.RUnlock()
	return DBMetrics{
		GetCount:        m.GetCount,
		SelectCount:     m.SelectCount,
		WriteCount:      m.WriteCount,
		MaxInFlight:     m.MaxInFlight,
		GetLatencies:    m.GetLatencies,
		SelectLatencies: m.SelectLatencies,
		WriteLatencies:  m.WriteLatencies,
	}
}

// GetMetrics returns the DB metrics structure from the context, errors out if it is not there.
func GetMetrics(ctx context.Context) (*DBMetrics, error) {
	val := ctx.Value(ctxDBMetrics)
	metrics, ok := val.(*DBMetrics)
	if !ok {
		return nil, ucerr.Errorf("Can't find DB metric data in context")
	}
	return metrics, nil
}

// InitContext adds a DB metrics struct to the context to allow keeping track of DB calls during a request
func InitContext(ctx context.Context) context.Context {
	val := ctx.Value(ctxDBMetrics)
	if _, ok := val.(*DBMetrics); !ok {
		return context.WithValue(ctx, ctxDBMetrics, &DBMetrics{})
	}
	return ctx
}

// ResetContext resets/adds a DB metrics struct to the context to allow keeping track of DB calls during a request
func ResetContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxDBMetrics, &DBMetrics{})
}

// RecordWrite records a DB write
func RecordWrite(ctx context.Context, duration time.Duration, inFlight int) {
	metricsData, err := GetMetrics(ctx)
	if err != nil {
		return
	}

	metricsData.metricsLock.Lock()
	defer metricsData.metricsLock.Unlock()

	metricsData.WriteCount++
	metricsData.WriteLatencies += duration
	if int(inFlight) > metricsData.MaxInFlight {
		metricsData.MaxInFlight = inFlight
	}
}

// RecordGet records fetching an object from the DB
func RecordGet(ctx context.Context, duration time.Duration, inFlight int) {
	metricsData, err := GetMetrics(ctx)
	if err != nil {
		return
	}

	metricsData.metricsLock.Lock()
	defer metricsData.metricsLock.Unlock()

	metricsData.GetCount++
	metricsData.GetLatencies += duration
	if inFlight > metricsData.MaxInFlight {
		metricsData.MaxInFlight = inFlight
	}
}

// RecordSelect records a select of multiple objects from the DB
func RecordSelect(ctx context.Context, duration time.Duration, inFlight int) {
	metricsData, err := GetMetrics(ctx)
	if err != nil {
		return
	}

	metricsData.metricsLock.Lock()
	defer metricsData.metricsLock.Unlock()

	metricsData.SelectCount++
	metricsData.SelectLatencies += duration
	if inFlight > metricsData.MaxInFlight {
		metricsData.MaxInFlight = inFlight
	}
}
