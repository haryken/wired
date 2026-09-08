/* Web Settings policy selector. AP lifecycle remains in the isolated service. */

function wifiModeApi() {
    return 'http://' + window.location.hostname + ':8081/api/mods/WifiSetup/mode';
}

async function loadWifiSetupMode() {
    const statusEl = document.getElementById('wifiModeStatus');
    try {
        const r = await fetch(wifiModeApi(), { cache: 'no-store' });
        const j = await r.json();
        if (!r.ok || j.status === 'error') throw new Error(j.message || ('HTTP ' + r.status));
        const mode = j.mode === 'ble' ? 'ble' : 'hotspot';
        document.querySelectorAll('input[name="wifiSetupMode"]').forEach((el) => {
            el.checked = el.value === mode;
        });
        showWifiModeGuide(mode);
        if (statusEl) {
            statusEl.style.display = 'block';
            statusEl.textContent = (mode === 'hotspot' ? 'Hotspot' : 'Bluetooth') +
                (j.ap ? ' · AP đang bật' : '') +
                (j.client_ssid ? ' · WiFi: ' + j.client_ssid : '');
        }
    } catch (e) {
        if (statusEl) {
            statusEl.style.display = 'block';
            statusEl.textContent = 'Lỗi đọc chế độ: ' + (e.message || e);
        }
    }
}

function showWifiModeGuide(mode) {
    const ble = document.getElementById('wifiModeGuideBle');
    const hotspot = document.getElementById('wifiModeGuideHotspot');
    if (ble) ble.style.display = mode === 'ble' ? 'block' : 'none';
    if (hotspot) hotspot.style.display = mode === 'hotspot' ? 'block' : 'none';
}

async function setWifiSetupMode(mode) {
    const statusEl = document.getElementById('wifiModeStatus');
    showWifiModeGuide(mode);
    if (statusEl) {
        statusEl.style.display = 'block';
        statusEl.textContent = 'Đang lưu chế độ ' + (mode === 'hotspot' ? 'Hotspot…' : 'Bluetooth…');
    }
    try {
        const r = await fetch(wifiModeApi(), {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: new URLSearchParams({ mode }).toString(),
        });
        const j = await r.json();
        if (!r.ok || j.status === 'error') throw new Error(j.message || ('HTTP ' + r.status));
        await loadWifiSetupMode();
        if (statusEl && mode === 'hotspot' && (j.ap_deferred || j.online)) {
            statusEl.textContent = 'Đã chọn Hotspot — WiFi hiện tại được giữ nguyên; mất mạng mới bật AP.';
        }
    } catch (e) {
        if (statusEl) statusEl.textContent = 'Lỗi đổi chế độ: ' + (e.message || e);
        await loadWifiSetupMode();
    }
}
