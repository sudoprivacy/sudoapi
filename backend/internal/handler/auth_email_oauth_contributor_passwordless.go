// sudoapi: Google contributor OAuth passwordless signup.

package handler

import (
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func emailOAuthPendingSessionAllowsContributorPasswordSkip(session *dbent.PendingAuthSession, provider string) bool {
	if session == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(provider), "google") &&
		strings.EqualFold(strings.TrimSpace(session.ProviderType), "google") &&
		pendingSessionStringValue(session.UpstreamIdentityClaims, "account_role") == service.RoleAccountContributor
}
