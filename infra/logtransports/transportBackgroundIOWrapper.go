package logtransports

// A transport wrapper that moves the IO operation to a background thread by accumulating logged data and flushing it
// on a given time interval

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Queue size limits to for when the writer thread falls behind the event transformation.
// At these values:
//
//     Since we use a single mutex to guard insertion into the queue - the insertion point is effectively single threaded. This means the
//     max insertion speed is limited to how quickly a single thread can create LogRecords and append them to the queue. The transport threads
//     read the queue every 100 ms and will fall behind if the number of events inserted takes longer than 100 ms to process. The numbers below
//     control what happens if the transport thread falls behind and events are dropped to protect the process from running out of memory.
//     Below is the analysis for how much load each transport can handle:
//
//     LogServerTransport can process up to 5,000,0000 events a second for events without payload without dropping any of them. Because
//     events without payload are aggregated into batches, regardless of their total number they result in one call to logserver a second
//     which posts at most one counter for each event in the map for events without payload. Each event with unique payload results in a
//     separate entry, but since they are compact we can accumulate a really large batch without problems. If we start generating more of these
//     events we'll need to send them more than once a second, but for now there is no issue. There is no back off on posting message via
//     LogServerTransport so messages just accumulate until max of 200,000 and then get dropped, since we don't use this code path today
//     (it is meant for code running on customer's machines) - I left it as is.
//
//
//     FileTransport can process up to 1,5000,000 log lines a second without dropping any events. This translates to about 250MB a sec or
//     15 Gb every minute. At this rate we would fill up the disk in a few minutes. It will start dropping events if we log them faster,
//     than a single thread can write them to disk. If that ever becomes a problem we will need to make that transport multithreaded and
//     stripe the file.
//
//     GoTransport is synchronous and can process about 100,000 log lines a second before blocking the calling code. We should
//
//     Kinesis logger can process 3,500 log lines a second outside DC and 35,000 log lines a second for in region stream in DC. This is by far
//     the lowest limit and it is driven by the size of the batch 500 items and roundtrip cost to the kinesis server. Two mechanisms for increasing
//     the throughput rate are packing more log lines into each batch (will add shortly) and adding multiple threads to make multiple
//     post to kinesis at the same time (will add only if needed)

const (
	debugBackOffSize   = 2000000
	infoBackOffSize    = 3500000
	warningBackoffSize = 5000000
	errorBackOffSize   = 5500000
	maxQueueSize       = 6000000
)

type logRecord struct {
	timestamp time.Time
	event     uclog.LogEvent
	next      *logRecord
}

// wrappedIOTransport defines the interface for wrapped transport that performs IO on background thread
type wrappedIOTransport interface {
	init(ctx context.Context) (*uclog.TransportConfig, error)
	writeMessages(ctx context.Context, logRecords *logRecord, startTime time.Time, count int)
	getIOInterval() time.Duration
	getMaxLogLevel() uclog.LogLevel
	getTransportName() string
	supportsCounters() bool
	flushIOResources()
	closeIOResources()
	getFailedAPICallsCount() int64
}

type transportBackgroundIOWrapper struct {
	wrapped           wrappedIOTransport
	transportName     string
	writeMutex        sync.Mutex
	postMutext        sync.Mutex
	diskRecords       *logRecord
	writeTicker       time.Ticker
	queueSize         int64
	droppedEventCount int64
	sentEventCount    int64
	runningBGThread   bool
	done              chan bool
	exitBG            chan bool
	flushChan         chan bool
}

// newTransportBackgroundIOWrapper returns a wrapper around a uninitialized wrappedIOTransport
func newTransportBackgroundIOWrapper(wrapped wrappedIOTransport) *transportBackgroundIOWrapper {
	return &transportBackgroundIOWrapper{
		wrapped:       wrapped,
		transportName: wrapped.getTransportName(),
	}
}

func (t *transportBackgroundIOWrapper) Init() (*uclog.TransportConfig, error) {
	ctx := context.Background() // TODO we may want to create unique context for background operations
	c, err := t.wrapped.init(ctx)
	if err != nil {
		return c, ucerr.Wrap(err)
	}
	t.queueSize = 0
	// Launch the file writer thread to prevent excessive disk seeks when many threads log at once
	t.writeTicker = *time.NewTicker(t.wrapped.getIOInterval())
	t.done = make(chan bool)
	t.exitBG = make(chan bool)
	t.flushChan = make(chan bool)
	go func() {
		t.runningBGThread = true
		for {
			select {
			case <-t.done:
				t.writeMutex.Lock()
				t.writeToIO(ctx)
				t.wrapped.closeIOResources()
				t.runningBGThread = false
				t.writeMutex.Unlock()
				t.exitBG <- true
				return
			case <-t.flushChan:
				t.writeMutex.Lock()
				t.writeToIO(ctx)
				t.wrapped.flushIOResources()
				t.writeMutex.Unlock()
			case <-t.writeTicker.C:
				t.writeMutex.Lock()
				t.writeToIO(ctx)
				t.writeMutex.Unlock()
			}
		}
	}()

	return c, nil
}

