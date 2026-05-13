// sudoapi: Idempotent admin data import.

package admin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func (h *AccountHandler) buildExistingAccountImportKeys(ctx context.Context) (map[string]struct{}, error) {
	accounts, err := h.listAccountsFiltered(ctx, "", "", "", "", 0, "", "created_at", "desc")
	if err != nil {
		return nil, err
	}
	keys := make(map[string]struct{}, len(accounts))
	for i := range accounts {
		markAccountDataIdentitySeen(keys, accountDataIdentityKeys(accounts[i].Platform, accounts[i].Type, accounts[i].Credentials, accounts[i].Extra))
	}
	return keys, nil
}

func buildCanonicalProxyKey(protocol, host string, port int, username, password string) string {
	return fmt.Sprintf("%s|%s|%d|%s|%s",
		strings.ToLower(strings.TrimSpace(protocol)),
		strings.ToLower(strings.TrimSpace(host)),
		port,
		strings.TrimSpace(username),
		strings.TrimSpace(password),
	)
}

func accountDataIdentityKeys(platform, accountType string, credentials, extra map[string]any) []string {
	prefix := strings.ToLower(strings.TrimSpace(platform)) + "|" + strings.ToLower(strings.TrimSpace(accountType)) + "|"
	keys := make([]string, 0, 8)
	addValueKey := func(kind, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		keys = append(keys, prefix+kind+":"+hashDataIdentityValue(value))
	}
	addMapString := func(kind string, values map[string]any, candidates ...string) {
		for _, candidate := range candidates {
			if value := dataIdentityString(values, candidate); value != "" {
				addValueKey(kind+":"+strings.ToLower(candidate), value)
				return
			}
		}
	}

	addMapString("extra", extra, "crs_account_id")
	addMapString("credentials", credentials, "api_key")
	addMapString("credentials", credentials, "refresh_token")
	addMapString("credentials", credentials, "session_key", "session_token")
	addMapString("credentials", credentials, "chatgpt_account_id", "account_id")
	addMapString("credentials", credentials, "chatgpt_user_id", "user_id")
	addMapString("credentials", credentials, "access_token")
	addMapString("credentials", credentials, "id_token")

	if len(keys) == 0 {
		addMapString("credentials", credentials, "email")
	}
	if fingerprint := dataMapFingerprint(credentials); fingerprint != "" {
		keys = append(keys, prefix+"credentials:"+fingerprint)
	}

	return keys
}

func hasAnyAccountDataIdentity(seen map[string]struct{}, keys []string) bool {
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			return true
		}
	}
	return false
}

func markAccountDataIdentitySeen(seen map[string]struct{}, keys []string) {
	for _, key := range keys {
		seen[key] = struct{}{}
	}
}

func dataIdentityString(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	case float64:
		return strings.TrimSpace(strconv.FormatFloat(v, 'f', -1, 64))
	case float32:
		return strings.TrimSpace(strconv.FormatFloat(float64(v), 'f', -1, 32))
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	default:
		return ""
	}
}

func dataMapFingerprint(values map[string]any) string {
	if len(values) == 0 {
		return ""
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return hashDataIdentityValue(string(raw))
}

func hashDataIdentityValue(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}
