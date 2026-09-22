// sudoapi: Record why the account pool came up empty.

package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Handlers build the client-facing message by concatenating the error text, so
// the reason must stay out of Error(). This pins the exact string an API caller
// sees — the one quoted in the outage report — byte for byte.
func TestNoAccountReasonKeepsTheClientMessageUnchanged(t *testing.T) {
	err := noAccountsBecause(context.Background(), "group=7 platform=anthropic: no schedulable accounts in this group")

	assert.Equal(t, ErrNoAvailableAccounts.Error(), err.Error(),
		"the reason must not reach Error(); it would ship group ids and counts to API callers")
	assert.Equal(t, "No available accounts: no available accounts",
		"No available accounts: "+err.Error())
}

// Every existing classifier reaches this error through errors.Is, including the
// 404-vs-503 split in classifyNoAccountError and the ops error logger.
func TestNoAccountReasonStaysTheSentinel(t *testing.T) {
	err := noAccountsBecause(context.Background(), "group=7 platform=anthropic: all 3 candidates are at their session limit")

	assert.True(t, errors.Is(err, ErrNoAvailableAccounts))
	assert.True(t, errors.Is(fmt.Errorf("wrapped: %w", err), ErrNoAvailableAccounts))
}

func TestNoAccountReasonReadsBackTheReason(t *testing.T) {
	reason := "group=7 platform=anthropic model=claude-opus-4-8: all 5 schedulable accounts were filtered out"
	err := noAccountsBecause(context.Background(), "%s", reason)

	assert.Equal(t, reason, NoAccountReason(err))
	assert.Equal(t, reason, NoAccountReason(fmt.Errorf("wrapped: %w", err)),
		"the reason must survive wrapping, since selection errors are wrapped on the way out")
}

// The three conditions need different operator responses — add accounts, wait or
// raise quota, raise concurrency — so they must not read alike.
func TestNoAccountReasonDistinguishesTheThreeConditions(t *testing.T) {
	emptyPool := noAccountsBecause(context.Background(), "group=%v platform=%s: no schedulable accounts in this group", 7, "anthropic")
	allFiltered := noAccountsBecause(context.Background(), "group=%v platform=%s model=%s: all %d schedulable accounts were filtered out", 7, "anthropic", "claude-opus-4-8", 5)
	sessionLimit := noAccountsBecause(context.Background(), "group=%v platform=%s: all %d candidates are at their session limit", 7, "anthropic", 5)

	reasons := []string{NoAccountReason(emptyPool), NoAccountReason(allFiltered), NoAccountReason(sessionLimit)}
	seen := map[string]bool{}
	for _, r := range reasons {
		require.NotEmpty(t, r)
		require.False(t, seen[r], "two conditions produced the same reason: %s", r)
		seen[r] = true
	}
}

func TestNoAccountReasonIsEmptyForAPlainSentinel(t *testing.T) {
	assert.Empty(t, NoAccountReason(ErrNoAvailableAccounts))
	assert.Empty(t, NoAccountReason(nil))
	assert.Empty(t, NoAccountReason(errors.New("something else")))
}
