function activateSection(target) {
    if (!target) return;
    if (target === '#chess') target = '#games'; // legacy
    const prev = document.querySelector('.tab-content.active');
    const prevId = prev ? '#' + prev.id : '';
    const tabs = document.querySelectorAll('.tabs button');
    tabs.forEach((b) => b.classList.toggle('active', b.dataset.target === target));
    document.querySelectorAll('.tab-content').forEach((c) => c.classList.remove('active'));
    const panel = document.querySelector(target);
    if (panel) panel.classList.add('active');
    const sel = document.getElementById('navSelect');
    if (sel && sel.value !== target) sel.value = target;
    syncDrawerActive(target);
    try {
        history.replaceState(null, '', target);
    } catch (_) {}
    if (prevId === '#logs' && target !== '#logs' && typeof logsOnHide === 'function') {
        logsOnHide();
    }
    if (target === '#logs' && typeof logsOnShow === 'function') {
        logsOnShow();
    }
    if (prevId === '#freetime' && target !== '#freetime' && typeof ftOnHide === 'function') {
        ftOnHide();
    }
    if (target === '#freetime' && typeof ftOnShow === 'function') {
        ftOnShow();
    }
    if (target === '#wifi' && typeof wifiRefreshStatus === 'function') {
        wifiRefreshStatus().then(() => {
            if (typeof wifiCanJoin !== 'undefined' && wifiCanJoin && typeof wifiScan === 'function') {
                wifiScan();
            }
        });
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

/* Mobile left drawer — items cloned from .tabs (same targets / labels / i18n) */
const navBurger = document.getElementById('navBurger');
const navDrawer = document.getElementById('navDrawer');
const navBackdrop = document.getElementById('navBackdrop');
const navDrawerClose = document.getElementById('navDrawerClose');
const navDrawerList = document.getElementById('navDrawerList');

function isMobileNav() {
    return window.matchMedia('(max-width: 768px)').matches;
}

function syncDrawerActive(target) {
    if (!navDrawerList) return;
    navDrawerList.querySelectorAll('button').forEach((b) => {
        b.classList.toggle('active', b.dataset.target === target);
    });
}

function buildNavDrawer() {
    if (!navDrawerList) return;
    navDrawerList.innerHTML = '';
    document.querySelectorAll('.tabs button').forEach((src) => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.dataset.target = src.dataset.target;
        const i18nKey = src.getAttribute('data-i18n');
        if (i18nKey) btn.setAttribute('data-i18n', i18nKey);
        btn.textContent = src.textContent.trim();
        btn.addEventListener('click', () => {
            activateSection(btn.dataset.target);
            closeNavDrawer();
        });
        navDrawerList.appendChild(btn);
    });
    const active = document.querySelector('.tabs button.active');
    syncDrawerActive(active ? active.dataset.target : '#botsettings');
}

function openNavDrawer() {
    if (!navDrawer || !isMobileNav()) return;
    navDrawer.classList.add('is-open');
    navDrawer.setAttribute('aria-hidden', 'false');
    if (navBackdrop) {
        navBackdrop.hidden = false;
        // force reflow for fade
        void navBackdrop.offsetWidth;
        navBackdrop.classList.add('is-open');
    }
    if (navBurger) navBurger.setAttribute('aria-expanded', 'true');
    document.body.classList.add('nav-drawer-open');
}

function closeNavDrawer() {
    if (!navDrawer) return;
    navDrawer.classList.remove('is-open');
    navDrawer.setAttribute('aria-hidden', 'true');
    if (navBackdrop) {
        navBackdrop.classList.remove('is-open');
        window.setTimeout(() => {
            if (!navBackdrop.classList.contains('is-open')) navBackdrop.hidden = true;
        }, 220);
    }
    if (navBurger) navBurger.setAttribute('aria-expanded', 'false');
    document.body.classList.remove('nav-drawer-open');
}

function toggleNavDrawer() {
    if (navDrawer && navDrawer.classList.contains('is-open')) closeNavDrawer();
    else openNavDrawer();
}

buildNavDrawer();
if (navBurger) navBurger.addEventListener('click', toggleNavDrawer);
if (navDrawerClose) navDrawerClose.addEventListener('click', closeNavDrawer);
if (navBackdrop) navBackdrop.addEventListener('click', closeNavDrawer);
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeNavDrawer();
});
window.addEventListener('resize', () => {
    if (!isMobileNav()) closeNavDrawer();
});

function refreshDrawerLabels() {
    if (!navDrawerList) return;
    const srcs = document.querySelectorAll('.tabs button');
    const btns = navDrawerList.querySelectorAll('button');
    if (btns.length !== srcs.length) {
        buildNavDrawer();
        return;
    }
    btns.forEach((b, i) => {
        if (srcs[i]) b.textContent = srcs[i].textContent.trim();
    });
}

document.addEventListener('DOMContentLoaded', () => {
    window.setTimeout(refreshDrawerLabels, 0);
});

if (window.WireOSI18n && typeof window.WireOSI18n.setLang === 'function') {
    const _setLang = window.WireOSI18n.setLang.bind(window.WireOSI18n);
    window.WireOSI18n.setLang = async function () {
        const r = await _setLang.apply(null, arguments);
        refreshDrawerLabels();
        return r;
    };
}

// Legacy hashes → bot settings child sections
const legacyBotHash = {
    '#mainmods': 'bot-perf',
    '#cww': 'bot-cww',
    '#sensitivity': 'bot-sens',
    '#alexa': 'bot-alexa',
};

// Deep-link / restore hash
if (legacyBotHash[location.hash]) {
    activateSection('#botsettings');
    const sec = legacyBotHash[location.hash];
    setTimeout(() => {
        if (typeof showBotSection === 'function') showBotSection(sec);
    }, 0);
} else if (location.hash === '#chess') {
    // Legacy bookmark → games lobby
    activateSection('#games');
} else if (location.hash && document.querySelector(location.hash)) {
    activateSection(location.hash);
} else {
    const wifiPath = (location.pathname || '').replace(/\/+$/, '');
    if (wifiPath === '/wifi' || location.hostname === '10.3.141.1') {
        activateSection('#wifi');
    } else {
        activateSection('#botsettings');
        fetch('/api/mods/WifiSetup/status').then(r => r.json()).then(j => {
            if (j && j.ap) activateSection('#wifi');
        }).catch(() => {});
    }
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