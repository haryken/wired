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

function otaSetStartDisabled(disabled) {
    ['otaStartBtn', 'otaOfficialBtn', 'otaUploadBtn'].forEach((id) => {
        const b = document.getElementById(id);
        if (b) b.disabled = disabled;
    });
}

function otaConfirm(opts) {
    const modal = document.getElementById('otaConfirmModal');
    const box = modal.querySelector('.wos-modal');
    const okBtn = document.getElementById('otaConfirmOk');
    const cancelBtn = document.getElementById('otaConfirmCancel');
    box.classList.toggle('warn', !!opts.warn);
    document.getElementById('otaConfirmIcon').textContent = opts.icon || '⬇️';
    document.getElementById('otaConfirmTitle').textContent = opts.title || '';
    const sub = document.getElementById('otaConfirmSub');
    sub.textContent = opts.sub || '';
    sub.hidden = !opts.sub;
    const list = document.getElementById('otaConfirmList');
    list.innerHTML = '';
    (opts.items || []).forEach((it) => {
        const li = document.createElement('li');
        if (it.warn) li.className = 'warn';
        const ic = document.createElement('span');
        ic.textContent = it.icon || '•';
        const tx = document.createElement('span');
        tx.textContent = it.text;
        li.append(ic, tx);
        list.appendChild(li);
    });
    okBtn.textContent = opts.okText || 'OK';
    cancelBtn.textContent = otaTr('aud.cf_cancel', 'Huỷ');
    modal.hidden = false;
    okBtn.focus();
    return new Promise((resolve) => {
        const close = (result) => {
            modal.hidden = true;
            okBtn.onclick = cancelBtn.onclick = modal.onclick = null;
            document.removeEventListener('keydown', onKey);
            resolve(result);
        };
        const onKey = (e) => { if (e.key === 'Escape') close(false); };
        okBtn.onclick = () => close(true);
        cancelBtn.onclick = () => close(false);
        modal.onclick = (e) => { if (e.target === modal) close(false); };
        document.addEventListener('keydown', onKey);
    });
}

function otaConfirmItems() {
    return [
        { icon: '⏱️', text: otaTr('aud.cf_time', 'Robot tải và cài trong khoảng 5–10 phút, mắt robot tắt trong lúc cài.') },
        { icon: '🔒', text: otaTr('aud.cf_safe', 'Nếu cập nhật lỗi, robot vẫn giữ hệ điều hành cũ.') },
        { icon: '🔌', text: otaTr('aud.cf_power', 'Không tắt nguồn robot trong lúc cập nhật.'), warn: true },
    ];
}

function otaSetManualMode(mode) {
    document.getElementById('otaManualFile').hidden = mode !== 'file';
    document.getElementById('otaManualUrl').hidden = mode !== 'url';
    document.getElementById('otaModeFile').classList.toggle('active', mode === 'file');
    document.getElementById('otaModeUrl').classList.toggle('active', mode === 'url');
}

function otaFilePicked() {
    const input = document.getElementById('otaFile');
    const file = input.files && input.files[0];
    const label = document.getElementById('otaFileLabel');
    document.getElementById('otaFilePick').classList.toggle('picked', !!file);
    label.textContent = file
        ? otaTr('aud.file_picked', '✅ {name} ({size}) — bấm để đổi file', { name: file.name, size: formatBytes(file.size) })
        : otaTr('aud.file_pick', '📁 Chọn file .ota');
}

function startOtaUpload() {
    const input = document.getElementById('otaFile');
    const file = input.files && input.files[0];
    if (!file) {
        setAutoUpdateStatus(otaTr('aud.up_none', 'Chọn file .ota trước.'));
        return;
    }
    if (!/\.ota$/i.test(file.name)) {
        setAutoUpdateStatus(otaTr('aud.up_bad_name', 'Chọn file có đuôi .ota.'));
        return;
    }
    otaConfirm({
        icon: '📤',
        title: otaTr('aud.cf_title_file', 'Tải file này lên robot?'),
        sub: file.name + ' · ' + formatBytes(file.size),
        items: [
            { icon: '🖥️', text: otaTr('aud.cf_keep_open', 'Giữ trang này mở đến khi tải lên xong.') },
            { icon: '⚠️', text: otaTr('aud.cf_trust', 'Chỉ dùng file .ota từ nguồn tin cậy.') },
        ].concat(otaConfirmItems()),
        okText: otaTr('aud.cf_ok_file', '📤 Tải lên'),
        warn: true,
    }).then((ok) => { if (ok) uploadOtaFile(file); });
}

