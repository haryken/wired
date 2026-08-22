function wifiT(key, fallback) {
    if (typeof t === 'function') return t(key, fallback);
    return fallback || key;
}

let wifiOpenNet = false;
let wifiScanBusy = false;
let wifiCanJoin = false;
let wifiDidAutoScan = false;
let wifiApSsid = '';

function wifiSetJoinMode(canJoin, status) {
    wifiCanJoin = !!canJoin;
    const join = document.getElementById('wifiJoinPanel');
    const conn = document.getElementById('wifiConnectedPanel');
    const net = document.getElementById('wifiConnectedNet');
    if (join) join.hidden = !canJoin;
    if (conn) conn.hidden = canJoin;
    if (!canJoin && net) {
        const ssid = (status && status.client_ssid) || '';
        const ips = (status && status.ips) || [];
        net.innerHTML =
            '<span class="wifi-net-name">' + wifiEscape(ssid || '—') + '</span>' +
            '<span class="wifi-net-meta">' + wifiBars(100) +
            (ips.length ? ' · ' + wifiEscape(ips.join(', ')) : '') + '</span>';
    }
    if (!canJoin) {
        const list = document.getElementById('wifiNetworkList');
        if (list) list.innerHTML = '';
    }
}

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
        if (j.ap_ssid) wifiApSsid = j.ap_ssid;
        if (j.client_ssid) bits.push('WiFi nhà: ' + j.client_ssid);
        if (j.ips && j.ips.length) bits.push('IP: ' + j.ips.join(', '));
        line.textContent = bits.join(' · ');

        // Only allow join while robot hotspot is up. If already on home WiFi, read-only.
        const onHomeWifi = !j.ap && !!(j.client_ssid || (j.ips && j.ips.length));
        wifiSetJoinMode(!!j.ap, j);
        if (!j.ap && !onHomeWifi) {
            const hint = document.getElementById('wifiConnectedHint');
            const net = document.getElementById('wifiConnectedNet');
            if (hint) hint.textContent = wifiT('wifi.wait_hotspot', 'Chưa có WiFi nhà. Đợi robot mở hotspot rồi nối phone vào để bắt mạng.');
            if (net) net.innerHTML = '<span class="wifi-net-name">—</span>';
        } else if (onHomeWifi) {
            const hint = document.getElementById('wifiConnectedHint');
            if (hint) hint.textContent = wifiT('wifi.connected_only', 'Robot đang nối WiFi nhà. Chỉ xem trạng thái — đổi mạng khi mất kết nối (qua hotspot).');
        }

        const hint = document.getElementById('wifiScanHint');
        if (hint && j.ap && !hint.dataset.locked) {
            hint.textContent = wifiT('wifi.ap_pass_hint', 'Hotspot robot không cần mật khẩu. Ô bên dưới là mật khẩu WiFi NHÀ.');
        }
        if (j.ap && !wifiDidAutoScan) {
            wifiDidAutoScan = true;
            wifiScan();
        }
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
    if (!wifiCanJoin) return;
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
    if (!wifiCanJoin) return;
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

function wifiTogglePass() {
    const passEl = document.getElementById('wifiPass');
    const btn = document.getElementById('wifiPassToggle');
    if (!passEl || !btn) return;
    const show = passEl.type === 'password';
    passEl.type = show ? 'text' : 'password';
    btn.setAttribute('aria-pressed', show ? 'true' : 'false');
    btn.setAttribute('aria-label', show
        ? wifiT('wifi.hide_pass', 'Ẩn mật khẩu')
        : wifiT('wifi.show_pass', 'Hiện mật khẩu'));
}

function wifiSentModal(ok) {
    const ap = wifiApSsid || 'Vector XXXX';
    let root = document.getElementById('wifiSentModal');
    if (!root) {
        root = document.createElement('div');
        root.id = 'wifiSentModal';
        root.className = 'wireos-modal wifi-sent-modal';
        root.hidden = true;
        root.innerHTML =
            '<div class="wireos-modal-backdrop" data-wifi-sent-close></div>' +
            '<div class="wireos-modal-panel" role="dialog" aria-modal="true">' +
            '  <div class="wifi-sent-mark" aria-hidden="true">✓</div>' +
            '  <h3 id="wifiSentTitle" class="wireos-modal-title"></h3>' +
            '  <p id="wifiSentBody" class="wireos-modal-body"></p>' +
            '  <div class="wireos-modal-actions">' +
            '    <button type="button" class="wireos-modal-btn primary" data-wifi-sent-close></button>' +
            '  </div>' +
            '</div>';
        document.body.appendChild(root);
        root.querySelectorAll('[data-wifi-sent-close]').forEach((el) => {
            el.addEventListener('click', () => {
                root.hidden = true;
                document.body.classList.remove('wireos-modal-open');
            });
        });
    }
    const title = document.getElementById('wifiSentTitle');
    const body = document.getElementById('wifiSentBody');
    const btn = root.querySelector('.wireos-modal-btn');
    const mark = root.querySelector('.wifi-sent-mark');
    if (ok) {
        if (mark) mark.textContent = '✓';
        title.textContent = wifiT('wifi.sent_title', 'Đã gửi thành công');
        const bodyTpl = wifiT('wifi.sent_body',
            'Robot đã nhận WiFi nhà.\n\nĐợi khoảng 30 giây: hotspot sẽ tắt để thử vào mạng nhà.\n\n• Thành công: mở lại http://<IP-robot>:8080/\n• Thất bại: hotspot {ap} bật lại — nối phone vào {ap} rồi nhập lại mật khẩu.');
        body.textContent = bodyTpl.split('{ap}').join(ap);
        btn.textContent = wifiT('wifi.sent_ok', 'Đã hiểu');
    } else {
        if (mark) mark.textContent = '!';
        title.textContent = wifiT('wifi.sent_fail_title', 'Chưa gửi được');
        body.textContent = wifiT('wifi.sent_fail_body', 'Không gửi được tới robot. Kiểm tra phone còn nối hotspot rồi thử lại.');
        btn.textContent = wifiT('wifi.sent_ok', 'Đã hiểu');
    }
    if (mark) {
        mark.style.animation = 'none';
        void mark.offsetWidth;
        mark.style.animation = '';
    }
    root.hidden = false;
    document.body.classList.add('wireos-modal-open');
}

async function wifiApply() {
    if (!wifiCanJoin) {
        alert(wifiT('wifi.connected_only', 'Đang có WiFi nhà — chỉ bắt lại qua hotspot khi mất mạng.'));
        return;
    }
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
    const body = new URLSearchParams({ ssid: ssid.trim(), password: pass, from: 'hotspot' });
    try {
        const r = await fetch('/api/mods/WifiSetup/connect', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body,
        });
        const t = await r.text();
        if (log) log.textContent = t;
        if (!r.ok) {
            wifiSentModal(false);
            return;
        }
        wifiSentModal(true);
    } catch (e) {
        if (log) log.textContent = String(e);
        wifiSentModal(false);
    }
}

function wifiBoot() {
    wifiRefreshStatus();
    setInterval(wifiRefreshStatus, 8000);
    if (/[?&]wifi=sent\b/.test(location.search)) {
        wifiSentModal(true);
    }
}
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', wifiBoot);
} else {
    wifiBoot();
}
