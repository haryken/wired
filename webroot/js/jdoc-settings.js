function setJdocStatus(status) {
    const el = document.getElementById('jdocStatus');
    el.innerHTML = `<p>${status}</p>`;
}

async function setLocation() {
    const v = document.getElementById('location').value;
    setJdocStatus("Đang đặt vị trí... (Setting location...)")
    try {
        const res = await fetch(`/api/mods/JdocSettings/setLocation?location=${v}`);
        if (!res.ok) {
            const e = await res.json();
            setJdocStatus(`${e.status}: ${e.message}`);
        } else {
            getLocation()
            setJdocStatus('Đã lưu vị trí. (Location saved.)');
        }
    } catch (e) {
        setJdocStatus(`Lỗi mạng (network error): ${e.message}`);
    }
}

async function setTimezone() {
    const v = document.getElementById('timezone').value;
    setJdocStatus("Đang đặt múi giờ... (Setting timezone...)")
    try {
        const res = await fetch(`/api/mods/JdocSettings/setTimezone?timezone=${v}`);
        if (!res.ok) {
            const e = await res.json();
            setJdocStatus(`${e.status}: ${e.message}`);
        } else {
            getTimezone()
            setJdocStatus('Đã lưu múi giờ. (Timezone saved.)');
        }
    } catch (e) {
        setJdocStatus(`Lỗi mạng (network error): ${e.message}`);
    }
}

async function setTempUnits() {
    const v = document.getElementById('tUnits').value;
    setJdocStatus("Đang đặt đơn vị nhiệt... (Setting temp units...)")
    try {
        const res = await fetch(`/api/mods/JdocSettings/setFahrenheit?t=${v}`);
        if (!res.ok) {
            const e = await res.json();
            setJdocStatus(`${e.status}: ${e.message}`);
        } else {
            getTempUnits()
            setJdocStatus('Đã lưu đơn vị nhiệt. (Temp units saved.)');
        }
    } catch (e) {
        setJdocStatus(`Lỗi mạng (network error): ${e.message}`);
    }
}

async function getLocation() {
    try {
        const res = await fetch(`/api/mods/JdocSettings/getLocation`);
        if (!res.ok) {
            const e = await res.json();
            console.log(`${e.status}: ${e.message}`);
        } else {
            const e = await res.text();
            document.getElementById('location').value = e;
        }
    } catch (e) {
        console.log(`network error: ${e.message}`);
    }
}

async function getTimezone() {
    try {
        const res = await fetch(`/api/mods/JdocSettings/getTimezone`);
        if (!res.ok) {
            const e = await res.json();
            console.log(`${e.status}: ${e.message}`);
        } else {
            const e = await res.text();
            document.getElementById('timezone').value = e;
        }
    } catch (e) {
        console.log(`network error: ${e.message}`);
    }
}

async function getTempUnits() {
    try {
        const res = await fetch(`/api/mods/JdocSettings/getFahrenheit`);
        if (!res.ok) {
            const e = await res.json();
            console.log(`${e.status}: ${e.message}`);
        } else {
            const e = await res.text();
            document.getElementById('tUnits').value = e;
        }
    } catch (e) {
        console.log(`network error: ${e.message}`);
    }
}

async function getMasterVolume() {
    try {
        const res = await fetch(`/api/mods/JdocSettings/getVolume`);
        if (!res.ok) return;
        const v = (await res.text()).trim();
        document.querySelectorAll('input[name="vol"]').forEach((el) => {
            el.checked = el.value === v;
        });
    } catch (e) {
        console.log(`volume get: ${e.message}`);
    }
}

async function setMasterVolumeVal(level) {
    setJdocStatus('Đang đặt âm lượng... (Setting volume...)');
    try {
        const res = await fetch(`/api/mods/JdocSettings/setVolume?level=${level}`);
        if (!res.ok) {
            const e = await res.json();
            setJdocStatus(`${e.status}: ${e.message}`);
        } else {
            setJdocStatus('Đã cập nhật âm lượng. (Volume updated.)');
            getMasterVolume();
        }
    } catch (e) {
        setJdocStatus(`Lỗi mạng (network error): ${e.message}`);
    }
}

