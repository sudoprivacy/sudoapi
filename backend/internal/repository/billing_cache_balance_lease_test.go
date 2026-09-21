// sudoapi: Deducting must not extend the balance entry's lease.

//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The cached balance mirrors users.balance, and its TTL is the only thing that
// periodically re-validates the mirror against the row. Renewing that TTL on
// every deduction keeps an active user's entry alive indefinitely, so any drift
// it picks up — a top-up whose cache invalidation failed, say — stops healing.
func TestDeductUserBalanceKeepsTheRemainingLease(t *testing.T) {
	cache, mr := newMiniRedisCache(t)
	ctx := context.Background()
	const userID = int64(42)

	require.NoError(t, cache.SetUserBalance(ctx, userID, 10))

	elapsed := 2 * time.Minute
	mr.FastForward(elapsed)
	before := mr.TTL(billingBalanceKey(userID))
	require.Positive(t, before, "the entry must still be alive to be deducted from")

	require.NoError(t, cache.DeductUserBalance(ctx, userID, 2.5))

	after := mr.TTL(billingBalanceKey(userID))
	assert.LessOrEqual(t, after, before, "a deduction must not extend the lease")
	assert.LessOrEqual(t, after, billingCacheTTL-elapsed,
		"the entry must still expire on its original schedule")

	balance, err := cache.GetUserBalance(ctx, userID)
	require.NoError(t, err)
	assert.InDelta(t, 7.5, balance, 1e-9, "the deduction itself must still apply")
}

// Repeated deductions are the case that matters: this is what an active user
// looks like, and it is where a renewed lease turns into an entry that outlives
// every bound the cache is supposed to have.
func TestRepeatedDeductionsDoNotOutliveTheCacheTTL(t *testing.T) {
	cache, mr := newMiniRedisCache(t)
	ctx := context.Background()
	const userID = int64(43)

	require.NoError(t, cache.SetUserBalance(ctx, userID, 1000))

	step := time.Minute
	for elapsed := step; elapsed <= billingCacheTTL; elapsed += step {
		mr.FastForward(step)
		if mr.TTL(billingBalanceKey(userID)) <= 0 {
			break
		}
		require.NoError(t, cache.DeductUserBalance(ctx, userID, 1))
	}

	assert.Zero(t, mr.TTL(billingBalanceKey(userID)),
		"an entry deducted from every minute must still expire within billingCacheTTL")
}

// A deduction must never leave an entry that outlives the cache entirely, so an
// entry that somehow carries no expiry is given one rather than kept forever.
func TestDeductUserBalanceGivesAnUnexpiringEntryALease(t *testing.T) {
	cache, mr := newMiniRedisCache(t)
	ctx := context.Background()
	const userID = int64(44)

	key := billingBalanceKey(userID)
	require.NoError(t, mr.Set(key, "10"))
	require.Zero(t, mr.TTL(key), "precondition: the entry has no expiry")

	require.NoError(t, cache.DeductUserBalance(ctx, userID, 1))

	assert.Positive(t, mr.TTL(key), "a deduction must not leave an immortal entry")
	assert.LessOrEqual(t, mr.TTL(key), billingCacheTTL)
}
