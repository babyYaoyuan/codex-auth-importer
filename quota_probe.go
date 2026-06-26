package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const (
	codexQuotaURL       = "https://chatgpt.com/backend-api/wham/usage"
	codexQuotaUserAgent = "codex_cli_rs/0.76.0 (Debian 13.0.0; x86_64) WindowsTerminal"
)

type codexQuotaProbeResult struct {
	Checked bool
	Valid   bool
	Message string
}

type codexQuotaHTTPDoer func(pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error)

func probeCodexQuota(raw json.RawMessage) codexQuotaProbeResult {
	return probeCodexQuotaWithDoer(raw, callHostHTTPDo)
}

func probeCodexQuotaWithDoer(raw json.RawMessage, do codexQuotaHTTPDoer) codexQuotaProbeResult {
	fields, errExtract := extractCodexQuotaFields(raw)
	if errExtract != nil {
		return codexQuotaProbeResult{Checked: true, Valid: false, Message: errExtract.Error()}
	}
	headers := http.Header{
		"Authorization": []string{"Bearer " + fields.AccessToken},
		"Content-Type":  []string{"application/json"},
		"User-Agent":    []string{codexQuotaUserAgent},
	}
	if fields.AccountID != "" {
		headers.Set("Chatgpt-Account-Id", fields.AccountID)
	}
	resp, errDo := do(pluginapi.HTTPRequest{
		Method:  http.MethodGet,
		URL:     codexQuotaURL,
		Headers: headers,
	})
	if errDo != nil {
		return codexQuotaProbeResult{Checked: true, Valid: false, Message: fmt.Sprintf("额度查询失败：%v", errDo)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return codexQuotaProbeResult{Checked: true, Valid: false, Message: quotaHTTPErrorMessage(resp.StatusCode, resp.Body)}
	}
	if !quotaResponseLooksValid(resp.Body) {
		return codexQuotaProbeResult{Checked: true, Valid: false, Message: "额度响应不是有效 JSON"}
	}
	return codexQuotaProbeResult{Checked: true, Valid: true, Message: "额度查询成功"}
}

type codexQuotaFields struct {
	AccessToken string
	AccountID   string
}

func extractCodexQuotaFields(raw json.RawMessage) (codexQuotaFields, error) {
	if len(raw) == 0 {
		return codexQuotaFields{}, fmt.Errorf("缺少认证文件内容")
	}
	var metadata map[string]any
	if errUnmarshal := json.Unmarshal(raw, &metadata); errUnmarshal != nil {
		return codexQuotaFields{}, fmt.Errorf("认证文件 JSON 无法解析")
	}
	if tokens, ok := objectValue(metadata["tokens"]); ok {
		copyStringIfMissing(metadata, "access_token", tokens["access_token"])
	}
	accessToken := firstString(metadata["access_token"], metadata["api_key"])
	if accessToken == "" {
		return codexQuotaFields{}, fmt.Errorf("缺少 access_token")
	}
	accountID := firstString(
		metadata["account_id"],
		metadata["accountId"],
		metadata["chatgpt_account_id"],
		metadata["chatgptAccountId"],
	)
	if account, ok := objectValue(metadata["account"]); ok && accountID == "" {
		accountID = firstString(account["id"], account["account_id"], account["accountId"])
	}
	return codexQuotaFields{
		AccessToken: accessToken,
		AccountID:   accountID,
	}, nil
}

func quotaResponseLooksValid(raw []byte) bool {
	var obj map[string]any
	if errUnmarshal := json.Unmarshal(raw, &obj); errUnmarshal != nil {
		return false
	}
	return len(obj) > 0
}

func quotaHTTPErrorMessage(statusCode int, raw []byte) string {
	message := quotaBodyMessage(raw)
	if message == "" {
		message = http.StatusText(statusCode)
	}
	if message == "" {
		message = strings.TrimSpace(string(raw))
	}
	if message == "" {
		return fmt.Sprintf("额度查询失败：HTTP %d", statusCode)
	}
	return fmt.Sprintf("额度查询失败：HTTP %d %s", statusCode, message)
}

func quotaBodyMessage(raw []byte) string {
	var body any
	if errUnmarshal := json.Unmarshal(raw, &body); errUnmarshal != nil {
		return trimMessage(string(raw))
	}
	return messageFromAny(body)
}

func messageFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return trimMessage(typed)
	case map[string]any:
		for _, key := range []string{"message", "detail", "error_description"} {
			if msg := messageFromAny(typed[key]); msg != "" {
				return msg
			}
		}
		if errObj, ok := typed["error"].(map[string]any); ok {
			if msg := messageFromAny(errObj); msg != "" {
				return msg
			}
		}
		return messageFromAny(typed["error"])
	default:
		return ""
	}
}

func trimMessage(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 240 {
		return value[:240] + "..."
	}
	return value
}
