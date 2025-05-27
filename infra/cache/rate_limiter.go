package cache

import (
	"context"
	"time"

	"userclouds.com/infra/ucerr"
)

// ReleaseRateLimitSlot will release a rate limit slot for the rate limitable item
func ReleaseRateLimitSlot[item RateLimitableItem](
	ctx context.Context,
	m Manager,
	i item,
) (int64, error) {
	if err := i.Validate(); err != nil {
		return 0, ucerr.Wrap(err)
	}

	totalSlots, err := m.Provider.ReleaseRateLimitSlot(ctx, i.GetRateLimitKeys(m.N))
	if err != nil {
		return 0, ucerr.Wrap(err)
	}

	return totalSlots, nil
}

// ReserveRateLimitSlot will check whether a rate limit slot can be reserved for
// the rate limitable item, actually reserving the slot if requested
func ReserveRateLimitSlot[item RateLimitableItem](
	ctx context.Context,
	m Manager,
	i item,
	takeSlot bool,
) (bool, int64, error) {
	if err := i.Validate(); err != nil {
		return false, 0, ucerr.Wrap(err)
	}

	ttl := i.TTL(m.T)
	limit := i.GetRateLimit()
	keys := i.GetRateLimitKeys(m.N)
	reserved, totalSlots, err := m.Provider.ReserveRateLimitSlot(ctx, keys, limit, ttl, takeSlot)
	if err != nil {
		return false, 0, ucerr.Wrap(err)
	}

	return reserved, totalSlots, nil
}

// WaitForRateLimitSlot will wait until a rate limit slot is can be reserved for the
// rate limitable item, reserving the slot once available
func WaitForRateLimitSlot[item RateLimitableItem](
	ctx context.Context,
	m Manager,
	i item,
	waitTimeMilliseconds int,
) (int64, error) {
	if waitTimeMilliseconds <= 0 {
		return 0, ucerr.Errorf("waitTimeMilliseconds must be greater than zero: '%d'", waitTimeMilliseconds)
	}

	for ctx.Err() == nil {
		reserved, numSlots, err := ReserveRateLimitSlot(ctx, m, i, true)
		if err != nil {
			return 0, ucerr.Wrap(err)
		}
		if reserved {
			return numSlots, nil
		}
		time.Sleep(time.Millisecond * time.Duration(waitTimeMilliseconds))
	}

	return 0, ucerr.Errorf("Either context is canceled or wait time exceeded max limit")
}
