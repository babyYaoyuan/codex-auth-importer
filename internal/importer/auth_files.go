package importer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type codexAuthFileEntry struct {
	Name          string `json:"name"`
	Status        string `json:"status,omitempty"`
	Expired       bool   `json:"expired"`
	Valid         bool   `json:"valid"`
	ValidReason   string `json:"valid_reason,omitempty"`
	QuotaChecked  bool   `json:"quota_checked,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	ExpiredReason string `json:"expired_reason,omitempty"`
	StatusMessage string `json:"status_message,omitempty"`
	Email         string `json:"email,omitempty"`
	Account       string `json:"account,omitempty"`
	LastRefresh   string `json:"last_refresh,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
	Source        string `json:"source,omitempty"`
}

func codexAuthFilesFromHost(entries []pluginapi.HostAuthFileEntry) []codexAuthFileEntry {
	return codexAuthFilesFromHostAt(entries, nil, time.Now())
}

func codexAuthFilesFromHostAt(entries []pluginapi.HostAuthFileEntry, rawByAuthIndex map[string]json.RawMessage, now time.Time) []codexAuthFileEntry {
	return codexAuthFilesFromHostWithQuotaAt(entries, rawByAuthIndex, nil, now)
}

func codexAuthFilesFromHostWithQuotaAt(entries []pluginapi.HostAuthFileEntry, rawByAuthIndex map[string]json.RawMessage, quotaByAuthIndex map[string]codexQuotaProbeResult, now time.Time) []codexAuthFileEntry {
	out := make([]codexAuthFileEntry, 0, len(entries))
	for _, entry := range entries {
		if !isCodexAuthEntry(entry) || strings.TrimSpace(entry.Name) == "" {
			continue
		}
		statusExpired, statusReason := expiredCodexStatus(entry)
		tokenExpired, expiresAt, tokenReason := codexAuthJSONExpired(rawByAuthIndex[entry.AuthIndex], now)
		expired := statusExpired || tokenExpired
		expiredReason := tokenReason
		if expiredReason == "" {
			expiredReason = statusReason
		}
		valid := !expired
		validReason := ""
		quota := quotaByAuthIndex[entry.AuthIndex]
		if quota.Checked {
			valid = quota.Valid
			expired = !quota.Valid
			validReason = quota.Message
			if !quota.Valid {
				expiredReason = quota.Message
			}
		}
		out = append(out, codexAuthFileEntry{
			Name:          entry.Name,
			Status:        entry.Status,
			Expired:       expired,
			Valid:         valid,
			ValidReason:   validReason,
			QuotaChecked:  quota.Checked,
			ExpiresAt:     formatTime(expiresAt),
			ExpiredReason: expiredReason,
			StatusMessage: entry.StatusMessage,
			Email:         entry.Email,
			Account:       entry.Account,
			LastRefresh:   formatTime(entry.LastRefresh),
			UpdatedAt:     formatTime(entry.UpdatedAt),
			Source:        entry.Source,
		})
	}
	return out
}

func isCodexAuthEntry(entry pluginapi.HostAuthFileEntry) bool {
	return strings.EqualFold(strings.TrimSpace(entry.Type), providerCodex) ||
		strings.EqualFold(strings.TrimSpace(entry.Provider), providerCodex)
}

func isExpiredCodexStatus(entry pluginapi.HostAuthFileEntry) bool {
	expired, _ := expiredCodexStatus(entry)
	return expired
}

func expiredCodexStatus(entry pluginapi.HostAuthFileEntry) (bool, string) {
	status := strings.ToLower(strings.TrimSpace(entry.Status))
	message := strings.ToLower(strings.TrimSpace(entry.StatusMessage))
	switch {
	case entry.Unavailable:
		return true, "认证文件不可用"
	case strings.Contains(status, "expired"):
		return true, "状态已过期"
	case strings.Contains(message, "expired") || strings.Contains(message, "过期"):
		return true, "状态消息提示已过期"
	default:
		return false, ""
	}
}

