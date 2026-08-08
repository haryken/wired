// Xiaozhi UI helpers

const XZ_DEFAULT_OTA = 'https://api.tenclass.net/';

let xzModeBusy = false;
let xzSaveBusy = false;
let xzAppliedMode = null; // 'xiaozhi' | 'vosk'

function xzSetElStatus(elId, msg, isError) {
    const el = document.getElementById(elId);
    if (!el) return;
    el.textContent = msg || '';
    el.style.color = isError ? '#f87171' : '#4ade80';
    el.style.display = msg ? 'block' : 'none';
}

function xzSetModeStatus(msg, isError) {
    xzSetElStatus('xiaozhiStatus', msg, isError);
}

function xzSetConfigStatus(msg, isError) {
    xzSetElStatus('xzConfigStatus', msg, isError);
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
    const convCont = document.getElementById('xzConvContinuous');
    const convSingle = document.getElementById('xzConvSingle');

    if (ota) ota.value = (cfg.ota_base_url && cfg.ota_base_url.trim()) ? cfg.ota_base_url : XZ_DEFAULT_OTA;
    if (ep) ep.value = cfg.endpoint || '';
    if (did) did.value = cfg.device_id || '';
    if (cid) cid.value = cfg.client_id || '';
    const conv = (cfg.conversation_mode === 'single') ? 'single' : 'continuous';
    if (convCont) convCont.checked = (conv === 'continuous');
    if (convSingle) convSingle.checked = (conv === 'single');

    const gviOn = document.getElementById('xzGameTtsOn');
    const gviOff = document.getElementById('xzGameTtsOff');
    if (gviOn && gviOff) {
        const on = !!cfg.game_google_tts_vi;
        gviOn.checked = on;
        gviOff.checked = !on;
    }

    const mode = cfg.enabled ? 'xiaozhi' : 'vosk';
    xzAppliedMode = mode;
    xzShowConfigForMode(mode);
}

async function xzLoad() {
    try {
        const resp = await fetch('/api/mods/Xiaozhi/get');
        if (!resp.ok) { xzSetModeStatus('Không tải được cấu hình (Failed to load config)', true); return; }
        const cfg = await resp.json();
        xzApplyCfg(cfg);
    } catch (e) {
        xzSetModeStatus('Lỗi tải: ' + e.message + ' (Error loading)', true);
    }
}

/** Persist config fields. Returns {ok, config?, message?}. Does not set UI status. */
async function xzSaveConfig() {
    let otaVal = document.getElementById('xzOTABaseURL').value.trim();
    if (!otaVal) otaVal = XZ_DEFAULT_OTA;
    const params = new URLSearchParams({
        ota_base_url: otaVal,
        device_id: document.getElementById('xzDeviceID').value.trim(),
        client_id: document.getElementById('xzClientID').value.trim(),
        enabled: (xzAppliedMode === 'vosk') ? 'false' : 'true',
        auto_apply_ota_websocket: 'true',
        tts_mode: 'xiaozhi',
        conversation_mode: xzSelectedConvMode(),
        idle_timeout_sec: '20',
        game_google_tts_vi: (document.getElementById('xzGameTtsOn') && document.getElementById('xzGameTtsOn').checked) ? 'true' : 'false',
    });
    const resp = await fetch('/api/mods/Xiaozhi/save?' + params.toString(), { method: 'POST' });
    const j = await resp.json();
    if (j.status === 'success') {
        if (j.config) xzApplyCfg(j.config);
        return { ok: true, config: j.config };
    }
    return { ok: false, message: j.message || 'unknown' };
}

