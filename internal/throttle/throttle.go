package throttle

// This file is based on code from https://github.com/go-chi/chi/blob/master/middleware/throttle.go and is licensed under the MIT license.

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/queuemetrics"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucmetrics"
	"userclouds.com/internal/multitenant"
)

const throttleSubsystem = ucmetrics.Subsystem("throttle")

var (
	inFlightQueueSize = ucmetrics.CreateGauge(throttleSubsystem, "in_flight_queue_size", "Number of in-flight requests", "tenant_id")
	backlogQueueSize  = ucmetrics.CreateGauge(throttleSubsystem, "backlog_queue_size", "Number of requests in the backlog")
)

// ErrCapacityExceeded is returned when the server is at capacity and cannot accept any more requests.
var ErrCapacityExceeded = ucerr.Friendlyf(nil, "Server capacity exceeded.")

// ErrTimedOut is returned when a request times out while waiting in the queue.
var ErrTimedOut = ucerr.Friendlyf(nil, "Timed out while waiting for a pending request to complete.")

// ErrContextCanceled is returned when the context is canceled while waiting for a request to complete.
var ErrContextCanceled = ucerr.Friendlyf(nil, "Context was canceled.")

// Opts represents a set of throttling options.
type Opts struct {
	RetryAfterFn        func(ctxDone bool) time.Duration
	InflightLimitTenant int           // number of inflight requests per tenant
	BacklogLimitTenant  int           // number of requests that can be queued up per tenant
	BacklogLimit        int           // number of requests that can be queued up in total
	BacklogTimeout      time.Duration // maximum time a request can be queued before being rejected
}

// token represents a place holder representing a single in flight or backlogged request
type token struct{}

// throttler limits number of currently processed requests at a time.
type throttler struct {
	backlogTokens          chan queueItem                 // channel for holding the backlog of tokens representing requests
	perTenantQueues        map[uuid.UUID]*perTenantRecord // per tenant queues for inflight tokens and in order request queue
	retryAfterFn           func(ctxDone bool) time.Duration
	backlogTimeout         time.Duration
	perTenantInflightLimit int // number of inflight requests per tenant (controls the size of perTenantQueues[ID].inflightTokens)
	perTenantBacklogLimit  int // number of requests that can be queued up per tenant (effectivily control size of perTenantQueues[ID].requestQueue and what portion of backlogTokens is used for a tenant)
	sync.RWMutex
}

// queueItem represents a request that is waiting to be processed
type queueItem struct {
	respChannel chan token
}

// perTenantRecord holds the inflight tokens and request queue for a tenant
type perTenantRecord struct {
	inflightTokens chan token
	requestQueue   chan queueItem
}

// LimitWithBacklogQueue is a middleware that limits number of currently processed
// requests at a time and provides a backlog for holding a finite number of
// pending requests.
func LimitWithBacklogQueue(ctx context.Context, limitInflightTenant, limitBacklogTenant, backlogLimit int, backlogTimeout time.Duration) middleware.Middleware {
	return LimitWithOptsQueue(ctx, Opts{InflightLimitTenant: limitInflightTenant, BacklogLimitTenant: limitBacklogTenant, BacklogLimit: backlogLimit, BacklogTimeout: backlogTimeout})
}

// getPerTenantQueues retrieves the per tenant queues for a given tenant ID and initializes them if needed
func getPerTenantQueues(thrttlr *throttler, tenantID uuid.UUID) *perTenantRecord {
	var queues *perTenantRecord
	thrttlr.RLock()
	queues, ok := thrttlr.perTenantQueues[tenantID]
	thrttlr.RUnlock()

	// Tenant is already setup so return the queues
	if ok {
		return queues
	}

	thrttlr.Lock()
	if queues, ok = thrttlr.perTenantQueues[tenantID]; !ok {
		tokens := make(chan token, thrttlr.perTenantInflightLimit)

		// Fill tokens into throttling to ensure on X requests at a time are allowed
		for range thrttlr.perTenantInflightLimit {
			tokens <- token{}
		}

		// Create a request queue for this tenant to enforce ordering of incoming requests
		requestQueue := make(chan queueItem, thrttlr.perTenantBacklogLimit)
		queues = &perTenantRecord{inflightTokens: tokens, requestQueue: requestQueue}
		thrttlr.perTenantQueues[tenantID] = queues
		tenantIDStr := tenantID.String()
		// Start a goroutine to process the request queue for this tenant
		go func() {
			for queueItem := range requestQueue {
				tok := <-tokens
				queueItem.respChannel <- tok
				inFlightQueueSize.WithLabelValues(tenantIDStr).Dec()
			}
			uclog.Debugf(context.Background(), "throttle: request queue for tenant %s closed", tenantID)
		}()
	}
	thrttlr.Unlock()

	return queues
}

