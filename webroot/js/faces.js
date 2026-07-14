function facesSetStatus(msg, isError) {
    const el = document.getElementById('facesStatus');
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

async function facesFetchText(url, opts) {
    const res = await fetch(url, opts);
    const txt = await res.text();
    if (!res.ok) {
        throw new Error(txt || `http ${res.status}`);
    }
    return txt;
}

async function facesFetchJson(url, opts) {
    const res = await fetch(url, opts);
    const txt = await res.text();
    if (!res.ok) throw new Error(txt || `http ${res.status}`);
    try { return JSON.parse(txt); }
    catch { throw new Error('bad json from server'); }
}

async function facesRefresh() {
    facesSetStatus('', false);
    const listEl = document.getElementById('facesList');
    listEl.innerHTML = '<p>Đang tải... (Loading...)</p>';

    try {
        const faces = await facesFetchJson('/api/mods/Faces/getFaces');

        if (!Array.isArray(faces) || faces.length === 0) {
            listEl.innerHTML = '<p>Chưa có khuôn mặt nào. (No enrolled faces.)</p>';
            return;
        }

        const rows = faces.map(f => {
            const id = Number(f.id);
            const name = f.name;
            const age = Number(f.secondssincefirstenrolled);

            return `
        <div class="face-card">
            <div class="face-row">
                <div>
                    <div><b>${name || '(chưa đặt tên / unnamed)'}</b></div>
                    <div class="face-meta">
                        id: ${id} — giây từ lần ghi đầu: ${age} (seconds since first enrolled)
                    </div>
                </div>
                <div class="face-actions">
                    <button type="button"
                        onclick="facePromptRename(${id}, '${name.replaceAll("'", "\\'")}')">
                        Đổi tên (Rename)
                    </button>
                    <button type="button" onclick="faceDelete(${id})">
                        Xóa (Delete)
                    </button>
                </div>
            </div>
        </div>
    `;
        }).join('');


        listEl.innerHTML = rows;
    } catch (e) {
        listEl.innerHTML = '';
        facesSetStatus(`Không tải được khuôn mặt: ${e.message} (failed to load)`, true);
    }
}

async function faceDelete(id) {
    facesSetStatus('', false);
    try {
        const qs = new URLSearchParams({ id: String(id) });
        await facesFetchText(`/api/mods/Faces/deleteFace?${qs.toString()}`);
        facesSetStatus('Đã xóa. (Deleted.)', false);
        facesRefresh();
    } catch (e) {
        facesSetStatus(`Xóa thất bại: ${e.message} (Delete failed)`, true);
    }
}

async function facePromptRename(id, currentName) {
    facesSetStatus('', false);
    const newName = prompt(`Đổi tên khuôn mặt id ${id} (Rename)`, currentName || '');
    if (newName === null) return;

    const trimmed = newName.trim();
    if (!trimmed) {
        facesSetStatus('Chưa nhập tên. (No name given.)', true);
        return;
    }

    try {
        const qs = new URLSearchParams({ id: String(id), name: trimmed });
        await facesFetchText(`/api/mods/Faces/renameFace?${qs.toString()}`);
        facesSetStatus('Đã đổi tên. (Renamed.)', false);
        facesRefresh();
    } catch (e) {
        facesSetStatus(`Đổi tên thất bại: ${e.message} (Rename failed)`, true);
    }
}

async function faceTrain() {
    facesSetStatus('', false);
    const input = document.getElementById('faceNewName');
    const name = (input.value || '').trim();
    if (!name) {
        facesSetStatus('Nhập tên trước. (Enter a name first.)', true);
        return;
    }

    try {
        const qs = new URLSearchParams({ name });
        await facesFetchText(`/api/mods/Faces/trainFace?${qs.toString()}`);
        facesSetStatus('Đã bắt đầu ghi — nhìn vào Vector. (Enroll started. Look at Vector.)', false);
        input.value = '';
        facesRefresh();
    } catch (e) {
        facesSetStatus(`Ghi thất bại: ${e.message} (train failed)`, true);
    }
}
