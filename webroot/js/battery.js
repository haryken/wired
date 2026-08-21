/* Header battery — poll only when bot setting "Hiện pin" is on (default off). */
(function () {
    'use strict';

    const STORAGE_KEY = 'wireos.showBatteryUi';
    const root = document.getElementById('headerBattery');
    if (!root) return;

    const segs = root.querySelectorAll('.hbatt-seg');
    const pctEl = root.querySelector('.hbatt-pct');
    const tipEl = root.querySelector('.hbatt-tip');
    const SEGMENTS = segs.length || 6;
    const POLL_MS = 3000;
    let timer = null;

    function isEnabled() {
        try {
            return localStorage.getItem(STORAGE_KEY) === '1';
        } catch (_) {
            return false;
        }
    }

    function setEnabled(on) {
        try {
            localStorage.setItem(STORAGE_KEY, on ? '1' : '0');
        } catch (_) {}
        applyEnabled(on);
        syncBotRadios(on);
        const st = document.getElementById('batteryUiStatus');
        if (st) {
            st.style.display = 'block';
            st.textContent = on
                ? (typeof t === 'function' ? t('batt.enabled', 'Đã bật hiện pin.') : 'Đã bật hiện pin.')
                : (typeof t === 'function' ? t('batt.disabled', 'Đã tắt hiện pin.') : 'Đã tắt hiện pin.');
        }
    }

    function syncBotRadios(on) {
        document.querySelectorAll('input[name="showBatteryUi"]').forEach((el) => {
            el.checked = (el.value === 'on') === !!on;
        });
    }

    function batteryTone(percent) {
        if (percent < 20) return 'level-low';
        if (percent < 60) return 'level-mid';
        return 'level-high';
    }

    function setClasses(tone, charging) {
        const wasHidden = root.hidden;
        root.className = 'hbatt ' + tone + (charging ? ' is-charging' : '');
        root.hidden = wasHidden;
    }

    function paintSegments(filled) {
        for (let i = 0; i < segs.length; i++) {
            if (i < filled) segs[i].classList.add('on');
            else segs[i].classList.remove('on');
        }
    }

    function showUnknown(msg) {
        setClasses('level-unk', false);
        paintSegments(0);
        pctEl.textContent = '--%';
        tipEl.innerHTML = '<b>Pin</b><br/>??%<br/> (' + (msg || 'Unable to connect') + ')';
        root.title = tipEl.textContent.replace(/\n/g, ' ');
    }

    async function refresh() {
        if (!isEnabled()) return;
        let data;
        try {
            const r = await fetch('/api/mods/Battery/get', { cache: 'no-store' });
            if (!r.ok) throw new Error('HTTP ' + r.status);
            data = await r.json();
        } catch (e) {
            showUnknown('Unable to connect');
            return;
        }

        let percent = typeof data.percent === 'number' ? data.percent : null;
        if (percent === null) {
            showUnknown('no percent');
            return;
        }

        let level = typeof data.level === 'number' ? data.level : 2;
        const onCharger = !!data.on_charger;
        const charging = !!data.charging;
        const volts = typeof data.volts === 'number' ? data.volts : 0;
        if (level === 1 && !onCharger) {
            percent = Math.min(15, percent);
        }

        const filled = Math.max(0, Math.min(SEGMENTS, Math.round((percent / 100) * SEGMENTS)));
        const showCharge = charging || onCharger;

        setClasses(batteryTone(percent), showCharge);
        paintSegments(filled);
        pctEl.textContent = percent + '%';

        tipEl.innerHTML =
            '<b>Vector</b><br/>' + percent + '%<br/> (' +
            (volts ? volts.toFixed(2) : '?') + 'V)' +
            '<br/>' + (data.summary || '');
        root.title = (data.summary || (percent + '%')).trim();
    }

    function stopPoll() {
        if (timer) {
            clearInterval(timer);
            timer = null;
        }
    }

    function startPoll() {
        stopPoll();
        refresh();
        timer = setInterval(refresh, POLL_MS);
    }

    function applyEnabled(on) {
        if (on) {
            root.hidden = false;
            startPoll();
        } else {
            root.hidden = true;
            stopPoll();
        }
    }

    window.setShowBatteryUi = setEnabled;
    window.isShowBatteryUi = isEnabled;
    window.syncBatteryUiRadios = function () {
        syncBotRadios(isEnabled());
    };

    applyEnabled(isEnabled());
    syncBotRadios(isEnabled());
})();