func (t *transportBackgroundIOWrapper) writeToIO(ctx context.Context) {
	var records *logRecord

	t.postMutext.Lock()
	records = t.diskRecords
	t.diskRecords = nil
	t.postMutext.Unlock()

	// Reverse the records since they are Last -> First order and count them
	var next *logRecord
	var recordCount = 0
	var startTime time.Time

	// Grab the earliest time in the batch
	if records != nil {
		startTime = records.timestamp
	}
	// Reverse the records
	for records != nil {
		tmp := records.next
		records.next = next
		next = records
		records = tmp
		recordCount++
	}
	records = next
	qs := float64(atomic.AddInt64(&t.queueSize, int64(-recordCount)))
	queueSize.WithLabelValues(t.transportName).Set(qs)
	atomic.AddInt64(&t.sentEventCount, int64(recordCount))
	sentEventCount.WithLabelValues(t.transportName).Add(float64(recordCount))
	t.wrapped.writeMessages(ctx, records, startTime, recordCount)
}

func (t *transportBackgroundIOWrapper) queueCapacityBackoff() uclog.LogLevel {
	// Default case when the queue is not overloaded
	queueSize := atomic.LoadInt64(&t.queueSize)
	if queueSize < debugBackOffSize {
		return uclog.LogLevelVerbose
	}
	if queueSize < infoBackOffSize {
		return uclog.LogLevelDebug
	}
	if queueSize < warningBackoffSize {
		return uclog.LogLevelInfo
	}
	if queueSize < errorBackOffSize {
		return uclog.LogLevelWarning
	}
	if queueSize < maxQueueSize {
		return uclog.LogLevelError
	}
	// Drop the message
	return uclog.LogLevelNone
}

func (t *transportBackgroundIOWrapper) writeLogRecord(event *uclog.LogEvent) {
	// Check if the queue has space for this event to protect from OOM when the writer is too slow to keep up
	bL := t.queueCapacityBackoff()
	if bL < event.LogLevel || bL <= uclog.LogLevelWarning && event.LogLevel == uclog.LogLevelNonMessage {
		atomic.AddInt64(&t.droppedEventCount, 1)
		droppedCalls.WithLabelValues(string(t.transportName)).Inc()
		return
	}

	// Append the record to the front of the linked list
	t.postMutext.Lock()
	var record = logRecord{time.Now().UTC(), *event, t.diskRecords}
	t.diskRecords = &record
	qs := atomic.AddInt64(&t.queueSize, 1)
	t.postMutext.Unlock()
	queueSize.WithLabelValues(t.transportName).Set(float64(qs))
}

func (t *transportBackgroundIOWrapper) Write(ctx context.Context, event uclog.LogEvent) {
	if t.wrapped.supportsCounters() {
		t.writeLogRecord(&event)
	} else if event.Message != "" && event.LogLevel <= t.wrapped.getMaxLogLevel() {
		t.writeLogRecord(&event)
	}
}

func (t *transportBackgroundIOWrapper) GetStats() uclog.LogTransportStats {
	queueSize := atomic.LoadInt64(&t.queueSize)
	sentEventCount := atomic.LoadInt64(&t.sentEventCount)
	droppedEventCount := atomic.LoadInt64(&t.droppedEventCount)
	return uclog.LogTransportStats{
		Name:                t.transportName,
		QueueSize:           queueSize,
		DroppedEventCount:   droppedEventCount,
		SentEventCount:      sentEventCount,
		FailedAPICallsCount: t.wrapped.getFailedAPICallsCount(),
	}

}

func (t *transportBackgroundIOWrapper) GetName() string {
	return t.transportName
}

func (t *transportBackgroundIOWrapper) Flush() error {
	if t.runningBGThread {
		t.flushChan <- true
	}
	return nil
}

func (t *transportBackgroundIOWrapper) Close() {
	if t.runningBGThread {
		// Send signal to background thread to perform final flush
		t.done <- true
		// Wait for the flush to finish
		<-t.exitBG
	}
}
