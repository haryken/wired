/* Greet when Vector sees a person (xiaozhi.json face_greet, read by vic-cloud). */

function faceGreetT(key, fallback) {
    return (typeof t === 'function') ? t(key, fallback) : fallback;
}

function faceGreetStatus(msg) {
    const el = document.getElementById('faceGreetStatus');
    if (!el) return;
    el.style.display = msg ? 'block' : 'none';
    el.textContent = msg || '';
}

function faceGreetShow(cfg) {
    const on = !!cfg.face_greet;
    document.querySelectorAll('input[name="faceGreetMode"]').forEach((el) => {
        el.checked = el.value === (on ? 'on' : 'off');
    });
    let msg = on
        ? faceGreetT('facegreet.st_on', 'Đang bật: robot sẽ tự chào khi thấy người.')
        : faceGreetT('facegreet.st_off', 'Đang tắt.');
    if (on && !cfg.enabled) {
        msg += ' ' + faceGreetT('facegreet.st_xz_off', '(AI giao tiếp đang tắt — chưa có tác dụng.)');
    }
    faceGreetStatus(msg);
}

async function loadFaceGreet() {
    try {
        const r = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (!r.ok) throw new Error('HTTP ' + r.status);
        faceGreetShow(await r.json());
    } catch (e) {
        faceGreetStatus(faceGreetT('facegreet.err_load', 'Lỗi đọc cài đặt chào:') + ' ' + (e.message || e));
    }
}

async function setFaceGreet(on) {
    faceGreetStatus(faceGreetT('facegreet.saving', 'Đang lưu…'));
    try {
        const r = await fetch('/api/mods/Xiaozhi/set_face_greet?' + new URLSearchParams({ enabled: on ? 'true' : 'false' }).toString(), {
            method: 'POST',
        });
        const j = await r.json().catch(() => ({}));
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        faceGreetShow(j.config || { face_greet: on, enabled: true });
    } catch (e) {
        faceGreetStatus(faceGreetT('facegreet.err_save', 'Lỗi lưu cài đặt chào:') + ' ' + (e.message || e));
        await loadFaceGreet();
    }
}