func codexAuthJSONExpired(raw json.RawMessage, now time.Time) (bool, time.Time, string) {
	if len(raw) == 0 {
		return false, time.Time{}, ""
	}
	var metadata map[string]any
	if errUnmarshal := json.Unmarshal(raw, &metadata); errUnmarshal != nil {
		return false, time.Time{}, ""
	}
	if tokens, ok := objectValue(metadata["tokens"]); ok {
		copyStringIfMissing(metadata, "id_token", tokens["id_token"])
		copyStringIfMissing(metadata, "access_token", tokens["access_token"])
		copyStringIfMissing(metadata, "refresh_token", tokens["refresh_token"])
		if _, exists := metadata["expired"]; !exists {
			metadata["expired"] = firstValue(tokens["expired"], tokens["expires_at"])
		}
	}

	var detectedExpiresAt time.Time
	if expiresAt, ok := parseExpiryValue(firstValue(metadata["expired"], metadata["expires_at"])); ok {
		if !expiresAt.After(now) {
			return true, expiresAt, "expired 字段已过期"
		}
		detectedExpiresAt = earliestTime(detectedExpiresAt, expiresAt)
	}
	if expiresAt, ok := jwtExpiresAt(firstString(metadata["access_token"])); ok {
		if !expiresAt.After(now) {
			return true, expiresAt, "access_token 已过期"
		}
		detectedExpiresAt = earliestTime(detectedExpiresAt, expiresAt)
	}
	return false, detectedExpiresAt, ""
}

func parseExpiryValue(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case string:
		raw := strings.TrimSpace(typed)
		if raw == "" {
			return time.Time{}, false
		}
		if parsed, errParse := time.Parse(time.RFC3339, raw); errParse == nil {
			return parsed.UTC(), true
		}
		if parsed, errParse := strconv.ParseFloat(raw, 64); errParse == nil {
			return unixExpiry(parsed)
		}
	case float64:
		return unixExpiry(typed)
	case int:
		return time.Unix(int64(typed), 0).UTC(), true
	case int64:
		return time.Unix(typed, 0).UTC(), true
	case json.Number:
		parsed, errParse := typed.Float64()
		if errParse != nil {
			return time.Time{}, false
		}
		return unixExpiry(parsed)
	}
	return time.Time{}, false
}

func unixExpiry(value float64) (time.Time, bool) {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return time.Time{}, false
	}
	seconds := int64(value)
	if value > 1_000_000_000_000 {
		seconds = int64(value / 1000)
	}
	return time.Unix(seconds, 0).UTC(), true
}

func jwtExpiresAt(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, errDecode := base64.RawURLEncoding.DecodeString(parts[1])
	if errDecode != nil {
		payload, errDecode = base64.URLEncoding.DecodeString(padBase64URL(parts[1]))
		if errDecode != nil {
			return time.Time{}, false
		}
	}
	var claims map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	if errDecodeJSON := decoder.Decode(&claims); errDecodeJSON != nil {
		return time.Time{}, false
	}
	expiresAt, ok := parseExpiryValue(claims["exp"])
	if !ok {
		return time.Time{}, false
	}
	return expiresAt, true
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func earliestTime(current, next time.Time) time.Time {
	if current.IsZero() || next.Before(current) {
		return next
	}
	return current
}

func authGetKey(entry pluginapi.HostAuthFileEntry) string {
	if key := strings.TrimSpace(entry.AuthIndex); key != "" {
		return key
	}
	if key := strings.TrimSpace(entry.ID); key != "" {
		return key
	}
	return strings.TrimSpace(entry.Name)
}

func authGetErrorMessage(entry pluginapi.HostAuthFileEntry, err error) string {
	if err == nil {
		return ""
	}
	if strings.TrimSpace(entry.StatusMessage) != "" {
		return entry.StatusMessage
	}
	return fmt.Sprintf("读取认证内容失败：%v", err)
}
