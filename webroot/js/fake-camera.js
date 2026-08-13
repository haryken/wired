function fakeCamT(key, fallback) {
    return (typeof t === 'function') ? t(key, fallback) : fallback;
}

function fakeCamSetStatus(msg, isError) {
    const el = document.getElementById('fakeCamStatus');
    if (!el) return;
    if (!msg) {
        el.style.display = 'none';
        el.textContent = '';
        return;
    }
    el.style.display = 'block';
    el.textContent = msg;
    el.className = 'faces-status ' + (isError ? 'error' : 'ok');
}

function fakeCamSetModeLabel(mode) {
    const el = document.getElementById('fakeCamModeLabel');
    if (!el) return;
    if (mode === 'on') {
        el.textContent = fakeCamT('fakecam.mode_on', 'Mode: On (always fake)');
    } else if (mode === 'off') {
        el.textContent = fakeCamT('fakecam.mode_off', 'Mode: Off (never fake)');
    } else {
        el.textContent = fakeCamT('fakecam.mode_auto', 'Mode: Auto (fake only if camera fails)');
    }
}

async function fakeCamLoad() {
    try {
        const res = await fetch('/api/mods/FakeCamera/get_mode');
        if (!res.ok) return;
        const j = await res.json();
        const mode = (j.mode || 'auto').toLowerCase();
        fakeCamSetModeLabel(mode);
        fakeCamSetStatus('', false);
    } catch (_) { /* ignore */ }
}

async function fakeCamSet(mode) {
    const params = new URLSearchParams();
    params.set('mode', mode);
    try {
        const res = await fetch('/api/mods/FakeCamera/set_mode?' + params.toString(), { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(j.message || `http ${res.status}`);
        fakeCamSetModeLabel(mode);
        fakeCamSetStatus(fakeCamT('fakecam.saved', 'Saved. Applies in about 0.5s.'), false);
    } catch (e) {
        fakeCamSetStatus(String(e.message || e), true);
    }
}
