function wifiT(key, fallback) {
    if (typeof t === 'function') return t(key, fallback);
    return fallback || key;
}

let wifiOpenNet = false;
let wifiScanBusy = false;

async function wifiRefreshStatus() {
    const line = document.getElementById('wifiStatusLine');
    const log = document.getElementById('wifiLog');
    if (!line) return;
    try {
        const r = await fetch('/api/mods/WifiSetup/status');
        const j = await r.json();
        const bits = [];
        if (j.ap) bits.push('Hotspot mở: ' + (j.ap_ssid || '?') + ' (không mật khẩu)');
        else bits.push('Hotspot: off');
        if (j.client_ssid) bits.push('WiFi nhà: ' + j.client_ssid);
        if (j.ips && j.ips.length) bits.push('IP: ' + j.ips.join(', '));
        line.textContent = bits.join(' · ');
        const hint = document.getElementById('wifiScanHint');
        if (hint && j.ap && !hint.dataset.locked) {
            hint.textContent = wifiT('wifi.ap_pass_hint', 'Hotspot robot không cần mật khẩu. Ô bên dưới là mật khẩu WiFi NHÀ.');
        }
        const ssidEl = document.getElementById('wifiSsid');
        if (ssidEl && !ssidEl.value && j.client_ssid) ssidEl.placeholder = j.client_ssid;
        if (log && !log.dataset.touched) log.textContent = JSON.stringify(j, null, 2);
    } catch (e) {
        line.textContent = 'Không đọc được trạng thái WiFi';
    }
}

function wifiEscape(s) {
    return String(s).replace(/[&<>"']/g, (c) => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
    }[c]));
}

function wifiBars(pct) {
    const n = pct >= 75 ? 4 : pct >= 50 ? 3 : pct >= 25 ? 2 : 1;
    let h = '<span class="wifi-bars" aria-hidden="true">';
    for (let i = 1; i <= 4; i++) h += '<i class="' + (i <= n ? 'on' : '') + '"></i>';
    return h + '</span>';
}

function wifiSelectNet(ssid, secure) {
    const ssidEl = document.getElementById('wifiSsid');
    const passEl = document.getElementById('wifiPass');
    if (ssidEl) ssidEl.value = ssid;
    wifiOpenNet = !secure;
    document.querySelectorAll('#wifiNetworkList .wifi-net').forEach((el) => {
        el.classList.toggle('selected', el.dataset.ssid === ssid);
    });
    if (!passEl) return;
    if (wifiOpenNet) {
        passEl.removeAttribute('minlength');
        passEl.placeholder = wifiT('wifi.unlocked', 'mạng mở — để trống');
        passEl.value = '';
    } else {
        passEl.setAttribute('minlength', '8');
        passEl.placeholder = '';
        passEl.focus();
    }
}

async function wifiScan() {
    const list = document.getElementById('wifiNetworkList');
    const hint = document.getElementById('wifiScanHint');
    const btn = document.getElementById('wifiScanBtn');
    if (!list || wifiScanBusy) return;
    wifiScanBusy = true;
    if (btn) btn.disabled = true;
    if (hint) hint.textContent = wifiT('wifi.scanning', 'Đang quét mạng 2.4 GHz…');
    list.innerHTML = '';
    try {
        const r = await fetch('/api/mods/WifiSetup/scan');
        const j = await r.json();
        const nets = j.networks || [];
        if (!nets.length) {
            if (hint) {
                hint.textContent = j.ap
                    ? wifiT('wifi.scan_ap', 'Đang hotspot — quét có thể trống. Gõ tay SSID.')
                    : wifiT('wifi.scan_empty', 'Không thấy mạng. Gõ tay SSID.');
            }
            return;
        }
        if (hint) {
            hint.textContent = j.cached
                ? wifiT('wifi.scan_ap', 'Đang hotspot: hiện list đã quét trước khi mở hotspot.')
                : wifiT('wifi.pick', 'Chọn mạng rồi nhập mật khẩu.');
        }
        nets.forEach((n) => {
            const b = document.createElement('button');
            b.type = 'button';
            b.className = 'wifi-net';
            b.dataset.ssid = n.ssid;
            b.innerHTML = '<span class="wifi-net-name">' + wifiEscape(n.ssid) + '</span>' +
                '<span class="wifi-net-meta">' + wifiBars(n.signal || 0) +
                (n.secure ? ' 🔒' : '') + '</span>';
            b.onclick = () => wifiSelectNet(n.ssid, !!n.secure);
            list.appendChild(b);
        });
    } catch (e) {
        if (hint) hint.textContent = wifiT('wifi.scan_fail', 'Không quét được. Gõ tay SSID.');
    } finally {
        wifiScanBusy = false;
        if (btn) btn.disabled = false;
    }
}

async function wifiApply() {
    const ssid = (document.getElementById('wifiSsid') || {}).value || '';
    const pass = (document.getElementById('wifiPass') || {}).value || '';
    const log = document.getElementById('wifiLog');
    if (!ssid.trim()) {
        alert(wifiT('wifi.ssid_ph', 'Nhập SSID WiFi nhà'));
        return;
    }
    if (!wifiOpenNet && pass.length < 8) {
        alert('Mật khẩu WiFi nhà phải từ 8 ký tự');
        return;
    }
    if (log) {
        log.dataset.touched = '1';
        log.textContent = 'Đang gửi…';
    }
    const fromHotspot = location.hostname === '10.3.141.1' || (location.pathname || '').replace(/\/+$/, '') === '/wifi';
    const body = new URLSearchParams({ ssid: ssid.trim(), password: pass });
    if (fromHotspot) body.set('from', 'hotspot');
    else body.set('from', 'lan');
    try {
        const r = await fetch('/api/mods/WifiSetup/connect', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body,
        });
        const t = await r.text();
        if (log) log.textContent = t;
        alert('Đã gửi. Robot sẽ rớt mạng ~30–60 giây rồi vào WiFi mới. Mở lại http://<IP-mới>:8080/');
    } catch (e) {
        if (log) log.textContent = String(e);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    wifiRefreshStatus();
    setInterval(wifiRefreshStatus, 8000);
});
