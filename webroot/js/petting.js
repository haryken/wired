let pettingLoading = false;

function setPettingStatus(msg, isError) {
    const el = document.getElementById('pettingStatus');
    if (!el) return;
    el.style.display = msg ? 'block' : 'none';
    el.classList.toggle('error', !!isError);
    el.innerHTML = msg ? `<p>${msg}</p>` : '';
}

function parseBoolFlag(v) {
    const s = String(v == null ? '' : v).trim().toLowerCase();
    return s !== 'false' && s !== '0' && s !== 'off' && s !== 'no';
}

function setNamedRadios(name, enabled) {
    document.querySelectorAll(`input[name="${name}"]`).forEach((el) => {
        el.checked = el.value === (enabled ? 'true' : 'false');
    });
}

function selectedNamedEnabled(name) {
    const on = document.querySelector(`input[name="${name}"][value="true"]`);
    return !!(on && on.checked);
}

async function loadPetting() {
    pettingLoading = true;
    try {
        const res = await fetch('/api/mods/Petting/get');
        const text = await res.text();
        let petting = true;
        let touch = true;
        try {
            const j = JSON.parse(text);
            petting = parseBoolFlag(j.petting);
            touch = parseBoolFlag(j.touch);
        } catch (_) {
            // Legacy plain true/false
            petting = parseBoolFlag(text);
        }
        setNamedRadios('pettingEnabled', petting);
        setNamedRadios('touchSensorEnabled', touch);
        setPettingStatus('', false);
    } catch (e) {
        setPettingStatus('Không đọc được cài đặt petting: ' + e.message, true);
    } finally {
        pettingLoading = false;
    }
}

async function savePetting() {
    if (pettingLoading) return;
    const enabled = selectedNamedEnabled('pettingEnabled') ? 'true' : 'false';
    setPettingStatus('Đang lưu…', false);
    try {
        const res = await fetch('/api/mods/Petting/set?enabled=' + encodeURIComponent(enabled));
        if (!res.ok) {
            let msg = 'Lỗi lưu';
            try {
                const j = await res.json();
                if (j.message) msg = j.message;
            } catch (_) {}
            setPettingStatus(msg, true);
            await loadPetting();
            return;
        }
        setPettingStatus(
            enabled === 'true'
                ? 'Đã bật petting / tiếng rừ rừ (áp dụng trong ~1 giây).'
                : 'Đã tắt petting / tiếng rừ rừ (áp dụng trong ~1 giây).',
            false
        );
    } catch (e) {
        setPettingStatus('Lỗi: ' + e.message, true);
        await loadPetting();
    }
}

async function saveTouchSensor() {
    if (pettingLoading) return;
    const enabled = selectedNamedEnabled('touchSensorEnabled') ? 'true' : 'false';
    setPettingStatus('Đang lưu…', false);
    try {
        const res = await fetch('/api/mods/Petting/set_touch?enabled=' + encodeURIComponent(enabled));
        if (!res.ok) {
            let msg = 'Lỗi lưu';
            try {
                const j = await res.json();
                if (j.message) msg = j.message;
            } catch (_) {}
            setPettingStatus(msg, true);
            await loadPetting();
            return;
        }
        setPettingStatus(
            enabled === 'true'
                ? 'Đã bật cảm biến lưng (wake / timer / … hoạt động lại, ~1 giây).'
                : 'Đã tắt cảm biến lưng — chặn mọi tính năng chạm (hết báo ảo, ~1 giây).',
            false
        );
    } catch (e) {
        setPettingStatus('Lỗi: ' + e.message, true);
        await loadPetting();
    }
}

document.addEventListener('DOMContentLoaded', () => {
    loadPetting();
});
