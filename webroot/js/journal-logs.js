// Live journalctl viewer for :8080 (JournalLogs mod)

const LOGS_MAX_LINES = 4000;
const LOGS_DEFAULT_UNIT = 'vic-cloud';
const LOGS_TAIL_N = 200;

let logsUnits = [];
let logsCurrentUnit = LOGS_DEFAULT_UNIT;
let logsEs = null;
let logsPaused = false;
let logsLines = [];
let logsAutoScroll = true;
let logsVisible = false;

function logsSetStatus(msg, isError) {
    const el = document.getElementById('logsStatus');
    if (!el) return;
    el.textContent = msg || '';
    el.style.color = isError ? '#f87171' : '#94a3b8';
    el.style.display = msg ? 'block' : 'none';
}

function logsUpdateMeta(active) {
    const el = document.getElementById('logsMeta');
    if (!el) return;
    const st = active || '';
    const follow = logsPaused ? 'paused' : 'following';
    el.textContent = logsCurrentUnit
        ? `${logsCurrentUnit} · ${st || '?'} · ${follow}`
        : '';
}

function logsRenderBox() {
    const box = document.getElementById('logsBox');
    if (!box) return;
    box.textContent = logsLines.length ? logsLines.join('\n') : '(trống)';
    if (logsAutoScroll) box.scrollTop = box.scrollHeight;
}

function logsAppendLine(line) {
    if (logsPaused) return;
    logsLines.push(line);
    if (logsLines.length > LOGS_MAX_LINES) {
        logsLines = logsLines.slice(logsLines.length - LOGS_MAX_LINES);
    }
    logsRenderBox();
}

function logsStopFollow() {
    if (logsEs) {
        try { logsEs.close(); } catch (_) {}
        logsEs = null;
    }
}

function logsStartFollow(unit) {
    logsStopFollow();
    if (!unit || !logsVisible) return;
    logsPaused = false;
    const pauseBtn = document.getElementById('logsPauseBtn');
    if (pauseBtn) pauseBtn.textContent = 'Tạm dừng (Pause)';

    const url = `/api/mods/JournalLogs/follow?unit=${encodeURIComponent(unit)}&n=${LOGS_TAIL_N}`;
    logsSetStatus('Đang kết nối journalctl -f…');
    try {
        logsEs = new EventSource(url);
    } catch (e) {
        logsSetStatus('Không mở được EventSource: ' + e.message, true);
        return;
    }

    logsEs.addEventListener('log', (ev) => {
        logsAppendLine(ev.data);
        logsSetStatus('');
    });
    logsEs.addEventListener('logerr', (ev) => {
        if (ev.data) logsAppendLine('[error] ' + ev.data);
        logsSetStatus(ev.data || 'Lỗi journalctl', true);
    });
    logsEs.onerror = () => {
        if (logsEs && logsEs.readyState === EventSource.CLOSED) {
            logsSetStatus('Mất kết nối follow — bấm lại service để thử.', true);
            logsStopFollow();
        }
    };
    logsEs.onopen = () => {
        logsSetStatus('Đang theo dõi (journalctl -f)…');
        logsUpdateMeta(logsUnitActive(unit));
    };
    logsEs.onmessage = (ev) => {
        if (ev.data) logsAppendLine(ev.data);
    };
}

function logsUnitActive(unit) {
    const u = logsUnits.find((x) => x.unit === unit);
    return u ? u.active : '';
}

function logsSelectUnit(unit) {
    if (!unit) return;
    logsCurrentUnit = unit;
    logsLines = [];
    logsRenderBox();
    document.querySelectorAll('.logs-unit-btn').forEach((b) => {
        b.classList.toggle('active', b.dataset.unit === unit);
    });
    logsUpdateMeta(logsUnitActive(unit));
    logsStartFollow(unit);
}

function logsRenderUnitTabs() {
    const wrap = document.getElementById('logsUnitTabs');
    if (!wrap) return;
    wrap.innerHTML = '';
    logsUnits.forEach((u) => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'logs-unit-btn' + (u.unit === logsCurrentUnit ? ' active' : '');
        btn.dataset.unit = u.unit;
        const badge = (u.active === 'active') ? '●' : '○';
        btn.innerHTML = `${badge} ${u.label}<br><small>${u.active || '?'}</small>`;
        btn.title = u.unit + ' — ' + (u.active || 'unknown');
        btn.onclick = () => logsSelectUnit(u.unit);
        wrap.appendChild(btn);
    });
}

async function logsLoadUnits() {
    try {
        const res = await fetch('/api/mods/JournalLogs/units');
        if (!res.ok) throw new Error('HTTP ' + res.status);
        logsUnits = await res.json();
        if (!Array.isArray(logsUnits) || !logsUnits.length) {
            logsSetStatus('Không có unit nào trong allowlist.', true);
            return;
        }
        if (!logsUnits.some((u) => u.unit === logsCurrentUnit)) {
            logsCurrentUnit = logsUnits[0].unit;
        }
        logsRenderUnitTabs();
        logsUpdateMeta(logsUnitActive(logsCurrentUnit));
    } catch (e) {
        logsSetStatus('Không tải danh sách service: ' + e.message, true);
    }
}

function logsTogglePause() {
    logsPaused = !logsPaused;
    const pauseBtn = document.getElementById('logsPauseBtn');
    if (logsPaused) {
        if (pauseBtn) pauseBtn.textContent = 'Tiếp tục (Resume)';
        logsStopFollow();
        logsSetStatus('Đã tạm dừng — buffer giữ nguyên.');
    } else {
        if (pauseBtn) pauseBtn.textContent = 'Tạm dừng (Pause)';
        logsStartFollow(logsCurrentUnit);
    }
    logsUpdateMeta(logsUnitActive(logsCurrentUnit));
}

function logsClear() {
    logsLines = [];
    logsRenderBox();
}

async function logsCopy() {
    const text = logsLines.length ? logsLines.join('\n') : '';
    if (!text) {
        logsSetStatus('Không có gì để copy.', true);
        return;
    }
    try {
        if (navigator.clipboard && navigator.clipboard.writeText) {
            await navigator.clipboard.writeText(text);
        } else {
            const ta = document.createElement('textarea');
            ta.value = text;
            document.body.appendChild(ta);
            ta.select();
            document.execCommand('copy');
            document.body.removeChild(ta);
        }
        logsSetStatus('Đã copy ' + logsLines.length + ' dòng vào clipboard.');
    } catch (e) {
        logsSetStatus('Copy thất bại: ' + e.message, true);
    }
}

function logsOnShow() {
    logsVisible = true;
    logsLoadUnits().then(() => {
        if (logsVisible) logsStartFollow(logsCurrentUnit);
    });
}

function logsOnHide() {
    logsVisible = false;
    logsStopFollow();
    logsSetStatus('');
}

document.addEventListener('DOMContentLoaded', () => {
    const box = document.getElementById('logsBox');
    if (box) {
        box.addEventListener('scroll', () => {
            const nearBottom = box.scrollHeight - box.scrollTop - box.clientHeight < 40;
            logsAutoScroll = nearBottom;
        });
    }
});
