let otaPollTimer = null;
let otaLastPhase = '';
let otaLastPct = -1;
let otaFailStreak = 0;

async function setAutoUpdateStatus(status) {
    const el = document.getElementById('autoUpdateStatus');
    el.innerHTML = `<p>${status}</p>`;
    show('autoUpdateStatus');
}

function otaLogAppend(line) {
    const box = document.getElementById('otaLogBox');
    if (!box) return;
    box.style.display = 'block';
    const t = new Date().toLocaleTimeString();
    box.textContent += `[${t}] ${line}\n`;
    box.scrollTop = box.scrollHeight;
}

function otaShowProgress(showIt) {
    const wrap = document.getElementById('otaProgressWrap');
    if (wrap) wrap.style.display = showIt ? 'block' : 'none';
}

function otaSetProgress(pct, phase, detail, mode) {
    otaShowProgress(true);
    const fill = document.getElementById('otaProgressFill');
    const pctEl = document.getElementById('otaPctLabel');
    const phaseEl = document.getElementById('otaPhaseLabel');
    const detailEl = document.getElementById('otaDetailLine');
    const p = Math.max(0, Math.min(100, pct || 0));
    if (fill) {
        fill.style.width = `${p}%`;
        fill.classList.remove('ok', 'err');
        if (mode === 'ok') fill.classList.add('ok');
        if (mode === 'err') fill.classList.add('err');
    }
    if (pctEl) pctEl.textContent = `${p}%`;
    if (phaseEl) phaseEl.textContent = phase || '—';
    if (detailEl) detailEl.textContent = detail || '';
}

function formatBytes(n) {
    const x = Number(n);
    if (!x || x < 0) return '';
    if (x < 1024) return `${x} B`;
    if (x < 1024 * 1024) return `${(x / 1024).toFixed(1)} KB`;
    return `${(x / (1024 * 1024)).toFixed(1)} MB`;
}

function validateOtaUrlClient(raw) {
    const url = (raw || '').trim();
    if (!url) return { ok: false, msg: 'Nhập URL file .ota trước. (Enter an .ota URL first.)' };
    if (/\s/.test(url)) return { ok: false, msg: 'URL không được có khoảng trắng. (No spaces in URL.)' };
    if (!/^https?:\/\//i.test(url)) {
        return { ok: false, msg: 'URL phải bắt đầu bằng http:// hoặc https://' };
    }
    let u;
    try {
        u = new URL(url);
    } catch (_) {
        return { ok: false, msg: 'URL không hợp lệ. (Invalid URL.)' };
    }
    if (u.protocol !== 'http:' && u.protocol !== 'https:') {
        return { ok: false, msg: 'Chỉ chấp nhận http/https. (http/https only.)' };
    }
    if (!u.hostname) {
        return { ok: false, msg: 'Thiếu host trong URL. (Host required.)' };
    }
    const file = (u.pathname.split('/').pop() || '').toLowerCase();
    if (!file.endsWith('.ota')) {
        return { ok: false, msg: 'Đường dẫn phải kết thúc bằng .ota (path must end with .ota)' };
    }
    return { ok: true, url };
}

function markOtaUrlField(ok) {
    const input = document.getElementById('otaUrl');
    if (!input) return;
    input.style.borderColor = ok ? '#555' : '#f87171';
}

async function checkAutoUpdateStatus() {
    // Background auto-update UI removed — URL update only.
}

async function autoUpdateInhibit() {}
async function autoUpdateAllow() {}

