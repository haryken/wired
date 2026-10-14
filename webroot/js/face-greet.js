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

const FACE_GREET_EVERY_DEFAULT = 5;
const FACE_GREET_EVERY_CHOICES = [0, 1, 2, 3, 5, 10, 15, 30, 60, 120];

// Unset = default 5 min; 0 = no wait.
function faceGreetEveryMin(cfg) {
    const n = parseInt(cfg.face_greet_every_min, 10);
    return n >= 0 ? n : FACE_GREET_EVERY_DEFAULT;
}

function faceGreetFillEvery(selected) {
    const box = document.getElementById('faceGreetEvery');
    if (!box) return;
    const choices = FACE_GREET_EVERY_CHOICES.includes(selected)
        ? FACE_GREET_EVERY_CHOICES
        : FACE_GREET_EVERY_CHOICES.concat([selected]).sort((a, b) => a - b);
    box.innerHTML = '';
    choices.forEach((n) => {
        const chip = document.createElement('button');
        chip.type = 'button';
        chip.className = 'face-greet-chip' + (n === selected ? ' selected' : '');
        chip.setAttribute('role', 'radio');
        chip.setAttribute('aria-checked', n === selected ? 'true' : 'false');
        chip.textContent = n === 0
            ? faceGreetT('facegreet.every_none', 'Không chờ')
            : faceGreetT('facegreet.every_min', '{n} phút').replace('{n}', n);
        if (n === FACE_GREET_EVERY_DEFAULT) {
            const tag = document.createElement('span');
            tag.className = 'face-greet-chip-tag';
            tag.textContent = faceGreetT('facegreet.every_default_tag', 'mặc định');
            chip.appendChild(tag);
        }
        chip.onclick = () => {
            if (n !== selected) setFaceGreetEvery(n);
        };
        box.appendChild(chip);
    });
}

function faceGreetShow(cfg) {
    const on = !!cfg.face_greet;
    const every = faceGreetEveryMin(cfg);
    document.querySelectorAll('input[name="faceGreetMode"]').forEach((el) => {
        el.checked = el.value === (on ? 'on' : 'off');
    });
    faceGreetFillEvery(every);
    let msg;
    if (!on) msg = faceGreetT('facegreet.st_off', 'Đang tắt.');
    else if (every === 0) msg = faceGreetT('facegreet.st_on_now', 'Đang bật: robot chào ngay khi thấy người (không chờ), miễn là đang không trò chuyện.');
    else msg = faceGreetT('facegreet.st_on', 'Đang bật: robot sẽ tự chào khi thấy người (cách nhau ít nhất {n} phút).').replace('{n}', every);
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
    await faceGreetPost({ enabled: on ? 'true' : 'false' });
}

async function setFaceGreetEvery(min) {
    await faceGreetPost({ every_min: String(min) });
}

async function faceGreetPost(params) {
    faceGreetStatus(faceGreetT('facegreet.saving', 'Đang lưu…'));
    try {
        const r = await fetch('/api/mods/Xiaozhi/set_face_greet?' + new URLSearchParams(params).toString(), {
            method: 'POST',
        });
        const j = await r.json().catch(() => ({}));
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        if (j.config) faceGreetShow(j.config);
        else await loadFaceGreet();
    } catch (e) {
        faceGreetStatus(faceGreetT('facegreet.err_save', 'Lỗi lưu cài đặt chào:') + ' ' + (e.message || e));
        await loadFaceGreet();
    }
}
