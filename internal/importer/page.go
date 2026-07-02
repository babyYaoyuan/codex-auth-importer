package importer

import (
	_ "embed"
	"strconv"
	"strings"
)

//go:embed web/import.html
var importPageTemplate string

func (p *Plugin) RenderImportPage() []byte {
	return renderImportPage(p.managementKey)
}

func renderImportPage(managementKey string) []byte {
	page := strings.ReplaceAll(importPageTemplate, "__MANAGEMENT_KEY__", jsStringLiteral(managementKey))
	return []byte(page)
}

func jsStringLiteral(value string) string {
	literal := strconv.Quote(value)
	return strings.ReplaceAll(literal, "</", "<\\/")
}