async function startOtaFromURL() {
    const input = document.getElementById('otaUrl');
    const v = validateOtaUrlClient(input.value);
    if (!v.ok) {
        markOtaUrlField(false);
        setAutoUpdateStatus(v.msg);
        return;
    }
    markOtaUrlField(true);

    const ok = await wireosConfirm({
        title: 'Cập nhật OS',
        message: 'Bắt đầu cập nhật OS từ URL này?\nRobot sẽ tắt mắt trong lúc tải/cài, rồi có thể tự khởi động lại.\n\n' + v.url,
        ok: 'Bắt đầu cập nhật',
        cancel: 'Hủy',
    });
    if (!ok) return;

    const box = document.getElementById('otaLogBox');
    if (box) {
        box.style.display = 'block';
        box.textContent = '';
    }
    otaLastPhase = '';
    otaLastPct = -1;
    otaFailStreak = 0;
    otaSetProgress(0, 'starting', v.url, '');
    setAutoUpdateStatus('Đang kiểm tra URL rồi khởi động... (Checking URL, then starting...)');
    otaLogAppend('Kiểm tra URL... (Probing URL)');
    document.getElementById('otaStartBtn').disabled = true;
    try {
        const qs = new URLSearchParams({ url: v.url });
        const res = await fetch(`/api/mods/AutoUpdate/startFromURL?${qs.toString()}`, { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            markOtaUrlField(false);
            otaSetProgress(0, 'rejected', j.message || String(res.status), 'err');
            setAutoUpdateStatus(`Không cập nhật: ${j.message || res.status} (Rejected)`);
            otaLogAppend('Từ chối: ' + (j.message || res.status));
            document.getElementById('otaStartBtn').disabled = false;
            return;
        }
        setAutoUpdateStatus('Đã bắt đầu — xem thanh tiến trình + log bên dưới. (Started — watch progress + log.)');
        otaLogAppend('Đã start update-engine — URL OK');
        startOtaPoll();
    } catch (e) {
        otaSetProgress(0, 'error', e.message, 'err');
        setAutoUpdateStatus(`Lỗi mạng: ${e.message} (Network error)`);
        otaLogAppend('Lỗi mạng: ' + e.message);
        document.getElementById('otaStartBtn').disabled = false;
    }
}

function startOtaPoll() {
    if (otaPollTimer) clearInterval(otaPollTimer);
    otaPollTimer = setInterval(pollOtaStatus, 1500);
    pollOtaStatus();
}

function stopOtaPoll() {
    if (otaPollTimer) clearInterval(otaPollTimer);
    otaPollTimer = null;
}

async function pollOtaStatus() {
    try {
        const res = await fetch('/api/mods/AutoUpdate/status');
        const j = await res.json();
        otaFailStreak = 0;

        const phase = j.phase || 'waiting';
        const pct = typeof j.percent === 'number' ? j.percent : 0;
        const ver = j.update_version || '';
        const cur = j.current_version || '';
        const unit = j.unit_active || '';
        const wrote = formatBytes(j.progress);
        const total = formatBytes(j.expected);
        const dl = formatBytes(j.expected_download_size);

        let detail = `Hiện tại: ${cur || '?'}`;
        if (ver) detail += ` → OTA: ${ver}`;
        if (wrote && total) detail += ` | ghi: ${wrote}/${total}`;
        else if (dl) detail += ` | tải ~${dl}`;
        if (unit) detail += ` | service: ${unit}`;

        let mode = '';
        if (j.error) mode = 'err';
        else if (j.done) mode = 'ok';
        otaSetProgress(j.done ? 100 : pct, phase, detail, mode);

        if (phase !== otaLastPhase || pct !== otaLastPct) {
            otaLogAppend(`phase=${phase} ${pct}%` + (ver ? ` ver=${ver}` : '') + (unit ? ` unit=${unit}` : ''));
            otaLastPhase = phase;
            otaLastPct = pct;
        }

        if (j.journal) {
            const box = document.getElementById('otaLogBox');
            // Keep our timeline + journal tail separated
            const marker = '--- journalctl update-engine ---\n';
            const base = (box.textContent || '').split(marker)[0];
            box.textContent = base + marker + j.journal;
            box.scrollTop = box.scrollHeight;
        }

        if (j.error) {
            setAutoUpdateStatus(`<span style="color:#f87171">Thất bại: ${j.error}</span> (Failed)`);
            otaLogAppend('ERROR: ' + j.error);
            stopOtaPoll();
            document.getElementById('otaStartBtn').disabled = false;
            await refreshOtaLogOnly();
            return;
        }

        if (j.done) {
            setAutoUpdateStatus('<span style="color:#4ade80">Thành công — đang reboot...</span> (Success — rebooting)');
            otaLogAppend('DONE — rebooting robot');
            stopOtaPoll();
            await fetch('/api/mods/AutoUpdate/reboot', { method: 'POST' }).catch(() => {});
            return;
        }

        setAutoUpdateStatus(`Đang cập nhật: ${pct}% (${phase})`);
    } catch (e) {
        otaFailStreak += 1;
        otaLogAppend('Mất kết nối tạm thời (' + otaFailStreak + ') — có thể đang reboot');
        setAutoUpdateStatus('Mất kết nối (robot có thể đang reboot). Giữ trang; thử F5 sau 1–2 phút. (Lost connection — maybe rebooting.)');
        if (otaFailStreak >= 8) {
            stopOtaPoll();
            document.getElementById('otaStartBtn').disabled = false;
            otaSetProgress(100, 'offline', 'Không còn phản hồi từ robot', 'ok');
        }
    }
}

async function refreshOtaLogOnly() {
    try {
        const res = await fetch('/api/mods/AutoUpdate/log');
        const txt = await res.text();
        const box = document.getElementById('otaLogBox');
        if (!box) return;
        box.style.display = 'block';
        otaLogAppend('--- file ota.log ---');
        box.textContent += txt + '\n';
        box.scrollTop = box.scrollHeight;
    } catch (e) {
        otaLogAppend('Không đọc được ota.log: ' + e.message);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const input = document.getElementById('otaUrl');
    if (!input) return;
    input.addEventListener('input', () => {
        const v = validateOtaUrlClient(input.value);
        if (!(input.value || '').trim()) {
            markOtaUrlField(true);
            return;
        }
        markOtaUrlField(v.ok);
    });
    // Always show progress + log chrome so the Update tab looks complete before start.
    otaShowProgress(true);
    const box = document.getElementById('otaLogBox');
    if (box) box.style.display = 'block';
});
