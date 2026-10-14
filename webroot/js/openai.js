// Direct-API settings (AI giao tiếp → Tự chọn AI | OpenAI Realtime | Gemini
// Live | Grok). All live in openai.json and share one card; oaiLayout picks
// the fields: 'pipeline' | 'openai' (Realtime) | 'gemini' | 'grok'.

let oaiBusy = false;
let oaiLayout = 'openai';

function oaiT(key, fallback) {
    return (typeof t === 'function') ? t(key, fallback) : fallback;
}

function oaiSetStatus(msg, isError) {
    const el = document.getElementById('oaiStatus');
    if (!el) return;
    el.textContent = msg || '';
    el.style.whiteSpace = 'pre-line';
    el.style.color = isError ? '#f87171' : '#4ade80';
    el.style.display = msg ? 'block' : 'none';
}

function oaiVal(id) {
    const el = document.getElementById(id);
    return el ? el.value.trim() : '';
}

function oaiSetVal(id, v) {
    const el = document.getElementById(id);
    if (el) el.value = (v === undefined || v === null) ? '' : v;
}

function oaiSelectedMode() {
    return oaiLayout === 'pipeline' ? 'pipeline' : 'realtime';
}

const OAI_TTS_RADIOS = { edge: 'oaiTtsEdge', google: 'oaiTtsGoogle', gemini: 'oaiTtsGemini', openai: 'oaiTtsOpenAI' };

function oaiSelectedTts() {
    for (const e of Object.keys(OAI_TTS_RADIOS)) {
        const r = document.getElementById(OAI_TTS_RADIOS[e]);
        if (r && r.checked) return e;
    }
    return 'openai';
}

function oaiSetTts(engine) {
    if (!OAI_TTS_RADIOS[engine]) engine = 'openai';
    Object.keys(OAI_TTS_RADIOS).forEach(function (e) {
        const r = document.getElementById(OAI_TTS_RADIOS[e]);
        if (r) r.checked = (e === engine);
    });
    oaiShowTtsFields();
}

function oaiShow(id, on) {
    const el = document.getElementById(id);
    if (el) el.style.display = on ? '' : 'none';
}

function oaiShowTtsFields() {
    oaiRefresh();
}

const OAI_GROQ_BASE = 'https://api.groq.com/openai/v1';
const OAI_GEMINI_BASE = 'https://generativelanguage.googleapis.com/v1beta/openai';

// Pipeline stages: each picks a provider; the base URL input is only
// visible for "custom", the others fill it in.
const OAI_STAGES = {
    stt: {
        box: 'oaiSttStage', base: 'oaiSttBase', model: 'oaiSttModel', key: 'oaiSttKey', keyBox: 'oaiSttKeyBox',
        radios: { groq: 'oaiSttProvGroq', openai: 'oaiSttProvOpenAI', custom: 'oaiSttProvCustom' },
        models: { groq: 'whisper-large-v3-turbo', openai: 'gpt-4o-mini-transcribe' },
    },
    llm: {
        box: 'oaiLlmStage', base: 'oaiLlmBase', model: 'oaiLlmModel', key: 'oaiLlmKey', keyBox: 'oaiLlmKeyBox',
        radios: { gemini: 'oaiLlmProvGemini', groq: 'oaiLlmProvGroq', openai: 'oaiLlmProvOpenAI', custom: 'oaiLlmProvCustom' },
        models: { gemini: 'gemini-3.5-flash-lite', groq: 'openai/gpt-oss-120b', openai: 'gpt-4o-mini' },
    },
};
const OAI_PROV_BASE = { groq: OAI_GROQ_BASE, gemini: OAI_GEMINI_BASE, openai: '' };
const oaiCustomBase = { stt: '', llm: '' };
let oaiLastCfg = null;

// One saved key per provider: a step shows the key of the provider it uses,
// so switching provider brings that provider's key along. Custom servers keep
// a key per step.
const OAI_PROV_KEY = {
    groq: { param: 'groq_api_key', has: 'has_groq_key', hint: 'groq_key_hint' },
    gemini: { param: 'gemini_api_key', has: 'has_gemini_key', hint: 'gemini_key_hint' },
};
const OAI_CUSTOM_KEY = {
    stt: { param: 'stt_api_key', has: 'has_stt_key', hint: 'stt_key_hint' },
    llm: { param: 'llm_api_key', has: 'has_llm_key', hint: 'llm_key_hint' },
};
const oaiCurProv = { stt: '', llm: '' };
// Last model per provider, so switching away and back restores it.
const oaiProvModel = { stt: {}, llm: {} };

