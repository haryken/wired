function activateSection(target) {
    if (!target) return;
    const tabs = document.querySelectorAll('.tabs button');
    tabs.forEach((b) => b.classList.toggle('active', b.dataset.target === target));
    document.querySelectorAll('.tab-content').forEach((c) => c.classList.remove('active'));
    const panel = document.querySelector(target);
    if (panel) panel.classList.add('active');
    const sel = document.getElementById('navSelect');
    if (sel && sel.value !== target) sel.value = target;
    try {
        history.replaceState(null, '', target);
    } catch (_) {}
    if (target === '#alexa' && typeof alexaRefresh === 'function') {
        alexaRefresh();
    }
}

const tabs = document.querySelectorAll('.tabs button');
tabs.forEach((btn) => btn.addEventListener('click', () => {
    activateSection(btn.dataset.target);
}));

const navSelect = document.getElementById('navSelect');
if (navSelect) {
    navSelect.addEventListener('change', () => activateSection(navSelect.value));
}

// Legacy hashes → bot settings child sections
const legacyBotHash = {
    '#mainmods': 'bot-perf',
    '#cww': 'bot-cww',
    '#sensitivity': 'bot-sens',
};

// Deep-link / restore hash
if (legacyBotHash[location.hash]) {
    activateSection('#botsettings');
    const sec = legacyBotHash[location.hash];
    setTimeout(() => {
        if (typeof showBotSection === 'function') showBotSection(sec);
    }, 0);
} else if (location.hash && document.querySelector(location.hash)) {
    activateSection(location.hash);
} else {
    activateSection('#botsettings');
}

function hide(id) { document.getElementById(id).style.display = 'none'; }
function show(id) { document.getElementById(id).style.display = 'block'; }

function showVoiceMode(mode) {
    const xz = document.getElementById('voice-mode-xz');
    const vosk = document.getElementById('voice-mode-vosk');
    if (!xz || !vosk) return;
    const isXz = mode !== 'vosk';
    xz.style.display = isXz ? 'block' : 'none';
    vosk.style.display = isXz ? 'none' : 'block';
    document.querySelectorAll('.voice-mode-btn').forEach((b) => {
        b.classList.toggle('active', b.dataset.voiceMode === (isXz ? 'xz' : 'vosk'));
    });
}

async function GetCurrent(mod) {
    let res = await fetch(`/api/mods/${mod}/get`);
    return res.text();
}

function SetModStatus(msg) {
    const div = document.getElementById('modStatus');
    div.innerHTML = `<h3>${msg}</h3>`;
    div.style.display = msg ? 'block' : 'none';
}

function HideModStatus() { hide('modStatus'); }

async function UpdateAllMods() {
    hide('restartNeeded');
    hide('showDuringVicRestart');

    const data = await GetCurrent('FreqChange');
    document.getElementsByName('frequency')
        .forEach(rb => { if (rb.value == data) rb.checked = true; });
    checkAutoUpdateStatus();
    setSensitivity();
    getTimezone()
    getLocation()
    getTempUnits()
    getMasterVolume()
    getEyePreset()
    facesRefresh()
}

UpdateAllMods();