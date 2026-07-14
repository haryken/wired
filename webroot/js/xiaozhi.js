// Xiaozhi UI helpers

function xzSetStatus(msg, isError) {
    const el = document.getElementById('xiaozhiStatus');
    if (!el) return;
    el.textContent = msg;
    el.style.color = isError ? '#f87171' : '#4ade80';
    el.style.display = msg ? 'block' : 'none';
}

function xzApplyCfg(cfg) {
    if (!cfg) return;
    const ota = document.getElementById('xzOTABaseURL');
    const ep = document.getElementById('xzEndpoint');
    const did = document.getElementById('xzDeviceID');
    const cid = document.getElementById('xzClientID');
    const tok = document.getElementById('xzToken');
    const en = document.getElementById('xzEnabled');
    const auto = document.getElementById('xzAutoApplyOTA');
    const ttsSel = document.getElementById('xzTTSMode');
    const convSel = document.getElementById('xzConvMode');
    const idle = document.getElementById('xzIdleTimeout');

    if (ota) ota.value = cfg.ota_base_url || '';
    if (ep) ep.value = cfg.endpoint || '';
    if (did) did.value = cfg.device_id || '';
    if (cid) cid.value = cfg.client_id || '';
    if (tok) tok.value = cfg.token || '';
    if (en) en.checked = !!cfg.enabled;
    if (auto) auto.checked = !!cfg.auto_apply_ota_websocket;
    if (ttsSel) ttsSel.value = cfg.tts_mode || 'xiaozhi';
    if (convSel) convSel.value = cfg.conversation_mode || 'continuous';
    if (idle) idle.value = cfg.idle_timeout_sec || 20;
}

async function xzLoad() {
    try {
        const resp = await fetch('/api/mods/Xiaozhi/get');
        if (!resp.ok) { xzSetStatus('Không tải được cấu hình (Failed to load config)', true); return; }
        const cfg = await resp.json();
        xzApplyCfg(cfg);
    } catch (e) {
        xzSetStatus('Lỗi tải: ' + e.message + ' (Error loading)', true);
    }
}

async function xzSave() {
    const params = new URLSearchParams({
        ota_base_url: document.getElementById('xzOTABaseURL').value.trim(),
        endpoint: document.getElementById('xzEndpoint').value.trim(),
        device_id: document.getElementById('xzDeviceID').value.trim(),
        client_id: document.getElementById('xzClientID').value.trim(),
        token: document.getElementById('xzToken').value.trim(),
        enabled: document.getElementById('xzEnabled').checked ? 'true' : 'false',
        auto_apply_ota_websocket: document.getElementById('xzAutoApplyOTA').checked ? 'true' : 'false',
        tts_mode: document.getElementById('xzTTSMode').value,
        conversation_mode: document.getElementById('xzConvMode').value,
        idle_timeout_sec: document.getElementById('xzIdleTimeout').value,
    });
    try {
        const resp = await fetch('/api/mods/Xiaozhi/save?' + params.toString(), { method: 'POST' });
        const j = await resp.json();
        if (j.status === 'success') {
            if (j.config) xzApplyCfg(j.config);
            else await xzLoad();
            xzSetStatus('Đã lưu. (Saved.)', false);
        } else {
            xzSetStatus('Lỗi lưu: ' + (j.message || 'unknown') + ' (Save error)', true);
        }
    } catch (e) {
        xzSetStatus('Lỗi lưu: ' + e.message + ' (Save error)', true);
    }
}

async function xzGenerateCode() {
    xzSetStatus('Đang gọi máy chủ... (Contacting server...)', false);
    document.getElementById('xzCodeBox').style.display = 'none';
    try {
        const resp = await fetch('/api/mods/Xiaozhi/generate_code', { method: 'POST' });
        if (!resp.ok) {
            const t = await resp.text();
            xzSetStatus('Lỗi: ' + t + ' (Error)', true);
            return;
        }
        const j = await resp.json();
        if (j.status === 'error') {
            xzSetStatus('Lỗi: ' + (j.message || 'unknown') + ' (Error)', true);
            return;
        }
        if (j.device_id) document.getElementById('xzDeviceID').value = j.device_id;
        if (j.client_id) document.getElementById('xzClientID').value = j.client_id;
        if (j.code) {
            document.getElementById('xzCode').textContent = j.code;
            document.getElementById('xzCodeBox').style.display = 'block';
            xzSetStatus('Đã nhận mã — nhập tại xiaozhi.me. (Enter code at xiaozhi.me.)', false);
        } else {
            xzSetStatus('Không có mã (có thể đã ghép). (No code; maybe already paired.)', false);
        }
        await xzLoad();
    } catch (e) {
        xzSetStatus('Lỗi: ' + e.message + ' (Error)', true);
    }
}

async function xzRefresh() {
    xzSetStatus('Đang làm mới token... (Refreshing...)', false);
    try {
        const resp = await fetch('/api/mods/Xiaozhi/refresh', { method: 'POST' });
        const j = await resp.json();
        if (j.status === 'success') {
            xzSetStatus('Đã làm mới token. (Token refreshed.)', false);
            await xzLoad();
        } else {
            xzSetStatus('Lỗi làm mới: ' + (j.message || 'unknown') + ' (Refresh error)', true);
        }
    } catch (e) {
        xzSetStatus('Lỗi làm mới: ' + e.message + ' (Refresh error)', true);
    }
}

async function xzUnpair() {
    if (!confirm('Xóa token và hủy ghép Xiaozhi? (Clear token and unpair?)')) return;
    try {
        const resp = await fetch('/api/mods/Xiaozhi/unpair', { method: 'POST' });
        const j = await resp.json();
        if (j.status === 'success') {
            xzSetStatus('Đã hủy ghép. (Unpaired.)', false);
            await xzLoad();
        } else {
            xzSetStatus('Lỗi hủy ghép: ' + (j.message || 'unknown') + ' (Unpair error)', true);
        }
    } catch (e) {
        xzSetStatus('Lỗi hủy ghép: ' + e.message + ' (Unpair error)', true);
    }
}

document.addEventListener('DOMContentLoaded', function () {
    // Always load so F5 / tab switch shows disk values.
    xzLoad();
    document.querySelectorAll('.tabs button[data-target="#xiaozhi"]').forEach(function (btn) {
        btn.addEventListener('click', function () { xzLoad(); });
    });
});
