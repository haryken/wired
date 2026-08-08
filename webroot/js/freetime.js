let ftRunning = false;
let ftPollTimer = null;
let ftLastSnap = null;

function ftSetStatus(msg, isError) {
    const el = document.getElementById('ftStatus');
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

function ftCamStart() {
    const box = document.getElementById('ftCamBox');
    const img = document.getElementById('ftCamImg');
    const ph = document.getElementById('ftCamPlaceholder');
    if (!box || !img) return;
    if (ph) ph.hidden = true;
    img.hidden = false;
    img.src = '/api/mods/Control/cam-stream?' + Date.now();
}

function ftCamStop() {
    const img = document.getElementById('ftCamImg');
    const ph = document.getElementById('ftCamPlaceholder');
    if (img) {
        img.removeAttribute('src');
        img.hidden = true;
    }
    if (ph) ph.hidden = false;
    fetch('/api/mods/Control/stop_cam', { method: 'POST' }).catch(() => {});
}

function ftDrawOverlay(snap) {
    const canvas = document.getElementById('ftOverlay');
    const img = document.getElementById('ftCamImg');
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const box = document.getElementById('ftCamBox');
    const w = (box && box.clientWidth) || 640;
    const h = Math.round(w * ((snap && snap.imgH) || 360) / ((snap && snap.imgW) || 640));
    if (canvas.width !== w || canvas.height !== h) {
        canvas.width = w;
        canvas.height = h;
    }
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    if (!snap) return;

    const iw = snap.imgW || 640;
    const ih = snap.imgH || 360;
    const sx = canvas.width / iw;
    const sy = canvas.height / ih;
    const showFaces = document.getElementById('ftShowFaces')?.checked !== false;
    const showObjs = document.getElementById('ftShowObjs')?.checked !== false;
    const showMotion = document.getElementById('ftShowMotion')?.checked !== false;

    if (showObjs && Array.isArray(snap.objects)) {
        snap.objects.forEach((o) => {
            const x = o.rect.x * sx;
            const y = o.rect.y * sy;
            const rw = o.rect.w * sx;
            const rh = o.rect.h * sy;
            ctx.strokeStyle = '#38bdf8';
            ctx.lineWidth = 2;
            ctx.strokeRect(x, y, rw, rh);
            const label = (o.type || 'OBJ') + ' #' + o.id;
            ctx.font = '12px sans-serif';
            ctx.fillStyle = 'rgba(0,0,0,0.55)';
            const tw = ctx.measureText(label).width + 8;
            ctx.fillRect(x, Math.max(0, y - 16), tw, 16);
            ctx.fillStyle = '#7dd3fc';
            ctx.fillText(label, x + 4, Math.max(12, y - 4));
        });
    }

    if (showFaces && Array.isArray(snap.faces)) {
        snap.faces.forEach((f) => {
            const x = f.rect.x * sx;
            const y = f.rect.y * sy;
            const rw = f.rect.w * sx;
            const rh = f.rect.h * sy;
            ctx.strokeStyle = '#4ade80';
            ctx.lineWidth = 2;
            ctx.strokeRect(x, y, rw, rh);
            const parts = [];
            if (f.name) parts.push(f.name);
            if (f.expression) parts.push(f.expression);
            parts.push('face ' + f.id);
            const label = parts.join(' · ');
            ctx.font = '12px sans-serif';
            ctx.fillStyle = 'rgba(0,0,0,0.55)';
            const tw = ctx.measureText(label).width + 8;
            ctx.fillRect(x, Math.max(0, y - 16), tw, 16);
            ctx.fillStyle = '#bbf7d0';
            ctx.fillText(label, x + 4, Math.max(12, y - 4));
        });
    }

    if (showMotion && snap.motion) {
        const mx = snap.motion.x * sx;
        const my = snap.motion.y * sy;
        const area = Math.max(0, Number(snap.motion.area) || 0);
        const r = Math.max(10, Math.min(80, Math.sqrt(area) * Math.min(canvas.width, canvas.height) * 0.35));
        ctx.beginPath();
        ctx.arc(mx, my, r, 0, Math.PI * 2);
        ctx.strokeStyle = '#fbbf24';
        ctx.lineWidth = 2;
        ctx.stroke();
        ctx.beginPath();
        ctx.arc(mx, my, 3, 0, Math.PI * 2);
        ctx.fillStyle = '#fbbf24';
        ctx.fill();
        ctx.font = '12px sans-serif';
        ctx.fillStyle = 'rgba(0,0,0,0.55)';
        const label = 'motion';
        const tw = ctx.measureText(label).width + 8;
        ctx.fillRect(mx + 6, Math.max(0, my - 16), tw, 16);
        ctx.fillStyle = '#fde68a';
        ctx.fillText(label, mx + 10, Math.max(12, my - 4));
    }

    // Keep overlay aligned over the img if natural size differs later.
    void img;
}

function ftUpdateHud(snap) {
    const stim = document.getElementById('ftStimVal');
    const fc = document.getElementById('ftFaceCount');
    const oc = document.getElementById('ftObjCount');
    const mv = document.getElementById('ftMotionVal');
    const list = document.getElementById('ftFaceList');
    if (stim) stim.textContent = snap ? (Number(snap.stim) || 0).toFixed(3) : '—';
    if (fc) fc.textContent = snap && snap.faces ? snap.faces.length : 0;
    if (oc) oc.textContent = snap && snap.objects ? snap.objects.length : 0;
    if (mv) {
        if (snap && snap.motion) {
            mv.textContent = `${Math.round(snap.motion.x)},${Math.round(snap.motion.y)}`;
        } else {
            mv.textContent = '—';
        }
    }
    if (list) {
        if (!snap || !snap.faces || !snap.faces.length) {
            list.textContent = '';
        } else {
            list.innerHTML = snap.faces.map((f) => {
                const name = f.name || ('#' + f.id);
                const expr = f.expression ? ` — ${f.expression}` : '';
                return `<div>${name}${expr}</div>`;
            }).join('');
        }
    }
}

async function ftPollOnce() {
    try {
        const res = await fetch('/api/mods/FreeTime/snapshot');
        if (!res.ok) return;
        const snap = await res.json();
        ftLastSnap = snap;
        ftUpdateHud(snap);
        ftDrawOverlay(snap);
    } catch (_) { /* ignore */ }
}

async function ftOnShow() {
    if (ftRunning) return;
    ftRunning = true;
    ftSetStatus('Đang bật vision… (Starting vision…)', false);
    try {
        const res = await fetch('/api/mods/FreeTime/start', { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            ftSetStatus(`${j.message || res.status}`, true);
            ftRunning = false;
            return;
        }
        ftCamStart();
        ftSetStatus('Đang xem freeplay. (Watching freeplay.)', false);
        ftPollTimer = setInterval(() => {
            if (!ftRunning) {
                ftOnHide();
                return;
            }
            ftPollOnce();
        }, 120);
        ftPollOnce();
    } catch (e) {
        ftSetStatus(e.message || String(e), true);
        ftRunning = false;
    }
}

async function ftOnHide() {
    ftRunning = false;
    if (ftPollTimer) {
        clearInterval(ftPollTimer);
        ftPollTimer = null;
    }
    ftCamStop();
    try {
        await fetch('/api/mods/FreeTime/stop', { method: 'POST' });
    } catch (_) {}
    ftLastSnap = null;
    ftUpdateHud(null);
    const canvas = document.getElementById('ftOverlay');
    if (canvas) {
        const ctx = canvas.getContext('2d');
        ctx.clearRect(0, 0, canvas.width, canvas.height);
    }
    ftSetStatus('');
}

document.getElementById('ftShowFaces')?.addEventListener('change', () => ftDrawOverlay(ftLastSnap));
document.getElementById('ftShowObjs')?.addEventListener('change', () => ftDrawOverlay(ftLastSnap));
document.getElementById('ftShowMotion')?.addEventListener('change', () => ftDrawOverlay(ftLastSnap));
window.addEventListener('resize', () => ftDrawOverlay(ftLastSnap));
document.getElementById('ftCamImg')?.addEventListener('load', () => ftDrawOverlay(ftLastSnap));

// Deep-link: main.js may activate this tab before this script loads.
if (document.getElementById('freetime')?.classList.contains('active')) {
    ftOnShow();
}
