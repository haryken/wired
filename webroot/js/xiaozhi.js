// Xiaozhi UI helpers

const XZ_DEFAULT_OTA = 'https://api.tenclass.net/';

let xzModeBusy = false;
let xzSaveBusy = false;
let xzAppliedMode = null; // 'xiaozhi' | 'pipeline' | 'gemini' | 'openai' | 'grok' | 'vosk'

function xzSetElStatus(elId, msg, isError) {
    const el = document.getElementById(elId);
    if (!el) return;
    el.textContent = msg || '';
    el.style.color = isError ? '#f87171' : '#4ade80';
    el.style.display = msg ? 'block' : 'none';
}

function xzSetModeStatus(msg, isError) {
    xzSetElStatus('xiaozhiStatus', msg, isError);
}

function xzSetConfigStatus(msg, isError) {
    xzSetElStatus('xzConfigStatus', msg, isError);
}

function xzSetPoolStatus(msg, isError) {
    xzSetElStatus('xzPoolStatus', msg, isError);
}

// Direct-API modes: vic-cloud talks to the AI service itself (openai.json).
// 'pipeline' (Tự chọn AI) runs as backend openai with openai.json mode pipeline.
const XZ_MODE_RADIOS = {
    xiaozhi: 'xzModeXiaozhi', pipeline: 'xzModePipeline', gemini: 'xzModeGemini',
    openai: 'xzModeOpenAI', grok: 'xzModeGrok', vosk: 'xzModeVosk',
};

function xzIsDirectMode(mode) {
    return mode === 'openai' || mode === 'pipeline' || mode === 'gemini' || mode === 'grok';
}

function xzSelectedMode() {
    for (const m of Object.keys(XZ_MODE_RADIOS)) {
        const r = document.getElementById(XZ_MODE_RADIOS[m]);
        if (r && r.checked) return m;
    }
    return 'xiaozhi';
}

function xzT(key, fallback) {
    return (typeof t === 'function') ? t(key, fallback) : fallback;
}

const XZ_GAIN_CHOICES = [100, 130, 160, 200, 250, 300, 400, 500, 600, 800, 1000];
// Chip row id -> its status line; both rows edit the same tts_gain_pct.
const XZ_GAIN_BOXES = { xzGainChips: 'xzGainStatus', volGainChips: 'volGainStatus' };

function xzRenderGain(pct) {
    const cur = pct > 0 ? pct : 100;
    const choices = XZ_GAIN_CHOICES.includes(cur)
        ? XZ_GAIN_CHOICES
        : XZ_GAIN_CHOICES.concat([cur]).sort((a, b) => a - b);
    Object.entries(XZ_GAIN_BOXES).forEach(([boxId, statusId]) => {
        const box = document.getElementById(boxId);
        if (!box) return;
        box.innerHTML = '';
        choices.forEach((n) => {
            const chip = document.createElement('button');
            chip.type = 'button';
            chip.className = 'face-greet-chip' + (n === cur ? ' selected' : '');
            chip.setAttribute('role', 'radio');
            chip.setAttribute('aria-checked', n === cur ? 'true' : 'false');
            chip.textContent = n + '%';
            if (n === 100) {
                const tag = document.createElement('span');
                tag.className = 'face-greet-chip-tag';
                tag.textContent = xzT('xz.gain_orig', 'gốc');
                chip.appendChild(tag);
            }
            chip.onclick = () => {
                if (n !== cur) xzSetGain(n, statusId);
            };
            box.appendChild(chip);
        });
    });
}

function xzGainStatus(statusId, msg, isErr) {
    Object.values(XZ_GAIN_BOXES).forEach((id) => {
        const el = document.getElementById(id);
        if (!el) return;
        const show = id === statusId && !!msg;
        el.style.display = show ? 'block' : 'none';
        el.textContent = show ? msg : '';
        el.classList.toggle('error', show && !!isErr);
    });
}

