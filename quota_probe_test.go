package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestProbeCodexQuotaWithDoerMarksSuccessfulQuotaAsValid(t *testing.T) {
	raw := json.RawMessage(`{"access_token":"token-123","account_id":"account-123"}`)

	got := probeCodexQuotaWithDoer(raw, func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("Method = %q, want GET", req.Method)
		}
		if req.URL != codexQuotaURL {
			t.Fatalf("URL = %q, want %q", req.URL, codexQuotaURL)
		}
		if req.Headers.Get("Authorization") != "Bearer token-123" {
			t.Fatalf("Authorization = %q", req.Headers.Get("Authorization"))
		}
		if req.Headers.Get("Chatgpt-Account-Id") != "account-123" {
			t.Fatalf("Chatgpt-Account-Id = %q", req.Headers.Get("Chatgpt-Account-Id"))
		}
		if req.Headers.Get("User-Agent") != codexQuotaUserAgent {
			t.Fatalf("User-Agent = %q", req.Headers.Get("User-Agent"))
		}
		return pluginapi.HTTPResponse{StatusCode: http.StatusOK, Body: []byte(`{"usage":{},"windows":[]}`)}, nil
	})

	if !got.Checked || !got.Valid {
		t.Fatalf("probe result = %#v, want checked valid", got)
	}
}

func TestProbeCodexQuotaWithDoerMarksInvalidatedTokenAsInvalid(t *testing.T) {
	raw := json.RawMessage(`{"tokens":{"access_token":"token-123"}}`)

	got := probeCodexQuotaWithDoer(raw, func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
		return pluginapi.HTTPResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       []byte(`{"detail":"Your authentication token has been invalidated. Please try signing in again."}`),
		}, nil
	})

	if !got.Checked || got.Valid {
		t.Fatalf("probe result = %#v, want checked invalid", got)
	}
	if !strings.Contains(got.Message, "401") || !strings.Contains(got.Message, "invalidated") {
		t.Fatalf("Message = %q, want HTTP 401 invalidated detail", got.Message)
	}
}

func TestProbeCodexQuotaWithDoerRequiresAccessToken(t *testing.T) {
	called := false

	got := probeCodexQuotaWithDoer(json.RawMessage(`{"type":"codex"}`), func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
		called = true
		return pluginapi.HTTPResponse{}, nil
	})

	if called {
		t.Fatal("HTTP doer was called without access_token")
	}
	if !got.Checked || got.Valid || !strings.Contains(got.Message, "access_token") {
		t.Fatalf("probe result = %#v, want missing access_token invalid", got)
	}
}
