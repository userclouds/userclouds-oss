package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"userclouds.com/infra/assert"
)

func testSupportedRateLimitsSingleThreaded[ProviderType Provider](ctx context.Context, t *testing.T, p ProviderType) {
	m := NewManager(p, testNameProv{}, testTTLProv{})

	trli := newTestRateLimitItem(4, 2, 2)
	reserved, numSlots, err := ReserveRateLimitSlot(ctx, m, trli, true)
	assert.NoErr(t, err)
	assert.True(t, reserved)
	assert.Equal(t, numSlots, int64(1))

	numSlots, err = ReleaseRateLimitSlot(ctx, m, trli)
	assert.NoErr(t, err)
	assert.Equal(t, numSlots, int64(0))

	numSlots, err = ReleaseRateLimitSlot(ctx, m, trli)
	assert.NoErr(t, err)
	assert.Equal(t, numSlots, int64(0))

	reserved, numSlots, err = ReserveRateLimitSlot(ctx, m, trli, true)
	assert.NoErr(t, err)
	assert.True(t, reserved)
	assert.Equal(t, numSlots, int64(1))

	numSlots, err = WaitForRateLimitSlot(ctx, m, trli, trli.retryPause())
	assert.NoErr(t, err)
	assert.Equal(t, numSlots, int64(2))

	trli.minBucket = 1
	trli.maxBucket = 4
	reserved, numSlots, err = ReserveRateLimitSlot(ctx, m, trli, true)
	assert.NoErr(t, err)
	assert.True(t, reserved)
	assert.Equal(t, numSlots, int64(3))

	trli.maxBucket = 5
	numSlots, err = WaitForRateLimitSlot(ctx, m, trli, trli.retryPause())
	assert.NoErr(t, err)
	assert.Equal(t, numSlots, int64(4))

	reserved, numSlots, err = ReserveRateLimitSlot(ctx, m, trli, true)
	assert.NoErr(t, err)
	assert.False(t, reserved)
	assert.Equal(t, numSlots, int64(4))

	trli.minBucket = 3
	trli.maxBucket = 6
	reserved, numSlots, err = ReserveRateLimitSlot(ctx, m, trli, true)
	assert.NoErr(t, err)
	assert.True(t, reserved)
	assert.Equal(t, numSlots, int64(3))

	numSlots, err = WaitForRateLimitSlot(ctx, m, trli, trli.retryPause())
	assert.NoErr(t, err)
	assert.Equal(t, numSlots, int64(4))

	reserved, numSlots, err = ReserveRateLimitSlot(ctx, m, trli, true)
	assert.NoErr(t, err)
	assert.False(t, reserved)
	assert.Equal(t, numSlots, int64(4))
}

func testSupportedRateLimitsMultiThreaded[ProviderType Provider](ctx context.Context, t *testing.T, p ProviderType) {
	m := NewManager(p, testNameProv{}, testTTLProv{})

	const numThreads = 100
	wg := sync.WaitGroup{}

	trli := newTestRateLimitItem(15, 1, 3)
	var trlis []testRateLimitItem
	for i := 1; i <= 3; i++ {
		trlis = append(trlis, trli)
		trlis[len(trlis)-1].rateLimit = 5
		trlis[len(trlis)-1].minBucket = i
		trlis[len(trlis)-1].maxBucket = i
	}

	for i := range numThreads {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			var err error
			switch {
			case threadID < 33:
				_, _, err = ReserveRateLimitSlot(ctx, m, trlis[0], true)
			case threadID < 66:
				_, _, err = ReserveRateLimitSlot(ctx, m, trlis[1], true)
			default:
				_, _, err = ReserveRateLimitSlot(ctx, m, trlis[2], true)
			}
			assert.NoErr(t, err)
		}(i)
	}
	wg.Wait()

	reserved, numSlots, err := ReserveRateLimitSlot(ctx, m, trli, true)
	assert.NoErr(t, err)
	assert.False(t, reserved)
	assert.Equal(t, numSlots, int64(15))

	for i := range numThreads {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			_, err := ReleaseRateLimitSlot(ctx, m, trli)
			assert.NoErr(t, err)
		}(i)
	}
	wg.Wait()

	numSlots, err = ReleaseRateLimitSlot(ctx, m, trli)
	assert.NoErr(t, err)
	assert.Equal(t, numSlots, int64(0))

	for i := 1; i <= 5; i++ {
		reserved, numSlots, err := ReserveRateLimitSlot(ctx, m, trli, true)
		assert.NoErr(t, err)
		assert.True(t, reserved)
		assert.Equal(t, numSlots, int64(i))
	}

	for i := range numThreads {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			_, err := WaitForRateLimitSlot(ctx, m, trli, trli.retryPause())
			assert.NoErr(t, err)
			time.Sleep(time.Millisecond * time.Duration(trli.retryPause()))
			_, err = ReleaseRateLimitSlot(ctx, m, trli)
			assert.NoErr(t, err)
		}(i)
	}
	wg.Wait()

	reserved, numSlots, err = ReserveRateLimitSlot(ctx, m, trli, true)
	assert.NoErr(t, err)
	assert.True(t, reserved)
	assert.Equal(t, numSlots, int64(6))

	for i := range numThreads {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			_, err := ReleaseRateLimitSlot(ctx, m, trli)
			assert.NoErr(t, err)
		}(i)
	}
	wg.Wait()

	numSlots, err = ReleaseRateLimitSlot(ctx, m, trli)
	assert.NoErr(t, err)
	assert.Equal(t, numSlots, int64(0))
}

func testUnsupportedRateLimits[ProviderType Provider](ctx context.Context, t *testing.T, p ProviderType) {
	m := NewManager(p, testNameProv{}, testTTLProv{})

	trli := newTestRateLimitItem(2, 1, 1)

	const numThreads = 100
	wg := sync.WaitGroup{}
	for i := range numThreads {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			numSlots, err := ReleaseRateLimitSlot(ctx, m, trli)
			assert.NoErr(t, err)
			assert.Equal(t, numSlots, int64(0))

			numSlots, err = ReleaseRateLimitSlot(ctx, m, trli)
			assert.NoErr(t, err)
			assert.Equal(t, numSlots, int64(0))

			reserved, numSlots, err := ReserveRateLimitSlot(ctx, m, trli, true)
			assert.NoErr(t, err)
			assert.True(t, reserved)
			assert.Equal(t, numSlots, int64(1))

			reserved, numSlots, err = ReserveRateLimitSlot(ctx, m, trli, true)
			assert.NoErr(t, err)
			assert.True(t, reserved)
			assert.Equal(t, numSlots, int64(1))

			numSlots, err = WaitForRateLimitSlot(ctx, m, trli, trli.retryPause())
			assert.NoErr(t, err)
			assert.Equal(t, numSlots, int64(1))

			numSlots, err = WaitForRateLimitSlot(ctx, m, trli, trli.retryPause())
			assert.NoErr(t, err)
			assert.Equal(t, numSlots, int64(1))
		}(i)
	}
	wg.Wait()
}