async function xzLoadGain() {
    try {
        const r = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (!r.ok) return;
        const cfg = await r.json();
        xzRenderGain(cfg.tts_gain_pct);
    } catch (e) {
        // Chips keep their last state; the AI tab reloads the full config.
    }
}

async function xzSetGain(pct, statusId) {
    try {
        const r = await fetch('/api/mods/Xiaozhi/set_tts_gain?' + new URLSearchParams({ pct: String(pct) }).toString(), { method: 'POST' });
        const j = await r.json().catch(() => ({}));
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        xzRenderGain(j.config ? j.config.tts_gain_pct : pct);
        xzGainStatus(statusId, xzT('xz.gain_saved', 'Đã lưu {n}% — áp dụng từ lần gọi “Hey Vector” tiếp theo.').replace('{n}', pct), false);
    } catch (e) {
        xzGainStatus(statusId, xzT('xz.gain_err', 'Lỗi lưu âm lượng:') + ' ' + (e.message || e), true);
    }
}

function xzSelectedPreset() {
    const custom = document.getElementById('xzPresetCustom');
    if (custom && custom.checked) return 'custom';
    const course = document.getElementById('xzPresetCourse');
    if (course && course.checked) return 'course';
    return 'vi_pool';
}

function xzShowPresetUI(preset) {
    if (preset !== 'custom' && preset !== 'course') preset = 'vi_pool';
    const radios = { vi_pool: 'xzPresetVi', course: 'xzPresetCourse', custom: 'xzPresetCustom' };
    const blocks = { vi_pool: 'xzPoolHelp', course: 'xzCourseFields', custom: 'xzCustomFields' };
    Object.keys(radios).forEach(function (p) {
        const r = document.getElementById(radios[p]);
        if (r) r.checked = (p === preset);
        const b = document.getElementById(blocks[p]);
        if (b) b.style.display = (p === preset) ? 'block' : 'none';
    });
    if (preset === 'course') xzRenderCourse();
    xzSetCoursePolling(preset === 'course');
}

function xzShowConfigForMode(mode) {
    const block = document.getElementById('xzConfigBlock');
    if (block) block.style.display = (mode === 'xiaozhi') ? 'block' : 'none';
    const direct = xzIsDirectMode(mode);
    const oaiBlock = document.getElementById('oaiConfigBlock');
    if (oaiBlock) oaiBlock.style.display = direct ? 'block' : 'none';
    Object.keys(XZ_MODE_RADIOS).forEach(function (m) {
        const r = document.getElementById(XZ_MODE_RADIOS[m]);
        if (r) r.checked = (m === mode);
    });
    document.querySelectorAll('.mode-card[data-mode]').forEach(function (card) {
        const on = card.dataset.mode === mode;
        card.classList.toggle('selected', on);
        card.setAttribute('aria-checked', on ? 'true' : 'false');
    });
    if (typeof showVoiceMode === 'function') showVoiceMode(mode === 'vosk' ? 'vosk' : 'xz');
    const gainBox = document.getElementById('xzGainBox');
    if (gainBox) gainBox.style.display = mode === 'vosk' ? 'none' : '';
    if (mode === 'xiaozhi') xzShowPresetUI(xzSelectedPreset());
    if (direct && typeof oaiSetLayout === 'function') oaiSetLayout(mode);
    if (direct && typeof oaiLoad === 'function') oaiLoad();
}

function xzSelectedConvMode() {
    const single = document.getElementById('xzConvSingle');
    return (single && single.checked) ? 'single' : 'continuous';
}

