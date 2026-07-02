package importer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const (
	pluginID                = "codex-auth-importer"
	resourcePath            = "/import"
	managementPath          = "/plugins/codex-auth-importer/import"
	authFilesManagementPath = "/plugins/codex-auth-importer/auth-files"
	defaultPluginLogo       = "https://raw.githubusercontent.com/router-for-me/CLIProxyAPI/main/docs/logo.png"
)

type Config struct {
	Version       string
	ManagementKey string
	Host          HostClient
}

type HostClient interface {
	AuthSave(name string, rawJSON json.RawMessage) (pluginapi.HostAuthSaveResponse, error)
	AuthList() ([]pluginapi.HostAuthFileEntry, error)
	AuthGet(authIndex string) (pluginapi.HostAuthGetResponse, error)
	HTTPDo(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error)
}

type Plugin struct {
	version       string
	managementKey string
	host          HostClient
}

func New(config Config) *Plugin {
	version := strings.TrimSpace(config.Version)
	if version == "" {
		version = "dev"
	}
	host := config.Host
	if host == nil {
		host = noHostClient{}
	}
	return &Plugin{
		version:       version,
		managementKey: strings.TrimSpace(config.ManagementKey),
		host:          host,
	}
}

func (p *Plugin) HandleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		return okEnvelope(p.pluginRegistration())
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
		return p.handleManagement(request)
	default:
		return ErrorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func (p *Plugin) pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             pluginID,
			Version:          p.version,
			Author:           "babyYaoyuan",
			GitHubRepository: "https://github.com/babyYaoyuan/codex-auth-importer",
			Logo:             defaultPluginLogo,
			ConfigFields:     []pluginapi.ConfigField{},
		},
		Capabilities: registrationCapabilities{
			ManagementAPI: true,
		},
	}
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

type noHostClient struct{}

func (noHostClient) AuthSave(string, json.RawMessage) (pluginapi.HostAuthSaveResponse, error) {
	return pluginapi.HostAuthSaveResponse{}, fmt.Errorf("host auth save callback unavailable")
}

func (noHostClient) AuthList() ([]pluginapi.HostAuthFileEntry, error) {
	return nil, fmt.Errorf("host auth list callback unavailable")
}

func (noHostClient) AuthGet(string) (pluginapi.HostAuthGetResponse, error) {
	return pluginapi.HostAuthGetResponse{}, fmt.Errorf("host auth get callback unavailable")
}

func (noHostClient) HTTPDo(pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
	return pluginapi.HTTPResponse{}, fmt.Errorf("host http callback unavailable")
}
