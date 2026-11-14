function facesT(key, fallback) {
    if (window.I18N && typeof I18N.t === 'function') return I18N.t(key, fallback);
    return fallback != null ? fallback : key;
}

function facesEsc(s) {
    return String(s == null ? '' : s)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function facesInitial(name) {
    const s = String(name || '').trim();
    if (!s) return '?';
    return s.charAt(0).toUpperCase();
}

function facesFmtAge(sec) {
    const n = Number(sec);
    if (!Number.isFinite(n) || n < 0) return '—';
    if (n < 60) return Math.floor(n) + 's';
    if (n < 3600) return Math.floor(n / 60) + 'm';
    if (n < 86400) return Math.floor(n / 3600) + 'h';
    return Math.floor(n / 86400) + 'd';
}

function facesSetStatus(msg, isError) {
    const el = document.getElementById('facesStatus');
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

function facesSetCount(n) {
    const el = document.getElementById('facesCount');
    if (!el) return;
    if (n == null || n < 0) {
        el.textContent = '';
        return;
    }
    el.textContent = facesT('faces.count', '{n} khuôn mặt')
        .replace('{n}', String(n));
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
    listEl.innerHTML = `<p class="faces-loading">${facesEsc(facesT('faces.loading', 'Đang tải…'))}</p>`;
    facesSetCount(-1);

    try {
        const faces = await facesFetchJson('/api/mods/Faces/getFaces');

        if (!Array.isArray(faces) || faces.length === 0) {
            listEl.innerHTML = `<p class="faces-empty">${facesEsc(facesT('faces.empty', 'Chưa có khuôn mặt nào.'))}</p>`;
            facesSetCount(0);
            return;
        }

        facesSetCount(faces.length);

        const rows = faces.map(f => {
            const id = Number(f.id);
            const name = f.name || '';
            const age = facesFmtAge(f.secondssincefirstenrolled);
            const display = name || facesT('faces.unnamed', '(chưa đặt tên)');
            const initial = facesInitial(name);
            const renameArg = JSON.stringify(name);

            return `
        <div class="face-card">
            <div class="face-row">
                <div class="face-identity">
                    <div class="face-avatar" aria-hidden="true">${facesEsc(initial)}</div>
                    <div>
                        <div class="face-name">${facesEsc(display)}</div>
                        <div class="face-meta">${facesEsc(facesT('faces.meta', 'id {id} · ghi lần đầu {age} trước')
                            .replace('{id}', String(id))
                            .replace('{age}', age))}</div>
                    </div>
                </div>
                <div class="face-actions">
                    <button type="button" class="face-btn-rename"
                        onclick='facePromptRename(${id}, ${renameArg})'>
                        ${facesEsc(facesT('faces.rename', 'Đổi tên'))}
                    </button>
                    <button type="button" class="face-btn-delete" onclick="faceDelete(${id})">
                        ${facesEsc(facesT('faces.delete', 'Xóa'))}
                    </button>
                </div>
            </div>
        </div>
    `;
        }).join('');

        listEl.innerHTML = rows;
    } catch (e) {
        listEl.innerHTML = '';
        facesSetCount(-1);
        facesSetStatus(facesT('faces.err_load', 'Không tải được khuôn mặt:') + ' ' + e.message, true);
    }
}

async function faceDelete(id) {
    facesSetStatus('', false);
    try {
        const qs = new URLSearchParams({ id: String(id) });
        await facesFetchText(`/api/mods/Faces/deleteFace?${qs.toString()}`);
        facesSetStatus(facesT('faces.deleted', 'Đã xóa.'), false);
        facesRefresh();
    } catch (e) {
        facesSetStatus(facesT('faces.err_delete', 'Xóa thất bại:') + ' ' + e.message, true);
    }
}

async function facePromptRename(id, currentName) {
    facesSetStatus('', false);
    const newName = prompt(
        facesT('faces.rename_prompt', 'Đổi tên khuôn mặt').replace('{id}', String(id)),
        currentName || ''
    );
    if (newName === null) return;

    const trimmed = newName.trim();
    if (!trimmed) {
        facesSetStatus(facesT('faces.err_noname', 'Chưa nhập tên.'), true);
        return;
    }

    try {
        const qs = new URLSearchParams({ id: String(id), name: trimmed });
        await facesFetchText(`/api/mods/Faces/renameFace?${qs.toString()}`);
        facesSetStatus(facesT('faces.renamed', 'Đã đổi tên.'), false);
        facesRefresh();
    } catch (e) {
        facesSetStatus(facesT('faces.err_rename', 'Đổi tên thất bại:') + ' ' + e.message, true);
    }
}

async function faceTrain() {
    facesSetStatus('', false);
    const input = document.getElementById('faceNewName');
    const name = (input.value || '').trim();
    if (!name) {
        facesSetStatus(facesT('faces.err_noname', 'Nhập tên trước.'), true);
        input.focus();
        return;
    }

    try {
        const qs = new URLSearchParams({ name });
        await facesFetchText(`/api/mods/Faces/trainFace?${qs.toString()}`);
        facesSetStatus(facesT('faces.enrolling', 'Đã bắt đầu ghi — nhìn vào Vector (đèn lưng nhấp nháy).'), false);
        input.value = '';
        facesRefresh();
    } catch (e) {
        facesSetStatus(facesT('faces.err_train', 'Ghi thất bại:') + ' ' + e.message, true);
    }
}