function xzApplyCfg(cfg) {
    if (!cfg) return;
    xzRenderGain(cfg.tts_gain_pct);
    const ota = document.getElementById('xzOTABaseURL');
    const ep = document.getElementById('xzEndpoint');
    const did = document.getElementById('xzDeviceID');
    const cid = document.getElementById('xzClientID');
    const convCont = document.getElementById('xzConvContinuous');
    const convSingle = document.getElementById('xzConvSingle');

    if (ota) ota.value = (cfg.ota_base_url && cfg.ota_base_url.trim()) ? cfg.ota_base_url : XZ_DEFAULT_OTA;
    if (ep) ep.value = cfg.endpoint || '';
    if (did) did.value = cfg.device_id || '';
    if (cid) cid.value = cfg.client_id || '';
    const conv = (cfg.conversation_mode === 'single') ? 'single' : 'continuous';
    if (convCont) convCont.checked = (conv === 'continuous');
    if (convSingle) convSingle.checked = (conv === 'single');

    const gviOn = document.getElementById('xzGameTtsOn');
    const gviOff = document.getElementById('xzGameTtsOff');
    if (gviOn && gviOff) {
        const on = !!cfg.game_google_tts_vi;
        gviOn.checked = on;
        gviOff.checked = !on;
    }

    xzLastCfg = cfg;
    xzCourseSel = null;
    xzCourseDirty = false;
    const preset = (cfg.identity_mode === 'vi_pool' || cfg.identity_mode === 'course'
        || cfg.identity_mode === 'custom') ? cfg.identity_mode
        : (cfg.device_id ? 'custom' : 'vi_pool');
    xzShowPresetUI(preset);

    const using = document.getElementById('xzPoolUsing');
    if (using && preset === 'vi_pool') {
        using.textContent = cfg.device_id
            ? (xzT('xz.pool_mac', 'MAC đang dùng: ') + cfg.device_id)
            : '';
    }

    let mode = !cfg.enabled ? 'vosk'
        : xzIsDirectMode(cfg.backend) ? cfg.backend : 'xiaozhi';
    if (mode === 'openai') {
        // Only get / set_enabled report openai_mode; otherwise keep the choice.
        if (cfg.openai_mode) mode = cfg.openai_mode === 'pipeline' ? 'pipeline' : 'openai';
        else if (xzAppliedMode === 'pipeline') mode = 'pipeline';
    }
    xzAppliedMode = mode;
    xzShowConfigForMode(mode);
}

async function xzLoad() {
    try {
        const resp = await fetch('/api/mods/Xiaozhi/get');
        if (!resp.ok) { xzSetModeStatus('Không tải được cấu hình (Failed to load config)', true); return; }
        const cfg = await resp.json();
        xzApplyCfg(cfg);
    } catch (e) {
        xzSetModeStatus('Lỗi tải: ' + e.message + ' (Error loading)', true);
    }
}

async function xzOnPresetRadio() {
    const preset = xzSelectedPreset();
    xzShowPresetUI(preset);
    if (xzSelectedMode() !== 'xiaozhi') return;
    if (preset === 'course' && !(xzLastCfg && xzLastCfg.identity_mode === 'course')) {
        // Pick the lesson first; "Lưu & áp dụng" switches (one reconnect).
        xzSetElStatus('xzCourseStatus', xzT('xz.course_pick', 'Chọn khóa học, giọng, bài rồi bấm Lưu & áp dụng.'), false);
        return;
    }
    xzAppliedMode = null;
    await xzSetListenMode('xiaozhi');
}

// --- English-teaching course preset ---

let xzLastCfg = null;
let xzCourses = null;
let xzCourseSel = null; // {idx, voice, book, unit} being edited
let xzCourseDirty = false; // form edited since the last load — polling must not overwrite it
let xzCoursePollTimer = null;
const XZ_COURSE_POLL_MS = 4000;

async function xzLoadCourses() {
    if (xzCourses) return xzCourses;
    try {
        const resp = await fetch('/api/mods/Xiaozhi/courses');
        if (!resp.ok) return null;
        const j = await resp.json();
        xzCourses = Array.isArray(j.courses) ? j.courses : null;
    } catch (e) {
        xzCourses = null;
    }
    return xzCourses;
}

function xzCourseByIdx(idx) {
    return (xzCourses || []).find(function (c) { return c.idx === idx; }) || null;
}

