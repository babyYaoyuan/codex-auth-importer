# Codex Auth Importer

Management API plugin for CLIProxyAPI that imports a Codex CLI `auth.json` file and saves it as a CLIProxyAPI Codex auth file.

## Build

```bash
make test
make build
```

The plugin binary is written to `dist/codex-auth-importer.dylib` on macOS.

## Install locally

From this project:

```bash
make build
mkdir -p ../CLIProxyAPI/plugins
cp dist/codex-auth-importer.dylib ../CLIProxyAPI/plugins/
```

Then enable it in the CLIProxyAPI `config.yaml`:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    codex-auth-importer:
      enabled: true
      priority: 1
```

Start CLIProxyAPI and open:

```text
http://127.0.0.1:8317/v0/resource/plugins/codex-auth-importer/import
```

## Release

```bash
make release
```

The release archive is written under `dist/release/` and uses this name format:

```text
codex-auth-importer_<version>_<goos>_<goarch>.zip
```