function uploadOtaFile(file) {
    const box = document.getElementById('otaLogBox');
    if (box) box.textContent = '';
    otaLastPhase = '';
    otaLastPct = -1;
    otaFailStreak = 0;
    otaSetStartDisabled(true);
    otaLogAppend(`Upload ${file.name} (${formatBytes(file.size)})`);

    const xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/mods/AutoUpdate/uploadOta');
    xhr.setRequestHeader('Content-Type', 'application/octet-stream');
    xhr.upload.onprogress = (e) => {
        if (!e.lengthComputable) return;
        const pct = Math.round((e.loaded * 100) / e.total);
        const line = otaTr('aud.up_progress', 'Đang tải file lên robot: {pct}% ({done}/{total})',
            { pct, done: formatBytes(e.loaded), total: formatBytes(e.total) });
        otaSetProgress(pct, 'upload', line, '');
        setAutoUpdateStatus(line);
    };
    xhr.upload.onload = () => {
        otaSetProgress(100, 'upload', otaTr('aud.up_verify', 'Đã tải lên, robot đang kiểm tra file…'), '');
        setAutoUpdateStatus(otaTr('aud.up_verify', 'Đã tải lên, robot đang kiểm tra file…'));
    };
    const fail = (msg) => {
        const line = otaTr('aud.up_failed', 'Tải lên thất bại: {msg}', { msg });
        otaSetProgress(0, 'error', line, 'err');
        setAutoUpdateStatus(`<span style="color:#f87171">${line.replace(/</g, '&lt;')}</span>`);
        otaLogAppend(line);
        otaSetStartDisabled(false);
    };
    xhr.onload = () => {
        let j = {};
        try { j = JSON.parse(xhr.responseText); } catch (_) {}
        if (xhr.status !== 200 || j.status !== 'success') {
            fail(j.message || `HTTP ${xhr.status}`);
            return;
        }
        const line = otaTr('aud.up_started', 'File hợp lệ ({ver}) — robot bắt đầu cài.', { ver: j.update_version || '?' });
        setAutoUpdateStatus(line.replace(/</g, '&lt;'));
        otaLogAppend(line);
        startOtaPoll();
    };
    xhr.onerror = () => fail(otaTr('aud.up_network', 'mất kết nối tới robot'));
    xhr.send(file);
}

/* --- Official OTA (fixed GitHub link) --- */
let otaOfficialLast = null;
let otaOfficialChecking = false;

function otaTr(key, fallback, vars) {
    let s = typeof t === 'function' ? t(key, fallback) : fallback;
    Object.entries(vars || {}).forEach(([k, v]) => {
        s = s.split('{' + k + '}').join(v);
    });
    return s;
}

function otaSetUpdateBadge(on) {
    document.querySelectorAll('[data-target="#aud"]').forEach((b) => b.classList.toggle('has-update', on));
    const burger = document.getElementById('navBurger');
    if (burger) burger.classList.toggle('has-update', on);
}

function renderOfficialOta() {
    const card = document.getElementById('otaOfficialCard');
    if (!card) return;
    const title = document.getElementById('otaOfficialTitle');
    const detail = document.getElementById('otaOfficialDetail');
    const btn = document.getElementById('otaOfficialBtn');
    const banner = document.getElementById('otaUpdateBanner');
    const bannerText = document.getElementById('otaUpdateBannerText');
    const j = otaOfficialLast;
    const unknown = otaTr('aud.off_unknown_ver', 'không rõ');
    const cur = (j && j.current_version) || unknown;
    const ver = (j && j.update_version) || unknown;

    let state = 'checking';
    if (!otaOfficialChecking && j) {
        if (!j.reachable) state = 'offline';
        else if (!j.exists) state = 'none';
        else if (j.same) state = 'latest';
        else state = 'new';
    }
    card.dataset.state = state;

    const curLine = otaTr('aud.off_current', 'Robot đang chạy: {cur}', { cur });
    const curEl = document.getElementById('otaCurrentLine');
    if (curEl && j && j.current_version) curEl.textContent = curLine;
    if (state === 'checking') {
        title.textContent = '🔍 ' + otaTr('aud.off_checking', 'Đang kiểm tra bản cập nhật…');
        detail.textContent = '';
    } else if (state === 'offline') {
        title.textContent = '⚠️ ' + otaTr('aud.off_offline_title', 'Không kiểm tra được bản cập nhật');
        detail.textContent = otaTr('aud.off_offline_detail', 'Robot chưa vào được Internet. Thử lại sau.') + '\n' + curLine;
    } else if (state === 'none') {
        title.textContent = 'ℹ️ ' + otaTr('aud.off_none_title', 'Chưa có bản cập nhật mới');
        detail.textContent = curLine;
    } else if (state === 'latest') {
        title.textContent = '✅ ' + otaTr('aud.off_latest_title', 'Robot đã cài bản cập nhật này rồi');
        detail.textContent = curLine;
    } else {
        title.textContent = '🆕 ' + otaTr('aud.off_new_title', 'Có bản cập nhật mới!');
        let d = otaTr('aud.off_new_detail', 'Phiên bản mới: {new}\nRobot đang chạy: {cur}', { new: ver, cur });
        const size = formatBytes(j.size);
        if (size) d += '\n' + otaTr('aud.off_size', 'Dung lượng: {size}', { size });
        detail.textContent = d;
    }

    const isNew = state === 'new' && !otaPollTimer;
    btn.hidden = !isNew;
    otaSetUpdateBadge(isNew);
    if (banner) {
        banner.hidden = !isNew;
        if (isNew && bannerText) {
            bannerText.textContent = otaTr('aud.off_banner', 'Có bản cập nhật mới cho robot: {new}', { new: ver });
        }
    }
    const recheck = document.getElementById('otaOfficialRecheck');
    if (recheck) recheck.disabled = otaOfficialChecking;
}