function oaiStageKeySpec(stage, prov) {
    return prov === 'custom' ? OAI_CUSTOM_KEY[stage] : (OAI_PROV_KEY[prov] || null);
}

// Step 2 on Groq while step 1 is on Groq: the key is entered once, in step 1.
function oaiLlmKeyShared() {
    return oaiSelectedProv('llm') === 'groq' && oaiSelectedProv('stt') === 'groq';
}

function oaiGeminiKeyShared() {
    return oaiSelectedProv('llm') === 'gemini';
}

function oaiProvFromBase(stage, base) {
    const b = (base || '').toLowerCase();
    if (!b || b.indexOf('api.openai.com') >= 0) return 'openai';
    if (b.indexOf('api.groq.com') >= 0) return 'groq';
    if (stage === 'llm' && b.indexOf('generativelanguage.googleapis.com') >= 0) return 'gemini';
    return 'custom';
}

function oaiSelectedProv(stage) {
    const radios = OAI_STAGES[stage].radios;
    for (const p of Object.keys(radios)) {
        const r = document.getElementById(radios[p]);
        if (r && r.checked) return p;
    }
    return 'openai';
}

function oaiCheckProv(stage, prov) {
    const radios = OAI_STAGES[stage].radios;
    Object.keys(radios).forEach(function (p) {
        const r = document.getElementById(radios[p]);
        if (r) r.checked = (p === prov);
    });
}

function oaiSetModelPlaceholder(stage, prov) {
    const el = document.getElementById(OAI_STAGES[stage].model);
    if (el) el.placeholder = OAI_STAGES[stage].models[prov] || '';
}

// oaiSetProv is a user choice: fill in the server address and the model for
// that provider. The key field is emptied so a key typed for the previous
// provider is never sent to the new one; the saved key shows in the hint.
function oaiSetProv(stage, prov) {
    const s = OAI_STAGES[stage];
    const prev = oaiCurProv[stage];
    oaiCheckProv(stage, prov);
    if (prev !== prov) {
        if (prev) oaiProvModel[stage][prev] = oaiVal(s.model);
        if (prev === 'custom') oaiCustomBase[stage] = oaiVal(s.base);
        oaiCurProv[stage] = prov;
        oaiSetVal(s.base, prov === 'custom' ? oaiCustomBase[stage] : OAI_PROV_BASE[prov]);
        oaiSetVal(s.model, oaiProvModel[stage][prov] || s.models[prov] || '');
        oaiSetVal(s.key, '');
        oaiSetModelPlaceholder(stage, prov);
    }
    oaiRefresh();
}

function oaiOnProv(stage) {
    oaiSetProv(stage, oaiSelectedProv(stage));
    if (oaiSelectedProv(stage) === 'custom') {
        const el = document.getElementById(OAI_STAGES[stage].base);
        if (el && !el.value) el.focus();
    }
}

function oaiShowProvFields(stage) {
    const box = document.getElementById(OAI_STAGES[stage].box);
    if (!box) return;
    const prov = oaiSelectedProv(stage);
    box.querySelectorAll('[data-prov]').forEach(function (el) {
        el.style.display = el.getAttribute('data-prov').split(' ').indexOf(prov) >= 0 ? '' : 'none';
    });
}

function oaiKeyNeeded() {
    return [oaiT('oai.key_need', '⚠ Chưa có key — dán key vào ô trên rồi bấm Lưu.'), true];
}

function oaiStageKeyHint(stage, c) {
    const prov = oaiSelectedProv(stage);
    const spec = oaiStageKeySpec(stage, prov);
    if (!spec) return ['', false];
    if (c[spec.has]) return [oaiKeyHint(true, c[spec.hint]), false];
    if (prov !== 'custom') return oaiKeyNeeded();
    // A custom server may not need a key (Ollama on the LAN).
    const savedCustom = oaiProvFromBase(stage, stage === 'stt' ? c.stt_base_url : c.llm_base_url) === 'custom';
    if (savedCustom && Array.isArray(c.missing_keys) && c.missing_keys.indexOf(stage) >= 0) {
        return [oaiT('oai.missing_key', '⚠ Chưa có key — máy chủ này cần key, thiếu thì robot sẽ không trả lời.'), true];
    }
    return ['', false];
}

