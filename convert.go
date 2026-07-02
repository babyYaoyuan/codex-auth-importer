package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const providerCodex = "codex"

var (
	safeFilePartPattern       = regexp.MustCompile(`[^\p{L}\p{N}@._+\-#()（） ]+`)
	repeatedWhitespacePattern = regexp.MustCompile(`\s+`)
	repeatedDashPattern       = regexp.MustCompile(`-+`)
)

type convertedAuth struct {
	FileName     string
	JSON         json.RawMessage
	SourceFormat string
	Email        string
	AccountID    string
}

type importOptions struct {
	RequestedName string
	UploadedName  string
}

type codexAuthFile struct {
	Type         string `json:"type"`
	IDToken      string `json:"id_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id,omitempty"`
	Email        string `json:"email,omitempty"`
	Expired      any    `json:"expired,omitempty"`
	LastRefresh  any    `json:"last_refresh,omitempty"`
}

type jwtClaims struct {
	Email         string        `json:"email"`
	CodexAuthInfo codexAuthInfo `json:"https://api.openai.com/auth"`
}

type codexAuthInfo struct {
	ChatGPTAccountID string `json:"chatgpt_account_id"`
}

func convertCodexAuthJSON(raw []byte, opts importOptions) (convertedAuth, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return convertedAuth{}, errors.New("auth json is empty")
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return convertedAuth{}, fmt.Errorf("invalid auth json: %w", err)
	}

	if strings.EqualFold(stringValue(metadata["type"]), providerCodex) {
		return normalizeCLIProxyCodexAuth(metadata, opts)
	}
	if tokens, ok := objectValue(metadata["tokens"]); ok {
		return convertCodexCLIAuth(metadata, tokens, opts)
	}
	return convertedAuth{}, errors.New("unsupported auth json: expected CLIProxyAPI codex auth or Codex CLI auth.json with tokens")
}

func normalizeCLIProxyCodexAuth(metadata map[string]any, opts importOptions) (convertedAuth, error) {
	refreshToken := firstString(metadata["refresh_token"])
	if refreshToken == "" {
		if tokens, ok := objectValue(metadata["tokens"]); ok {
			refreshToken = firstString(tokens["refresh_token"])
			copyStringIfMissing(metadata, "id_token", tokens["id_token"])
			copyStringIfMissing(metadata, "access_token", tokens["access_token"])
			copyStringIfMissing(metadata, "refresh_token", tokens["refresh_token"])
			copyStringIfMissing(metadata, "account_id", tokens["account_id"])
		}
	}
	if refreshToken == "" {
		return convertedAuth{}, errors.New("codex auth is missing refresh_token")
	}

	metadata["type"] = providerCodex
	claims := parseJWTClaims(firstString(metadata["id_token"]))
	if stringValue(metadata["email"]) == "" && claims.Email != "" {
		metadata["email"] = claims.Email
	}
	if stringValue(metadata["account_id"]) == "" && claims.CodexAuthInfo.ChatGPTAccountID != "" {
		metadata["account_id"] = claims.CodexAuthInfo.ChatGPTAccountID
	}
	if _, ok := metadata["last_refresh"]; !ok {
		metadata["last_refresh"] = time.Now().UTC().Format(time.RFC3339)
	}

	output, err := marshalPretty(metadata)
	if err != nil {
		return convertedAuth{}, err
	}
	email := stringValue(metadata["email"])
	accountID := stringValue(metadata["account_id"])
	return convertedAuth{
		FileName:     targetFileName(opts, email, accountID, refreshToken),
		JSON:         output,
		SourceFormat: "cliproxy-codex",
		Email:        email,
		AccountID:    accountID,
	}, nil
}