/** Stored lesson for a course from the last loaded config (clamped like the backend). */
function xzStoredLesson(idx) {
    const cfg = xzLastCfg || {};
    const c = xzCourseByIdx(idx) || (xzCourses || [])[0];
    if (!c) return null;
    const key = String(c.idx);
    const voice = (cfg.course_voice && cfg.course_voice[key] === 1) ? 1 : 0;
    let book = (cfg.course_book && cfg.course_book[key]) || 0;
    if (book < 0 || book >= c.books.length) book = 0;
    return { idx: c.idx, voice: voice, book: book, unit: xzStoredUnit(c, book) };
}

function xzStoredUnit(c, book) {
    const cfg = xzLastCfg || {};
    let unit = (cfg.course_unit && cfg.course_unit[c.idx + '.' + book]) || 1;
    if (unit < 1 || unit > c.books[book].units.length) unit = 1;
    return unit;
}

/** Lesson the robot is using now (saved config), or null when not in course mode. */
function xzActiveLesson() {
    const cfg = xzLastCfg || {};
    if (cfg.identity_mode !== 'course' || !xzCourses) return null;
    return xzStoredLesson(cfg.course_idx || 1);
}

/** Radio card; label may be a string or a DOM node. */
function xzRadio(name, value, checked, label, onchange) {
    const lbl = document.createElement('label');
    const inp = document.createElement('input');
    inp.type = 'radio';
    inp.name = name;
    inp.value = String(value);
    inp.checked = checked;
    inp.addEventListener('change', function () {
        xzCourseDirty = true;
        onchange();
    });
    lbl.appendChild(inp);
    if (typeof label === 'string') {
        const span = document.createElement('span');
        span.textContent = label;
        lbl.appendChild(span);
    } else {
        lbl.appendChild(label);
    }
    return lbl;
}

function xzRenderCourseActive() {
    const box = document.getElementById('xzCourseActive');
    const lessonEl = document.getElementById('xzCourseActiveLesson');
    const metaEl = document.getElementById('xzCourseActiveMeta');
    const using = document.getElementById('xzCourseUsing');
    const cfg = xzLastCfg || {};
    const a = xzActiveLesson();
    const c = a && xzCourseByIdx(a.idx);
    if (box) box.style.display = c ? 'block' : 'none';
    if (c) {
        const parts = [c.name];
        if (c.books.length > 1) parts.push(c.books[a.book].name);
        parts.push(c.books[a.book].units[a.unit - 1]);
        if (lessonEl) lessonEl.textContent = parts.join(' → ');
        const meta = [xzT('xz.course_voice', 'Giọng:') + ' ' + c.voices[a.voice].label];
        if (cfg.student_name) meta.push(xzT('xz.course_student', 'Tên học sinh:') + ' ' + cfg.student_name);
        if (metaEl) metaEl.textContent = meta.join(' · ');
    }
    if (using) {
        using.textContent = (c && cfg.device_id)
            ? (xzT('xz.course_using', 'Đang dùng: ') + 'MAC ' + cfg.device_id + ' · client ' + (cfg.client_id || '—'))
            : '';
    }
}

function xzRenderCourseUnits() {
    const el = document.getElementById('xzCourseUnits');
    const sel = xzCourseSel;
    const c = sel && xzCourseByIdx(sel.idx);
    if (!el || !c) return;
    const a = xzActiveLesson();
    el.innerHTML = '';
    c.books[sel.book].units.forEach(function (name, i) {
        const body = document.createElement('span');
        const num = document.createElement('span');
        num.className = 'xz-unit-num';
        num.textContent = '#' + (i + 1);
        if (a && a.idx === sel.idx && a.book === sel.book && a.unit === i + 1) {
            const now = document.createElement('span');
            now.className = 'xz-unit-now';
            now.textContent = xzT('xz.course_now', 'đang học');
            num.appendChild(now);
        }
        body.appendChild(num);
        body.appendChild(document.createTextNode(name));
        el.appendChild(xzRadio('xzCourseUnit', i + 1, i + 1 === sel.unit, body, function () {
            xzCourseSel.unit = i + 1;
        }));
    });
}

