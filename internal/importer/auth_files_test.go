package importer

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestCodexAuthFilesFromHostFiltersAndMarksExpired(t *testing.T) {
	lastRefresh := time.Date(2026, 6, 23, 1, 2, 3, 0, time.UTC)
	got := codexAuthFilesFromHost([]pluginapi.HostAuthFileEntry{
		{Name: "codex-ok.json", Type: "codex", Status: "ok", Email: "a@example.com", LastRefresh: lastRefresh, Source: "file"},
		{Name: "gemini.json", Type: "gemini", Status: "ok"},
		{Name: "codex-expired.json", Provider: "codex", Status: "error", StatusMessage: "token expired"},
	})
	if len(got) != 2 {
		t.Fatalf("codexAuthFilesFromHost() len = %d, want 2", len(got))
	}
	if got[0].Name != "codex-ok.json" || got[0].Expired {
		t.Fatalf("first codex entry = %#v, want non-expired codex-ok.json", got[0])
	}
	if got[0].LastRefresh != "2026-06-23T01:02:03Z" {
		t.Fatalf("LastRefresh = %q, want RFC3339 UTC", got[0].LastRefresh)
	}
	if got[1].Name != "codex-expired.json" || !got[1].Expired {
		t.Fatalf("second codex entry = %#v, want expired codex-expired.json", got[1])
	}
}

func TestCodexAuthFilesFromHostUsesExpiredField(t *testing.T) {
	now := time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)
	rawByAuthIndex := map[string]json.RawMessage{
		"codex-1": []byte(`{"type":"codex","refresh_token":"refresh","expired":"2026-06-23T07:59:00Z"}`),
	}

	got := codexAuthFilesFromHostAt([]pluginapi.HostAuthFileEntry{
		{Name: "codex-expired-field.json", AuthIndex: "codex-1", Type: "codex", Status: "ok"},
	}, rawByAuthIndex, now)

	if len(got) != 1 {
		t.Fatalf("codexAuthFilesFromHostAt() len = %d, want 1", len(got))
	}
	if !got[0].Expired {
		t.Fatalf("Expired = false, want true for expired field: %#v", got[0])
	}
	if got[0].ExpiresAt != "2026-06-23T07:59:00Z" {
		t.Fatalf("ExpiresAt = %q, want expired field timestamp", got[0].ExpiresAt)
	}
	if got[0].ExpiredReason != "expired 字段已过期" {
		t.Fatalf("ExpiredReason = %q, want expired field reason", got[0].ExpiredReason)
	}
}

func TestCodexAuthFilesFromHostUsesAccessTokenExp(t *testing.T) {
	now := time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)
	rawByAuthIndex := map[string]json.RawMessage{
		"codex-1": []byte(`{"type":"codex","access_token":"` + fakeJWTWithExp(t, now.Add(-time.Minute)) + `","refresh_token":"refresh"}`),
		"codex-2": []byte(`{"type":"codex","access_token":"` + fakeJWTWithExp(t, now.Add(time.Hour)) + `","refresh_token":"refresh"}`),
	}

	got := codexAuthFilesFromHostAt([]pluginapi.HostAuthFileEntry{
		{Name: "codex-expired-token.json", AuthIndex: "codex-1", Type: "codex", Status: "ok"},
		{Name: "codex-valid-token.json", AuthIndex: "codex-2", Type: "codex", Status: "ok"},
	}, rawByAuthIndex, now)

	if len(got) != 2 {
		t.Fatalf("codexAuthFilesFromHostAt() len = %d, want 2", len(got))
	}
	if !got[0].Expired || got[0].ExpiredReason != "access_token 已过期" {
		t.Fatalf("first entry = %#v, want expired access_token", got[0])
	}
	if got[1].Expired {
		t.Fatalf("second entry = %#v, want valid future access_token", got[1])
	}
}

