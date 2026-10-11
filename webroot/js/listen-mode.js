/* Xiaozhi mic mode: fixed ~6s window vs open until the server detects end of speech. */

function listenModeStatus(msg) {
    const el = document.getElementById('listenModeStatus');
    if (!el) return;
    el.style.display = msg ? 'block' : 'none';
    el.textContent = msg || '';
}

function listenModeShow(cfg) {
    const mode = cfg.listen_mode === 'vad' ? 'vad' : 'fixed';
    document.querySelectorAll('input[name="listenMode"]').forEach((el) => {
        el.checked = el.value === mode;
    });
    let msg = mode === 'vad' ? 'Đang dùng: theo giọng nói.' : 'Đang dùng: cố định ~6 giây.';
    if (!cfg.enabled) msg += ' (Xiaozhi đang tắt — robot dùng Vosk, chế độ này chưa có tác dụng.)';
    listenModeStatus(msg);
}

async function loadListenMode() {
    try {
        const r = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (!r.ok) throw new Error('HTTP ' + r.status);
        listenModeShow(await r.json());
    } catch (e) {
        listenModeStatus('Lỗi đọc chế độ micro: ' + (e.message || e));
    }
}

async function setListenMode(mode) {
    listenModeStatus('Đang lưu…');
    try {
        const r = await fetch('/api/mods/Xiaozhi/set_listen_mode?' + new URLSearchParams({ mode }).toString(), {
            method: 'POST',
        });
        const j = await r.json().catch(() => ({}));
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        listenModeShow(j.config || { listen_mode: mode, enabled: true });
    } catch (e) {
        listenModeStatus('Lỗi lưu chế độ micro: ' + (e.message || e));
        await loadListenMode();
    }
}
