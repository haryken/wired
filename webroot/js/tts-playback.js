/* AI voice playback: classic (gap keepalive) vs smooth (ESP-like prefill). */

function ttsPlayT(key, fallback) {
    return (typeof t === 'function') ? t(key, fallback) : fallback;
}

function ttsPlayStatus(msg) {
    const el = document.getElementById('ttsPlaybackStatus');
    if (!el) return;
    el.style.display = msg ? 'block' : 'none';
    el.textContent = msg || '';
}

function ttsPlayShow(cfg) {
    const mode = (cfg.tts_playback === 'smooth') ? 'smooth' : 'classic';
    document.querySelectorAll('input[name="ttsPlaybackMode"]').forEach((el) => {
        el.checked = el.value === mode;
    });
    let msg = mode === 'smooth'
        ? ttsPlayT('ttp.st_smooth', 'Đang dùng mới: prefill PCM rồi ghi liên tục (ít giật hơn, kiểu ESP32).')
        : ttsPlayT('ttp.st_classic', 'Đang dùng cũ: FIFO + pad silence giữa câu (ổn định như hiện tại).');
    if (cfg.enabled === false) {
        msg += ' ' + ttsPlayT('ttp.st_xz_off', '(AI giao tiếp đang tắt — chưa có tác dụng.)');
    }
    ttsPlayStatus(msg);
}

async function loadTTSPlayback() {
    try {
        const r = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (!r.ok) throw new Error('HTTP ' + r.status);
        ttsPlayShow(await r.json());
    } catch (e) {
        ttsPlayStatus(ttsPlayT('ttp.err_load', 'Lỗi đọc cài đặt:') + ' ' + (e.message || e));
    }
}

async function setTTSPlayback(mode) {
    ttsPlayStatus(ttsPlayT('ttp.saving', 'Đang lưu…'));
    try {
        const r = await fetch('/api/mods/Xiaozhi/set_tts_playback?' + new URLSearchParams({
            mode: mode,
        }).toString(), { method: 'POST' });
        const j = await r.json().catch(() => ({}));
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        if (j.config) ttsPlayShow(j.config);
        else await loadTTSPlayback();
    } catch (e) {
        ttsPlayStatus(ttsPlayT('ttp.err_save', 'Lỗi lưu cài đặt:') + ' ' + (e.message || e));
        await loadTTSPlayback();
    }
}
