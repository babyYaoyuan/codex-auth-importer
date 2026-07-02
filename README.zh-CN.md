# Codex Auth Importer

[English](README.md) | 简体中文

`codex-auth-importer` 是一个 CLIProxyAPI Management API 插件，用于导入 Codex CLI 的 `auth.json`，并保存成 CLIProxyAPI 可识别的 Codex 认证文件。

## 功能

- 导入 Codex CLI 的 `~/.codex/auth.json`。
- 自动把 Codex CLI 的嵌套 token 信息转换成 CLIProxyAPI Codex auth JSON。
- 读取已有 CLIProxyAPI Codex 认证文件，并支持选择某个文件名作为替换目标。
- 通过 Codex 额度接口校验现有认证文件是否可用。只有额度查询成功，才显示为有效。
- 提供独立 Management 资源页面，不需要额外前端构建。

## 要求

- CLIProxyAPI v7.2.30 或更新版本。
- 本地开发需要 Go 1.26+。
- 构建动态库时需要启用 CGO。
- 构建时写入 CLIProxyAPI 管理密钥，或者在 CLIProxyAPI 进程环境变量中设置 `CODEX_AUTH_IMPORTER_MANAGEMENT_KEY`。

## CLIProxyAPI 配置

在 `config.yaml` 中启用插件：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    codex-auth-importer:
      enabled: true
      priority: 1
```

插件页面地址：

```text
/v0/resource/plugins/codex-auth-importer/import
```

自助页面会调用插件已有 Management API 完成导入和刷新。页面不会渲染管理密钥输入框，但当动态库内置了密钥或进程环境变量提供了密钥时，会自动在请求中携带 `X-Management-Key`。

插件注册的 Management API：

```text
POST /v0/management/plugins/codex-auth-importer/import
POST /v0/management/plugins/codex-auth-importer/auth-files
```

## 本地构建

```bash
make test
make vet
make build
```

默认输出：

- macOS: `dist/codex-auth-importer.dylib`
- Linux: `dist/codex-auth-importer.so`
- Windows: `dist/codex-auth-importer.dll`

指定平台构建：

```bash
make build GOOS=linux GOARCH=amd64
make build GOOS=darwin GOARCH=arm64
make build GOOS=windows GOARCH=amd64
```

构建时内置管理密钥：

```bash
make build MANAGEMENT_KEY='YOUR_MANAGEMENT_KEY'
```

也可以在 CLIProxyAPI 进程环境变量中设置 `CODEX_AUTH_IMPORTER_MANAGEMENT_KEY`，然后重启 CLIProxyAPI。

## 在 Apple Silicon 上编译 Linux amd64 插件

如果服务器是 Linux Docker `x86_64`/`amd64`，可以在 macOS 本地用 Zig 直接编译 `.so`：

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

验证产物：

```bash
file dist/codex-auth-importer.so
```

期望包含：

```text
ELF 64-bit LSB shared object, x86-64
```

也可以直接通过 Makefile 构建并打包：

```bash
make package GOOS=linux GOARCH=amd64 CC="zig cc -target x86_64-linux-gnu" PLUGIN_BIN=dist/codex-auth-importer.so
```

如果要把管理密钥内置到 Linux 包：

```bash
make package GOOS=linux GOARCH=amd64 CC="zig cc -target x86_64-linux-gnu" PLUGIN_BIN=dist/codex-auth-importer.so MANAGEMENT_KEY='YOUR_MANAGEMENT_KEY'
```

## 本地安装

```bash
make build
mkdir -p ../CLIProxyAPI/plugins
cp dist/codex-auth-importer.dylib ../CLIProxyAPI/plugins/
```

启动或重启 CLIProxyAPI，然后打开：

```text
http://127.0.0.1:8317/v0/resource/plugins/codex-auth-importer/import
```

## Linux Docker 服务器安装

先构建 `dist/codex-auth-importer.so`，然后只上传动态库：

```bash
scp dist/codex-auth-importer.so \
  root@YOUR_SERVER:/opt/apps/cli-proxy-api/plugins/codex-auth-importer.so
```

如果 CLIProxyAPI 运行在 Docker 中，把插件目录挂载进容器：

```yaml
services:
  cli-proxy-api:
    volumes:
      - ${CLI_PROXY_CONFIG_PATH:-./config.yaml}:/CLIProxyAPI/config.yaml
      - ${CLI_PROXY_AUTH_PATH:-./auths}:/root/.cli-proxy-api
      - ${CLI_PROXY_LOG_PATH:-./logs}:/CLIProxyAPI/logs
      - ./plugins:/CLIProxyAPI/plugins
```

替换插件后重启容器：

```bash
docker compose restart
```

## 有效性判断

已有文件列表不会只相信本地 JWT 时间。插件会：

1. 通过 CLIProxyAPI host auth callback 读取保存的 auth JSON。
2. 提取 Codex `access_token` 和可选 account ID。
3. 通过 CLIProxyAPI host HTTP callback 请求 `https://chatgpt.com/backend-api/wham/usage`。
4. 只有额度请求返回成功 HTTP 状态且响应是有效 JSON，才标记为有效。

如果服务端返回 `401 Your authentication token has been invalidated`，页面会显示不可用。

## 安全说明

- 上传的 Codex `auth.json` 只在导入请求中解析并转换。
- 插件不会额外保存原始 Codex CLI 文件格式。
- 自助页面不渲染管理密钥输入框，也不会把管理密钥写入浏览器存储。
- 内置密钥模式会把密钥写入页面脚本中，用于让浏览器请求通过 CLIProxyAPI 管理鉴权。
- 插件不会记录 token。

## 发布

```bash
make release
```

发布包位于 `dist/release/`，命名格式：

```text
codex-auth-importer_<version>_<goos>_<goarch>.zip
```

压缩包包含动态库、README、CHANGELOG、VERSION 和 LICENSE。旁边会生成 `.zip.sha256` 校验文件。

GitHub Actions 会构建 Linux、macOS、Windows 包。推送 `v0.2.4` 这类 tag 时会自动创建 GitHub Release。
