// Xiaozhi UI helpers

const XZ_DEFAULT_OTA = 'https://api.tenclass.net/';

let xzModeBusy = false;
let xzSaveBusy = false;
let xzAppliedMode = null; // 'xiaozhi' | 'pipeline' | 'gemini' | 'openai' | 'grok' | 'qwen' | 'vosk'
let xzPreviewMode = null; // card selected for reading; Apply commits this

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
    openai: 'xzModeOpenAI', grok: 'xzModeGrok', qwen: 'xzModeQwen', vosk: 'xzModeVosk',
};

function xzIsDirectMode(mode) {
    return mode === 'openai' || mode === 'pipeline' || mode === 'gemini' || mode === 'grok' || mode === 'qwen';
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

// 30% ≈ quiet like old Wwise VOLUME_LOW; 100% = current full ALSA loudness.
const XZ_GAIN_CHOICES = [30, 45, 60, 75, 100, 130, 160, 200, 250, 300, 400, 500, 600, 800, 1000];
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
            if (n === 30 || n === 100) {
                const tag = document.createElement('span');
                tag.className = 'face-greet-chip-tag';
                tag.textContent = n === 30
                    ? xzT('xz.gain_quiet', 'nhỏ')
                    : xzT('xz.gain_loud', 'to');
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

function xzModeLabel(mode) {
    if (mode === 'pipeline') return xzT('xz.pl_card', 'Tự chọn AI');
    if (mode === 'gemini') return 'Gemini Live';
    if (mode === 'openai') return xzT('xz.oai_card', 'OpenAI Realtime');
    if (mode === 'grok') return 'Grok Voice';
    if (mode === 'qwen') return 'Qwen Live';
    if (mode === 'vosk') return 'Vosk';
    return 'Xiaozhi';
}

function xzUpdateModeApplyHint() {
    const hint = document.getElementById('xzModeApplyHint');
    const btn = document.getElementById('xzModeApplyBtn');
    const preview = xzPreviewMode || xzSelectedMode();
    const same = preview && preview === xzAppliedMode;
    if (btn) btn.disabled = !!xzModeBusy || same;
    if (!hint) return;
    if (same) {
        hint.textContent = xzT('xz.apply_current', 'Đây là chế độ đang chạy trên robot.');
    } else {
        hint.textContent = xzT('xz.apply_hint_sel', 'Đã chọn {mode} — bấm Áp dụng để đổi trên robot.')
            .replace('{mode}', xzModeLabel(preview));
    }
    document.querySelectorAll('.mode-card-apply').forEach(function (b) {
        const card = b.closest('.mode-card');
        const m = card && card.dataset.mode;
        b.disabled = !!xzModeBusy;
        b.classList.toggle('is-current', m === xzAppliedMode);
    });
}

function xzShowConfigForMode(mode) {
    xzPreviewMode = mode;
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
        const m = card.dataset.mode;
        const preview = m === mode;
        const applied = m === xzAppliedMode;
        card.classList.toggle('selected', preview);
        card.classList.toggle('is-applied', applied);
        card.setAttribute('aria-checked', preview ? 'true' : 'false');
    });
    if (typeof showVoiceMode === 'function') showVoiceMode(mode === 'vosk' ? 'vosk' : 'xz');
    const gainBox = document.getElementById('xzGainBox');
    if (gainBox) gainBox.style.display = mode === 'vosk' ? 'none' : '';
    if (mode === 'xiaozhi') xzShowPresetUI(xzSelectedPreset());
    if (direct && typeof oaiSetLayout === 'function') oaiSetLayout(mode);
    if (direct && typeof oaiLoad === 'function') oaiLoad();
    xzUpdateModeApplyHint();
    xzUpdatePresetApplyHint();
}

var xzConvAppliedMode = 'continuous';
var xzConvPreviewMode = 'continuous';

function convModeStatus(msg) {
    const el = document.getElementById('convModeStatus');
    if (!el) return;
    el.style.display = msg ? 'block' : 'none';
    el.textContent = msg || '';
}