function oaiRefreshHints() {
    const c = oaiLastCfg;
    if (!c) return;
    const pl = oaiSelectedMode() === 'pipeline';
    const kh = document.getElementById('oaiKeyHint');
    if (kh) kh.textContent = oaiKeyHint(c.has_key, c.key_hint);

    [['stt', 'oaiSttKeyHint'], ['llm', 'oaiLlmKeyHint']].forEach(function (p) {
        const el = document.getElementById(p[1]);
        if (!el) return;
        const h = pl ? oaiStageKeyHint(p[0], c) : ['', false];
        el.textContent = h[0];
        el.style.color = h[1] ? '#fbbf24' : '';
    });

    const gk = document.getElementById('oaiGeminiKeyHint');
    if (gk) {
        const h = c.has_gemini_key ? [oaiKeyHint(true, c.gemini_key_hint), false] : oaiKeyNeeded();
        gk.textContent = h[0];
        gk.style.color = h[1] ? '#fbbf24' : '';
    }

    const gh = document.getElementById('glKeyHint');
    if (gh) {
        gh.textContent = c.has_gemini_key
            ? oaiKeyHint(true, c.gemini_key_hint)
            : oaiT('gl.key_missing', '⚠ Chưa có key Gemini — robot sẽ không trả lời.');
        gh.style.color = c.has_gemini_key ? '' : '#fbbf24';
    }

    const xh = document.getElementById('gkKeyHint');
    if (xh) {
        xh.textContent = c.has_xai_key
            ? oaiKeyHint(true, c.xai_key_hint)
            : oaiT('gk.key_missing', '⚠ Chưa có key xAI — robot sẽ không trả lời.');
        xh.style.color = c.has_xai_key ? '' : '#fbbf24';
    }
}

// oaiRefresh shows only what the current choices need.
function oaiRefresh() {
    const gl = oaiLayout === 'gemini';
    const gk = oaiLayout === 'grok';
    const pl = oaiLayout === 'pipeline';
    const rt = oaiLayout === 'openai';
    const tts = oaiSelectedTts();
    const stt = oaiSelectedProv('stt');
    const llm = oaiSelectedProv('llm');

    oaiShow('oaiTitle', rt);
    oaiShow('oaiPlTitle', pl);
    oaiShow('oaiHelp', rt);
    oaiShow('oaiPlHelp', pl);
    oaiShow('glTitle', gl);
    oaiShow('glHelp', gl);
    oaiShow('glFields', gl);
    oaiShow('gkTitle', gk);
    oaiShow('gkHelp', gk);
    oaiShow('gkFields', gk);
    oaiShow('oaiRtFields', rt);
    oaiShow('oaiPlFields', pl);

    const usedBy = [];
    if (pl) {
        if (stt === 'openai') usedBy.push(oaiT('oai.stage_stt', 'Bước 1 · Nghe'));
        if (llm === 'openai') usedBy.push(oaiT('oai.stage_llm', 'Bước 2 · Suy nghĩ'));
        if (tts === 'openai') usedBy.push(oaiT('oai.stage_tts', 'Bước 3 · Nói'));
    }
    const needOpenAI = rt || usedBy.length > 0;
    oaiShow('oaiKeyRow', needOpenAI);
    oaiShow('oaiBaseURLRow', needOpenAI);
    oaiShow('oaiVisionRow', needOpenAI);
    const ub = document.getElementById('oaiKeyUsedBy');
    if (ub) {
        ub.textContent = usedBy.length ? oaiT('oai.key_used_by', 'Cần cho:') + ' ' + usedBy.join(', ') : '';
        ub.style.display = usedBy.length ? '' : 'none';
    }

    const ttsBoxes = { oaiTtsEdgeFields: 'edge', oaiTtsGoogleFields: 'google', oaiTtsGeminiFields: 'gemini', oaiTtsOpenAIFields: 'openai' };
    Object.keys(ttsBoxes).forEach(function (id) { oaiShow(id, tts === ttsBoxes[id]); });

    const voiceRow = document.getElementById('oaiVoiceRow');
    if (voiceRow) {
        const home = pl ? document.getElementById('oaiTtsOpenAIFields') : document.getElementById('oaiRtFields');
        if (home && voiceRow.nextElementSibling !== home && voiceRow.parentNode !== home) {
            if (pl) home.appendChild(voiceRow);
            else home.parentNode.insertBefore(voiceRow, home);
        }
        voiceRow.style.display = (rt || (pl && tts === 'openai')) ? '' : 'none';
    }

    oaiShowProvFields('stt');
    oaiShowProvFields('llm');
    const sharedGroq = oaiLlmKeyShared();
    if (sharedGroq) oaiShow('oaiLlmKeyBox', false);
    oaiShow('oaiLlmKeyShared', sharedGroq);
    const sharedGemini = oaiGeminiKeyShared();
    oaiShow('oaiGeminiKeyRow', !sharedGemini);
    oaiShow('oaiGeminiKeyShared', sharedGemini);
    oaiRefreshHints();
}

