let photosLightboxId = null;

function photosSetStatus(msg, isError) {
    const el = document.getElementById('photosStatus');
    if (!el) return;
    if (!msg) {
        el.style.display = 'none';
        el.textContent = '';
        el.className = 'faces-status';
        return;
    }
    el.style.display = 'block';
    el.textContent = msg;
    el.className = 'faces-status ' + (isError ? 'error' : 'ok');
}

function photosFormatTime(ts) {
    if (!ts) return '';
    try {
        return new Date(ts * 1000).toLocaleString();
    } catch (_) {
        return String(ts);
    }
}

async function photosRefresh() {
    photosSetStatus('', false);
    const grid = document.getElementById('photosGrid');
    if (!grid) return;
    grid.innerHTML = '<p>Đang tải... (Loading...)</p>';

    try {
        const res = await fetch('/api/mods/Photos/list');
        const txt = await res.text();
        if (!res.ok) throw new Error(txt || `http ${res.status}`);
        const list = JSON.parse(txt);

        if (!Array.isArray(list) || list.length === 0) {
            grid.innerHTML = '<p>Chưa có ảnh nào. Hãy bảo Vector chụp ảnh (Xiaozhi / “take a photo”). <em>(No photos yet.)</em></p>';
            return;
        }

        grid.innerHTML = list.map((p) => {
            const id = Number(p.id);
            const ts = Number(p.timestamp_utc) || 0;
            return `
        <div class="photo-card" data-id="${id}">
            <button type="button" class="photo-thumb-btn" onclick="photosOpenLightbox(${id})" title="Xem (View)">
                <img class="photo-thumb" src="/api/mods/Photos/thumb?id=${id}" alt="photo ${id}" loading="lazy">
            </button>
            <div class="photo-meta">#${id} · ${photosFormatTime(ts)}</div>
            <button type="button" class="photo-del" onclick="photosDelete(${id})">Xóa (Delete)</button>
        </div>`;
        }).join('');
    } catch (e) {
        grid.innerHTML = '';
        photosSetStatus(`Không tải được ảnh: ${e.message} (failed to load)`, true);
    }
}

function photosOpenLightbox(id) {
    photosLightboxId = id;
    const box = document.getElementById('photoLightbox');
    const img = document.getElementById('photoLightboxImg');
    if (!box || !img) return;
    img.src = `/api/mods/Photos/image?id=${id}`;
    box.style.display = 'flex';
}

function photosCloseLightbox(ev) {
    // Backdrop click: only close when the outer overlay is the target
    if (ev && ev.target && ev.target.id !== 'photoLightbox' && ev.type === 'click') {
        return;
    }
    const box = document.getElementById('photoLightbox');
    const img = document.getElementById('photoLightboxImg');
    if (box) box.style.display = 'none';
    if (img) img.removeAttribute('src');
    photosLightboxId = null;
}

async function photosDelete(id) {
    const ok = await wireosConfirm({
        title: 'Xóa ảnh',
        message: 'Xóa ảnh #' + id + '?',
        ok: 'Xóa',
        cancel: 'Hủy',
    });
    if (!ok) return;
    try {
        const res = await fetch(`/api/mods/Photos/delete?id=${id}`);
        const txt = await res.text();
        if (!res.ok) throw new Error(txt || `http ${res.status}`);
        if (photosLightboxId === id) photosCloseLightbox();
        photosSetStatus(`Đã xóa #${id}. (Deleted.)`, false);
        await photosRefresh();
    } catch (e) {
        photosSetStatus(`Xóa thất bại: ${e.message}`, true);
    }
}

async function photosDeleteFromLightbox() {
    if (photosLightboxId == null) return;
    await photosDelete(photosLightboxId);
}