function xzNormalizeConvMode(mode) {
    if (mode === 'single' || mode === 'stand' || mode === 'ambient' || mode === 'open' || mode === 'gate') {
        return mode;
    }
    return 'continuous';
}

function convModeLabel(mode) {
    mode = xzNormalizeConvMode(mode);
    if (mode === 'single') return xzT('xz.conv_single', 'Một lượt');
    if (mode === 'stand') return xzT('xz.conv_stand', 'Đứng nghe');
    if (mode === 'ambient') return xzT('xz.conv_ambient', 'Kéo dài');
    if (mode === 'open') return xzT('xz.conv_open', 'Mở sẵn');
    if (mode === 'gate') return xzT('xz.conv_gate', 'Ngưỡng');
    return xzT('xz.conv_cont', 'Liên tục');
}

function xzConvDetailCopy(mode) {
    mode = xzNormalizeConvMode(mode);
    const map = {
        continuous: {
            flow: [
                xzT('xz.conv_cont_flow_p1', 'Hey Vector → ting + lưng xanh, robot đứng yên → bạn nói → robot trả lời.'),
                xzT('xz.conv_cont_flow_p2', 'Trả lời xong mic mở lại ngay để nói tiếp — không cần Hey Vector mỗi câu.'),
                xzT('xz.conv_cont_flow_p3', 'Im khoảng 10 giây thì hết lượt nghe; robot có thể đi lại.'),
            ],
            tech: [
                xzT('xz.conv_cont_tech_p1', 'Mic mở theo giọng nói: nói xong mới gửi lên AI; im ~10 giây thì đóng lượt nghe.'),
                xzT('xz.conv_cont_tech_p2', 'Giữa các câu trong cùng cuộc chat vẫn giữ kết nối AI — khỏi gọi lại Hey Vector.'),
                xzT('xz.conv_cont_tech_p3', 'Đây là kiểu gốc của OpenAI / Gemini / Grok / Tự chọn AI.'),
            ],
            extra: xzT('xz.conv_cont_extra', 'Phù hợp chat nhanh: nói — trả lời — nói tiếp.'),
        },
        stand: {
            flow: [
                xzT('xz.conv_stand_flow_p1', 'Giống Liên tục: ting, đứng yên, lưng xanh — không đi lung tung lúc chat.'),
                xzT('xz.conv_stand_flow_p2', 'Im lâu vẫn đứng nghe tiếp (~2 phút); không tự cắt sau khoảng 10 giây im.'),
                xzT('xz.conv_stand_flow_p3', 'Trả lời xong vẫn đứng nghe trong cùng cuộc trò chuyện; tạm biệt hoặc hết phiên thì thôi.'),
            ],
            tech: [
                xzT('xz.conv_stand_tech_p1', 'Khác Liên tục: không cắt cứng ~10 giây im — cửa sổ nghe dài như ESP32 (~2 phút).'),
                xzT('xz.conv_stand_tech_p2', 'Im hết ~2 phút mới tắt mic và đóng kênh AI; nói “tạm biệt” cũng kết thúc sớm.'),
                xzT('xz.conv_stand_tech_p3', 'Sau mỗi câu trả lời, robot đứng nghe lại trong cùng cuộc chat.'),
            ],
            extra: xzT('xz.conv_stand_extra', 'Dùng khi muốn đứng yên, chat thong thả, không sợ bị cắt sớm.'),
        },
        ambient: {
            flow: [
                xzT('xz.conv_ambient_flow_p1', 'Ban đầu đứng nghe như thường; sau đó robot có thể đi lại.'),
                xzT('xz.conv_ambient_flow_p2', 'Trong lúc đi vẫn nghe khi bạn nói — lưng vẫn xanh.'),
                xzT('xz.conv_ambient_flow_p3', 'Im vài phút thì tắt — muốn nói tiếp phải Hey Vector lại.'),
            ],
            tech: [
                xzT('xz.conv_ambient_tech_p1', 'Sau đoạn đứng nghe, robot được phép đi; mic vẫn cầm sẵn khi bạn nói.'),
                xzT('xz.conv_ambient_tech_p2', 'Im lâu hơn Liên tục (khoảng vài phút) mới tắt hoàn toàn đến lần wake sau.'),
                xzT('xz.conv_ambient_tech_p3', 'Trong phiên vẫn giữ kênh AI giữa các câu; hết idle thì phải Hey Vector lại.'),
            ],
            extra: xzT('xz.conv_ambient_extra', 'Hợp khi muốn robot vừa đi vừa còn nghe được bạn.'),
        },
        open: {
            flow: [
                xzT('xz.conv_open_flow_p1', 'Có ting và lưng xanh — robot vẫn đi lại bình thường (không đứng đông).'),
                xzT('xz.conv_open_flow_p2', 'Mic sẵn sàng đến khi bạn nói xong, rồi robot trả lời.'),
                xzT('xz.conv_open_flow_p3', 'Im lâu (~2 phút) thì đóng kênh; muốn nói tiếp cần đánh thức lại.'),
            ],
            tech: [
                xzT('xz.conv_open_tech_p1', 'Không bắt robot đứng yên kiểu “nghe lệnh” — vẫn đi trong lúc chờ bạn nói.'),
                xzT('xz.conv_open_tech_p2', 'Mic mở theo giọng nói; hết câu thì đóng mic để robot nói, rồi mở lại lượt sau.'),
                xzT('xz.conv_open_tech_p3', 'Giữ kết nối AI trong phiên; idle ~2 phút (giống ESP32) thì đóng kênh.'),
            ],
            extra: xzT('xz.conv_open_extra', 'Hợp khi muốn chat mà robot vẫn đi quanh.'),
        },
        gate: {
            flow: [
                xzT('xz.conv_gate_flow_p1', 'Giống Mở sẵn: ting, lưng xanh, robot vẫn đi.'),
                xzT('xz.conv_gate_flow_p2', 'Chỉ nhận khi bạn nói đủ rõ — bớt nghe nhầm tiếng ồn / TV.'),
                xzT('xz.conv_gate_flow_p3', 'Chọn mức RMS bên dưới: thấp = dễ nghe hơn; cao = lọc ồn mạnh hơn.'),
            ],
            tech: [
                xzT('xz.conv_gate_tech_p1', 'Cùng kiểu Mở sẵn, thêm sàn độ to (RMS). Mặc định 800.'),
                xzT('xz.conv_gate_tech_p2', 'Tiếng nhỏ hơn ngưỡng không gửi lên AI — giảm kích hoạt nhầm.'),
                xzT('xz.conv_gate_tech_p3', 'Idle / đóng kênh giống Mở sẵn (~2 phút im).'),
            ],
            extra: xzT('xz.conv_gate_extra', 'Hợp phòng ồn hoặc khi robot hay nghe nhầm.'),
        },
        single: {
            flow: [
                xzT('xz.conv_single_flow_p1', 'Một câu hỏi → một câu trả lời rồi tắt mic.'),
                xzT('xz.conv_single_flow_p2', 'Muốn nói tiếp phải Hey Vector (hoặc nút lưng) lại.'),
                xzT('xz.conv_single_flow_p3', 'Tiết kiệm pin / ít bị nghe nhầm giữa các lần gọi.'),
            ],
            tech: [
                xzT('xz.conv_single_tech_p1', 'Chỉ một vòng hỏi–đáp; xong là đóng mic.'),
                xzT('xz.conv_single_tech_p2', 'Không giữ kênh AI giữa các lần — mỗi lần gọi là phiên mới.'),
                xzT('xz.conv_single_tech_p3', 'Gọi khi im lặng hầu như không cần vì mic không mở sẵn lâu.'),
            ],
            extra: xzT('xz.conv_single_extra', 'Hợp hỏi một câu rồi thôi, hoặc nhà có nhiều tiếng ồn.'),
        },
    };
    return map[mode] || map.continuous;
}

