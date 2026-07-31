// Xiaozhi UI helpers

let xzModeBusy = false;
let xzAppliedMode = null; // 'xiaozhi' | 'vosk' — last mode known from disk / set_enabled

function xzSetStatus(msg, isError) {
    const el = document.getElementById('xiaozhiStatus');
    if (!el) return;
    el.textContent = msg;
    el.style.color = isError ? '#f87171' : '#4ade80';
    el.style.display = msg ? 'block' : 'none';
}

function xzSelectedMode() {
    const vosk = document.getElementById('xzModeVosk');
    return (vosk && vosk.checked) ? 'vosk' : 'xiaozhi';
}

function xzShowConfigForMode(mode) {
    const block = document.getElementById('xzConfigBlock');
    if (block) block.style.display = (mode === 'xiaozhi') ? 'block' : 'none';
    const xzRadio = document.getElementById('xzModeXiaozhi');
    const voskRadio = document.getElementById('xzModeVosk');
    if (xzRadio) xzRadio.checked = (mode === 'xiaozhi');
    if (voskRadio) voskRadio.checked = (mode === 'vosk');
}

function xzSelectedConvMode() {
    const single = document.getElementById('xzConvSingle');
    return (single && single.checked) ? 'single' : 'continuous';
}

function xzApplyCfg(cfg) {
    if (!cfg) return;
    const ota = document.getElementById('xzOTABaseURL');
    const ep = document.getElementById('xzEndpoint');
    const did = document.getElementById('xzDeviceID');
    const cid = document.getElementById('xzClientID');
    const auto = document.getElementById('xzAutoApplyOTA');
    const idle = document.getElementById('xzIdleTimeout');
    const convCont = document.getElementById('xzConvContinuous');
    const convSingle = document.getElementById('xzConvSingle');

    if (ota) ota.value = cfg.ota_base_url || '';
    if (ep) ep.value = cfg.endpoint || '';
    if (did) did.value = cfg.device_id || '';
    if (cid) cid.value = cfg.client_id || '';
    if (auto) auto.checked = !!cfg.auto_apply_ota_websocket;
    if (idle) idle.value = cfg.idle_timeout_sec || 20;
    const conv = (cfg.conversation_mode === 'single') ? 'single' : 'continuous';
    if (convCont) convCont.checked = (conv === 'continuous');
    if (convSingle) convSingle.checked = (conv === 'single');

    const mode = cfg.enabled ? 'xiaozhi' : 'vosk';
    xzAppliedMode = mode;
    xzShowConfigForMode(mode);
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

/** Save Xiaozhi config fields only (not listen mode). */
async function xzSave() {
    const params = new URLSearchParams({
        ota_base_url: document.getElementById('xzOTABaseURL').value.trim(),
        endpoint: document.getElementById('xzEndpoint').value.trim(),
        device_id: document.getElementById('xzDeviceID').value.trim(),
        client_id: document.getElementById('xzClientID').value.trim(),
        // Keep current mode; mode changes go through set_enabled.
        enabled: (xzAppliedMode === 'vosk') ? 'false' : 'true',
        auto_apply_ota_websocket: document.getElementById('xzAutoApplyOTA').checked ? 'true' : 'false',
        // Xiaozhi listen mode always uses Xiaozhi TTS (acapela here is invalid).
        tts_mode: 'xiaozhi',
        conversation_mode: xzSelectedConvMode(),
        idle_timeout_sec: document.getElementById('xzIdleTimeout').value,
    });
    try {
        const resp = await fetch('/api/mods/Xiaozhi/save?' + params.toString(), { method: 'POST' });
        const j = await resp.json();
        if (j.status === 'success') {
            if (j.config) xzApplyCfg(j.config);
            else await xzLoad();
            xzSetStatus('Đã lưu cấu hình Xiaozhi. (Config saved.)', false);
        } else {
            xzSetStatus('Lỗi lưu: ' + (j.message || 'unknown') + ' (Save error)', true);
        }
    } catch (e) {
        xzSetStatus('Lỗi lưu: ' + e.message + ' (Save error)', true);
    }
}

async function xzSetListenMode(mode) {
    if (xzModeBusy) return;
    const wantXz = (mode === 'xiaozhi');
    if (xzAppliedMode === mode) {
        xzShowConfigForMode(mode);
        return;
    }
    xzModeBusy = true;
    xzShowConfigForMode(mode);
    xzSetStatus(wantXz
        ? 'Đang bật Xiaozhi (tắt Vosk)… đang restart cloud…'
        : 'Đang bật Vosk (tắt Xiaozhi)… đang restart cloud…', false);
    try {
        const params = new URLSearchParams({ enabled: wantXz ? 'true' : 'false' });
        const resp = await fetch('/api/mods/Xiaozhi/set_enabled?' + params.toString(), { method: 'POST' });
        const j = await resp.json();
        if (j.status !== 'success') {
            xzSetStatus('Lỗi đổi chế độ: ' + (j.message || 'unknown'), true);
            await xzLoad();
            return;
        }
        if (j.config) xzApplyCfg(j.config);
        else {
            xzAppliedMode = mode;
            xzShowConfigForMode(mode);
        }
        xzSetStatus(wantXz
            ? 'Đã chuyển sang Xiaozhi. Đợi ~5–10s rồi Hey Vector. (Xiaozhi on.)'
            : 'Đã chuyển sang Vosk. Đợi ~5–10s rồi Hey Vector. (Vosk on.)', false);
    } catch (e) {
        xzSetStatus('Lỗi đổi chế độ: ' + e.message, true);
        await xzLoad();
    } finally {
        xzModeBusy = false;
    }
}

function xzOnModeRadio() {
    xzSetListenMode(xzSelectedMode());
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

document.addEventListener('DOMContentLoaded', function () {
    xzLoad();
    document.querySelectorAll('.tabs button[data-target="#xiaozhi"]').forEach(function (btn) {
        btn.addEventListener('click', function () { xzLoad(); });
    });
    const sel = document.getElementById('navSelect');
    if (sel) {
        sel.addEventListener('change', function () {
            if (sel.value === '#xiaozhi') xzLoad();
        });
    }
});
