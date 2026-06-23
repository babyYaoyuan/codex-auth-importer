package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestConvertCodexCLIAuth(t *testing.T) {
	raw := `{
	  "tokens": {
	    "id_token": "` + fakeJWT(t, "user@example.com", "acc_123") + `",
	    "access_token": "access-1",
	    "refresh_token": "refresh-1"
	  },
	  "last_refresh": "2026-06-23T00:00:00Z"
	}`

	converted, err := convertCodexAuthJSON([]byte(raw), importOptions{})
	if err != nil {
		t.Fatalf("convertCodexAuthJSON() error = %v", err)
	}
	if converted.SourceFormat != "codex-cli" {
		t.Fatalf("SourceFormat = %q, want codex-cli", converted.SourceFormat)
	}
	if !strings.HasPrefix(converted.FileName, "codex-user_at_example.com-") {
		t.Fatalf("FileName = %q, want generated email-based name", converted.FileName)
	}

	var out map[string]any
	if err := json.Unmarshal(converted.JSON, &out); err != nil {
		t.Fatalf("converted JSON is invalid: %v", err)
	}
	if out["type"] != "codex" {
		t.Fatalf("type = %#v, want codex", out["type"])
	}
	if out["refresh_token"] != "refresh-1" {
		t.Fatalf("refresh_token = %#v, want refresh-1", out["refresh_token"])
	}
	if out["email"] != "user@example.com" {
		t.Fatalf("email = %#v, want user@example.com", out["email"])
	}
	if out["account_id"] != "acc_123" {
		t.Fatalf("account_id = %#v, want acc_123", out["account_id"])
	}
}

func TestNormalizeCLIProxyCodexAuth(t *testing.T) {
	raw := `{
	  "type": "codex",
	  "tokens": {
	    "id_token": "` + fakeJWT(t, "native@example.com", "acc_native") + `",
	    "refresh_token": "refresh-native"
	  }
	}`

	converted, err := convertCodexAuthJSON([]byte(raw), importOptions{RequestedName: "my-codex"})
	if err != nil {
		t.Fatalf("convertCodexAuthJSON() error = %v", err)
	}
	if converted.SourceFormat != "cliproxy-codex" {
		t.Fatalf("SourceFormat = %q, want cliproxy-codex", converted.SourceFormat)
	}
	if converted.FileName != "my-codex.json" {
		t.Fatalf("FileName = %q, want my-codex.json", converted.FileName)
	}

	var out map[string]any
	if err := json.Unmarshal(converted.JSON, &out); err != nil {
		t.Fatalf("converted JSON is invalid: %v", err)
	}
	if out["refresh_token"] != "refresh-native" {
		t.Fatalf("refresh_token = %#v, want refresh-native", out["refresh_token"])
	}
	if out["email"] != "native@example.com" {
		t.Fatalf("email = %#v, want native@example.com", out["email"])
	}
}

func TestConvertRejectsMissingRefreshToken(t *testing.T) {
	raw := `{"tokens":{"access_token":"access-1"}}`
	if _, err := convertCodexAuthJSON([]byte(raw), importOptions{}); err == nil {
		t.Fatal("convertCodexAuthJSON() error = nil, want missing refresh_token error")
	}
}

func TestSafeAuthFileNameRejectsTraversal(t *testing.T) {
	if got := safeAuthFileName("../auth.json"); got != "auth.json" {
		t.Fatalf("safeAuthFileName traversal basename = %q, want auth.json", got)
	}
	if got := safeAuthFileName("bad/name.json"); got != "name.json" {
		t.Fatalf("safeAuthFileName nested basename = %q, want name.json", got)
	}
}

func fakeJWT(t *testing.T, email, accountID string) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	claims, err := json.Marshal(map[string]any{
		"email": email,
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": accountID,
		},
	})
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(claims)
	return header + "." + payload + ".sig"
}