function applyGateRMS(n) {
    const sel = document.getElementById('xzGateRMS');
    if (!sel) return;
    const v = String(n > 0 ? n : 800);
    let found = false;
    Array.from(sel.options).forEach(function (o) {
        if (o.value === v) found = true;
    });
    if (!found) {
        const o = document.createElement('option');
        o.value = v;
        o.textContent = v;
        sel.appendChild(o);
    }
    sel.value = v;
    document.querySelectorAll('.xz-conv-gate-chip').forEach(function (btn) {
        btn.classList.toggle('is-on', btn.getAttribute('data-rms') === v);
    });
}

function xzConvPickGateRMS(n) {
    applyGateRMS(n);
    // Preview only until Apply — keep gate tab selected.
    if (xzConvPreviewMode !== 'gate') xzConvSelectTab('gate');
    else xzConvRenderDetail('gate');
}

function xzConvSyncHiddenRadios(mode) {
    document.querySelectorAll('input[name="xzConvMode"]').forEach(function (el) {
        el.checked = el.value === mode;
    });
}

function xzConvFillPoints(el, lines) {
    if (!el) return;
    el.innerHTML = '';
    (lines || []).forEach(function (t) {
        if (!t) return;
        const li = document.createElement('li');
        li.textContent = t;
        el.appendChild(li);
    });
}

