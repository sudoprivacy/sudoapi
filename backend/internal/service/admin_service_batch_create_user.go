// sudoapi: CSV-style admin batch user creation.

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// BatchCreateUsersMaxRows caps a single batch-create request to protect DB and
// default-subscription assignment from runaway payloads.
const BatchCreateUsersMaxRows = 500

// BatchCreateUsersInput is the request for AdminService.BatchCreateUsers.
// SkipOnError=true continues after a row-level failure (CSV-import semantics).
// SkipOnError=false stops at the first row-level failure; rows already created
// remain created (no global rollback).
type BatchCreateUsersInput struct {
	Rows        []CreateUserInput
	SkipOnError bool
}

// BatchCreateUserResult is the per-row outcome of BatchCreateUsers.
// Index is 1-based and reflects the position in the original Rows slice.
type BatchCreateUserResult struct {
	Index     int
	Email     string
	Success   bool
	UserID    int64
	ErrorCode string
	ErrorMsg  string
}

// BatchCreateUsersOutput aggregates per-row results for BatchCreateUsers.
// Aborted=true when SkipOnError was false and processing stopped after a failure.
type BatchCreateUsersOutput struct {
	Total   int
	Created int
	Failed  int
	Aborted bool
	Results []BatchCreateUserResult
}

// BatchCreateUsers creates multiple users in one call. Each row goes through
// the same code path as CreateUser (password hash, default subscriptions).
// Validation, in-payload duplicate detection, and per-row error reporting
// are handled here so the caller can surface per-row outcomes to the user.
func (s *adminServiceImpl) BatchCreateUsers(ctx context.Context, input *BatchCreateUsersInput) (*BatchCreateUsersOutput, error) {
	if input == nil || len(input.Rows) == 0 {
		return nil, infraerrors.BadRequest("BATCH_EMPTY", "no rows to create")
	}
	if len(input.Rows) > BatchCreateUsersMaxRows {
		return nil, infraerrors.BadRequest("BATCH_TOO_LARGE", fmt.Sprintf("batch size %d exceeds max %d", len(input.Rows), BatchCreateUsersMaxRows))
	}

	out := &BatchCreateUsersOutput{
		Total:   len(input.Rows),
		Results: make([]BatchCreateUserResult, 0, len(input.Rows)),
	}

	seen := make(map[string]int, len(input.Rows))
	for i := range input.Rows {
		row := input.Rows[i]
		normalized := strings.ToLower(strings.TrimSpace(row.Email))
		result := BatchCreateUserResult{Index: i + 1, Email: row.Email}

		switch {
		case normalized == "":
			result.ErrorCode = "INVALID_EMAIL"
			result.ErrorMsg = "email is required"
		case !isLikelyEmail(normalized):
			result.ErrorCode = "INVALID_EMAIL"
			result.ErrorMsg = "email format is invalid"
		case len(row.Password) < 6:
			result.ErrorCode = "WEAK_PASSWORD"
			result.ErrorMsg = "password must be at least 6 characters"
		case row.Balance != nil && *row.Balance < 0:
			result.ErrorCode = "INVALID_BALANCE"
			result.ErrorMsg = "balance must be >= 0"
		case row.Concurrency < 0:
			result.ErrorCode = "INVALID_CONCURRENCY"
			result.ErrorMsg = "concurrency must be >= 0"
		case row.RPMLimit < 0:
			result.ErrorCode = "INVALID_RPM"
			result.ErrorMsg = "rpm must be >= 0"
		}

		if result.ErrorCode == "" {
			if prev, ok := seen[normalized]; ok {
				result.ErrorCode = "DUPLICATE_IN_PAYLOAD"
				result.ErrorMsg = fmt.Sprintf("email duplicates row %d", prev)
			} else {
				seen[normalized] = result.Index
			}
		}

		if result.ErrorCode != "" {
			out.Results = append(out.Results, result)
			out.Failed++
			if !input.SkipOnError {
				out.Aborted = true
				return out, nil
			}
			continue
		}

		row.Email = normalized
		user, err := s.CreateUser(ctx, &row)
		if err != nil {
			result.ErrorMsg = err.Error()
			switch {
			case errors.Is(err, ErrEmailExists):
				result.ErrorCode = "EMAIL_EXISTS"
			default:
				result.ErrorCode = "CREATE_FAILED"
			}
			out.Results = append(out.Results, result)
			out.Failed++
			if !input.SkipOnError {
				out.Aborted = true
				return out, nil
			}
			continue
		}

		result.Success = true
		result.UserID = user.ID
		out.Results = append(out.Results, result)
		out.Created++
	}

	return out, nil
}

// isLikelyEmail is a cheap format check; canonical validation happens during
// user creation in the repository layer. Mirrors what gin's "email" binding
// accepts in spirit without pulling in a regex library.
func isLikelyEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	local := s[:at]
	domain := s[at+1:]
	if strings.ContainsAny(local, " \t\r\n") || strings.ContainsAny(domain, " \t\r\n") {
		return false
	}
	dot := strings.IndexByte(domain, '.')
	if dot <= 0 || dot == len(domain)-1 {
		return false
	}
	return true
}
