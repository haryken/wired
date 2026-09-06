/* Bot Settings → Bắt WiFi (BLE vs Hotspot). Default BLE. */

async function loadWifiSetupMode() {
    const statusEl = document.getElementById('wifiModeStatus');
    try {
        const r = await fetch('/api/mods/WifiSetup/mode', { cache: 'no-store' });
        const j = await r.json();
        if (!r.ok || j.status === 'error') {
            throw new Error(j.message || ('HTTP ' + r.status));
        }
        const mode = (j.mode === 'hotspot') ? 'hotspot' : 'ble';
        document.querySelectorAll('input[name="wifiSetupMode"]').forEach((el) => {
            el.checked = (el.value === mode);
        });
        showWifiModeGuide(mode);
        if (statusEl) {
            statusEl.style.display = 'block';
            const ap = j.ap ? ' · AP đang bật' : '';
            const ssid = j.client_ssid ? (' · WiFi: ' + j.client_ssid) : '';
            statusEl.textContent = (mode === 'hotspot' ? 'Hotspot' : 'Bluetooth') + ap + ssid;
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
    const hs = document.getElementById('wifiModeGuideHotspot');
    if (ble) ble.style.display = (mode === 'ble') ? 'block' : 'none';
    if (hs) hs.style.display = (mode === 'hotspot') ? 'block' : 'none';
}

async function setWifiSetupMode(mode) {
    const statusEl = document.getElementById('wifiModeStatus');
    showWifiModeGuide(mode);
    if (statusEl) {
        statusEl.style.display = 'block';
        statusEl.textContent = (mode === 'hotspot')
            ? 'Đang lưu chế độ Hotspot…'
            : 'Đang lưu chế độ Bluetooth…';
    }
    try {
        const body = new URLSearchParams({ mode: mode });
        const r = await fetch('/api/mods/WifiSetup/mode', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: body.toString(),
        });
        const j = await r.json();
        if (!r.ok || j.status === 'error') {
            throw new Error(j.message || ('HTTP ' + r.status));
        }
        const out = (j.mode === 'hotspot') ? 'hotspot' : 'ble';
        document.querySelectorAll('input[name="wifiSetupMode"]').forEach((el) => {
            el.checked = (el.value === out);
        });
        showWifiModeGuide(out);
        if (statusEl) {
            if (out === 'hotspot') {
                statusEl.textContent = j.ap_deferred || j.online
                    ? 'Đã chọn Hotspot — giữ WiFi nhà. Mất mạng mới phát hotspot.'
                    : (j.ap
                        ? 'Đã bật hotspot. Nối WiFi robot → http://192.168.4.1/'
                        : 'Đã chọn Hotspot — đang mở AP…');
            } else {
                statusEl.textContent = j.online
                    ? 'Đã chọn Bluetooth — giữ WiFi nhà. Mất mạng sẽ dùng BLE.'
                    : 'Đã chọn Bluetooth. Ấn lưng 2 lần hoặc WIFI BY BLUE trên mặt.';
            }
        }
    } catch (e) {
        if (statusEl) {
            statusEl.style.display = 'block';
            statusEl.textContent = 'Lỗi đổi chế độ: ' + (e.message || e);
        }
        loadWifiSetupMode();
    }
}
