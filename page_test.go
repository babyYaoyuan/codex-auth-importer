package main

import (
	"strings"
	"testing"
)

func TestImportPageDoesNotRenderManagementKeyInput(t *testing.T) {
	page := string(renderImportPage())
	for _, forbidden := range []string{
		`id="key"`,
		"codexAuthImporter.managementKey",
		"localStorage",
		"sessionStorage",
	} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("renderImportPage() contains %q, want self-service page without management key input or browser storage", forbidden)
		}
	}
}

func TestImportPageProvidesSubscriptionStatusAndSelection(t *testing.T) {
	page := string(renderImportPage())
	for _, required := range []string{
		"订阅状态",
		"刷新状态",
		"选择替换",
		"auth-files",
		"refreshCodexFiles();",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want subscription status and selection UI", required)
		}
	}
}

func TestImportPageUsesWideTableLayout(t *testing.T) {
	page := string(renderImportPage())
	for _, required := range []string{
		"max-width:1680px",
		`<col class="col-subscription">`,
		`<col class="col-account">`,
		"min-width:1160px",
		"<th>操作</th>",
		"position:sticky",
		`aria-live="polite"`,
		"formatDateTime(file.expires_at)",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want wide readable subscription table layout", required)
		}
	}
}

func TestImportPageUsesBlockingBusyOverlay(t *testing.T) {
	page := string(renderImportPage())
	for _, required := range []string{
		`id="busyOverlay"`,
		"正在刷新订阅状态",
		"请等待当前操作完成",
		"setBusy(true, '正在刷新订阅状态')",
		`appMain.toggleAttribute('inert', active)`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want blocking refresh/import loading overlay", required)
		}
	}
}

func TestImportPageSelectionOnlyUsesSelectButton(t *testing.T) {
	page := string(renderImportPage())
	if !strings.Contains(page, `event.target.closest('.select-file')`) {
		t.Fatalf("renderImportPage() missing select button click handler")
	}
	if strings.Contains(page, `event.target.closest('.row')`) {
		t.Fatalf("renderImportPage() selects rows by clicking the whole row, want select button only")
	}
}

func TestImportPageSupportsCustomSubscriptionName(t *testing.T) {
	page := string(renderImportPage())
	for _, required := range []string{
		"订阅名称",
		`id="customName"`,
		"请先选择 auth.json",
		"const customNameInput = document.getElementById('customName');",
		"customNameInput.disabled = !hasFile;",
		"例如 张三 @ Plus",
		"const hasName = currentSubscriptionName() !== '';",
		"const requestedName = currentSubscriptionName();",
		"name: requestedName",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want custom subscription name support", required)
		}
	}
}

func TestImportPageFormatsDateTimeWithSeconds(t *testing.T) {
	page := string(renderImportPage())
	for _, required := range []string{
		"function padDatePart(value)",
		"return y + '-' + m + '-' + d + ' ' + hh + ':' + mm + ':' + ss;",
		"formatDateTime(file.expires_at)",
		"formatDateTime(file.last_refresh)",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want YYYY-MM-DD HH:mm:ss date formatting", required)
		}
	}
}

func TestImportPageUsesManagementActionEndpoints(t *testing.T) {
	page := string(renderImportPage())
	for _, required := range []string{
		"apiURL('auth-files')",
		"apiURL('import')",
		"/v0/management/plugins/codex-auth-importer/auth-files",
		"/v0/management/plugins/codex-auth-importer/import",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want management action endpoints", required)
		}
	}
}

func TestImportPageInjectsManagementKeyHeader(t *testing.T) {
	oldKey := managementKey
	managementKey = "test-management-key"
	t.Cleanup(func() {
		managementKey = oldKey
	})

	page := string(renderImportPage())
	for _, required := range []string{
		`const MANAGEMENT_KEY = "test-management-key";`,
		`headers['X-Management-Key'] = MANAGEMENT_KEY;`,
		"ensureManagementKey()",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("renderImportPage() missing %q, want baked management key request header", required)
		}
	}
}