/** One button: save config, then OTA activation check. */
async function xzSaveAndActivate() {
    if (xzSaveBusy) return;
    xzSaveBusy = true;
    const btn = document.getElementById('xzSaveActivateBtn');
    if (btn) btn.disabled = true;
    const codeBox = document.getElementById('xzCodeBox');
    if (codeBox) codeBox.style.display = 'none';
    xzSetConfigStatus('Đang lưu cấu hình…', false);

    try {
        const saved = await xzSaveConfig();
        if (!saved.ok) {
            xzSetConfigStatus('Lỗi lưu: ' + (saved.message || 'unknown'), true);
            return;
        }

        xzSetConfigStatus('Đã lưu. Đang kiểm tra kích hoạt trên máy chủ…', false);
        const resp = await fetch('/api/mods/Xiaozhi/generate_code', { method: 'POST' });
        if (!resp.ok) {
            xzSetConfigStatus('Đã lưu, nhưng lỗi lấy mã: ' + (await resp.text()), true);
            return;
        }
        const j = await resp.json();
        if (j.status === 'error') {
            xzSetConfigStatus('Đã lưu, nhưng lỗi: ' + (j.message || 'unknown'), true);
            return;
        }
        if (j.device_id) document.getElementById('xzDeviceID').value = j.device_id;
        if (j.client_id) document.getElementById('xzClientID').value = j.client_id;

        await xzLoad();

        const did = document.getElementById('xzDeviceID').value || j.device_id || '';
        if (j.code) {
            document.getElementById('xzCode').textContent = j.code;
            if (codeBox) codeBox.style.display = 'block';
            xzSetConfigStatus(
                'Đã lưu. Chưa kích hoạt — nhập mã bên dưới tại xiaozhi.me để liên kết robot với tài khoản Xiaozhi.',
                false
            );
        } else {
            xzSetConfigStatus(
                'Đã lưu. ID này đã kích hoạt / liên kết với tài khoản trên hệ thống Xiaozhi rồi. Đánh thức robot (Hey Vector) và dùng bình thường.',
                false
            );
            if (did) {
                xzSetConfigStatus(
                    'Đã lưu. Device ID ' + did + ' đã liên kết với tài khoản Xiaozhi. Đánh thức robot (Hey Vector) và dùng bình thường.',
                    false
                );
            }
        }
    } catch (e) {
        xzSetConfigStatus('Lỗi: ' + e.message, true);
    } finally {
        xzSaveBusy = false;
        if (btn) btn.disabled = false;
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
    xzSetModeStatus(wantXz
        ? 'Đang bật Xiaozhi (tắt Vosk)… đang restart cloud…'
        : 'Đang bật Vosk (tắt Xiaozhi)… đang restart cloud…', false);
    try {
        const params = new URLSearchParams({ enabled: wantXz ? 'true' : 'false' });
        const resp = await fetch('/api/mods/Xiaozhi/set_enabled?' + params.toString(), { method: 'POST' });
        const j = await resp.json();
        if (j.status !== 'success') {
            xzSetModeStatus('Lỗi đổi chế độ: ' + (j.message || 'unknown'), true);
            await xzLoad();
            return;
        }
        if (j.config) xzApplyCfg(j.config);
        else {
            xzAppliedMode = mode;
            xzShowConfigForMode(mode);
        }
        xzSetModeStatus(wantXz
            ? 'Đã chuyển sang Xiaozhi. Đợi ~5–10s rồi Hey Vector.'
            : 'Đã chuyển sang Vosk. Đợi ~5–10s rồi Hey Vector.', false);
    } catch (e) {
        xzSetModeStatus('Lỗi đổi chế độ: ' + e.message, true);
        await xzLoad();
    } finally {
        xzModeBusy = false;
    }
}

function xzOnModeRadio() {
    xzSetListenMode(xzSelectedMode());
}

async function xzSaveGameGoogleVi() {
    const onEl = document.getElementById('xzGameTtsOn');
    if (!onEl) return;
    const on = !!onEl.checked;
    try {
        const params = new URLSearchParams({ enabled: on ? 'true' : 'false' });
        const resp = await fetch('/api/mods/Xiaozhi/set_game_google_tts_vi?' + params.toString(), { method: 'POST' });
        const j = await resp.json();
        if (j.status !== 'success') {
            xzSetElStatus('xzGameTtsStatus', 'Lỗi lưu: ' + (j.message || 'unknown'), true);
            await xzLoad();
            return;
        }
        if (j.config) xzApplyCfg(j.config);
        xzSetElStatus('xzGameTtsStatus', on
            ? 'Đã bật Google VI — chọn trong tab Game → chế độ bình luận.'
            : 'Đã tắt Google VI.', false);
    } catch (e) {
        xzSetElStatus('xzGameTtsStatus', 'Lỗi: ' + e.message, true);
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
