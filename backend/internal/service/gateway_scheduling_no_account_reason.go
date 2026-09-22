// sudoapi: Record why the account pool came up empty.

package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Account selection can come up empty for reasons that need completely different
// operator responses — the group has no schedulable accounts at all, or it has
// them but every one was filtered out, or candidates exist and are all at their
// session limit. All three returned the bare ErrNoAvailableAccounts, so the 503
// read the same either way; classifyNoAccountError says as much, and works around
// it by re-deriving what it can from model_mapping alone, which cannot see
// transient state.
//
// noAccountReason carries that missing piece. Its Error() deliberately returns
// the bare sentinel text: handlers build the client-facing message as
// "No available accounts: " + err.Error(), so anything added here would ship
// group ids and account counts to API callers. The reason is for operators and
// is read back with NoAccountReason.
type noAccountReason struct {
	reason string
}

func (e *noAccountReason) Error() string { return ErrNoAvailableAccounts.Error() }

func (e *noAccountReason) Unwrap() error { return ErrNoAvailableAccounts }

// noAccountsBecause reports an empty pool along with the condition that emptied
// it, and logs that condition once at the point it is decided. The request-scoped
// logger carries request_id, which is the id the caller was given, so a reported
// 503 can be traced straight to why the pool was empty.
func noAccountsBecause(ctx context.Context, format string, args ...any) error {
	reason := fmt.Sprintf(format, args...)
	logger.FromContext(ctx).Warn("account selection found no available accounts",
		zap.String("reason", reason))
	return &noAccountReason{reason: reason}
}

// NoAccountReason returns why selection came up empty, or "" when the error did
// not record one.
func NoAccountReason(err error) string {
	var reasoned *noAccountReason
	if errors.As(err, &reasoned) {
		return reasoned.reason
	}
	return ""
}
