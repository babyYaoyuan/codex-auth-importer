package importer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type importRequest struct {
	Name     string `json:"name,omitempty"`
	Filename string `json:"filename,omitempty"`
	Content  string `json:"content"`
}

type importResponse struct {
	SavedName    string `json:"saved_name"`
	SavedPath    string `json:"saved_path,omitempty"`
	SourceFormat string `json:"source_format"`
	Email        string `json:"email,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
	Version      string `json:"version"`
}

type codexAuthFilesResponse struct {
	Files   []codexAuthFileEntry `json:"files"`
	Version string               `json:"version"`
}

func (p *Plugin) handleManagement(raw []byte) ([]byte, error) {
	var req managementRequest
	if len(raw) > 0 {
		if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
			return nil, fmt.Errorf("decode management request: %w", errUnmarshal)
		}
	}
	if strings.EqualFold(req.Method, http.MethodPost) && strings.HasSuffix(req.Path, authFilesManagementPath) {
		return p.handleCodexAuthFiles()
	}
	if strings.EqualFold(req.Method, http.MethodPost) && strings.HasSuffix(req.Path, managementPath) {
		return p.handleImport(req)
	}
	return okEnvelope(htmlResponse(http.StatusOK, p.RenderImportPage()))
}

func (p *Plugin) handleCodexAuthFiles() ([]byte, error) {
	files, errList := p.host.AuthList()
	if errList != nil {
		return okEnvelope(jsonResponse(http.StatusBadGateway, map[string]any{
			"error": errList.Error(),
		}))
	}
	rawByAuthIndex := make(map[string]json.RawMessage)
	quotaByAuthIndex := make(map[string]codexQuotaProbeResult)
	for i := range files {
		if !isCodexAuthEntry(files[i]) {
			continue
		}
		authIndex := strings.TrimSpace(files[i].AuthIndex)
		if authIndex == "" {
			quotaByAuthIndex[files[i].AuthIndex] = codexQuotaProbeResult{
				Checked: true,
				Valid:   false,
				Message: "缺少 auth_index，无法查询额度",
			}
			continue
		}
		authFile, errGet := p.host.AuthGet(authIndex)
		if errGet != nil {
			message := authGetErrorMessage(files[i], errGet)
			files[i].StatusMessage = message
			quotaByAuthIndex[authIndex] = codexQuotaProbeResult{
				Checked: true,
				Valid:   false,
				Message: message,
			}
			continue
		}
		rawByAuthIndex[authIndex] = authFile.JSON
		quotaByAuthIndex[authIndex] = probeCodexQuotaWithDoer(authFile.JSON, p.host.HTTPDo)
	}
	return okEnvelope(jsonResponse(http.StatusOK, codexAuthFilesResponse{
		Files:   codexAuthFilesFromHostWithQuotaAt(files, rawByAuthIndex, quotaByAuthIndex, time.Now()),
		Version: p.version,
	}))
}

func (p *Plugin) handleImport(req managementRequest) ([]byte, error) {
	var payload importRequest
	if errUnmarshal := json.Unmarshal(req.Body, &payload); errUnmarshal != nil {
		return okEnvelope(jsonResponse(http.StatusBadRequest, map[string]any{
			"error": "invalid JSON request body",
		}))
	}
	converted, errConvert := convertCodexAuthJSON([]byte(payload.Content), importOptions{
		RequestedName: payload.Name,
		UploadedName:  payload.Filename,
	})
	if errConvert != nil {
		return okEnvelope(jsonResponse(http.StatusBadRequest, map[string]any{
			"error": errConvert.Error(),
		}))
	}

	saved, errSave := p.host.AuthSave(converted.FileName, converted.JSON)
	if errSave != nil {
		return okEnvelope(jsonResponse(http.StatusBadGateway, map[string]any{
			"error": errSave.Error(),
		}))
	}
	return okEnvelope(jsonResponse(http.StatusOK, importResponse{
		SavedName:    saved.Name,
		SavedPath:    saved.Path,
		SourceFormat: converted.SourceFormat,
		Email:        converted.Email,
		AccountID:    converted.AccountID,
		Version:      p.version,
	}))
}
