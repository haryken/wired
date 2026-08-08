/* Header battery — Option 3 horizontal segmented bar. Same /api/mods/Battery/get poll. */
(function () {
    'use strict';

    const root = document.getElementById('headerBattery');
    if (!root) return;

    const segs = root.querySelectorAll('.hbatt-seg');
    const pctEl = root.querySelector('.hbatt-pct');
    const tipEl = root.querySelector('.hbatt-tip');
    const SEGMENTS = segs.length || 6;
    const POLL_MS = 3000;

    function batteryTone(percent) {
        if (percent < 20) return 'level-low';
        if (percent < 60) return 'level-mid';
        return 'level-high';
    }

    function setClasses(tone, charging) {
        root.className = 'hbatt ' + tone + (charging ? ' is-charging' : '');
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

        // Keep existing WirePod-ish LOW cap when robot reports low + off dock.
        let level = typeof data.level === 'number' ? data.level : 2;
        const onCharger = !!data.on_charger;
        const charging = !!data.charging;
        const volts = typeof data.volts === 'number' ? data.volts : 0;
        if (level === 1 && !onCharger) {
            percent = Math.min(15, percent);
        }

        const filled = Math.max(0, Math.min(SEGMENTS, Math.round((percent / 100) * SEGMENTS)));
        // Show charging chrome when actively charging OR sitting on the dock.
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

    refresh();
    setInterval(refresh, POLL_MS);
})();
