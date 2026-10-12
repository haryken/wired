/* Xiaozhi music source: server's own music vs device self.music.play (YouTube music server). */

const MUSIC_DEFAULT_URL = 'https://youtube.kytuoi.com';

function musicStatus(msg) {
    const el = document.getElementById('musicStatus');
    if (!el) return;
    el.style.display = msg ? 'block' : 'none';
    el.textContent = msg || '';
}

function musicSelectedSource() {
    const el = document.querySelector('input[name="musicSource"]:checked');
    return el ? el.value : 'server';
}

function musicShow(cfg) {
    const source = cfg.music_source === 'youtube' ? 'youtube' : 'server';
    document.querySelectorAll('input[name="musicSource"]').forEach((el) => {
        el.checked = el.value === source;
    });
    const url = document.getElementById('musicServerUrl');
    if (url) url.value = cfg.music_server_url || MUSIC_DEFAULT_URL;
    let msg = source === 'youtube'
        ? 'Đang dùng: nhạc YouTube từ ' + (cfg.music_server_url || MUSIC_DEFAULT_URL) + '.'
        : 'Đang dùng: nhạc server gốc Xiaozhi.';
    if (!cfg.enabled) msg += ' (Xiaozhi đang tắt — chưa có tác dụng.)';
    musicStatus(msg);
}

async function loadMusicSettings() {
    try {
        const r = await fetch('/api/mods/Xiaozhi/get', { cache: 'no-store' });
        if (!r.ok) throw new Error('HTTP ' + r.status);
        musicShow(await r.json());
    } catch (e) {
        musicStatus('Lỗi đọc cài đặt nhạc: ' + (e.message || e));
    }
}

async function saveMusicSettings() {
    const url = (document.getElementById('musicServerUrl') || {}).value || '';
    musicStatus('Đang lưu…');
    try {
        const q = new URLSearchParams({ source: musicSelectedSource(), url: url.trim() });
        const r = await fetch('/api/mods/Xiaozhi/set_music?' + q.toString(), { method: 'POST' });
        const j = await r.json().catch(() => ({}));
        if (!r.ok || j.status !== 'success') throw new Error(j.message || ('HTTP ' + r.status));
        musicShow(j.config || {});
    } catch (e) {
        musicStatus('Lỗi lưu cài đặt nhạc: ' + (e.message || e));
    }
}

function resetMusicServerUrl() {
    const url = document.getElementById('musicServerUrl');
    if (url) url.value = MUSIC_DEFAULT_URL;
    saveMusicSettings();
}

async function testMusicServer() {
    const url = ((document.getElementById('musicServerUrl') || {}).value || '').trim();
    musicStatus('Đang thử tìm và mở 1 bài trên server nhạc… (tối đa ~25 giây)');
    try {
        const r = await fetch('/api/mods/Xiaozhi/test_music_server?' + new URLSearchParams({ url }).toString(), {
            method: 'POST',
        });
        const j = await r.json().catch(() => ({}));
        if (!r.ok) throw new Error(j.message || ('HTTP ' + r.status));
        if (j.search_ok && j.stream_ok) {
            musicStatus('Server nhạc OK: tìm thấy "' + j.title + '" và mở được luồng nhạc.');
        } else if (j.search_ok) {
            musicStatus('Tìm được "' + j.title + '" nhưng KHÔNG mở được luồng nhạc: ' + (j.error || '?'));
        } else {
            musicStatus('Server nhạc lỗi khi tìm bài: ' + (j.error || '?'));
        }
    } catch (e) {
        musicStatus('Lỗi thử server nhạc: ' + (e.message || e));
    }
}
