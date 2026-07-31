let alexaPollTimer = null;

function alexaSetStatus(html) {
    const el = document.getElementById('alexaStatus');
    if (el) {
        el.innerHTML = `<p>${html}</p>`;
        show('alexaStatus');
    }
}

function alexaStopPoll() {
    if (alexaPollTimer) clearInterval(alexaPollTimer);
    alexaPollTimer = null;
}

function alexaStartPoll() {
    alexaStopPoll();
    alexaPollTimer = setInterval(alexaRefresh, 3000);
}

async function alexaRefresh() {
    try {
        const res = await fetch('/api/mods/Alexa/status');
        const j = await res.json();
        if (!res.ok) {
            alexaSetStatus(`Lỗi: ${j.message || res.status}`);
            return;
        }
        const btn = j.button_wakeword === 1 ? 'Alexa' : 'Hey Vector';
        let html = `<strong>${j.auth_label || '—'}</strong><br>`;
        html += `Nút lưng hiện tại: <b>${btn}</b> <small>(Back button)</small>`;
        if (j.auth_state === 3) {
            html += `<br><br>Mã kích hoạt hiện trên <b>mặt robot</b>. Nhập tại `;
            html += `<a href="https://amazon.com/code" target="_blank" style="color:#c4a0ff;">amazon.com/code</a>`;
            if (j.extra) html += `<br><small>URL: ${j.extra}</small>`;
        }
        if (j.auth_state === 4) {
            html += `<br><small>Giọng: nói "Alexa" hoặc "Hey Vector". Nút lưng = theo cài đặt bên dưới.</small>`;
        }
        alexaSetStatus(html);

        document.querySelectorAll('input[name="btnWake"]').forEach((el) => {
            el.checked = el.value === String(j.button_wakeword ?? 0);
        });

        if (j.auth_state === 4 || j.auth_state === 1) {
            alexaStopPoll();
        }
    } catch (e) {
        alexaSetStatus(`Lỗi mạng: ${e.message}`);
    }
}

async function alexaOptIn(enable) {
    const msg = enable
        ? 'Bắt đầu liên kết Alexa? Robot sẽ hiện mã trên mặt.\n\nStart Alexa linking?'
        : 'Hủy liên kết Alexa?\n\nSign out of Alexa?';
    if (!confirm(msg)) return;

    alexaSetStatus(enable
        ? 'Đang bắt đầu liên kết... (Starting...)'
        : 'Đang hủy liên kết... (Signing out...)');
    try {
        const res = await fetch(`/api/mods/Alexa/optIn?enable=${enable ? 'true' : 'false'}`, { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            alexaSetStatus(`Lỗi: ${j.message || res.status}`);
            return;
        }
        alexaStartPoll();
        setTimeout(alexaRefresh, 1500);
    } catch (e) {
        alexaSetStatus(`Lỗi mạng: ${e.message}`);
    }
}

async function alexaSetButtonWake(mode) {
    try {
        const res = await fetch(`/api/mods/Alexa/setButtonWake?mode=${mode}`, { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            alexaSetStatus(`Lỗi nút lưng: ${j.message || res.status}`);
            return;
        }
        alexaRefresh();
    } catch (e) {
        alexaSetStatus(`Lỗi mạng: ${e.message}`);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    // Refresh when opened via bot-alexa tile (showBotSection); no auto-poll on every page load.
});
