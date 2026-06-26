package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef int (*cliproxy_host_call_fn)(void*, const char*, const uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	cliproxy_host_call_fn call;
	cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);

static const cliproxy_host_api* stored_host;

static void store_host_api(const cliproxy_host_api* host) {
	stored_host = host;
}

static int call_host_api(const char* method, const uint8_t* request, size_t request_len, cliproxy_buffer* response) {
	if (stored_host == NULL || stored_host->call == NULL) {
		return 1;
	}
	return stored_host->call(stored_host->host_ctx, method, request, request_len, response);
}

static void free_host_buffer(void* ptr, size_t len) {
	if (stored_host != NULL && stored_host->free_buffer != NULL && ptr != NULL) {
		stored_host->free_buffer(ptr, len);
	}
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unsafe"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const (
	pluginID                = "codex-auth-importer"
	resourcePath            = "/import"
	managementPath          = "/plugins/codex-auth-importer/import"
	authFilesManagementPath = "/plugins/codex-auth-importer/auth-files"
	contentTypeHTML         = "text/html; charset=utf-8"
	contentTypeJSON         = "application/json; charset=utf-8"
	defaultPluginLogo       = "https://raw.githubusercontent.com/router-for-me/CLIProxyAPI/main/docs/logo.png"
)

var version = "0.2.4"

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type registration struct {
	SchemaVersion uint32                   `json:"schema_version"`
	Metadata      pluginapi.Metadata       `json:"metadata"`
	Capabilities  registrationCapabilities `json:"capabilities"`
}

type registrationCapabilities struct {
	ManagementAPI bool `json:"management_api"`
}

type managementRegistration struct {
	Routes    []managementRoute    `json:"routes,omitempty"`
	Resources []managementResource `json:"resources,omitempty"`
}

type managementRoute struct {
	Method      string `json:"Method"`
	Path        string `json:"Path"`
	Description string `json:"Description,omitempty"`
}

type managementResource struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu"`
	Description string `json:"Description"`
}

type managementRequest struct {
	Method         string
	Path           string
	Headers        http.Header
	Query          url.Values
	Body           []byte
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type managementResponse struct {
	StatusCode int         `json:"StatusCode"`
	Headers    http.Header `json:"Headers"`
	Body       []byte      `json:"Body"`
}

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

type authListResponse struct {
	Files []pluginapi.HostAuthFileEntry `json:"files"`
}

type codexAuthFilesResponse struct {
	Files   []codexAuthFileEntry `json:"files"`
	Version string               `json:"version"`
}

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	C.store_host_api(host)
	plugin.abi_version = C.uint32_t(pluginabi.ABIVersion)
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, errorEnvelope("invalid_method", "method is required"))
		return 1
	}
	var requestBytes []byte
	if request != nil && requestLen > 0 {
		requestBytes = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}
	raw, errHandle := handleMethod(C.GoString(method), requestBytes)
	if errHandle != nil {
		writeResponse(response, errorEnvelope("plugin_error", errHandle.Error()))
		return 1
	}
	writeResponse(response, raw)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, len C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
	_ = len
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		return okEnvelope(pluginRegistration())
	case pluginabi.MethodManagementRegister:
		return okEnvelope(managementRegistration{
			Routes: []managementRoute{
				{
					Method:      http.MethodPost,
					Path:        managementPath,
					Description: "Imports Codex CLI auth.json and saves a CLIProxyAPI Codex auth file.",
				},
				{
					Method:      http.MethodPost,
					Path:        authFilesManagementPath,
					Description: "Lists existing CLIProxyAPI Codex auth files for replacement.",
				},
			},
			Resources: []managementResource{{
				Path:        resourcePath,
				Menu:        "导入 Codex auth.json",
				Description: "Import Codex CLI auth.json as a CLIProxyAPI Codex auth file.",
			}},
		})
	case pluginabi.MethodManagementHandle:
		return handleManagement(request)
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             pluginID,
			Version:          version,
			Author:           "router-for-me",
			GitHubRepository: "https://github.com/router-for-me/CLIProxyAPI",
			Logo:             defaultPluginLogo,
			ConfigFields:     []pluginapi.ConfigField{},
		},
		Capabilities: registrationCapabilities{
			ManagementAPI: true,
		},
	}
}

func handleManagement(raw []byte) ([]byte, error) {
	var req managementRequest
	if len(raw) > 0 {
		if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
			return nil, fmt.Errorf("decode management request: %w", errUnmarshal)
		}
	}
	if strings.EqualFold(req.Method, http.MethodPost) && strings.HasSuffix(req.Path, authFilesManagementPath) {
		return handleCodexAuthFiles()
	}
	if strings.EqualFold(req.Method, http.MethodPost) && strings.HasSuffix(req.Path, managementPath) {
		return handleImport(req)
	}
	return okEnvelope(htmlResponse(http.StatusOK, renderImportPage()))
}

