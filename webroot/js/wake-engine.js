let wakeEngineLoading = false;
let wakeLocaleLoading = false;
let currentWakeEngine = 'thf';

function isWakeThf() {
    return currentWakeEngine === 'thf';
}

async function loadWakeEngine() {
    wakeEngineLoading = true;
    try {
        const res = await fetch('/api/mods/WakeEngine/get');
        const engine = (await res.text()).trim() || 'thf';
        currentWakeEngine = (engine === 'picovoice') ? 'picovoice' : 'thf';
        const pv = document.getElementById('wakeEnginePv');
        const thf = document.getElementById('wakeEngineThf');
        if (pv && thf) {
            pv.checked = currentWakeEngine === 'picovoice';
            thf.checked = currentWakeEngine === 'thf';
        }
        applyWakeEngineUi();
    } catch (e) {
        setWakeEngineStatus('Không đọc được engine: ' + e.message);
    } finally {
        wakeEngineLoading = false;
    }
}

function setWakeLocaleStatus(msg) {
    const el = document.getElementById('wakeLocaleStatus');
    if (el) el.innerHTML = `<p>${msg}</p>`;
}

async function loadWakeLocale() {
    wakeLocaleLoading = true;
    try {
        const res = await fetch('/api/mods/JdocSettings/getLocale');
        if (!res.ok) {
            setWakeLocaleStatus('Không đọc được locale');
            return;
        }
        const locale = (await res.text()).trim() || 'en-AU';
        let matched = false;
        document.querySelectorAll('input[name="wakeLocale"]').forEach((el) => {
            el.checked = el.value === locale;
            if (el.checked) matched = true;
        });
        if (!matched) {
            setWakeLocaleStatus('Model giọng hiện tại: ' + locale + ' (không có trong list)');
        } else {
            setWakeLocaleStatus('Model giọng hiện tại: ' + locale);
        }
    } catch (e) {
        setWakeLocaleStatus('Không đọc được locale: ' + e.message);
    } finally {
        wakeLocaleLoading = false;
    }
}

async function onWakeLocaleChange(locale) {
    if (wakeLocaleLoading) return;
    if (!isWakeThf()) {
        setWakeLocaleStatus('Locale chỉ dùng khi engine = THF');
        await loadWakeLocale();
        return;
    }
    setWakeLocaleStatus('Đang lưu ' + locale + '...');
    try {
        const res = await fetch('/api/mods/JdocSettings/setLocale?locale=' + encodeURIComponent(locale));
        if (!res.ok) {
            let msg = 'Lỗi lưu locale';
            try {
                const e = await res.json();
                if (e.message) msg = e.message;
            } catch (_) {}
            setWakeLocaleStatus(msg);
            await loadWakeLocale();
            return;
        }
        setWakeLocaleStatus('Đã lưu: ' + locale);
    } catch (e) {
        setWakeLocaleStatus('Lỗi: ' + e.message);
        await loadWakeLocale();
    }
}

function setNavDisabled(btn, disabled, title) {
    if (!btn) return;
    btn.classList.toggle('disabled', !!disabled);
    btn.setAttribute('aria-disabled', disabled ? 'true' : 'false');
    btn.title = title || '';
    if (disabled) {
        btn.dataset.wakeLocked = '1';
    } else {
        delete btn.dataset.wakeLocked;
    }
}

function setControlsDisabled(rootId, disabled) {
    const root = document.getElementById(rootId);
    if (!root) return;
    root.querySelectorAll('input, button, select, textarea').forEach((el) => {
        el.disabled = !!disabled;
    });
    root.style.opacity = disabled ? '0.45' : '';
    root.style.pointerEvents = disabled ? 'none' : '';
}

function selectedWakeEngineRadio() {
    const thf = document.getElementById('wakeEngineThf');
    return (thf && thf.checked) ? 'thf' : 'picovoice';
}

/** Nav locks / section gates follow the *saved* engine only (currentWakeEngine). */
function applyWakeEngineUi() {
    const useThf = isWakeThf();
    const thfHint = document.getElementById('wakeThfHint');
    if (thfHint) thfHint.style.display = useThf ? '' : 'none';

    const cwwNav = document.getElementById('navCww');
    const sensNav = document.getElementById('navSens');
    const localeNav = document.getElementById('navLocale');
    setNavDisabled(cwwNav, useThf, useThf ? 'Chỉ Picovoice' : '');
    setNavDisabled(sensNav, useThf, useThf ? 'Chỉ Picovoice' : '');
    setNavDisabled(localeNav, !useThf, !useThf ? 'Chỉ THF' : '');

    const cwwLock = document.getElementById('cwwEngineLock');
    const sensLock = document.getElementById('sensEngineLock');
    const localeLock = document.getElementById('localeEngineLock');
    if (cwwLock) cwwLock.style.display = useThf ? '' : 'none';
    if (sensLock) sensLock.style.display = useThf ? '' : 'none';
    if (localeLock) localeLock.style.display = useThf ? 'none' : '';

    setControlsDisabled('cwwControls', useThf);
    setControlsDisabled('sensControls', useThf);
    setControlsDisabled('localeControls', !useThf);

    // Close locked sections if currently open
    const closeIfOpen = (secId, navBtn) => {
        const sec = document.getElementById(secId);
        if (sec && sec.style.display !== 'none') {
            sec.style.display = 'none';
            if (navBtn) navBtn.classList.remove('active');
        }
    };
    if (useThf) {
        closeIfOpen('bot-cww', cwwNav);
        closeIfOpen('bot-sens', sensNav);
    } else {
        closeIfOpen('bot-locale', localeNav);
    }
}

/** Radio preview only — do not lock/unlock tiles until Save. */
function onWakeEngineRadio() {
    const pending = selectedWakeEngineRadio();
    if (pending !== currentWakeEngine) {
        setWakeEngineStatus('Đã chọn <b>' + pending + '</b> — bấm <b>Lưu engine + restart</b> để áp dụng. Tile phía trên vẫn theo engine hiện tại.');
    } else {
        setWakeEngineStatus('Engine hiện tại: <b>' + currentWakeEngine + '</b> (đã áp dụng).');
    }
}

function setWakeEngineStatus(msg) {
    const el = document.getElementById('wakeEngineStatus');
    if (el) el.innerHTML = msg ? `<p>${msg}</p>` : '';
}

async function saveWakeEngine() {
    const engine = selectedWakeEngineRadio();
    setWakeEngineStatus('Đang lưu ' + engine + '...');
    try {
        const res = await fetch('/api/mods/WakeEngine/set?engine=' + encodeURIComponent(engine));
        if (!res.ok) {
            setWakeEngineStatus('Lỗi lưu engine');
            await loadWakeEngine();
            return;
        }
        currentWakeEngine = engine;
        applyWakeEngineUi();
        setWakeEngineStatus('Đã lưu — đang restart anim...');
        await RestartVic();
        setWakeEngineStatus('Xong. Engine: ' + engine);
        await loadWakeEngine();
    } catch (e) {
        setWakeEngineStatus('Lỗi: ' + e.message);
        await loadWakeEngine();
    }
}

document.addEventListener('DOMContentLoaded', () => {
    loadWakeEngine();
    loadWakeLocale();
});
