let ctrlAssumed = false;
let ctrlMoving = false;

function setControlStatus(msg) {
    const el = document.getElementById('controlStatus');
    if (el) el.innerHTML = `<p>${msg}</p>`;
}

function ctrlSetDriveEnabled(on) {
    document.querySelectorAll('#control .ctrl-btn').forEach((b) => {
        b.disabled = !on;
    });
    const mirror = document.getElementById('ctrlMirrorSwitch');
    if (mirror) {
        mirror.disabled = !on;
        if (!on) mirror.checked = false;
    }
    const sayIn = document.getElementById('ctrlSayText');
    const sayBtn = document.getElementById('ctrlSayBtn');
    if (sayIn) sayIn.disabled = !on;
    if (sayBtn) sayBtn.disabled = !on;
    const assume = document.getElementById('ctrlAssumeBtn');
    const release = document.getElementById('ctrlReleaseBtn');
    if (assume) assume.classList.toggle('is-active', !!on);
    if (release) release.classList.toggle('is-active', !on);
}

async function ctrlAssume() {
    try {
        setControlStatus('Đang chiếm quyền... (Assuming...)');
        const res = await fetch('/api/mods/Control/assume', { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            setControlStatus(`${j.status || 'error'}: ${j.message || res.status}`);
            return;
        }
        ctrlAssumed = true;
        ctrlSetDriveEnabled(true);
        setControlStatus('Đã chiếm quyền — đang mở camera. (Assumed — opening camera.)');
        ctrlCamStart();
    } catch (e) {
        setControlStatus(`Lỗi mạng (network error): ${e.message}`);
    }
}

async function ctrlRelease() {
    setControlStatus('Đang nhả quyền... (Releasing...)');
    try {
        await ctrlWheels(0, 0);
        await ctrlLift(0);
        await ctrlHead(0);
        await fetch('/api/mods/Control/release', { method: 'POST' });
        ctrlAssumed = false;
        ctrlMoving = false;
        ctrlSetDriveEnabled(false);
        ctrlCamStop();
        setControlStatus('Đã nhả — robot về trạng thái tự nhiên. (Released — freeplay / Xiaozhi.)');
    } catch (e) {
        setControlStatus(`Lỗi mạng (network error): ${e.message}`);
    }
}

async function ctrlWheels(lw, rw) {
    try {
        await fetch(`/api/mods/Control/wheels?lw=${lw}&rw=${rw}`, { method: 'POST' });
    } catch (e) {
        console.log('wheels', e);
    }
}

async function ctrlLift(speed) {
    try {
        await fetch(`/api/mods/Control/lift?speed=${speed}`, { method: 'POST' });
    } catch (e) {
        console.log('lift', e);
    }
}

async function ctrlHead(speed) {
    try {
        await fetch(`/api/mods/Control/head?speed=${speed}`, { method: 'POST' });
    } catch (e) {
        console.log('head', e);
    }
}

function ctrlHold(lw, rw, ev) {
    if (ev) {
        ev.preventDefault();
        try { ev.currentTarget.setPointerCapture(ev.pointerId); } catch (_) {}
    }
    if (!ctrlAssumed) {
        setControlStatus('Cần chiếm quyền trước. (Assume control first.)');
        return;
    }
    ctrlMoving = true;
    ctrlWheels(lw, rw);
}

function ctrlStop(ev) {
    if (ev) ev.preventDefault();
    ctrlMoving = false;
    if (ctrlAssumed) ctrlWheels(0, 0);
}

function ctrlStopAll(ev) {
    if (ev) ev.preventDefault();
    ctrlMoving = false;
    if (!ctrlAssumed) return;
    ctrlWheels(0, 0);
    ctrlLift(0);
    ctrlHead(0);
}

function ctrlLiftHold(speed, ev) {
    if (ev) {
        ev.preventDefault();
        try { ev.currentTarget.setPointerCapture(ev.pointerId); } catch (_) {}
    }
    if (!ctrlAssumed) {
        setControlStatus('Cần chiếm quyền trước. (Assume control first.)');
        return;
    }
    ctrlLift(speed);
}

function ctrlLiftStop(ev) {
    if (ev) ev.preventDefault();
    if (ctrlAssumed) ctrlLift(0);
}

function ctrlHeadHold(speed, ev) {
    if (ev) {
        ev.preventDefault();
        try { ev.currentTarget.setPointerCapture(ev.pointerId); } catch (_) {}
    }
    if (!ctrlAssumed) {
        setControlStatus('Cần chiếm quyền trước. (Assume control first.)');
        return;
    }
    ctrlHead(speed);
}

function ctrlHeadStop(ev) {
    if (ev) ev.preventDefault();
    if (ctrlAssumed) ctrlHead(0);
}

async function ctrlMirror(on) {
    if (!ctrlAssumed) {
        setControlStatus('Cần chiếm quyền trước. (Assume control first.)');
        const sw = document.getElementById('ctrlMirrorSwitch');
        if (sw) sw.checked = false;
        return;
    }
    try {
        const res = await fetch(`/api/mods/Control/mirror?enable=${on}`, { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            setControlStatus(`${j.status || 'error'}: ${j.message || res.status}`);
            const sw = document.getElementById('ctrlMirrorSwitch');
            if (sw) sw.checked = !on;
            return;
        }
        const sw = document.getElementById('ctrlMirrorSwitch');
        if (sw) sw.checked = !!on;
        setControlStatus(on ? 'Gương bật (Mirror ON)' : 'Gương tắt (Mirror OFF)');
    } catch (e) {
        setControlStatus(`Lỗi mạng (network error): ${e.message}`);
        const sw = document.getElementById('ctrlMirrorSwitch');
        if (sw) sw.checked = !on;
    }
}

function ctrlMirrorToggle(on) {
    ctrlMirror(!!on);
}

function ctrlCamPlaceholderHTML() {
    return '<div class="ctrl-cam-placeholder">Camera tắt — bật switch bên dưới hoặc Chiếm quyền.<br><em>Camera off — flip the switch below or Assume.</em></div>';
}

function ctrlSetCamSwitch(on) {
    const sw = document.getElementById('ctrlCamSwitch');
    if (sw) sw.checked = !!on;
}

function ctrlCamToggle(on) {
    if (on) ctrlCamStart();
    else ctrlCamStop();
}

function ctrlCamStart() {
    const box = document.getElementById('ctrlCamBox');
    if (!box) return;
    box.innerHTML = '';
    const img = document.createElement('img');
    img.alt = 'camera';
    img.decoding = 'sync';
    img.loading = 'eager';
    // Bust caches / prior MJPEG connections when restarting.
    img.src = '/api/mods/Control/cam-stream?t=' + Date.now();
    box.appendChild(img);
    ctrlSetCamSwitch(true);
    setControlStatus('Camera bật (Camera ON)');
}

function ctrlCamStop() {
    const box = document.getElementById('ctrlCamBox');
    if (box) {
        box.innerHTML = ctrlCamPlaceholderHTML();
    }
    ctrlSetCamSwitch(false);
    fetch('/api/mods/Control/stop_cam', { method: 'POST' }).catch(() => {});
    setControlStatus('Camera tắt (Camera OFF)');
}

window.addEventListener('beforeunload', () => {
    if (ctrlAssumed) {
        navigator.sendBeacon('/api/mods/Control/release');
    }
});

async function ctrlSayText() {
    const inp = document.getElementById('ctrlSayText');
    const text = (inp && inp.value || '').trim();
    if (!text) {
        setControlStatus('Nhập nội dung trước. (Enter text first.)');
        return;
    }
    if (!ctrlAssumed) {
        setControlStatus('Cần Chiếm quyền trước. (Assume control first.)');
        return;
    }
    try {
        setControlStatus('Đang gửi Say Text... (Saying...)');
        const res = await fetch('/api/mods/Control/say_text?text=' + encodeURIComponent(text), { method: 'POST' });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            setControlStatus(`${j.status || 'error'}: ${j.message || res.status}`);
            return;
        }
        setControlStatus('Đã gửi Say Text. (Sent.)');
    } catch (e) {
        setControlStatus(`Lỗi mạng: ${e.message}`);
    }
}

let ctrlProcessedAudioBlob = null;

document.addEventListener('DOMContentLoaded', () => {
    const fileInput = document.getElementById('ctrlAudioFile');
    if (!fileInput) return;
    fileInput.addEventListener('change', async () => {
        ctrlProcessedAudioBlob = null;
        const sendBtn = document.getElementById('ctrlAudioSendBtn');
        if (sendBtn) sendBtn.style.display = 'none';
        if (!fileInput.files.length) return;
        const file = fileInput.files[0];
        try {
            setControlStatus('Đang xử lý WAV… (Processing WAV…)');
            const arrayBuffer = await file.arrayBuffer();
            const audioContext = new (window.AudioContext || window.webkitAudioContext)();
            const audioBuffer = await audioContext.decodeAudioData(arrayBuffer);
            let mono = audioBuffer;
            if (audioBuffer.numberOfChannels > 1) {
                mono = audioContext.createBuffer(1, audioBuffer.length, audioBuffer.sampleRate);
                const out = mono.getChannelData(0);
                const c0 = audioBuffer.getChannelData(0);
                const c1 = audioBuffer.getChannelData(1);
                for (let i = 0; i < audioBuffer.length; i++) out[i] = 0.5 * (c0[i] + c1[i]);
            }
            const newSampleRate = 8000;
            const newLength = Math.round(mono.length * newSampleRate / mono.sampleRate);
            const resampled = audioContext.createBuffer(1, newLength, newSampleRate);
            const oldData = mono.getChannelData(0);
            const newData = resampled.getChannelData(0);
            for (let i = 0; i < newLength; i++) {
                const oldIndex = i * mono.sampleRate / newSampleRate;
                const i0 = Math.floor(oldIndex);
                const i1 = Math.min(i0 + 1, oldData.length - 1);
                const f = oldIndex - i0;
                newData[i] = oldData[i0] * (1 - f) + oldData[i1] * f;
            }
            ctrlProcessedAudioBlob = ctrlBufferToWave(resampled);
            const prev = document.getElementById('ctrlAudioPreview');
            if (prev) {
                prev.src = URL.createObjectURL(ctrlProcessedAudioBlob);
            }
            if (sendBtn) sendBtn.style.display = 'inline-block';
            setControlStatus('WAV sẵn sàng — bấm Gửi file. (Ready — Send Audio.)');
        } catch (e) {
            setControlStatus(`Lỗi xử lý audio: ${e.message}`);
        }
    });
});

function ctrlBufferToWave(abuffer) {
    const numOfChannels = 1;
    const length = abuffer.length * numOfChannels * 2 + 44;
    const buffer = new ArrayBuffer(length);
    const view = new DataView(buffer);
    let offset = 0;
    const setStr = (o, s) => { for (let i = 0; i < s.length; i++) view.setUint8(o + i, s.charCodeAt(i)); };
    setStr(offset, 'RIFF'); offset += 4;
    view.setUint32(offset, length - 8, true); offset += 4;
    setStr(offset, 'WAVE'); offset += 4;
    setStr(offset, 'fmt '); offset += 4;
    view.setUint32(offset, 16, true); offset += 4;
    view.setUint16(offset, 1, true); offset += 2;
    view.setUint16(offset, numOfChannels, true); offset += 2;
    view.setUint32(offset, 8000, true); offset += 4;
    view.setUint32(offset, 8000 * numOfChannels * 2, true); offset += 4;
    view.setUint16(offset, numOfChannels * 2, true); offset += 2;
    view.setUint16(offset, 16, true); offset += 2;
    setStr(offset, 'data'); offset += 4;
    view.setUint32(offset, length - offset - 4, true); offset += 4;
    const channelData = abuffer.getChannelData(0);
    for (let i = 0; i < channelData.length; i++) {
        view.setInt16(offset, channelData[i] * 0x7FFF, true);
        offset += 2;
    }
    return new Blob([buffer], { type: 'audio/wav' });
}

async function ctrlSendAudio() {
    if (!ctrlProcessedAudioBlob) {
        setControlStatus('Chưa có file đã xử lý. (No processed audio.)');
        return;
    }
    try {
        setControlStatus('Đang gửi audio… (Uploading…)');
        const fd = new FormData();
        fd.append('sound', ctrlProcessedAudioBlob, 'processed.wav');
        const res = await fetch('/api/mods/Control/play_sound', { method: 'POST', body: fd });
        const j = await res.json().catch(() => ({}));
        if (!res.ok) {
            setControlStatus(`${j.status || 'error'}: ${j.message || res.status}`);
            return;
        }
        setControlStatus('Đã phát audio trên robot. (Playing on robot.)');
    } catch (e) {
        setControlStatus(`Lỗi gửi audio: ${e.message}`);
    }
}
