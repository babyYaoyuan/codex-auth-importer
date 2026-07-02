# Codex Auth Importer

English | [简体中文](README.zh-CN.md)

`codex-auth-importer` is a CLIProxyAPI Management API plugin that imports a Codex CLI `auth.json` file and saves it as a CLIProxyAPI Codex auth file.

## Features

- Imports Codex CLI `auth.json` files from `~/.codex/auth.json`.
- Converts nested Codex token metadata into CLIProxyAPI's Codex auth JSON shape.
- Lists existing CLIProxyAPI Codex auth files and lets the user select a target file name for replacement.
- Validates existing Codex auth files by querying the Codex quota endpoint. A file is shown as valid only when quota retrieval succeeds.
- Provides a standalone Management resource page, so no extra frontend build is required.

## Requirements

- CLIProxyAPI v7.2.30 or newer.
- Go 1.26+ for local development.
- CGO enabled when building the shared library.
- A CLIProxyAPI management key baked into the plugin at build time, or provided by the `CODEX_AUTH_IMPORTER_MANAGEMENT_KEY` runtime environment variable.

## CLIProxyAPI Configuration

Enable plugins and this plugin in `config.yaml`:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    codex-auth-importer:
      enabled: true
      priority: 1
```

The resource page is available at:

```text
/v0/resource/plugins/codex-auth-importer/import
```

The self-service page calls the plugin's existing Management API endpoints for refresh and import. It does not render a management-key input, but it automatically sends `X-Management-Key` when a key is baked into the shared library or provided by `CODEX_AUTH_IMPORTER_MANAGEMENT_KEY`.

The plugin registers these Management API routes:

```text
POST /v0/management/plugins/codex-auth-importer/import
POST /v0/management/plugins/codex-auth-importer/auth-files
```

## Build

Run tests and build for the current platform:

```bash
make test
make vet
make build
```

The default output is:

- macOS: `dist/codex-auth-importer.dylib`
- Linux: `dist/codex-auth-importer.so`
- Windows: `dist/codex-auth-importer.dll`

Build for an explicit platform:

```bash
make build GOOS=linux GOARCH=amd64
make build GOOS=darwin GOARCH=arm64
make build GOOS=windows GOARCH=amd64
```

Build with a baked management key:

```bash
make build MANAGEMENT_KEY='YOUR_MANAGEMENT_KEY'
```

Alternatively, set `CODEX_AUTH_IMPORTER_MANAGEMENT_KEY` in the CLIProxyAPI process environment and restart CLIProxyAPI.

## Build Linux amd64 on Apple Silicon

For a Linux Docker server running `x86_64`/`amd64`, build the `.so` locally on macOS with Zig:

```bash
brew install zig

cd /path/to/codex-auth-importer
mkdir -p dist

CGO_ENABLED=1 \
GOOS=linux \
GOARCH=amd64 \
CC="zig cc -target x86_64-linux-gnu" \
go build \
  -trimpath \
  -buildmode=c-shared \
  -ldflags "-s -w -X main.version=$(cat VERSION)" \
  -o dist/codex-auth-importer.so \
  .
```

Verify the output:

```bash
file dist/codex-auth-importer.so
```

Expected output includes:

```text
ELF 64-bit LSB shared object, x86-64
```

The same build can be driven through the Makefile:

```bash
make package GOOS=linux GOARCH=amd64 CC="zig cc -target x86_64-linux-gnu" PLUGIN_BIN=dist/codex-auth-importer.so
```

To bake the management key into the Linux package:

```bash
make package GOOS=linux GOARCH=amd64 CC="zig cc -target x86_64-linux-gnu" PLUGIN_BIN=dist/codex-auth-importer.so MANAGEMENT_KEY='YOUR_MANAGEMENT_KEY'
```

## Install Locally

From this project:

```bash
make build
mkdir -p ../CLIProxyAPI/plugins
cp dist/codex-auth-importer.dylib ../CLIProxyAPI/plugins/
```

Start or restart CLIProxyAPI, then open:

```text
http://127.0.0.1:8317/v0/resource/plugins/codex-auth-importer/import
```

## Install on a Linux Docker Server

Build `dist/codex-auth-importer.so`, then upload only the shared library:

```bash
scp dist/codex-auth-importer.so \
  root@YOUR_SERVER:/opt/apps/cli-proxy-api/plugins/codex-auth-importer.so
```

If CLIProxyAPI runs in Docker, mount the plugin directory into the container:

```yaml
services:
  cli-proxy-api:
    volumes:
      - ${CLI_PROXY_CONFIG_PATH:-./config.yaml}:/CLIProxyAPI/config.yaml
      - ${CLI_PROXY_AUTH_PATH:-./auths}:/root/.cli-proxy-api
      - ${CLI_PROXY_LOG_PATH:-./logs}:/CLIProxyAPI/logs
      - ./plugins:/CLIProxyAPI/plugins
```

Restart the container after replacing the shared library:

```bash
docker compose restart
```

## Auth Validity Check

The existing-file list does not trust only local JWT timestamps. For each Codex auth file, the plugin:

1. Reads the saved auth JSON through CLIProxyAPI host auth callbacks.
2. Extracts the Codex `access_token` and optional account ID.
3. Calls `https://chatgpt.com/backend-api/wham/usage` through the CLIProxyAPI host HTTP callback.
4. Marks the auth file as valid only when the quota request returns a successful HTTP status and a valid JSON response.

Server-side invalidation errors such as `401 Your authentication token has been invalidated` are shown as unavailable.

## Security Notes

- The uploaded Codex `auth.json` is parsed in the plugin page request and converted before saving.
- The original Codex CLI file shape is not persisted separately by the plugin.
- The self-service page does not render a management-key input or store the key in browser storage.
- In baked-key mode, the key is sent to the browser as part of the page script so that fetch requests can pass CLIProxyAPI management authentication.
- The plugin does not log tokens.

## Release

Create a release package for the current platform:

```bash
make release
```

The release archive is written under `dist/release/` and uses this name format:

```text
codex-auth-importer_<version>_<goos>_<goarch>.zip
```

The package includes the shared library, README files, changelog, version file, and license. A `.zip.sha256` checksum file is generated next to the archive.

GitHub Actions builds Linux, macOS, and Windows packages. Pushing a tag like `v0.2.4` publishes a GitHub release with all archives and checksums.