async function xzRenderCourse() {
    if (!(await xzLoadCourses())) {
        xzSetElStatus('xzCourseStatus', xzT('xz.course_load_err', 'Không tải được danh sách khóa học.'), true);
        return;
    }
    if (!xzCourseSel) {
        xzCourseSel = xzStoredLesson((xzLastCfg && xzLastCfg.course_idx) || 1);
        xzCourseDirty = false;
        const stu = document.getElementById('xzCourseStudent');
        if (stu) stu.value = (xzLastCfg && xzLastCfg.student_name) || '';
    }
    const sel = xzCourseSel;
    const c = xzCourseByIdx(sel.idx);
    if (!c) return;

    const list = document.getElementById('xzCourseList');
    if (list) {
        list.innerHTML = '';
        xzCourses.forEach(function (co) {
            list.appendChild(xzRadio('xzCourseIdx', co.idx, co.idx === sel.idx, co.name, function () {
                xzCourseSel = xzStoredLesson(co.idx);
                xzRenderCourse();
            }));
        });
    }
    const voices = document.getElementById('xzCourseVoices');
    if (voices) {
        voices.innerHTML = '';
        c.voices.forEach(function (v, i) {
            voices.appendChild(xzRadio('xzCourseVoice', i, i === sel.voice, v.label, function () {
                xzCourseSel.voice = i;
            }));
        });
    }
    const books = document.getElementById('xzCourseBooks');
    const bookLbl = document.getElementById('xzCourseBookLbl');
    const multiBook = c.books.length > 1;
    if (bookLbl) bookLbl.style.display = multiBook ? '' : 'none';
    if (books) {
        books.style.display = multiBook ? '' : 'none';
        books.innerHTML = '';
        c.books.forEach(function (b, i) {
            books.appendChild(xzRadio('xzCourseBook', i, i === sel.book, b.name, function () {
                xzCourseSel.book = i;
                xzCourseSel.unit = xzStoredUnit(c, i);
                xzRenderCourseUnits();
            }));
        });
    }
    xzRenderCourseUnits();
    xzRenderCourseActive();
}

function xzCourseSignature(cfg) {
    return JSON.stringify([cfg.identity_mode, cfg.enabled, cfg.course_idx, cfg.course_voice,
        cfg.course_book, cfg.course_unit, cfg.student_name, cfg.device_id, cfg.client_id]);
}

/** Picks up lesson changes made on the robot (voice "next unit" via MCP self.otto.next_unit / prev_unit). */
async function xzPollCourse() {
    const block = document.getElementById('xzCourseFields');
    if (document.hidden || !block || block.offsetParent === null || xzSaveBusy) return;
    try {
        const resp = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (!resp.ok) return;
        const cfg = await resp.json();
        if (xzLastCfg && xzCourseSignature(cfg) === xzCourseSignature(xzLastCfg)) return;
        xzLastCfg = cfg;
        if (!xzCourseDirty) xzCourseSel = null;
        if (xzCourseSel) {
            xzRenderCourseUnits();
            xzRenderCourseActive();
        } else {
            xzRenderCourse();
        }
    } catch (e) {
        /* next tick */
    }
}

function xzSetCoursePolling(on) {
    if (on && !xzCoursePollTimer) {
        xzCoursePollTimer = setInterval(xzPollCourse, XZ_COURSE_POLL_MS);
    } else if (!on && xzCoursePollTimer) {
        clearInterval(xzCoursePollTimer);
        xzCoursePollTimer = null;
    }
}

