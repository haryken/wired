let stimRunning = false;
let stimTimer = null;
let stimData = [];
let stimHolding = false;
let stimHeldValue = 0.5;
let stimSliderBusy = false;

function stimT(key, fallback) {
    return (typeof t === 'function') ? t(key, fallback) : fallback;
}

function stimSetStatus(msg, isError) {
    const el = document.getElementById('stimStatus');
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

function stimSetSliderUI(value01) {
    const v = Math.max(0, Math.min(1, Number(value01) || 0));
    const slider = document.getElementById('stimSlider');
    const label = document.getElementById('stimSliderLabel');
    if (slider) slider.value = String(Math.round(v * 100));
    if (label) label.textContent = v.toFixed(2);
}

function stimSetModeLabel() {
    const el = document.getElementById('stimModeLabel');
    if (!el) return;
    if (stimHolding) {
        el.textContent = `${stimT('stim.mode_hold', 'Hold')} ${stimHeldValue.toFixed(2)}`;
    } else {
        el.textContent = stimT('stim.mode_auto', 'Auto');
    }
}

async function stimLoadHold() {
    try {
        const res = await fetch('/api/mods/StimStats/get_hold');
        if (!res.ok) return;
        const j = await res.json();
        stimHolding = !!j.hold;
        stimHeldValue = Number(j.value) || 0;
        if (stimHolding) stimSetSliderUI(stimHeldValue);
        stimSetModeLabel();
    } catch (_) { /* ignore */ }
}

async function stimApplyHold(hold, value) {
    const params = new URLSearchParams();
    params.set('hold', hold ? 'true' : 'false');
    if (hold) params.set('value', String(value));
    const res = await fetch('/api/mods/StimStats/set_stim?' + params.toString(), { method: 'POST' });
    const j = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(j.message || `http ${res.status}`);
    stimHolding = hold;
    if (hold) stimHeldValue = Number(value) || 0;
    stimSetModeLabel();
}

function stimSliderInput(raw) {
    const v = (Number(raw) || 0) / 100;
    stimSetSliderUI(v);
}

async function stimSliderCommit(raw) {
    if (stimSliderBusy) return;
    stimSliderBusy = true;
    const v = Math.max(0, Math.min(1, (Number(raw) || 0) / 100));
    stimSetSliderUI(v);
    try {
        await stimApplyHold(true, v);
        stimSetStatus(`${stimT('stim.mode_hold', 'Hold')} ${v.toFixed(2)}`, false);
    } catch (e) {
        stimSetStatus(e.message, true);
    } finally {
        stimSliderBusy = false;
    }
}

async function stimPreset(kind) {
    try {
        if (kind === 'auto') {
            await stimApplyHold(false, 0);
            stimSetStatus(stimT('stim.mode_auto', 'Auto'), false);
            return;
        }
        const v = Math.max(0, Math.min(1, Number(kind)));
        stimSetSliderUI(v);
        await stimApplyHold(true, v);
        stimSetStatus(`${stimT('stim.mode_hold', 'Hold')} ${v.toFixed(2)}`, false);
    } catch (e) {
        stimSetStatus(e.message, true);
    }
}

function stimDraw() {
    const canvas = document.getElementById('stimCanvas');
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const w = canvas.width;
    const h = canvas.height;
    ctx.clearRect(0, 0, w, h);
    ctx.fillStyle = '#1a1a1a';
    ctx.fillRect(0, 0, w, h);
    ctx.strokeStyle = '#444';
    ctx.beginPath();
    ctx.moveTo(8, h - 8);
    ctx.lineTo(w - 8, h - 8);
    ctx.moveTo(8, 8);
    ctx.lineTo(8, h - 8);
    ctx.stroke();

    if (stimData.length < 2) return;
    const pad = 12;
    const plotW = w - pad * 2;
    const plotH = h - pad * 2;
    ctx.strokeStyle = 'rgba(51, 237, 109, 1)';
    ctx.lineWidth = 2;
    ctx.beginPath();
    stimData.forEach((v, i) => {
        const x = pad + (i / Math.max(stimData.length - 1, 1)) * plotW;
        const y = pad + plotH * (1 - Math.max(0, Math.min(1, Number(v) || 0)));
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
    });
    ctx.stroke();
}

async function stimPollOnce() {
    try {
        const res = await fetch('/api/mods/StimStats/get_stim');
        if (!res.ok) return;
        const data = await res.json();
        stimData.push(Number(data) || 0);
        if (stimData.length > 24) stimData.shift();
        const label = document.getElementById('stimValueLabel');
        if (label) label.textContent = (Number(data) || 0).toFixed(3);
        stimDraw();
    } catch (_) { /* ignore poll errors */ }
}

async function stimStart() {
    stimStop();
    stimRunning = true;
    stimData = [];
    stimDraw();
    stimSetStatus('Đang theo dõi stim… (Streaming stimulation…)', false);
    stimLoadHold();
    try {
        const res = await fetch('/api/mods/StimStats/begin_stim', { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            stimSetStatus(`${j.message || res.status}`, true);
            stimRunning = false;
            return;
        }
        stimTimer = setInterval(() => {
            if (!stimRunning) {
                stimStop();
                return;
            }
            stimPollOnce();
        }, 500);
    } catch (e) {
        stimSetStatus(e.message, true);
        stimRunning = false;
    }
}

function stimStop() {
    stimRunning = false;
    if (stimTimer) {
        clearInterval(stimTimer);
        stimTimer = null;
    }
    fetch('/api/mods/StimStats/stop_stim', { method: 'POST' }).catch(() => {});
}

function statsSetStatus(msg, isError) {
    const el = document.getElementById('statsStatus');
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

async function statsRefresh() {
    const box = document.getElementById('statsSection');
    if (!box) return;
    statsSetStatus('', false);
    box.innerHTML = '<p>Đang tải… (Loading…)</p>';
    try {
        const res = await fetch('/api/mods/StimStats/get_stats', { method: 'POST' });
        const txt = await res.text();
        if (!res.ok) throw new Error(txt || `http ${res.status}`);
        const j = JSON.parse(txt);
        const secs = Number(j['Alive.seconds']) || 0;
        const days = Math.round(secs / (24 * 60 * 60));
        const trigger = j['BStat.ReactedToTriggerWord'] ?? '—';
        const utility = j['FeatureType.Utility'] ?? '—';
        const petMs = Number(j['Pet.ms']) || 0;
        const moved = Number(j['Stim.CumlPosDelta']) || 0;
        box.innerHTML = `
            <p>Số ngày sống <small>(Days alive)</small>: <b>${days}</b></p>
            <p>Phản hồi từ đánh thức <small>(Reacted to trigger)</small>: <b>${trigger}</b></p>
            <p>Tiện ích đã dùng <small>(Utility features)</small>: <b>${utility}</b></p>
            <p>Giây được vuốt <small>(Seconds petted)</small>: <b>${Math.round(petMs / 1000)}</b></p>
            <p>Quãng đường (cm) <small>(Distance moved)</small>: <b>${Math.round(moved / 100)}</b></p>
        `;
    } catch (e) {
        box.innerHTML = '';
        statsSetStatus(`Không tải được thống kê: ${e.message}`, true);
    }
}

window.addEventListener('beforeunload', () => {
    stimStop();
});