// oaiSetLayout shows the OpenAI card or the Gemini Live one (same form).
function oaiSetLayout(layout) {
    oaiLayout = ['pipeline', 'gemini', 'grok'].indexOf(layout) >= 0 ? layout : 'openai';
    oaiSetStatus('', false);
    oaiRefresh();
}

function oaiSelectValue(id, v) {
    const sel = document.getElementById(id);
    if (!sel || !v) return;
    if (!Array.from(sel.options).some(function (o) { return o.value === v; })) {
        const opt = document.createElement('option');
        opt.value = v;
        opt.textContent = v;
        sel.appendChild(opt);
    }
    sel.value = v;
}

// Presets only fill the form; nothing is saved until "Lưu & áp dụng".
function oaiPreset(name) {
    let focus = 'oaiSttKey';
    let msg = oaiT('oai.preset_free_msg', 'Đã chọn: Groq nghe, Gemini suy nghĩ, Edge nói. Dán key Groq ở Bước 1 và key Gemini ở Bước 2, rồi bấm Lưu.');
    if (name === 'openai') {
        oaiSetProv('stt', 'openai');
        oaiSetProv('llm', 'openai');
        oaiSetTts('openai');
        focus = 'oaiKey';
        msg = oaiT('oai.preset_openai_msg', 'Đã điền cấu hình OpenAI. Bấm Lưu để áp dụng.');
    } else {
        oaiSetProv('stt', 'groq');
        oaiSetProv('llm', 'gemini');
        oaiSetTts('edge');
    }
    oaiSetStatus(msg, false);
    const el = document.getElementById(focus);
    if (el) el.focus();
}

function oaiKeyHint(has, hint) {
    return has
        ? oaiT('oai.key_saved', 'Đã lưu key') + ' ' + (hint || '')
        : oaiT('oai.key_none', 'Chưa có key.');
}

function oaiApply(c) {
    if (!c) return;
    oaiSelectValue('oaiVoice', c.voice);
    oaiSelectValue('oaiEdgeVoice', c.edge_voice);
    oaiSelectValue('oaiGeminiVoice', c.gemini_voice);
    oaiSelectValue('glVoice', c.gemini_live_voice);
    oaiSetVal('glModel', c.gemini_live_model);
    oaiSelectValue('gkVoice', c.grok_voice);
    oaiSetVal('gkModel', c.grok_model);
    oaiSetVal('oaiRtModel', c.realtime_model);
    oaiSetVal('oaiSttBase', c.stt_base_url);
    oaiSetVal('oaiSttModel', c.stt_model);
    oaiSetVal('oaiLlmModel', c.llm_model);
    oaiSetVal('oaiLlmBase', c.llm_base_url);
    oaiSetVal('oaiTtsModel', c.tts_model);
    oaiSetVal('oaiGeminiModel', c.gemini_tts_model);
    oaiSetVal('oaiMemory', c.memory_turns);
    oaiSetVal('oaiInstructions', c.instructions);
    oaiSetVal('oaiBaseURL', c.base_url);
    oaiSetVal('oaiLanguage', c.language);
    oaiSetVal('oaiVisionModel', c.vision_model);
    ['oaiKey', 'oaiSttKey', 'oaiLlmKey', 'oaiGeminiKey', 'glKey', 'gkKey'].forEach(function (id) { oaiSetVal(id, ''); });
    oaiLastCfg = c;
    ['stt', 'llm'].forEach(function (stage) {
        const base = stage === 'stt' ? c.stt_base_url : c.llm_base_url;
        const prov = oaiProvFromBase(stage, base);
        oaiCustomBase[stage] = prov === 'custom' ? base : '';
        oaiCurProv[stage] = prov;
        oaiProvModel[stage] = {};
        oaiProvModel[stage][prov] = stage === 'stt' ? c.stt_model : c.llm_model;
        oaiCheckProv(stage, prov);
        oaiSetModelPlaceholder(stage, prov);
    });
    oaiSetTts(c.tts_engine);
}