// LimitWithOptsQueue is a middleware that limits number of currently processed requests using passed ThrottleOpts.
func LimitWithOptsQueue(ctx context.Context, opts Opts) middleware.Middleware {
	if opts.InflightLimitTenant < 1 {
		uclog.Fatalf(ctx, "middleware: Throttle expects limit > 0")
	}

	if opts.BacklogLimit < opts.InflightLimitTenant {
		uclog.Fatalf(ctx, "middleware: Throttle expects backlogLimit to be to be greater then or equal to the InflightLimitTenant")
	}

	if opts.BacklogLimitTenant < opts.InflightLimitTenant {
		uclog.Fatalf(ctx, "middleware: Throttle expects BacklogLimitTenant to be greater then or equal to the InflightLimitTenant")
	}

	thrttlr := throttler{
		backlogTokens:   make(chan queueItem, opts.BacklogLimit),
		perTenantQueues: make(map[uuid.UUID]*perTenantRecord),
		backlogTimeout:  opts.BacklogTimeout,
		retryAfterFn:    opts.RetryAfterFn,
		// TODO this values can be read of Tenant config so that they can vary by tenant
		perTenantInflightLimit: opts.InflightLimitTenant,
		perTenantBacklogLimit:  opts.BacklogLimitTenant,
	}

	// Fill tokens limiting the total number of requests the process will queue (note each tenant will add additional InflightLimitTenant number of tokens to the backlogTokens)
	for range opts.BacklogLimit {
		// Create a channel to receive the response
		respChannel := make(chan token)
		// Create a queue item with the response channel
		queueItem := queueItem{respChannel: respChannel}
		// Send the queue item to the response queue
		thrttlr.backlogTokens <- queueItem
	}
	backlogQueueSize.WithLabelValues().Add(float64(opts.BacklogLimit))

	return middleware.Func(func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			startTime := time.Now().UTC()

			ts := multitenant.MustGetTenantState(ctx)
			tsID := ts.ID.String()

			queues := getPerTenantQueues(&thrttlr, ts.ID)

			select {

			case <-ctx.Done():
				thrttlr.setRetryAfterHeaderIfNeeded(w, true)
				uchttp.Error(ctx, w, ErrCapacityExceeded, http.StatusTooManyRequests)
				return

			// Do a non-blocking select and if it fails, reject the request immediately
			case workItem := <-thrttlr.backlogTokens:
				backlogQueueSize.WithLabelValues().Dec()
				timedOut := false
				timer := time.NewTimer(thrttlr.backlogTimeout) // start the timer for the timeout of request in the backlog queue

				defer func() {
					if timedOut {
						go func() {
							// If the request times out while in the queue, the worker thread will still eventually get the expired request
							// this means that if the queue is at the limit and the request times out, we don't start accepting new requests until worker thread cleared the expired requests
							tok := <-workItem.respChannel
							queues.inflightTokens <- tok
							inFlightQueueSize.WithLabelValues(tsID).Inc()
							thrttlr.backlogTokens <- workItem
							backlogQueueSize.WithLabelValues().Inc()

						}()
					} else {
						thrttlr.backlogTokens <- workItem
						backlogQueueSize.WithLabelValues().Inc()
					}
				}()

				// Send the queue item to the per tenant request queue where the worker thread will pick it up
				// Once the item is picked up which means it now can proceed, the worker thread will send the token back on workItem.respChannel channel

				select {
				// Do a non-blocking select and if it fails, reject the request immediately
				case queues.requestQueue <- workItem:
					select {
					case <-timer.C:
						timedOut = true
						thrttlr.setRetryAfterHeaderIfNeeded(w, false)
						uchttp.Error(ctx, w, ErrTimedOut, http.StatusTooManyRequests)
						return
					case <-ctx.Done():
						timedOut = true
						timer.Stop()
						thrttlr.setRetryAfterHeaderIfNeeded(w, true)
						uchttp.Error(ctx, w, ErrContextCanceled, http.StatusTooManyRequests)
						return
					case tok := <-workItem.respChannel:
						defer func() {
							timer.Stop()
							queues.inflightTokens <- tok
							inFlightQueueSize.WithLabelValues(tsID).Inc()
						}()
						ctx = queuemetrics.SetQueueStats(ctx, time.Since(startTime)) // record the request counts in the context
						next.ServeHTTP(w, r.WithContext(ctx))
					}
				default:
					thrttlr.setRetryAfterHeaderIfNeeded(w, false)
					uchttp.Error(ctx, w, ErrCapacityExceeded, http.StatusTooManyRequests)
					return

				}
				return

			default:
				thrttlr.setRetryAfterHeaderIfNeeded(w, false)
				uchttp.Error(ctx, w, ErrCapacityExceeded, http.StatusTooManyRequests)
				return
			}
		}

		return http.HandlerFunc(fn)
	})
}

// setRetryAfterHeaderIfNeeded sets Retry-After HTTP header if corresponding retryAfterFn option of throttler is initialized.
func (thrttlr *throttler) setRetryAfterHeaderIfNeeded(w http.ResponseWriter, ctxDone bool) {
	if thrttlr.retryAfterFn == nil {
		return
	}
	w.Header().Set("Retry-After", strconv.Itoa(int(thrttlr.retryAfterFn(ctxDone).Seconds())))
}
