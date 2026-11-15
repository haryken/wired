/* After AI self-control finishes: detect "xong hành động" + reopen mic (or old closed-mic). */

function scrT(key, fallback) {
    return (typeof t === 'function') ? t(key, fallback) : fallback;
}

function scrStatus(msg) {
    const el = document.getElementById('selfControlResumeStatus');
    if (!el) return;
    el.style.display = msg ? 'block' : 'none';
    el.textContent = msg || '';
}

function scrShow(cfg) {
    const on = !!cfg.self_control_resume;
    document.querySelectorAll('input[name="selfControlResumeMode"]').forEach((el) => {
        el.checked = el.value === (on ? 'new' : 'old');
    });
    let msg = on
        ? scrT('scr.st_on', 'Đang dùng mới: xong self-control → AI nói “xong hành động” rồi mở micro.')
        : scrT('scr.st_off', 'Đang dùng cũ: xong self-control thì mic đóng đến khi Hey Vector / nút lưng.');
    if (on && !cfg.enabled) {
        msg += ' ' + scrT('scr.st_xz_off', '(AI giao tiếp đang tắt — chưa có tác dụng.)');
    }
    scrStatus(msg);
}

async function loadSelfControlResume() {
    try {
        const r = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (!r.ok) throw new Error('HTTP ' + r.status);
        scrShow(await r.json());
    } catch (e) {
        scrStatus(scrT('scr.err_load', 'Lỗi đọc cài đặt:') + ' ' + (e.message || e));
    }
}

async function setSelfControlResume(on) {
    scrStatus(scrT('scr.saving', 'Đang lưu…'));
    try {
        const r = await fetch('/api/mods/Xiaozhi/set_self_control_resume?' + new URLSearchParams({
            enabled: on ? 'true' : 'false',
        }).toString(), { method: 'POST' });
        const j = await r.json().catch(() => ({}));
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        if (j.config) scrShow(j.config);
        else await loadSelfControlResume();
    } catch (e) {
        scrStatus(scrT('scr.err_save', 'Lỗi lưu cài đặt:') + ' ' + (e.message || e));
        await loadSelfControlResume();
    }
}