function oaiMissingKeysText(c) {
    const missing = (c && Array.isArray(c.missing_keys)) ? c.missing_keys : [];
    if (!missing.length) return '';
    return oaiT('oai.missing_keys_save', '⚠ Còn thiếu key cho:') + ' ' +
        missing.map(function (s) { return oaiT('oai.check_' + s, s.toUpperCase()); }).join(', ') + '.';
}

async function oaiLoad() {
    try {
        const resp = await fetch('/api/mods/OpenAI/status');
        const j = await resp.json();
        if (j.status === 'success') oaiApply(j.config);
    } catch (e) {
        oaiSetStatus(oaiT('oai.err_load', 'Lỗi tải cấu hình OpenAI:') + ' ' + e.message, true);
    }
}

function oaiFormParams() {
    const p = new URLSearchParams();
    const put = function (id, param) {
        const v = oaiVal(id);
        if (v && param) p.set(param, v);
    };
    put('oaiKey', 'api_key');
    if (oaiLayout === 'pipeline') {
        put('oaiSttKey', (oaiStageKeySpec('stt', oaiSelectedProv('stt')) || {}).param);
        if (!oaiLlmKeyShared()) put('oaiLlmKey', (oaiStageKeySpec('llm', oaiSelectedProv('llm')) || {}).param);
        if (oaiSelectedTts() === 'gemini' && !oaiGeminiKeyShared()) put('oaiGeminiKey', 'gemini_api_key');
    }
    if (oaiLayout === 'gemini') put('glKey', 'gemini_api_key');
    if (oaiLayout === 'grok') put('gkKey', 'xai_api_key');
    p.set('gemini_live_model', oaiVal('glModel'));
    p.set('gemini_live_voice', oaiVal('glVoice'));
    p.set('grok_model', oaiVal('gkModel'));
    p.set('grok_voice', oaiVal('gkVoice'));
    // Gemini Live / Grok leave the OpenAI mode as it is.
    if (oaiLayout === 'openai' || oaiLayout === 'pipeline') p.set('mode', oaiSelectedMode());
    p.set('voice', oaiVal('oaiVoice'));
    p.set('realtime_model', oaiVal('oaiRtModel'));
    p.set('stt_base_url', oaiVal('oaiSttBase'));
    p.set('stt_model', oaiVal('oaiSttModel'));
    p.set('llm_model', oaiVal('oaiLlmModel'));
    p.set('llm_base_url', oaiVal('oaiLlmBase'));
    p.set('tts_model', oaiVal('oaiTtsModel'));
    p.set('tts_engine', oaiSelectedTts());
    p.set('edge_voice', oaiVal('oaiEdgeVoice'));
    p.set('gemini_voice', oaiVal('oaiGeminiVoice'));
    p.set('gemini_tts_model', oaiVal('oaiGeminiModel'));
    p.set('memory_turns', oaiVal('oaiMemory') || '10');
    const ins = document.getElementById('oaiInstructions');
    p.set('instructions', ins ? ins.value : '');
    p.set('base_url', oaiVal('oaiBaseURL'));
    p.set('language', oaiVal('oaiLanguage'));
    p.set('vision_model', oaiVal('oaiVisionModel'));
    return p;
}

async function oaiPost(endpoint, params) {
    const resp = await fetch('/api/mods/OpenAI/' + endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: params ? params.toString() : ''
    });
    return resp.json();
}

function oaiCustomURLMissing() {
    if (oaiLayout !== 'pipeline') return '';
    for (const stage of ['stt', 'llm']) {
        if (oaiSelectedProv(stage) === 'custom' && !oaiVal(OAI_STAGES[stage].base)) return OAI_STAGES[stage].base;
    }
    return '';
}