async function checkOfficialOta(force) {
    if (otaOfficialChecking) return;
    otaOfficialChecking = true;
    renderOfficialOta();
    try {
        const res = await fetch('/api/mods/AutoUpdate/checkOfficial' + (force ? '?force=1' : ''));
        otaOfficialLast = await res.json();
    } catch (e) {
        otaOfficialLast = { reachable: false, error: e.message };
    }
    otaOfficialChecking = false;
    renderOfficialOta();
}

async function startOfficialOta() {
    const j = otaOfficialLast;
    if (!j || !j.url || !j.exists) return;
    const ok = await otaConfirm({
        icon: '🚀',
        title: otaTr('aud.cf_title_auto', 'Cập nhật robot ngay?'),
        sub: otaTr('aud.cf_version', 'Phiên bản {ver}', { ver: j.update_version || otaTr('aud.off_unknown_ver', 'không rõ') }),
        items: otaConfirmItems(),
        okText: otaTr('aud.cf_ok_auto', '⬇️ Cập nhật ngay'),
    });
    if (!ok) return;
    await startOtaFromURL(j.url, true);
}

function otaOfficialFromBanner() {
    if (typeof activateSection === 'function') activateSection('#aud');
    window.scrollTo({ top: 0, behavior: 'smooth' });
    startOfficialOta();
}

async function startOtaFromURL(officialUrl, confirmed) {
    const input = document.getElementById('otaUrl');
    const v = officialUrl ? { ok: true, url: officialUrl } : validateOtaUrlClient(input.value);
    if (!v.ok) {
        markOtaUrlField(false);
        setAutoUpdateStatus(v.msg);
        return;
    }
    if (!officialUrl) markOtaUrlField(true);

    const ok = confirmed || await otaConfirm({
        icon: '🔗',
        title: otaTr('aud.cf_title_url', 'Cập nhật từ link này?'),
        sub: v.url.length > 70 ? v.url.slice(0, 67) + '…' : v.url,
        items: [{ icon: '⚠️', text: otaTr('aud.cf_trust', 'Chỉ dùng file .ota từ nguồn tin cậy.') }].concat(otaConfirmItems()),
        okText: otaTr('aud.cf_ok_url', '⬇️ Cập nhật'),
        warn: true,
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
    otaSetStartDisabled(true);
    try {
        const qs = new URLSearchParams({ url: v.url });
        const res = await fetch(`/api/mods/AutoUpdate/startFromURL?${qs.toString()}`, { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            markOtaUrlField(false);
            otaSetProgress(0, 'rejected', j.message || String(res.status), 'err');
            setAutoUpdateStatus(`Không cập nhật: ${j.message || res.status} (Rejected)`);
            otaLogAppend('Từ chối: ' + (j.message || res.status));
            otaSetStartDisabled(false);
            return;
        }
        setAutoUpdateStatus('Đã bắt đầu — xem thanh tiến trình + log bên dưới. (Started — watch progress + log.)');
        otaLogAppend('Đã start update-engine — URL OK');
        startOtaPoll();
    } catch (e) {
        otaSetProgress(0, 'error', e.message, 'err');
        setAutoUpdateStatus(`Lỗi mạng: ${e.message} (Network error)`);
        otaLogAppend('Lỗi mạng: ' + e.message);
        otaSetStartDisabled(false);
    }
}

function startOtaPoll() {
    if (otaPollTimer) clearInterval(otaPollTimer);
    otaPollTimer = setInterval(pollOtaStatus, 1500);
    pollOtaStatus();
    renderOfficialOta();
}

function stopOtaPoll() {
    if (otaPollTimer) clearInterval(otaPollTimer);
    otaPollTimer = null;
    renderOfficialOta();
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
            otaSetStartDisabled(false);
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
            otaSetStartDisabled(false);
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
    checkOfficialOta(false);
});

// Language switch rewrites labels and may rebuild the mobile drawer.
document.addEventListener('wireos-i18n', () => {
    renderOfficialOta();
    otaFilePicked();
});
