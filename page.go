package main

func renderImportPage() []byte {
	return []byte(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>导入 Codex auth.json</title>
  <style>
    :root{color-scheme:light dark;--bg:#f6f7f9;--panel:#fff;--text:#17202a;--muted:#5c6670;--line:#d9dee5;--accent:#2563eb;--accent-strong:#174bb8;--ok:#0f7b45;--err:#b42318}
    @media (prefers-color-scheme:dark){:root{--bg:#111418;--panel:#181d23;--text:#edf1f7;--muted:#aab3bf;--line:#303844;--accent:#6aa6ff;--accent-strong:#8bbcff;--ok:#5bd38e;--err:#ff7d73}}
    *{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:14px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
    main{max-width:760px;margin:0 auto;padding:36px 20px}.panel{background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:24px;box-shadow:0 10px 30px rgba(15,23,42,.06)}
    h1{font-size:22px;line-height:1.2;margin:0 0 18px}.grid{display:grid;gap:16px}label{display:grid;gap:7px;font-weight:600}small{color:var(--muted);font-weight:400}
    input{width:100%;border:1px solid var(--line);border-radius:6px;background:transparent;color:var(--text);padding:10px 11px;font:inherit}
    input[type=file]{padding:9px;background:rgba(127,127,127,.04)}button{border:0;border-radius:6px;background:var(--accent);color:#fff;padding:11px 14px;font:600 14px/1.2 inherit;cursor:pointer}
    .hint{color:var(--muted);font-size:12px;font-weight:400}
    button:hover{background:var(--accent-strong)}button:disabled{opacity:.55;cursor:not-allowed}.actions{display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin-top:4px}
    button.secondary{background:transparent;color:var(--text);border:1px solid var(--line)}button.secondary:hover{border-color:var(--accent);color:var(--accent);background:transparent}
    .files{margin-top:22px;border-top:1px solid var(--line);padding-top:20px}.files-head{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:12px}.files-title{font-weight:700}.files-list{display:grid;gap:8px}
    .file-row{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:10px;align-items:center;border:1px solid var(--line);border-radius:8px;padding:11px;background:rgba(127,127,127,.04)}
    .file-main{min-width:0}.file-name{font-weight:650;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.file-meta{color:var(--muted);font-size:12px;margin-top:3px}.status{display:inline-flex;align-items:center;border-radius:999px;padding:2px 8px;font-size:12px;border:1px solid var(--line);margin-right:6px}.status.expired{color:var(--err);border-color:rgba(180,35,24,.35)}.status.valid{color:var(--ok);border-color:rgba(15,123,69,.35)}
    pre{margin:18px 0 0;white-space:pre-wrap;overflow:auto;border:1px solid var(--line);border-radius:8px;padding:14px;background:rgba(127,127,127,.07);min-height:56px}
    .ok{color:var(--ok)}.err{color:var(--err)}
  </style>
</head>
<body>
<main>
  <section class="panel">
    <h1>导入 Codex auth.json</h1>
    <div class="grid">
      <label>Codex auth.json
        <input id="file" type="file" accept="application/json,.json">
        <span class="hint">macOS 选择文件时按 ⌘⇧G，输入 ~/.codex/，然后选择 auth.json。</span>
      </label>
      <label>保存文件名 <small>可选；留空会自动生成 codex-*.json</small>
        <input id="name" type="text" placeholder="codex-my-account.json" autocomplete="off">
      </label>
      <label>管理密钥 <small>用于调用 CLIProxyAPI 管理接口</small>
        <input id="key" type="password" autocomplete="current-password">
      </label>
      <div class="actions">
        <button id="submit" type="button">导入 Codex auth.json</button>
      </div>
    </div>
    <div class="files">
      <div class="files-head">
        <div class="files-title">已有 Codex 认证文件</div>
        <button id="refreshFiles" class="secondary" type="button">刷新已有文件</button>
      </div>
      <div id="filesList" class="files-list"></div>
    </div>
    <pre id="result">等待选择文件</pre>
  </section>
</main>
<script>
const fileInput = document.getElementById('file');
const nameInput = document.getElementById('name');
const keyInput = document.getElementById('key');
const button = document.getElementById('submit');
const refreshFilesButton = document.getElementById('refreshFiles');
const filesList = document.getElementById('filesList');
const result = document.getElementById('result');
const managementKeyNames = [
  'codexAuthImporter.managementKey',
  'cliproxy.managementKey',
  'cliproxy.management_key',
  'cliproxy.remoteManagementKey',
  'cpa.managementKey',
  'managementKey',
  'management-key',
  'remoteManagementKey'
];
const savedKey = sessionStorage.getItem('codexAuthImporter.managementKey') || findStoredManagementKey();
if (savedKey) keyInput.value = savedKey;

function show(text, className) {
  result.className = className || '';
  result.textContent = text;
}

function findStoredManagementKey() {
  for (const store of [localStorage, sessionStorage]) {
    for (const name of managementKeyNames) {
      try {
        const value = store.getItem(name);
        if (value && value.trim()) return value.trim();
      } catch (_) {}
    }
  }
  return '';
}

function buildHeaders() {
  const headers = {'Content-Type': 'application/json'};
  const managementKey = keyInput.value.trim() || findStoredManagementKey();
  if (managementKey) sessionStorage.setItem('codexAuthImporter.managementKey', managementKey);
  if (managementKey) headers['X-Management-Key'] = managementKey;
  return headers;
}

function escapeHTML(value) {
  return String(value || '').replace(/[&<>"']/g, ch => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  }[ch]));
}

function renderCodexFiles(files) {
  if (!Array.isArray(files) || files.length === 0) {
    filesList.innerHTML = '<div class="file-meta">没有读取到 Codex 认证文件</div>';
    return;
  }
  filesList.innerHTML = files.map(file => {
    const onlineChecked = Boolean(file.quota_checked);
    const statusText = onlineChecked ? (file.valid ? '有效' : '不可用') : (file.expired ? '已过期' : '未过期');
    const statusClass = (onlineChecked ? file.valid : !file.expired) ? 'valid' : 'expired';
    const expiresText = file.expires_at ? '到期 ' + file.expires_at : '';
    const reason = file.valid_reason || file.expired_reason;
    const meta = [file.email, file.account, file.status, reason, expiresText, file.status_message, file.last_refresh ? '刷新 ' + file.last_refresh : '']
      .filter(Boolean)
      .join(' · ');
    return '<div class="file-row">' +
      '<div class="file-main">' +
      '<div class="file-name" title="' + escapeHTML(file.name) + '">' + escapeHTML(file.name) + '</div>' +
      '<div class="file-meta"><span class="status ' + statusClass + '">' + statusText + '</span>' + escapeHTML(meta) + '</div>' +
      '</div>' +
      '<button class="secondary select-file" type="button" data-name="' + escapeHTML(file.name) + '">选择替换</button>' +
      '</div>';
  }).join('');
}

async function refreshCodexFiles() {
  refreshFilesButton.disabled = true;
  show('读取已有 Codex 认证文件...');
  try {
    const response = await fetch('/v0/management/plugins/codex-auth-importer/auth-files', {
      method: 'POST',
      credentials: 'same-origin',
      headers: buildHeaders(),
      body: '{}'
    });
    const text = await response.text();
    let payload;
    try { payload = JSON.parse(text); } catch (_) { payload = {error: text || response.statusText}; }
    if (response.status === 401 || response.status === 403) {
      show('管理密钥无效或缺失，请填写 CLIProxyAPI 管理密钥后重试。', 'err');
      return;
    }
    if (!response.ok || payload.error) {
      show(payload.error || ('HTTP ' + response.status), 'err');
      return;
    }
    renderCodexFiles(payload.files || []);
    show('已有文件已刷新', 'ok');
  } catch (error) {
    show(error && error.message ? error.message : String(error), 'err');
  } finally {
    refreshFilesButton.disabled = false;
  }
}

refreshFilesButton.addEventListener('click', refreshCodexFiles);
filesList.addEventListener('click', event => {
  const target = event.target.closest('.select-file');
  if (!target) return;
  nameInput.value = target.dataset.name || '';
  show('已选择替换：' + nameInput.value, 'ok');
});

button.addEventListener('click', async () => {
  const file = fileInput.files && fileInput.files[0];
  if (!file) {
    show('请先选择 auth.json 文件', 'err');
    return;
  }
  button.disabled = true;
  show('导入中...');
  try {
    const content = await file.text();
    const response = await fetch('/v0/management/plugins/codex-auth-importer/import', {
      method: 'POST',
      credentials: 'same-origin',
      headers: buildHeaders(),
      body: JSON.stringify({
        filename: file.name,
        name: nameInput.value.trim(),
        content
      })
    });
    const text = await response.text();
    let payload;
    try { payload = JSON.parse(text); } catch (_) { payload = {error: text || response.statusText}; }
    if (response.status === 401 || response.status === 403) {
      show('管理密钥无效或缺失，请填写 CLIProxyAPI 管理密钥后重试。', 'err');
      return;
    }
    if (!response.ok || payload.error) {
      show(payload.error || ('HTTP ' + response.status), 'err');
      return;
    }
    show(JSON.stringify(payload, null, 2), 'ok');
  } catch (error) {
    show(error && error.message ? error.message : String(error), 'err');
  } finally {
    button.disabled = false;
  }
});
</script>
</body>
</html>`)
}