function xzConvRenderDetail(mode) {
    mode = xzNormalizeConvMode(mode);
    const title = document.getElementById('xzConvDetailTitle');
    const flow = document.getElementById('xzConvDetailFlow');
    const tech = document.getElementById('xzConvDetailTech');
    const extra = document.getElementById('xzConvDetailExtra');
    const gateOpts = document.getElementById('xzConvGateOpts');
    const hint = document.getElementById('xzConvApplyHint');
    const btn = document.getElementById('xzConvApplyBtn');
    const copy = xzConvDetailCopy(mode);
    if (title) title.textContent = convModeLabel(mode);
    xzConvFillPoints(flow, copy.flow);
    xzConvFillPoints(tech, copy.tech);
    if (extra) {
        extra.textContent = copy.extra || '';
        extra.style.display = copy.extra ? 'block' : 'none';
    }
    if (gateOpts) gateOpts.style.display = mode === 'gate' ? 'flex' : 'none';
    const dirty = mode !== xzConvAppliedMode;
    if (btn) {
        btn.disabled = !dirty;
        btn.textContent = dirty
            ? xzT('conv.apply', 'Áp dụng')
            : xzT('conv.applied', 'Đang dùng');
    }
    if (hint) {
        hint.textContent = dirty
            ? xzT('conv.apply_hint', 'Chưa áp dụng — bấm Áp dụng để đổi trên robot.')
            : xzT('conv.applied_hint', 'Đây là chế độ đang chạy trên robot.');
    }
}

function xzConvPaintTabs() {
    document.querySelectorAll('.xz-conv-tab').forEach(function (btn) {
        const mode = btn.getAttribute('data-mode');
        btn.classList.toggle('is-preview', mode === xzConvPreviewMode);
        btn.classList.toggle('is-applied', mode === xzConvAppliedMode);
        btn.setAttribute('aria-selected', mode === xzConvPreviewMode ? 'true' : 'false');
    });
}

function xzConvSelectTab(mode) {
    xzConvPreviewMode = xzNormalizeConvMode(mode);
    xzConvSyncHiddenRadios(xzConvPreviewMode);
    xzConvPaintTabs();
    xzConvRenderDetail(xzConvPreviewMode);
}

function xzConvApplySelected() {
    const mode = xzNormalizeConvMode(xzConvPreviewMode);
    const rms = mode === 'gate' ? (document.getElementById('xzGateRMS') || {}).value : undefined;
    setConversationMode(mode, rms);
}