async function getEyePreset() {
    try {
        const res = await fetch(`/api/mods/JdocSettings/getEyeColor`);
        if (!res.ok) return;
        const j = await res.json();
        if (j.iscustom) {
            document.querySelectorAll('input[name="eye"]').forEach((el) => { el.checked = false; });
            const hue = typeof j.hue === 'number' ? j.hue : 0.5;
            const sat = typeof j.saturation === 'number' ? j.saturation : 1;
            const hueEl = document.getElementById('eyeHue');
            const satEl = document.getElementById('eyeSat');
            if (hueEl) hueEl.value = String(hue);
            if (satEl) satEl.value = String(sat);
            syncEyeCustomUIFromSliders();
        } else if (typeof j.preset === 'number') {
            document.querySelectorAll('input[name="eye"]').forEach((el) => {
                el.checked = el.value === String(j.preset);
            });
        }
    } catch (e) {
        console.log(`eye get: ${e.message}`);
    }
}

async function setEyePresetVal(preset) {
    setJdocStatus('Đang đổi màu mắt... (Setting eye color...)');
    try {
        const res = await fetch(`/api/mods/JdocSettings/setEyeColor?preset=${preset}`);
        if (!res.ok) {
            const e = await res.json();
            setJdocStatus(`${e.status}: ${e.message}`);
        } else {
            setJdocStatus('Đã đổi màu mắt. (Eye color updated.)');
            getEyePreset();
        }
    } catch (e) {
        setJdocStatus(`Lỗi mạng (network error): ${e.message}`);
    }
}

function hsvToHex(h, s, v) {
    // h,s,v in 0..1
    const i = Math.floor(h * 6);
    const f = h * 6 - i;
    const p = v * (1 - s);
    const q = v * (1 - f * s);
    const t = v * (1 - (1 - f) * s);
    let r, g, b;
    switch (i % 6) {
        case 0: r = v; g = t; b = p; break;
        case 1: r = q; g = v; b = p; break;
        case 2: r = p; g = v; b = t; break;
        case 3: r = p; g = q; b = v; break;
        case 4: r = t; g = p; b = v; break;
        default: r = v; g = p; b = q;
    }
    const to = (x) => Math.round(x * 255).toString(16).padStart(2, '0');
    return `#${to(r)}${to(g)}${to(b)}`;
}

function hexToHsv(hex) {
    const m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
    if (!m) return { h: 0.5, s: 1, v: 1 };
    const r = parseInt(m[1], 16) / 255;
    const g = parseInt(m[2], 16) / 255;
    const b = parseInt(m[3], 16) / 255;
    const max = Math.max(r, g, b);
    const min = Math.min(r, g, b);
    const d = max - min;
    let h = 0;
    if (d !== 0) {
        switch (max) {
            case r: h = ((g - b) / d + (g < b ? 6 : 0)) / 6; break;
            case g: h = ((b - r) / d + 2) / 6; break;
            default: h = ((r - g) / d + 4) / 6; break;
        }
    }
    const s = max === 0 ? 0 : d / max;
    return { h, s, v: max };
}

function syncEyeCustomUIFromSliders() {
    const hue = parseFloat(document.getElementById('eyeHue').value);
    const sat = parseFloat(document.getElementById('eyeSat').value);
    document.getElementById('eyeHueVal').textContent = hue.toFixed(2);
    document.getElementById('eyeSatVal').textContent = sat.toFixed(2);
    const hex = hsvToHex(hue, sat, 1);
    const picker = document.getElementById('eyeColorPicker');
    const swatch = document.getElementById('eyeCustomSwatch');
    if (picker) picker.value = hex;
    if (swatch) swatch.style.background = hex;
}

function onEyeSliderInput() {
    syncEyeCustomUIFromSliders();
}

function onEyePickerInput() {
    const hex = document.getElementById('eyeColorPicker').value;
    const { h, s } = hexToHsv(hex);
    document.getElementById('eyeHue').value = String(h);
    document.getElementById('eyeSat').value = String(Math.max(s, 0.15)); // keep some sat for eyes
    syncEyeCustomUIFromSliders();
}

