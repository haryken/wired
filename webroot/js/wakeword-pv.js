function setWakeStatus(status) {
    const el = document.getElementById('wakeWordStatus');
    el.innerHTML = `<p>${status}</p>`;
}

async function genWakeWord() {
    const kw = document.getElementById('keyword').value;
    setWakeStatus('Đang tạo từ đánh thức... (Generating...)');
    try {
        const res = await fetch(`/api/mods/WakeWordPV/request-model?keyword=${kw}`);
        if (!res.ok) {
            const e = await res.json();
            setWakeStatus(`${e.status}: ${e.message}`);
        } else {
            setWakeStatus('Đã tạo và cài — đang khởi động lại... (Installed. Restarting...)');
            await RestartVic();
            setWakeStatus('Từ đánh thức mới đã sẵn sàng. (New wake word ready.)');
        }
    } catch (e) {
        setWakeStatus(`Lỗi mạng (network error): ${e.message}`);
    } finally {
    }
}

async function revertDefaultWakeWord() {
    setWakeStatus('Đang xóa từ tùy chỉnh... (Deleting...)');
    await fetch('/api/mods/WakeWordPV/delete-model');
    setWakeStatus('Đã xóa — đang khởi động lại... (Deleted. Restarting...)');
    await RestartVic();
    setWakeStatus('Đã về mặc định. (Reverted to default.)');
}

async function RestartVic() {
    const tabsEl = document.querySelector('.tabs');
    const activePanel = document.querySelector('.tab-content.active');
    tabsEl.style.display = 'none';
    if (activePanel) activePanel.classList.remove('active');
    show('showDuringVicRestart');
    await fetch('/api/extra/restartvic', { method: 'POST' });
    hide('showDuringVicRestart');
    tabsEl.style.display = 'flex';
    document.querySelectorAll('.tab-content').forEach(c => c.style.display = '');
    if (activePanel) activePanel.classList.add('active');
}