function applyConversationRadios(cfg) {
    const mode = xzNormalizeConvMode(cfg && cfg.conversation_mode);
    xzConvAppliedMode = mode;
    xzConvPreviewMode = mode;
    xzConvSyncHiddenRadios(mode);
    applyGateRMS(cfg && cfg.gate_rms);
    xzConvPaintTabs();
    xzConvRenderDetail(mode);
    let msg = xzT('conv.using', 'Đang dùng: {name}.').replace('{name}', convModeLabel(mode));
    if (cfg && cfg.enabled === false) msg += ' ' + xzT('conv.off', '(AI giao tiếp đang tắt — chế độ này chưa có tác dụng.)');
    convModeStatus(msg);
}

async function loadConversationMode() {
    try {
        const r = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (!r.ok) throw new Error('HTTP ' + r.status);
        applyConversationRadios(await r.json());
    } catch (e) {
        convModeStatus(xzT('conv.load_err', 'Lỗi đọc cách hội thoại:') + ' ' + (e.message || e));
    }
}

function setGateRMS(rms) {
    xzConvSelectTab('gate');
    setConversationMode('gate', rms);
}

async function setConversationMode(mode, rms) {
    mode = xzNormalizeConvMode(mode);
    convModeStatus(xzT('conv.saving', 'Đang lưu…'));
    try {
        const params = { mode: mode };
        if (rms != null && rms !== '') params.rms = String(rms);
        else if (mode === 'gate') {
            const sel = document.getElementById('xzGateRMS');
            if (sel && sel.value) params.rms = sel.value;
        }
        const r = await fetch('/api/mods/Xiaozhi/set_conversation_mode?' + new URLSearchParams(params).toString(), { method: 'POST' });
        const j = await r.json().catch(function () { return {}; });
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        applyConversationRadios(j.config || { conversation_mode: mode, enabled: true });
    } catch (e) {
        convModeStatus(xzT('conv.save_err', 'Lỗi lưu cách hội thoại:') + ' ' + (e.message || e));
        await loadConversationMode();
    }
}

function xzSelectedConvMode() {
    return xzNormalizeConvMode(xzConvPreviewMode || xzConvAppliedMode);
}

