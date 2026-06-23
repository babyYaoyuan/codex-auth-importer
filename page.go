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
    button:hover{background:var(--accent-strong)}button:disabled{opacity:.55;cursor:not-allowed}.actions{display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin-top:4px}
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
      </label>
      <label>保存文件名 <small>可选；留空会自动生成 codex-*.json</small>
        <input id="name" type="text" placeholder="codex-my-account.json" autocomplete="off">
      </label>
      <label>管理密钥 <small>如果当前页面没有管理登录态，在这里填 remote-management.secret-key</small>
        <input id="key" type="password" autocomplete="current-password">
      </label>
      <div class="actions">
        <button id="submit" type="button">导入 Codex auth.json</button>
      </div>
    </div>
    <pre id="result">等待选择文件</pre>
  </section>
</main>
<script>
const fileInput = document.getElementById('file');
const nameInput = document.getElementById('name');
const keyInput = document.getElementById('key');
const button = document.getElementById('submit');
const result = document.getElementById('result');
const savedKey = sessionStorage.getItem('codexAuthImporter.managementKey');
if (savedKey) keyInput.value = savedKey;

function show(text, className) {
  result.className = className || '';
  result.textContent = text;
}

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
    const managementKey = keyInput.value.trim();
    if (managementKey) sessionStorage.setItem('codexAuthImporter.managementKey', managementKey);
    const headers = {'Content-Type': 'application/json'};
    if (managementKey) headers['X-Management-Key'] = managementKey;
    const response = await fetch('/v0/management/plugins/codex-auth-importer/import', {
      method: 'POST',
      credentials: 'same-origin',
      headers,
      body: JSON.stringify({
        filename: file.name,
        name: nameInput.value.trim(),
        content
      })
    });
    const text = await response.text();
    let payload;
    try { payload = JSON.parse(text); } catch (_) { payload = {error: text || response.statusText}; }
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