async function applyCustomEyeColor() {
    const hue = parseFloat(document.getElementById('eyeHue').value);
    const sat = parseFloat(document.getElementById('eyeSat').value);
    setJdocStatus('Đang áp màu tùy chỉnh... (Applying custom eye color...)');
    try {
        const qs = new URLSearchParams({
            hue: String(hue),
            saturation: String(sat),
        });
        const res = await fetch(`/api/mods/JdocSettings/setCustomEyeColor?${qs.toString()}`);
        if (!res.ok) {
            const e = await res.json();
            setJdocStatus(`${e.status}: ${e.message}`);
        } else {
            document.querySelectorAll('input[name="eye"]').forEach((el) => { el.checked = false; });
            setJdocStatus('Đã áp màu tùy chỉnh. (Custom eye color applied.)');
            getEyePreset();
        }
    } catch (e) {
        setJdocStatus(`Lỗi mạng (network error): ${e.message}`);
    }
}

function showBotSection(id) {
    // Gate engine-specific tiles
    if ((id === 'bot-sens' || id === 'bot-cww') && typeof isWakeThf === 'function' && isWakeThf()) {
        if (typeof setJdocStatus === 'function') {
            setJdocStatus(id === 'bot-cww'
                ? 'Từ đánh thức tùy chỉnh chỉ dùng khi engine = Picovoice.'
                : 'Độ nhạy chỉ dùng khi engine = Picovoice.');
        }
        return;
    }
    if (id === 'bot-locale' && typeof isWakeThf === 'function' && !isWakeThf()) {
        if (typeof setJdocStatus === 'function') {
            setJdocStatus('Độ nhạy từ đánh thức chỉ dùng khi engine = Hey Vector (THF).');
        }
        return;
    }

    document.querySelectorAll('.bot-section').forEach((el) => {
        el.style.display = 'none';
    });
    document.querySelectorAll('.bot-nav-item').forEach((el) => el.classList.remove('active'));
    const sec = document.getElementById(id);
    if (sec) sec.style.display = 'block';
    // highlight matching nav button
    document.querySelectorAll('.bot-nav-item').forEach((btn) => {
        if (btn.getAttribute('onclick') && btn.getAttribute('onclick').includes(id)) {
            btn.classList.add('active');
        }
    });
    if (id === 'bot-faces') facesRefresh();
    if (id === 'bot-photos' && typeof photosRefresh === 'function') photosRefresh();
    if (id === 'bot-volume') getMasterVolume();
    if (id === 'bot-eyes') getEyePreset();
    if (id === 'bot-sens' && typeof setSensitivity === 'function') setSensitivity();
    if (id === 'bot-locale' && typeof loadWakeLocale === 'function') loadWakeLocale();
    if (id === 'bot-battery' && typeof syncBatteryUiRadios === 'function') syncBatteryUiRadios();
    if (id === 'bot-stim' && typeof stimStart === 'function') stimStart();
    else if (typeof stimStop === 'function') stimStop();
    if (id === 'bot-stats' && typeof statsRefresh === 'function') statsRefresh();
    if (id === 'bot-alexa' && typeof alexaRefresh === 'function') alexaRefresh();
    if (id === 'bot-wifi-setup' && typeof loadWifiSetupMode === 'function') loadWifiSetupMode();
    if (id === 'bot-perf' && typeof GetCurrent === 'function') {
        GetCurrent('FreqChange').then((data) => {
            document.getElementsByName('frequency').forEach((rb) => {
                if (rb.value == data) rb.checked = true;
            });
        }).catch(() => {});
    }
}

// Back-compat no-ops if old callers exist
async function setMasterVolume() {
    const checked = document.querySelector('input[name="vol"]:checked');
    if (checked) setMasterVolumeVal(checked.value);
}
async function setEyePreset() {
    const checked = document.querySelector('input[name="eye"]:checked');
    if (checked) setEyePresetVal(checked.value);
}