async function xzSaveCourse() {
    if (xzSaveBusy || !xzCourseSel) return;
    xzSaveBusy = true;
    const btn = document.getElementById('xzCourseSaveBtn');
    if (btn) btn.disabled = true;
    xzSetElStatus('xzCourseStatus', xzT('xz.course_saving', 'Đang lưu…'), false);
    try {
        const params = new URLSearchParams({
            course_idx: String(xzCourseSel.idx),
            voice: String(xzCourseSel.voice),
            book: String(xzCourseSel.book),
            unit: String(xzCourseSel.unit),
            student_name: (document.getElementById('xzCourseStudent') || { value: '' }).value.trim(),
        });
        const resp = await fetch('/api/mods/Xiaozhi/set_course?' + params.toString(), { method: 'POST' });
        const j = await resp.json();
        if (j.status !== 'success') {
            xzSetElStatus('xzCourseStatus', xzT('xz.course_err', 'Lỗi lưu: ') + (j.message || 'unknown'), true);
            return;
        }
        if (j.config) xzApplyCfg(j.config);
        let msg;
        if (!j.identity_changed) msg = xzT('xz.course_saved_lesson', 'Đã lưu bài / tên — dùng ngay ở câu nói sau.');
        else if (j.applied) msg = xzT('xz.course_saved_apply', 'Đã lưu. Robot đang kết nối lại và sẽ tự chào.');
        else msg = xzT('xz.course_saved_restart', 'Đã lưu. Cloud khởi động lại — đợi ~10s rồi Hey Vector.');
        xzSetElStatus('xzCourseStatus', msg, false);
        xzSetModeStatus('', false);
    } catch (e) {
        xzSetElStatus('xzCourseStatus', 'Lỗi: ' + e.message, true);
    } finally {
        xzSaveBusy = false;
        if (btn) btn.disabled = false;
    }
}

/** Persist config fields. Returns {ok, config?, message?}. Does not set UI status. */
async function xzSaveConfig() {
    let otaVal = document.getElementById('xzOTABaseURL').value.trim();
    if (!otaVal) otaVal = XZ_DEFAULT_OTA;
    const params = new URLSearchParams({
        ota_base_url: otaVal,
        device_id: document.getElementById('xzDeviceID').value.trim(),
        client_id: document.getElementById('xzClientID').value.trim(),
        enabled: (xzAppliedMode === 'vosk') ? 'false' : 'true',
        auto_apply_ota_websocket: 'true',
        tts_mode: 'xiaozhi',
        conversation_mode: xzSelectedConvMode(),
        idle_timeout_sec: '20',
        identity_mode: xzSelectedPreset(),
        game_google_tts_vi: 'true',
    });
    const resp = await fetch('/api/mods/Xiaozhi/save?' + params.toString(), { method: 'POST' });
    const j = await resp.json();
    if (j.status === 'success') {
        if (j.config) xzApplyCfg(j.config);
        return { ok: true, config: j.config };
    }
    return { ok: false, message: j.message || 'unknown' };
}

/** One button: save config, then OTA activation check. */
async function xzSaveAndActivate() {
    if (xzSaveBusy) return;
    xzSaveBusy = true;
    const btn = document.getElementById('xzSaveActivateBtn');
    if (btn) btn.disabled = true;
    const codeBox = document.getElementById('xzCodeBox');
    if (codeBox) codeBox.style.display = 'none';
    xzSetConfigStatus('Đang lưu cấu hình…', false);

    try {
        const saved = await xzSaveConfig();
        if (!saved.ok) {
            xzSetConfigStatus('Lỗi lưu: ' + (saved.message || 'unknown'), true);
            return;
        }

        xzSetConfigStatus('Đã lưu. Đang kiểm tra kích hoạt trên máy chủ…', false);
        const resp = await fetch('/api/mods/Xiaozhi/generate_code', { method: 'POST' });
        if (!resp.ok) {
            xzSetConfigStatus('Đã lưu, nhưng lỗi lấy mã: ' + (await resp.text()), true);
            return;
        }
        const j = await resp.json();
        if (j.status === 'error') {
            xzSetConfigStatus('Đã lưu, nhưng lỗi: ' + (j.message || 'unknown'), true);
            return;
        }
        if (j.device_id) document.getElementById('xzDeviceID').value = j.device_id;
        if (j.client_id) document.getElementById('xzClientID').value = j.client_id;

        await xzLoad();

        const did = document.getElementById('xzDeviceID').value || j.device_id || '';
        if (j.code) {
            document.getElementById('xzCode').textContent = j.code;
            if (codeBox) codeBox.style.display = 'block';
            xzSetConfigStatus(
                'Đã lưu. Chưa kích hoạt — nhập mã bên dưới tại xiaozhi.me để liên kết robot với tài khoản Xiaozhi.',
                false
            );
        } else {
            xzSetConfigStatus(
                'Đã lưu. ID này đã kích hoạt / liên kết với tài khoản trên hệ thống Xiaozhi rồi. Đánh thức robot (Hey Vector) và dùng bình thường.',
                false
            );
            if (did) {
                xzSetConfigStatus(
                    'Đã lưu. Device ID ' + did + ' đã liên kết với tài khoản Xiaozhi. Đánh thức robot (Hey Vector) và dùng bình thường.',
                    false
                );
            }
        }
    } catch (e) {
        xzSetConfigStatus('Lỗi: ' + e.message, true);
    } finally {
        xzSaveBusy = false;
        if (btn) btn.disabled = false;
    }
}