func handleCodexAuthFiles() ([]byte, error) {
	files, errList := callHostAuthList()
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
		authFile, errGet := callHostAuthGet(authIndex)
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
		quotaByAuthIndex[authIndex] = probeCodexQuota(authFile.JSON)
	}
	return okEnvelope(jsonResponse(http.StatusOK, codexAuthFilesResponse{
		Files:   codexAuthFilesFromHostWithQuotaAt(files, rawByAuthIndex, quotaByAuthIndex, time.Now()),
		Version: version,
	}))
}

func handleImport(req managementRequest) ([]byte, error) {
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

	saved, errSave := callHostAuthSave(converted.FileName, converted.JSON)
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
		Version:      version,
	}))
}

func callHostAuthSave(name string, rawJSON json.RawMessage) (pluginapi.HostAuthSaveResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostAuthSave, pluginapi.HostAuthSaveRequest{
		Name: name,
		JSON: rawJSON,
	})
	if errCall != nil {
		return pluginapi.HostAuthSaveResponse{}, errCall
	}
	var resp pluginapi.HostAuthSaveResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return pluginapi.HostAuthSaveResponse{}, fmt.Errorf("decode host.auth.save result: %w", errUnmarshal)
	}
	return resp, nil
}

func callHostAuthList() ([]pluginapi.HostAuthFileEntry, error) {
	result, errCall := callHost(pluginabi.MethodHostAuthList, map[string]any{})
	if errCall != nil {
		return nil, errCall
	}
	var resp authListResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return nil, fmt.Errorf("decode host.auth.list result: %w", errUnmarshal)
	}
	return resp.Files, nil
}

func callHostAuthGet(authIndex string) (pluginapi.HostAuthGetResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostAuthGet, pluginapi.HostAuthGetRequest{
		AuthIndex: authIndex,
	})
	if errCall != nil {
		return pluginapi.HostAuthGetResponse{}, errCall
	}
	var resp pluginapi.HostAuthGetResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return pluginapi.HostAuthGetResponse{}, fmt.Errorf("decode host.auth.get result: %w", errUnmarshal)
	}
	return resp, nil
}

func callHostHTTPDo(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostHTTPDo, req)
	if errCall != nil {
		return pluginapi.HTTPResponse{}, errCall
	}
	var resp pluginapi.HTTPResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return pluginapi.HTTPResponse{}, fmt.Errorf("decode host.http.do result: %w", errUnmarshal)
	}
	return resp, nil
}

func callHost(method string, payload any) (json.RawMessage, error) {
	rawPayload, errMarshal := json.Marshal(payload)
	if errMarshal != nil {
		return nil, fmt.Errorf("marshal host callback payload %s: %w", method, errMarshal)
	}
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))

	var response C.cliproxy_buffer
	var requestPtr *C.uint8_t
	if len(rawPayload) > 0 {
		cPayload := C.CBytes(rawPayload)
		if cPayload == nil {
			return nil, fmt.Errorf("allocate host callback payload %s", method)
		}
		defer C.free(cPayload)
		requestPtr = (*C.uint8_t)(cPayload)
	}
	callCode := C.call_host_api(cMethod, requestPtr, C.size_t(len(rawPayload)), &response)
	var rawResponse []byte
	if response.ptr != nil && response.len > 0 {
		rawResponse = C.GoBytes(response.ptr, C.int(response.len))
	}
	if response.ptr != nil {
		C.free_host_buffer(response.ptr, response.len)
	}
	if len(rawResponse) == 0 {
		return nil, fmt.Errorf("host callback %s returned no response, code=%d", method, int(callCode))
	}

	var env envelope
	if errUnmarshal := json.Unmarshal(rawResponse, &env); errUnmarshal != nil {
		return nil, fmt.Errorf("decode host callback envelope %s: %w", method, errUnmarshal)
	}
	if !env.OK {
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("host callback %s failed", method)
	}
	if callCode != 0 {
		return nil, fmt.Errorf("host callback %s returned code=%d", method, int(callCode))
	}
	return append(json.RawMessage(nil), env.Result...), nil
}

func htmlResponse(statusCode int, body []byte) managementResponse {
	return managementResponse{
		StatusCode: statusCode,
		Headers: http.Header{
			"Content-Type": []string{contentTypeHTML},
		},
		Body: body,
	}
}

func jsonResponse(statusCode int, payload any) managementResponse {
	raw, errMarshal := json.MarshalIndent(payload, "", "  ")
	if errMarshal != nil {
		raw = []byte(`{"error":"failed to marshal response"}`)
	}
	return managementResponse{
		StatusCode: statusCode,
		Headers: http.Header{
			"Content-Type": []string{contentTypeJSON},
		},
		Body: raw,
	}
}

func okEnvelope(v any) ([]byte, error) {
	raw, errMarshal := json.Marshal(v)
	if errMarshal != nil {
		return nil, errMarshal
	}
	return json.Marshal(envelope{OK: true, Result: raw})
}

func errorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(envelope{OK: false, Error: &envelopeError{Code: code, Message: message}})
	return raw
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}
