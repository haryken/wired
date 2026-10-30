/* Nudge when the user stays quiet too long (xiaozhi.json silence_prompt_sec). */

function silencePromptT(key, fallback) {
    return (typeof t === 'function') ? t(key, fallback) : fallback;
}

function silencePromptStatus(msg, isErr) {
    const el = document.getElementById('silencePromptStatus');
    if (!el) return;
    el.style.display = msg ? 'block' : 'none';
    el.textContent = msg || '';
    el.classList.toggle('error', !!isErr && !!msg);
}

const SILENCE_PROMPT_CHOICES = [10, 15, 20, 25, 30, 35, 40, 45, 50];

function silencePromptSec(cfg) {
    const n = parseInt(cfg.silence_prompt_sec, 10);
    return n > 0 ? n : 0;
}

function silencePromptFillChips(selected) {
    const box = document.getElementById('silencePromptChips');
    if (!box) return;
    const choices = SILENCE_PROMPT_CHOICES.includes(selected) || selected === 0
        ? SILENCE_PROMPT_CHOICES
        : SILENCE_PROMPT_CHOICES.concat([selected]).filter((n) => n > 0).sort((a, b) => a - b);
    box.innerHTML = '';
    choices.forEach((n) => {
        const chip = document.createElement('button');
        chip.type = 'button';
        chip.className = 'face-greet-chip' + (n === selected ? ' selected' : '');
        chip.setAttribute('role', 'radio');
        chip.setAttribute('aria-checked', n === selected ? 'true' : 'false');
        chip.textContent = silencePromptT('silence.sec', '{n} giây').replace('{n}', n);
        chip.onclick = () => {
            if (n !== selected) setSilencePrompt(n);
        };
        box.appendChild(chip);
    });
}

function silencePromptShow(cfg) {
    const sec = silencePromptSec(cfg);
    const on = sec > 0;
    document.querySelectorAll('input[name="silencePromptMode"]').forEach((el) => {
        el.checked = el.value === (on ? 'on' : 'off');
    });
    const chipsWrap = document.getElementById('silencePromptEvery');
    if (chipsWrap) chipsWrap.style.display = on ? 'block' : 'none';
    silencePromptFillChips(on ? sec : 20);
    let msg;
    if (!on) {
        msg = silencePromptT('silence.st_off', 'Đang tắt — robot không tự gọi khi bạn im lặng.');
    } else {
        msg = silencePromptT('silence.st_on', 'Đang bật: im khoảng {n} giây thì robot gọi bạn nói chuyện tiếp.')
            .replace('{n}', sec);
    }
    if (on && !cfg.enabled) {
        msg += ' ' + silencePromptT('silence.st_xz_off', '(AI giao tiếp đang tắt — chưa có tác dụng.)');
    }
    silencePromptStatus(msg, false);
}

async function loadSilencePrompt() {
    try {
        const r = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (!r.ok) throw new Error('HTTP ' + r.status);
        silencePromptShow(await r.json());
    } catch (e) {
        silencePromptStatus(silencePromptT('silence.err_load', 'Lỗi đọc cài đặt:') + ' ' + (e.message || e), true);
    }
}

async function setSilencePromptMode(on) {
    if (!on) {
        await setSilencePrompt(0);
        return;
    }
    // Turning on without a prior value → default 20s (middle of 10..50).
    let sec = 20;
    try {
        const r = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (r.ok) {
            const cfg = await r.json();
            const cur = silencePromptSec(cfg);
            if (cur > 0) sec = cur;
        }
    } catch (_) {}
    await setSilencePrompt(sec);
}

async function setSilencePrompt(sec) {
    silencePromptStatus(silencePromptT('silence.saving', 'Đang lưu…'), false);
    try {
        const r = await fetch('/api/mods/Xiaozhi/set_silence_prompt?' + new URLSearchParams({
            sec: String(sec),
        }).toString(), { method: 'POST' });
        const j = await r.json().catch(() => ({}));
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        if (j.config) silencePromptShow(j.config);
        else await loadSilencePrompt();
    } catch (e) {
        silencePromptStatus(silencePromptT('silence.err_save', 'Lỗi lưu:') + ' ' + (e.message || e), true);
        await loadSilencePrompt();
    }
}
