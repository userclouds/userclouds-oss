package throttle

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/queuemetrics"
	"userclouds.com/infra/request"
	"userclouds.com/infra/uclog/responsewriter"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
)

func TestMiddleware(t *testing.T) {
	ctx := context.Background()
	mw := LimitWithBacklogQueue(ctx, 2 /* in flight per tenant */, 4 /*total per tenant*/, 9 /*total max*/, 50*time.Millisecond /*max wait time*/)

	h := func(code int, sleep time.Duration) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(sleep)
			w.WriteHeader(code)
			fmt.Fprintf(w, "%d", code)
		})
	}

	tenantID1 := uuid.Must(uuid.NewV4())
	tenantID2 := uuid.Must(uuid.NewV4())
	tenantID3 := uuid.Must(uuid.NewV4())

	// Make sure nothing crashed if we try to get stats when they were not set
	waitTime := queuemetrics.GetQueueStats(ctx)
	assert.Equal(t, waitTime, time.Duration(0))

	// Check simple case of initializing the middleware for multiple tenants
	ctx = multitenant.SetTenantState(ctx, &tenantmap.TenantState{ID: tenantID1})
	ctx = queuemetrics.InitContext(ctx)
	for _, code := range []int{http.StatusOK, http.StatusInternalServerError} {
		handler := mw.Apply(h(code, 0))
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		lrw := responsewriter.StatusResponseWriter{ResponseWriter: rr}
		handler.ServeHTTP(&lrw, req)
		assert.Equal(t, lrw.StatusCode, code)
	}

	waitTime = queuemetrics.GetQueueStats(ctx)
	assert.NotEqual(t, waitTime, time.Duration(0))

	ctx = multitenant.SetTenantState(ctx, &tenantmap.TenantState{ID: tenantID2})
	for _, code := range []int{http.StatusOK, http.StatusInternalServerError} {
		handler := mw.Apply(h(code, 0))
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		lrw := responsewriter.StatusResponseWriter{ResponseWriter: rr}
		handler.ServeHTTP(&lrw, req)
		assert.Equal(t, lrw.StatusCode, code)
	}

	// Check timeout case - we expect two requests to time out while in the backlog since we allow 2 inflight requests and 4 in the backlog
	workerSleep := 100 * time.Millisecond
	wg := sync.WaitGroup{}
	var failureCount atomic.Int32
	for _, code := range []int{http.StatusOK, http.StatusAccepted, http.StatusAlreadyReported, http.StatusInternalServerError} {
		wg.Add(1)
		ctx := queuemetrics.InitContext(request.SetRequestIDIfNotSet(multitenant.SetTenantState(ctx, &tenantmap.TenantState{ID: tenantID1}), uuid.Must(uuid.NewV4())))
		go func(ctx context.Context) {
			defer wg.Done()
			handler := mw.Apply(h(code, workerSleep))
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			lrw := responsewriter.StatusResponseWriter{ResponseWriter: rr}
			handler.ServeHTTP(&lrw, req)
			if lrw.StatusCode != code {
				failureCount.Add(1)
				assert.Equal(t, lrw.StatusCode, http.StatusTooManyRequests)
			}
		}(ctx)
	}
	wg.Wait()
	assert.Equal(t, failureCount.Load(), int32(2))

	// Try to queue too many requests for the same tenant - we expect the first 4 to be queued and the rest to be rejected
	mw = LimitWithBacklogQueue(ctx, 2 /* in flight per tenant */, 4 /*total per tenant*/, 9 /*total max*/, 1000*time.Millisecond /*max wait time*/)
	wg = sync.WaitGroup{}
	failureCount.Swap(0)
	for i := range 8 { // 2 inflight + 1 on the worker thread waiting for inflight slot + 4 in the backlog = 7 and the 8th will be rejected
		wg.Add(1)
		ctx := queuemetrics.InitContext(request.SetRequestIDIfNotSet(multitenant.SetTenantState(ctx, &tenantmap.TenantState{ID: tenantID1}), uuid.Must(uuid.NewV4())))

		// Take a short pause to ensure that the worker thread picks up 2 requests from the backlog queue into the inflight queue
		if i == 4 {
			time.Sleep(time.Millisecond)
		}

		go func(ctx context.Context) {
			defer wg.Done()
			handler := mw.Apply(h(http.StatusOK, 20*time.Millisecond))
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			lrw := responsewriter.StatusResponseWriter{ResponseWriter: rr}
			handler.ServeHTTP(&lrw, req)
			if lrw.StatusCode != http.StatusOK {
				failureCount.Add(1)
				assert.Equal(t, lrw.StatusCode, http.StatusTooManyRequests)
			}
		}(ctx)
	}
	wg.Wait()
	assert.Equal(t, failureCount.Load(), int32(1))

	// Try to exceed the overall queue limit - we expect the first 9 to be queued and the rest to be rejected
	contexts := []context.Context{
		multitenant.SetTenantState(ctx, &tenantmap.TenantState{ID: tenantID1}),
		multitenant.SetTenantState(ctx, &tenantmap.TenantState{ID: tenantID2}),
		multitenant.SetTenantState(ctx, &tenantmap.TenantState{ID: tenantID3}),
	}
	wg = sync.WaitGroup{}
	failureCount.Swap(0)
	for i := range 12 {
		wg.Add(1)
		ctx := queuemetrics.InitContext(request.SetRequestIDIfNotSet(contexts[i%len(contexts)], uuid.Must(uuid.NewV4())))
		go func(ctx context.Context) {
			defer wg.Done()
			handler := mw.Apply(h(http.StatusOK, 20*time.Millisecond))
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			lrw := responsewriter.StatusResponseWriter{ResponseWriter: rr}
			handler.ServeHTTP(&lrw, req)
			if lrw.StatusCode != http.StatusOK {
				failureCount.Add(1)
				assert.Equal(t, lrw.StatusCode, http.StatusTooManyRequests)
			}
		}(ctx)
	}

	wg.Wait()
	assert.Equal(t, failureCount.Load(), int32(3))

	// Validate that everything works as expected after errors
	for i := range 10 {
		ctx := contexts[i%len(contexts)]
		handler := mw.Apply(h(http.StatusOK, time.Millisecond))
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		lrw := responsewriter.StatusResponseWriter{ResponseWriter: rr}
		handler.ServeHTTP(&lrw, req)
		if lrw.StatusCode != http.StatusOK {
			failureCount.Add(1)
			assert.Equal(t, lrw.StatusCode, http.StatusTooManyRequests)
		}
	}

	// Perf check
	mw = LimitWithBacklogQueue(ctx, 10 /* in flight per tenant */, 500 /*total per tenant*/, 2000 /*total max*/, 200*time.Millisecond /*max wait time*/)
	failureCount.Swap(0)
	startTime := time.Now().UTC()
	for i := range 1000 {
		wg.Add(1)
		ctx := queuemetrics.InitContext(request.SetRequestIDIfNotSet(contexts[i%len(contexts)], uuid.Must(uuid.NewV4())))
		go func(ctx context.Context) {
			defer wg.Done()
			handler := mw.Apply(h(http.StatusOK, 0))
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			lrw := responsewriter.StatusResponseWriter{ResponseWriter: rr}
			handler.ServeHTTP(&lrw, req)
			if lrw.StatusCode != http.StatusOK {
				failureCount.Add(1)
				assert.Equal(t, lrw.StatusCode, http.StatusTooManyRequests)
			}
		}(ctx)
	}
	wg.Wait()
	assert.Equal(t, failureCount.Load(), int32(0))
	elapsedTimeWithHandler := time.Since(startTime)

	failureCount.Swap(0)
	startTime = time.Now().UTC()
	for i := range 1000 {
		wg.Add(1)
		ctx := queuemetrics.InitContext(request.SetRequestIDIfNotSet(contexts[i%len(contexts)], uuid.Must(uuid.NewV4())))
		go func(ctx context.Context) {
			defer wg.Done()
			handler := h(http.StatusOK, 0)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			lrw := responsewriter.StatusResponseWriter{ResponseWriter: rr}
			handler.ServeHTTP(&lrw, req)
			if lrw.StatusCode != http.StatusOK {
				failureCount.Add(1)
				assert.Equal(t, lrw.StatusCode, http.StatusTooManyRequests)
			}
		}(ctx)
	}
	wg.Wait()
	assert.Equal(t, failureCount.Load(), int32(0))
	elapsedTimeWithoutHandler := time.Since(startTime)

	// The penalty per request is around 2 microseconds on my old laptop but this may prove to be flaky on other machines
	assert.Equal(t, elapsedTimeWithHandler-elapsedTimeWithoutHandler < 200*time.Millisecond, true)
}