async function oaiSave() {
    if (oaiBusy) return;
    const noURL = oaiCustomURLMissing();
    if (noURL) {
        oaiSetStatus(oaiT('oai.err_custom_url', 'Bạn chọn "Máy chủ khác" — hãy nhập địa chỉ máy chủ trước khi lưu.'), true);
        document.getElementById(noURL).focus();
        return;
    }
    oaiBusy = true;
    oaiSetStatus(oaiT('oai.saving', 'Đang lưu…'), false);
    try {
        const j = await oaiPost('save', oaiFormParams());
        if (j.status !== 'success') {
            oaiSetStatus(oaiT('oai.err_save', 'Lỗi lưu:') + ' ' + (j.message || 'unknown'), true);
            return;
        }
        oaiApply(j.config);
        let msg = j.applied
            ? oaiT('oai.saved_live', 'Đã lưu — áp dụng từ lượt nói tiếp theo (không ngắt cuộc trò chuyện đang diễn ra).')
            : oaiT('oai.saved', 'Đã lưu.');
        if (oaiLayout === 'gemini') {
            const noKey = !j.config.has_gemini_live_key;
            if (noKey) msg += '\n' + oaiT('gl.key_missing', '⚠ Chưa có key Gemini — robot sẽ không trả lời.');
            oaiSetStatus(msg, noKey);
            return;
        }
        if (oaiLayout === 'grok') {
            const noKey = !j.config.has_xai_key;
            if (noKey) msg += '\n' + oaiT('gk.key_missing', '⚠ Chưa có key xAI — robot sẽ không trả lời.');
            oaiSetStatus(msg, noKey);
            return;
        }
        if (j.config.needs_openai_key) msg += ' ' + oaiT('oai.need_key', 'Nhớ nhập API key để dùng.');
        const missingText = oaiMissingKeysText(j.config);
        if (missingText) msg += '\n' + missingText;
        oaiSetStatus(msg, !!missingText);
    } catch (e) {
        oaiSetStatus(oaiT('oai.err_save', 'Lỗi lưu:') + ' ' + e.message, true);
    } finally {
        oaiBusy = false;
    }
}

async function oaiTest() {
    if (oaiBusy) return;
    oaiBusy = true;
    oaiSetStatus(oaiT('oai.testing', 'Đang kiểm tra…'), false);
    try {
        const params = oaiFormParams();
        if (oaiLayout === 'gemini' || oaiLayout === 'grok') params.set('target', oaiLayout);
        const j = await oaiPost('test', params);
        if (!Array.isArray(j.checks)) {
            oaiSetStatus(oaiT('oai.test_fail', 'Không kết nối được:') + ' ' + (j.message || 'unknown'), true);
            return;
        }
        let bad = false;
        const lines = j.checks.map(function (c) {
            const stage = oaiT('oai.check_' + c.stage, c.stage.toUpperCase());
            const where = c.host ? ' (' + c.host + ')' : '';
            if (!c.ok) {
                bad = true;
                return '✗ ' + stage + where + ': ' + (c.error || 'error');
            }
            if (c.free) {
                return '✓ ' + stage + ': ' + (c.error || oaiT('oai.check_free', 'miễn phí, không cần key') + ' (' + c.model + ')');
            }
            if (c.model && !c.model_found) {
                bad = true;
                return '⚠ ' + stage + where + ': ' + oaiT('oai.test_model_missing', 'Không thấy model trong danh sách:') + ' ' + c.model;
            }
            return '✓ ' + stage + where + ': ' + oaiT('oai.test_ok', 'Key dùng được') + ', ' + c.model;
        });
        oaiSetStatus(lines.join('\n'), bad);
    } catch (e) {
        oaiSetStatus(oaiT('oai.test_fail', 'Không kết nối được:') + ' ' + e.message, true);
    } finally {
        oaiBusy = false;
    }
}

async function oaiClearMemory() {
    if (!confirm(oaiT('oai.clear_confirm', 'Xóa toàn bộ trí nhớ hội thoại của robot?'))) return;
    try {
        const j = await oaiPost('clear_memory');
        oaiSetStatus(j.status === 'success'
            ? oaiT('oai.cleared', 'Đã xóa trí nhớ. Robot sẽ quên các cuộc trò chuyện trước (sau khi phiên hiện tại kết thúc).')
            : (oaiT('oai.err_save', 'Lỗi:') + ' ' + (j.message || 'unknown')), j.status !== 'success');
    } catch (e) {
        oaiSetStatus(e.message, true);
    }
}