function xzApplyCfg(cfg) {
    if (!cfg) return;
    xzRenderGain(cfg.tts_gain_pct);
    const ota = document.getElementById('xzOTABaseURL');
    const ep = document.getElementById('xzEndpoint');
    const did = document.getElementById('xzDeviceID');
    const cid = document.getElementById('xzClientID');
    if (ota) ota.value = (cfg.ota_base_url && cfg.ota_base_url.trim()) ? cfg.ota_base_url : XZ_DEFAULT_OTA;
    if (ep) ep.value = cfg.endpoint || '';
    if (did) did.value = cfg.device_id || '';
    if (cid) cid.value = cfg.client_id || '';
    applyConversationRadios(cfg);

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

function xzUpdatePresetApplyHint() {
    const hint = document.getElementById('xzPresetApplyHint');
    const btn = document.getElementById('xzPresetApplyBtn');
    const preset = xzSelectedPreset();
    const applied = (xzLastCfg && xzLastCfg.identity_mode) || 'vi_pool';
    const same = xzAppliedMode === 'xiaozhi' && preset === applied;
    if (btn) btn.disabled = !!xzModeBusy || !!xzSaveBusy;
    if (hint) {
        if (same) {
            hint.textContent = xzT('xz.preset_apply_current', 'Đây là cấu hình Xiaozhi đang chạy trên robot.');
        } else if (preset === 'course') {
            hint.textContent = xzT('xz.preset_apply_course', 'Chọn khóa / giọng / bài bên dưới rồi Áp dụng (hoặc Lưu & áp dụng).');
        } else if (preset === 'custom') {
            hint.textContent = xzT('xz.preset_apply_custom', 'Điền MAC / kích hoạt bên dưới rồi Áp dụng (hoặc Lưu & kích hoạt).');
        } else {
            hint.textContent = xzT('xz.preset_apply_hint', 'Chọn cấu hình để đọc — bấm Áp dụng khi muốn dùng.');
        }
    }
}

async function xzOnPresetRadio() {
    // Preview only — reading options must not switch the robot.
    const preset = xzSelectedPreset();
    xzShowPresetUI(preset);
    xzUpdatePresetApplyHint();
    if (preset === 'course' && !(xzLastCfg && xzLastCfg.identity_mode === 'course')) {
        xzSetElStatus('xzCourseStatus', xzT('xz.course_pick', 'Chọn khóa học, giọng, bài rồi bấm Lưu & áp dụng.'), false);
    }
}

async function xzApplyPreset(preset, ev) {
    if (ev) {
        ev.preventDefault();
        ev.stopPropagation();
    }
    if (preset === 'vi_pool' || preset === 'course' || preset === 'custom') {
        const radios = { vi_pool: 'xzPresetVi', course: 'xzPresetCourse', custom: 'xzPresetCustom' };
        const r = document.getElementById(radios[preset]);
        if (r) r.checked = true;
        xzShowPresetUI(preset);
    } else {
        preset = xzSelectedPreset();
    }
    xzUpdatePresetApplyHint();

    if (preset === 'course') {
        if (typeof xzSaveCourse === 'function') await xzSaveCourse();
        return;
    }
    if (preset === 'custom') {
        if (typeof xzSaveAndActivate === 'function') await xzSaveAndActivate();
        return;
    }
    // vi_pool — force apply even if already on Xiaozhi with another identity.
    await xzSetListenMode('xiaozhi', { force: true });
}

function xzApplySelectedPreset() {
    xzApplyPreset(xzSelectedPreset());
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

async function xzSetListenMode(mode, opts) {
    if (xzModeBusy) return;
    const force = !!(opts && opts.force);
    const wantOAI = xzIsDirectMode(mode);
    const wantXz = (mode === 'xiaozhi') || wantOAI;
    if (!force && xzAppliedMode === mode) {
        xzShowConfigForMode(mode);
        xzSetModeStatus(xzT('xz.already_on', 'Đang dùng chế độ này rồi.'), false);
        return;
    }
    xzModeBusy = true;
    xzUpdateModeApplyHint();
    xzShowConfigForMode(mode);
    xzSetModeStatus(mode === 'pipeline'
        ? xzT('xz.switching_pl', 'Đang bật Tự chọn AI…')
        : mode === 'gemini'
        ? xzT('xz.switching_gemini', 'Đang bật Gemini Live…')
        : mode === 'grok'
        ? xzT('xz.switching_grok', 'Đang bật Grok Voice…')
        : mode === 'qwen'
        ? xzT('xz.switching_qwen', 'Đang bật Qwen Live…')
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
                : mode === 'qwen'
                ? xzT('xz.on_qwen', 'Đã chuyển sang Qwen Live — robot kết nối lại và sẽ tự chào (cần key DashScope).')
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
        xzUpdateModeApplyHint();
        xzUpdatePresetApplyHint();
    } catch (e) {
        xzSetModeStatus('Lỗi đổi chế độ: ' + e.message, true);
        await xzLoad();
    } finally {
        xzModeBusy = false;
        xzUpdateModeApplyHint();
    }
}

function xzOnModeRadio() {
    // Radios are driven by cards; Apply commits.
    xzShowConfigForMode(xzSelectedMode());
}

// Card click = preview only (read info / show config). Apply commits.
function xzPickMode(mode) {
    if (xzModeBusy || !XZ_MODE_RADIOS[mode]) return;
    xzShowConfigForMode(mode);
}

function xzApplyModeFromCard(mode, ev) {
    if (ev) {
        ev.preventDefault();
        ev.stopPropagation();
    }
    if (xzModeBusy || !XZ_MODE_RADIOS[mode]) return;
    xzShowConfigForMode(mode);
    xzSetListenMode(mode);
}

function xzApplySelectedMode() {
    const mode = xzPreviewMode || xzSelectedMode();
    if (!XZ_MODE_RADIOS[mode]) return;
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