async function xzRenewPool() {
    if (xzSaveBusy) return;
    xzSaveBusy = true;
    const btn = document.getElementById('xzRenewBtn');
    if (btn) btn.disabled = true;
    xzSetPoolStatus(typeof t === 'function' ? t('xz.pool_renewing', 'Đang làm mới…') : 'Đang làm mới…', false);
    try {
        const resp = await fetch('/api/mods/Xiaozhi/renew_pool', { method: 'POST' });
        const j = await resp.json();
        if (j.status !== 'success') {
            xzSetPoolStatus((typeof t === 'function' ? t('xz.pool_renew_err', 'Lỗi làm mới: ') : 'Lỗi làm mới: ') + (j.message || 'unknown'), true);
            return;
        }
        if (j.config) xzApplyCfg(j.config);
        xzSetPoolStatus(j.applied
            ? xzT('xz.pool_renewed_live', 'Đã làm mới — robot kết nối lại và sẽ tự chào.')
            : xzT('xz.pool_renewed', 'Đã làm mới. Đánh thức robot (Hey Vector).'), false);
    } catch (e) {
        xzSetPoolStatus('Lỗi: ' + e.message, true);
    } finally {
        xzSaveBusy = false;
        if (btn) btn.disabled = false;
    }
}

async function xzSetListenMode(mode) {
    if (xzModeBusy) return;
    const wantOAI = xzIsDirectMode(mode);
    const wantXz = (mode === 'xiaozhi') || wantOAI;
    if (xzAppliedMode === mode) {
        xzShowConfigForMode(mode);
        return;
    }
    xzModeBusy = true;
    xzShowConfigForMode(mode);
    xzSetModeStatus(mode === 'pipeline'
        ? xzT('xz.switching_pl', 'Đang bật Tự chọn AI…')
        : mode === 'gemini'
        ? xzT('xz.switching_gemini', 'Đang bật Gemini Live…')
        : mode === 'grok'
        ? xzT('xz.switching_grok', 'Đang bật Grok Voice…')
        : wantOAI
        ? xzT('xz.switching_oai', 'Đang bật OpenAI…')
        : wantXz
        ? xzT('xz.switching_xz', 'Đang bật Xiaozhi (tắt Vosk)…')
        : xzT('xz.switching_vosk', 'Đang bật Vosk (tắt Xiaozhi)…'), false);
    try {
        const params = new URLSearchParams({ enabled: wantXz ? 'true' : 'false' });
        if (wantOAI) {
            params.set('backend', mode);
        } else if (wantXz) {
            params.set('backend', 'xiaozhi');
            params.set('identity_mode', xzSelectedPreset());
        }
        const resp = await fetch('/api/mods/Xiaozhi/set_enabled?' + params.toString(), { method: 'POST' });
        const j = await resp.json();
        if (j.status !== 'success') {
            xzSetModeStatus('Lỗi đổi chế độ: ' + (j.message || 'unknown'), true);
            await xzLoad();
            return;
        }
        if (j.config) xzApplyCfg(j.config);
        else {
            xzAppliedMode = mode;
            xzShowConfigForMode(mode);
        }
        let msg;
        if (wantOAI) {
            msg = !(j.applied || j.restarted)
                ? xzT('xz.on_saved', 'Đã lưu.')
                : mode === 'pipeline'
                ? xzT('xz.on_pl', 'Đã chuyển sang Tự chọn AI — robot kết nối lại và sẽ tự chào (cần key cho các bước đã chọn).')
                : mode === 'gemini'
                ? xzT('xz.on_gemini_live', 'Đã chuyển sang Gemini Live — robot kết nối lại và sẽ tự chào (cần key Gemini).')
                : mode === 'grok'
                ? xzT('xz.on_grok', 'Đã chuyển sang Grok Voice — robot kết nối lại và sẽ tự chào (cần key xAI).')
                : xzT('xz.on_oai_live', 'Đã chuyển sang OpenAI — robot kết nối lại và sẽ tự chào (cần API key).');
        } else if (j.restarted) {
            msg = wantXz
                ? xzT('xz.on_xiaozhi', 'Đã chuyển sang Xiaozhi. Đợi ~5–10s rồi Hey Vector.')
                : xzT('xz.on_vosk', 'Đã chuyển sang Vosk. Đợi ~5–10s rồi Hey Vector.');
        } else if (j.applied) {
            msg = wantXz
                ? xzT('xz.on_xiaozhi_live', 'Đã chuyển sang Xiaozhi — robot kết nối lại và sẽ tự chào.')
                : xzT('xz.on_vosk_live', 'Đã chuyển sang Vosk. Lần Hey Vector đầu có thể chậm vài giây (nạp Vosk).');
        } else {
            msg = xzT('xz.on_saved', 'Đã lưu.');
        }
        xzSetModeStatus(msg, false);
    } catch (e) {
        xzSetModeStatus('Lỗi đổi chế độ: ' + e.message, true);
        await xzLoad();
    } finally {
        xzModeBusy = false;
    }
}