func TestCodexAuthFilesFromHostUsesQuotaProbeAsValiditySource(t *testing.T) {
	now := time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)
	rawByAuthIndex := map[string]json.RawMessage{
		"codex-1": []byte(`{"type":"codex","access_token":"` + fakeJWTWithExp(t, now.Add(time.Hour)) + `","refresh_token":"refresh"}`),
		"codex-2": []byte(`{"type":"codex","access_token":"` + fakeJWTWithExp(t, now.Add(-time.Hour)) + `","refresh_token":"refresh"}`),
	}
	quotaByAuthIndex := map[string]codexQuotaProbeResult{
		"codex-1": {Checked: true, Valid: false, Message: "额度查询失败：HTTP 401 Your authentication token has been invalidated. Please try signing in again."},
		"codex-2": {Checked: true, Valid: true, Message: "额度查询成功"},
	}

	got := codexAuthFilesFromHostWithQuotaAt([]pluginapi.HostAuthFileEntry{
		{Name: "codex-invalidated.json", AuthIndex: "codex-1", Type: "codex", Status: "ok"},
		{Name: "codex-quota-ok.json", AuthIndex: "codex-2", Type: "codex", Status: "ok"},
	}, rawByAuthIndex, quotaByAuthIndex, now)

	if len(got) != 2 {
		t.Fatalf("codexAuthFilesFromHostWithQuotaAt() len = %d, want 2", len(got))
	}
	if !got[0].QuotaChecked || got[0].Valid || !got[0].Expired {
		t.Fatalf("first entry = %#v, want quota-checked invalid/expired", got[0])
	}
	if got[0].ExpiredReason != quotaByAuthIndex["codex-1"].Message {
		t.Fatalf("ExpiredReason = %q, want quota failure", got[0].ExpiredReason)
	}
	if !got[1].QuotaChecked || !got[1].Valid || got[1].Expired {
		t.Fatalf("second entry = %#v, want quota-checked valid even with expired JWT", got[1])
	}
}

func TestCodexAuthFilesFromHostDoesNotLetFutureExpiredFieldMaskExpiredAccessToken(t *testing.T) {
	now := time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)
	rawByAuthIndex := map[string]json.RawMessage{
		"codex-1": []byte(`{"type":"codex","expired":"2026-06-24T08:00:00Z","access_token":"` + fakeJWTWithExp(t, now.Add(-time.Minute)) + `","refresh_token":"refresh"}`),
	}

	got := codexAuthFilesFromHostAt([]pluginapi.HostAuthFileEntry{
		{Name: "codex-expired-token.json", AuthIndex: "codex-1", Type: "codex", Status: "ok"},
	}, rawByAuthIndex, now)

	if len(got) != 1 {
		t.Fatalf("codexAuthFilesFromHostAt() len = %d, want 1", len(got))
	}
	if !got[0].Expired || got[0].ExpiredReason != "access_token 已过期" {
		t.Fatalf("entry = %#v, want expired access_token despite future expired field", got[0])
	}
}

func TestCodexAuthFilesFromHostIgnoresExpiredIDTokenWhenAccessTokenIsValid(t *testing.T) {
	now := time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)
	rawByAuthIndex := map[string]json.RawMessage{
		"codex-1": []byte(`{"type":"codex","id_token":"` + fakeJWTWithExp(t, now.Add(-time.Hour)) + `","access_token":"` + fakeJWTWithExp(t, now.Add(time.Hour)) + `","refresh_token":"refresh"}`),
	}

	got := codexAuthFilesFromHostAt([]pluginapi.HostAuthFileEntry{
		{Name: "codex-valid-access-token.json", AuthIndex: "codex-1", Type: "codex", Status: "ok"},
	}, rawByAuthIndex, now)

	if len(got) != 1 {
		t.Fatalf("codexAuthFilesFromHostAt() len = %d, want 1", len(got))
	}
	if got[0].Expired {
		t.Fatalf("entry = %#v, want valid because access_token is still valid", got[0])
	}
	if got[0].ExpiresAt != "2026-06-23T09:00:00Z" {
		t.Fatalf("ExpiresAt = %q, want access_token expiration", got[0].ExpiresAt)
	}
}

func fakeJWTWithExp(t *testing.T, expiresAt time.Time) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	claims, err := json.Marshal(map[string]any{
		"exp": expiresAt.Unix(),
	})
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(claims)
	return header + "." + payload + ".sig"
}
