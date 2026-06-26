package main

import (
	"strings"
	"testing"
)

func TestImportPageExposesManagementKeyField(t *testing.T) {
	page := string(renderImportPage())
	for _, required := range []string{
		"管理密钥",
		`id="key"`,
		"codexAuthImporter.managementKey",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want manual management key field", required)
		}
	}
}

func TestImportPageProvidesExistingCodexFileSelection(t *testing.T) {
	page := string(renderImportPage())
	for _, required := range []string{
		"已有 Codex 认证文件",
		"刷新已有文件",
		"选择替换",
		"auth-files",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want existing Codex file selection UI", required)
		}
	}
}

func TestImportPageShowsMacFileDialogDirectoryHint(t *testing.T) {
	page := string(renderImportPage())
	for _, required := range []string{
		"⌘⇧G",
		"~/.codex/",
		"auth.json",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want macOS file dialog directory hint", required)
		}
	}
}
