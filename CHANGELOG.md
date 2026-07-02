# Changelog

## Unreleased

- Remove management-key input from the self-service import page.
- Route page refresh/import actions through the existing plugin Management API endpoints with a build-time or environment-provided management key.
- Redesign the import page around subscription status, target selection, and local auth file import.

## 0.2.4

- Validate existing Codex auth files by calling the Codex quota endpoint through the host HTTP callback.
- Mark an auth file as valid only when quota retrieval succeeds.
- Remove the local CLIProxyAPI module replacement and depend on the public v7.2.30 SDK.
- Add release packaging, GitHub Actions builds, license, checksum generation, and Chinese documentation.

## 0.2.3

- Add a macOS file picker hint for opening `~/.codex/` and selecting `auth.json`.

## 0.2.2

- Fix false expired status when `id_token` is expired but the Codex `access_token` is still valid.

## 0.2.1

- Fix Codex auth expiration detection by reading each auth JSON and checking `expired` plus JWT `exp` values.
- Show the detected expiration timestamp and reason in the auth file list.

## 0.2.0

- Add existing Codex auth file listing from CLIProxyAPI host auth callbacks.
- Show Codex auth status and expired-state hints before importing.
- Support selecting an existing Codex auth file to prefill the target file name for replacement.

## 0.1.0

- Add the initial Codex auth importer Management API plugin.
- Support converting Codex CLI `auth.json` files with nested `tokens` into CLIProxyAPI Codex auth JSON.
- Support pass-through import for existing CLIProxyAPI Codex auth JSON.
- Add release packaging with versioned artifacts and checksums.