func convertCodexCLIAuth(metadata map[string]any, tokens map[string]any, opts importOptions) (convertedAuth, error) {
	auth := codexAuthFile{
		Type:         providerCodex,
		IDToken:      firstString(tokens["id_token"], metadata["id_token"]),
		AccessToken:  firstString(tokens["access_token"], metadata["access_token"]),
		RefreshToken: firstString(tokens["refresh_token"], metadata["refresh_token"]),
		AccountID:    firstString(tokens["account_id"], metadata["account_id"]),
		Email:        firstString(metadata["email"]),
		Expired:      firstValue(metadata["expired"], tokens["expired"], metadata["expires_at"], tokens["expires_at"]),
		LastRefresh:  firstValue(metadata["last_refresh"], tokens["last_refresh"]),
	}
	if auth.RefreshToken == "" {
		return convertedAuth{}, errors.New("codex cli auth is missing tokens.refresh_token")
	}

	claims := parseJWTClaims(auth.IDToken)
	if auth.Email == "" {
		auth.Email = claims.Email
	}
	if auth.AccountID == "" {
		auth.AccountID = claims.CodexAuthInfo.ChatGPTAccountID
	}
	if auth.LastRefresh == nil {
		auth.LastRefresh = time.Now().UTC().Format(time.RFC3339)
	}

	output, err := marshalPretty(auth)
	if err != nil {
		return convertedAuth{}, err
	}
	return convertedAuth{
		FileName:     targetFileName(opts, auth.Email, auth.AccountID, auth.RefreshToken),
		JSON:         output,
		SourceFormat: "codex-cli",
		Email:        auth.Email,
		AccountID:    auth.AccountID,
	}, nil
}

func targetFileName(opts importOptions, email, accountID, refreshToken string) string {
	if name := safeAuthFileName(opts.RequestedName); name != "" {
		return name
	}
	label := "imported"
	if email != "" {
		label = email
	} else if accountID != "" {
		label = accountID
	}
	label = safeFilePart(label)
	if label == "" {
		label = "imported"
	}
	return fmt.Sprintf("codex-%s-%s.json", label, shortHash(firstNonEmpty(accountID, refreshToken, opts.UploadedName)))
}

func safeAuthFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return ""
	}
	name = strings.NewReplacer("/", "-", "\\", "-").Replace(name)
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}
	stem := strings.TrimSuffix(name, name[len(name)-len(".json"):])
	stem = safeFilePart(stem)
	if stem == "" {
		return ""
	}
	return stem + ".json"
}

func safeFilePart(value string) string {
	value = strings.TrimSpace(value)
	value = repeatedWhitespacePattern.ReplaceAllString(value, " ")
	value = safeFilePartPattern.ReplaceAllString(value, "-")
	value = repeatedDashPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, " .-_")
	if len([]rune(value)) > 80 {
		value = string([]rune(value)[:80])
		value = strings.Trim(value, " .-_")
	}
	return value
}

func parseJWTClaims(token string) jwtClaims {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(padBase64URL(parts[1]))
		if err != nil {
			return jwtClaims{}
		}
	}
	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return jwtClaims{}
	}
	return claims
}

func padBase64URL(value string) string {
	switch len(value) % 4 {
	case 2:
		return value + "=="
	case 3:
		return value + "="
	default:
		return value
	}
}

func objectValue(value any) (map[string]any, bool) {
	if typed, ok := value.(map[string]any); ok {
		return typed, true
	}
	return nil, false
}

func firstString(values ...any) string {
	for _, value := range values {
		if raw, ok := value.(string); ok {
			if trimmed := strings.TrimSpace(raw); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func firstValue(values ...any) any {
	for _, value := range values {
		switch typed := value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(typed) == "" {
				continue
			}
		}
		return value
	}
	return nil
}

func copyStringIfMissing(dst map[string]any, key string, value any) {
	if stringValue(dst[key]) != "" {
		return
	}
	if raw := firstString(value); raw != "" {
		dst[key] = raw
	}
}

func stringValue(value any) string {
	raw, _ := value.(string)
	return strings.TrimSpace(raw)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "codex-auth"
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:10]
}

func marshalPretty(value any) (json.RawMessage, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal converted auth json: %w", err)
	}
	return append(json.RawMessage(nil), raw...), nil
}
