/* DNS mode: dhcp (default) vs public 8.8.8.8 / 1.1.1.1 via ConnMan. */

function dnsT(key, fallback) {
    return (typeof t === 'function') ? t(key, fallback) : fallback;
}

function dnsStatus(msg, isErr) {
    const el = document.getElementById('dnsStatus');
    if (!el) return;
    el.style.display = msg ? 'block' : 'none';
    el.textContent = msg || '';
    el.classList.toggle('error', !!isErr && !!msg);
}

function dnsShow(cfg) {
    const mode = (cfg && cfg.mode === 'public') ? 'public' : 'dhcp';
    document.querySelectorAll('input[name="dnsMode"]').forEach((el) => {
        el.checked = el.value === mode;
    });
    let msg;
    if (mode === 'public') {
        msg = dnsT('dns.st_on', 'Đang dùng DNS công cộng (8.8.8.8 + 1.1.1.1).');
    } else {
        msg = dnsT('dns.st_off', 'Đang theo DNS router (DHCP) — như mặc định.');
    }
    if (cfg && cfg.deferred) {
        msg += ' ' + dnsT('dns.deferred', 'Đã lưu — áp dụng sau khi robot vào Wi‑Fi nhà.');
    }
    if (cfg && cfg.apply_error) {
        msg += ' ' + dnsT('dns.apply_warn', 'Áp dụng mạng:') + ' ' + cfg.apply_error;
    }
    dnsStatus(msg, !!cfg.apply_error);
    const hint = document.getElementById('dnsResolvHint');
    if (hint) {
        const ns = (cfg && cfg.active_nameservers) ? cfg.active_nameservers.join(', ') : '';
        hint.textContent = ns
            ? dnsT('dns.now', 'DNS đang dùng: {list}').replace('{list}', ns)
            : '';
    }
}

async function loadDnsSettings() {
    try {
        const r = await fetch('/api/mods/DnsSettings/get', { cache: 'no-store' });
        if (!r.ok) throw new Error('HTTP ' + r.status);
        dnsShow(await r.json());
    } catch (e) {
        dnsStatus(dnsT('dns.err_load', 'Lỗi đọc cài đặt DNS:') + ' ' + (e.message || e), true);
    }
}

async function setDnsMode(mode) {
    dnsStatus(dnsT('dns.saving', 'Đang lưu…'), false);
    try {
        const r = await fetch('/api/mods/DnsSettings/set?' + new URLSearchParams({ mode: mode }).toString(), {
            method: 'POST',
        });
        const j = await r.json().catch(() => ({}));
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        if (j.config) dnsShow(j.config);
        else await loadDnsSettings();
    } catch (e) {
        dnsStatus(dnsT('dns.err_save', 'Lỗi lưu DNS:') + ' ' + (e.message || e), true);
        await loadDnsSettings();
    }
}