function xzOnModeRadio() {
    xzSetListenMode(xzSelectedMode());
}

// The mode cards are the picker: a click anywhere on a card selects it.
function xzPickMode(mode) {
    if (xzModeBusy || !XZ_MODE_RADIOS[mode]) return;
    xzSetListenMode(mode);
}

function xzCardKey(ev, mode) {
    if (ev.key === 'Enter' || ev.key === ' ') {
        ev.preventDefault();
        xzPickMode(mode);
    }
}

async function xzSaveGameGoogleVi() {
    const onEl = document.getElementById('xzGameTtsOn');
    if (!onEl) return;
    const on = !!onEl.checked;
    try {
        const params = new URLSearchParams({ enabled: on ? 'true' : 'false' });
        const resp = await fetch('/api/mods/Xiaozhi/set_game_google_tts_vi?' + params.toString(), { method: 'POST' });
        const j = await resp.json();
        if (j.status !== 'success') {
            xzSetElStatus('xzGameTtsStatus', 'Lỗi lưu: ' + (j.message || 'unknown'), true);
            await xzLoad();
            return;
        }
        if (j.config) xzApplyCfg(j.config);
        xzSetElStatus('xzGameTtsStatus', on
            ? 'Đã bật Google VI — chọn trong tab Game → chế độ bình luận.'
            : 'Đã tắt Google VI.', false);
    } catch (e) {
        xzSetElStatus('xzGameTtsStatus', 'Lỗi: ' + e.message, true);
    }
}

document.addEventListener('DOMContentLoaded', function () {
    xzLoad();
    document.querySelectorAll('.tabs button[data-target="#xiaozhi"]').forEach(function (btn) {
        btn.addEventListener('click', function () { xzLoad(); });
    });
    const sel = document.getElementById('navSelect');
    if (sel) {
        sel.addEventListener('change', function () {
            if (sel.value === '#xiaozhi') xzLoad();
        });
    }
});
