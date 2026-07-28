async function loadWakeEngine() {
    try {
        const res = await fetch('/api/mods/WakeEngine/get');
        const engine = (await res.text()).trim() || 'picovoice';
        const pv = document.getElementById('wakeEnginePv');
        const thf = document.getElementById('wakeEngineThf');
        if (pv && thf) {
            pv.checked = engine !== 'thf';
            thf.checked = engine === 'thf';
        }
        onWakeEngineRadio();
    } catch (e) {
        setWakeEngineStatus('Không đọc được engine: ' + e.message);
    }
}

function onWakeEngineRadio() {
    const thf = document.getElementById('wakeEngineThf');
    const useThf = thf && thf.checked;
    const pvPanel = document.getElementById('wakePicovoicePanel');
    const thfPanel = document.getElementById('wakeThfPanel');
    if (pvPanel) pvPanel.style.display = useThf ? 'none' : '';
    if (thfPanel) thfPanel.style.display = useThf ? '' : 'none';
    // Sensitivity (bot-sens) is Picovoice-only — dim nav hint via status if THF
    const sensNav = document.querySelector('[onclick*="bot-sens"]');
    if (sensNav) {
        sensNav.style.opacity = useThf ? '0.45' : '';
        sensNav.title = useThf ? 'Chỉ Picovoice (Picovoice only)' : '';
    }
}

function setWakeEngineStatus(msg) {
    const el = document.getElementById('wakeEngineStatus');
    if (el) el.innerHTML = `<p>${msg}</p>`;
}

async function saveWakeEngine() {
    const thf = document.getElementById('wakeEngineThf');
    const engine = (thf && thf.checked) ? 'thf' : 'picovoice';
    setWakeEngineStatus('Đang lưu ' + engine + '...');
    try {
        const res = await fetch('/api/mods/WakeEngine/set?engine=' + encodeURIComponent(engine));
        if (!res.ok) {
            setWakeEngineStatus('Lỗi lưu engine');
            return;
        }
        setWakeEngineStatus('Đã lưu — đang restart anim...');
        await RestartVic();
        setWakeEngineStatus('Xong. Engine: ' + engine);
        await loadWakeEngine();
    } catch (e) {
        setWakeEngineStatus('Lỗi: ' + e.message);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    loadWakeEngine();
});
