const CHESS_GLYPH = {
    // White outline pieces (U+2654–2659) vs black filled (U+265A–265F)
    K: '♔', Q: '♕', R: '♖', B: '♗', N: '♘', P: '♙',
    k: '♚', q: '♛', r: '♜', b: '♝', n: '♞', p: '♟',
};

// Traditional set: Red uses person-radical variants where they differ.
const XQ_GLYPH = {
    K: '帥', A: '仕', E: '相', H: '傌', R: '俥', C: '炮', P: '兵',
    k: '將', a: '士', e: '象', h: '馬', r: '車', c: '砲', p: '卒',
};

const GAMES_STORAGE_KEY = 'wireos.games.session';

function gT(key, fallback) {
    if (typeof t === 'function') return t(key, fallback);
    return fallback != null ? fallback : key;
}

const GAMES_META = {
    chess: {
        get title() { return gT('games.title_chess', '♟️ Cờ vua'); },
        get nav() { return gT('games.chess', 'Cờ vua'); },
        get short() { return gT('games.chess', 'Cờ vua'); },
        api: 'Chess',
        ico: '♟️',
        helpKeys: ['games.chess_help1', 'games.chess_help2', 'games.chess_help3', 'games.chess_help4'],
        hintKey: 'games.hint_chess',
        get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
        get helpItems() {
            return this.helpKeys.map((k) => gT(k, ''));
        },
        get hint() { return gT('games.hint_chess', 'Chọn quân → chọn ô đến. Phong hậu tự động.'); },
    },
    xiangqi: {
        get title() { return gT('games.title_xq', '🀄 Cờ tướng'); },
        get nav() { return gT('games.xiangqi', 'Cờ tướng'); },
        get short() { return gT('games.xiangqi', 'Cờ tướng'); },
        api: 'Xiangqi',
        ico: '🀄',
        helpKeys: ['games.xq_help1', 'games.xq_help2', 'games.xq_help3', 'games.xq_help4'],
        hintKey: 'games.hint_xq',
        get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
        get helpItems() {
            return this.helpKeys.map((k) => gT(k, ''));
        },
        get hint() { return gT('games.hint_xq', 'Chọn quân → chọn ô đến. Đỏ đi trước.'); },
    },
    caro: {
        get title() { return gT('games.title_caro', '⭕ Cờ caro'); },
        get nav() { return gT('games.caro', 'Cờ caro'); },
        get short() { return gT('games.caro', 'Cờ caro'); },
        api: 'Caro',
        ico: '⭕',
        helpKeys: ['games.caro_help1', 'games.caro_help2', 'games.caro_help3', 'games.caro_help4'],
        hintKey: 'games.hint_caro',
        get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
        get helpItems() {
            return this.helpKeys.map((k) => gT(k, ''));
        },
        get hint() { return gT('games.hint_caro', 'Bấm ô để đặt X. Cấm 2 đầu (X).'); },
    },
    connect4: {
        get title() { return gT('games.title_c4', '🔴 Connect Four'); },
        get nav() { return gT('games.connect4', 'Connect Four'); },
        get short() { return gT('games.connect4', 'Connect Four'); },
        api: 'Connect4',
        ico: '🔴',
        helpKeys: ['games.c4_help1', 'games.c4_help2', 'games.c4_help3', 'games.c4_help4'],
        hintKey: 'games.hint_c4',
        get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
        get helpItems() {
            return this.helpKeys.map((k) => gT(k, ''));
        },
        get hint() { return gT('games.hint_c4', 'Bấm cột để thả quân.'); },
    },
    reversi: {
        get title() { return gT('games.title_rev', '⚫ Reversi'); },
        get nav() { return gT('games.reversi', 'Reversi'); },
        get short() { return gT('games.reversi', 'Reversi'); },
        api: 'Reversi',
        ico: '⚫',
        helpKeys: ['games.rev_help1', 'games.rev_help2', 'games.rev_help3', 'games.rev_help4'],
        hintKey: 'games.hint_rev',
        get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
        get helpItems() {
            return this.helpKeys.map((k) => gT(k, ''));
        },
        get hint() { return gT('games.hint_rev', 'Bấm ô hợp lệ để lật quân.'); },
    },
    checkers: {
        get title() { return gT('games.title_ck', '⬛ Cờ đam'); },
        get nav() { return gT('games.checkers', 'Cờ đam'); },
        get short() { return gT('games.checkers', 'Cờ đam'); },
        api: 'Checkers',
        ico: '⬛',
        helpKeys: ['games.ck_help1', 'games.ck_help2', 'games.ck_help3', 'games.ck_help4'],
        hintKey: 'games.hint_ck',
        get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
        get helpItems() {
            return this.helpKeys.map((k) => gT(k, ''));
        },
        get hint() { return gT('games.hint_ck', 'Chọn quân → ô đến. Ăn bắt buộc.'); },
    },
    go9: {
        get title() { return gT('games.title_go', '⚪ Cờ vây 9×9'); },
        get nav() { return gT('games.go9', 'Cờ vây'); },
        get short() { return gT('games.go9', 'Cờ vây'); },
        api: 'Go9',
        ico: '⚪',
        helpKeys: ['games.go_help1', 'games.go_help2', 'games.go_help3', 'games.go_help4'],
        hintKey: 'games.hint_go',
        get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
        get helpItems() {
            return this.helpKeys.map((k) => gT(k, ''));
        },
        get hint() { return gT('games.hint_go', 'Bấm ô đặt quân hoặc Pass.'); },
    },
};

let chessState = null;
let chessSelected = null;
let chessLegal = [];
let gamesXiaozhiAvailable = false;
let gamesGoogleVIAvailable = false;
let gamesCommentMode = 'saytext';
let gamesDifficulty = 'medium';
let gamesSession = loadGamesSession();

function normalizeDifficulty(d) {
    d = String(d || '').toLowerCase();
    if (d === 'easy' || d === 'de' || d === 'dễ' || d === '1') return 'easy';
    if (d === 'hard' || d === 'kho' || d === 'khó' || d === '3') return 'hard';
    return 'medium';
}

function gamesDifficultyLabel(d) {
    d = normalizeDifficulty(d);
    if (d === 'easy') return gT('games.diff_easy', 'Dễ');
    if (d === 'hard') return gT('games.diff_hard', 'Khó');
    return gT('games.diff_med', 'Trung bình');
}

function normalizeCommentMode(mode) {
    if (mode === 'xiaozhi') return 'xiaozhi';
    if (mode === 'google_vi' || mode === 'google' || mode === 'google-vi') return 'google_vi';
    return 'saytext';
}

function normalizeGameId(id) {
    if (id && GAMES_META[id]) return id;
    return null;
}

function loadGamesSession() {
    try {
        const raw = sessionStorage.getItem(GAMES_STORAGE_KEY);
        if (!raw) {
            return { pendingGame: null, activeGame: null, commentMode: 'saytext', difficulty: 'medium' };
        }
        const j = JSON.parse(raw);
        return {
            pendingGame: normalizeGameId(j.pendingGame),
            activeGame: normalizeGameId(j.activeGame),
            commentMode: normalizeCommentMode(j.commentMode),
            difficulty: normalizeDifficulty(j.difficulty),
        };
    } catch (_) {
        return { pendingGame: null, activeGame: null, commentMode: 'saytext', difficulty: 'medium' };
    }
}

function saveGamesSession() {
    try {
        sessionStorage.setItem(GAMES_STORAGE_KEY, JSON.stringify(gamesSession));
    } catch (_) {}
}

function chessSetMeta(msg) {
    const el = document.getElementById('chessMeta');
    if (el) el.textContent = msg || '';
}

function gamesActiveId() {
    return gamesSession.activeGame || gamesSession.pendingGame || 'chess';
}

function gamesApiBase() {
    const id = gamesActiveId();
    const meta = GAMES_META[id] || GAMES_META.chess;
    return '/api/mods/' + meta.api + '/';
}

async function gamesFetch(path, opts) {
    const res = await fetch(gamesApiBase() + path, opts || {});
    const j = await res.json().catch(() => ({}));
    if (!res.ok) {
        throw new Error(j.message || j.status || res.status);
    }
    return j;
}

/** Caps (comment modes) always come from Chess mod — shared announce path. */
async function chessFetchCaps(path, opts) {
    const res = await fetch('/api/mods/Chess/' + path, opts || {});
    const j = await res.json().catch(() => ({}));
    if (!res.ok) {
        throw new Error(j.message || j.status || res.status);
    }
    return j;
}

function gamesModeLabel(mode) {
    mode = normalizeCommentMode(mode);
    if (mode === 'xiaozhi') return gT('games.mode_xz', 'Giọng robot + Xiaozhi');
    if (mode === 'google_vi') return gT('games.mode_gvi', 'SayText Google (VI)');
    return gT('games.mode_st', 'Giọng robot bằng tiếng Anh');
}

function gamesFocusId() {
    return gamesSession.activeGame || gamesSession.pendingGame || null;
}

function gamesUpdateNavLabel() {
    const btn = document.getElementById('navGamesBtn');
    const sel = document.getElementById('navSelect');
    const opt = sel ? sel.querySelector('option[value="#games"]') : null;
    const id = gamesFocusId();
    let label = 'Game';
    let optLabel = 'Game';
    if (id && GAMES_META[id]) {
        label = GAMES_META[id].nav;
        optLabel = GAMES_META[id].short + ' (Game)';
    }
    if (btn) btn.textContent = label;
    if (opt) opt.textContent = optLabel;
}

function gamesUpdateHeading() {
    const h = document.getElementById('gamesHeading');
    if (!h) return;
    const id = gamesFocusId();
    if (id && GAMES_META[id]) {
        h.textContent = GAMES_META[id].title;
        h.removeAttribute('data-i18n');
    } else {
        h.setAttribute('data-i18n', 'games.title_lobby');
        h.textContent = gT('games.title_lobby', '🎮 Game');
    }
}

function gamesUpdatePlayHelp() {
    const id = gamesSession.activeGame || gamesSession.pendingGame || 'chess';
    const meta = GAMES_META[id] || GAMES_META.chess;
    const title = document.getElementById('gamesPlayHelpTitle');
    const list = document.getElementById('gamesPlayHelpList');
    const hint = document.getElementById('gamesPlayHint');
    const ico = document.getElementById('gamesPlayHelpIco');
    if (title) {
        title.textContent = meta.helpTitle || gT('games.guide', 'Hướng dẫn');
        title.setAttribute('data-i18n', 'games.guide');
    }
    if (ico) ico.textContent = meta.ico || '🎮';
    if (list) {
        const keys = meta.helpKeys || [];
        if (keys.length) {
            list.innerHTML = keys.map((k) => {
                const html = gT(k, '');
                return '<li data-i18n="' + k + '" data-i18n-html="1">' + html + '</li>';
            }).join('');
        } else {
            list.innerHTML = (meta.helpItems || []).map((tHtml) => '<li data-i18n-html="1">' + tHtml + '</li>').join('');
        }
        if (window.I18N && typeof window.I18N.apply === 'function') {
            window.I18N.apply(list);
        }
    }
    if (hint) {
        const hk = meta.hintKey || null;
        if (hk) {
            hint.setAttribute('data-i18n', hk);
            hint.textContent = gT(hk, meta.hint || '');
        } else {
            hint.textContent = meta.hint || '';
        }
    }
}

function gamesShowStep(step) {
    const pick = document.getElementById('gamesStepPick');
    const mode = document.getElementById('gamesStepMode');
    const chess = document.getElementById('gamesStepChess');
    if (pick) pick.hidden = step !== 'pick';
    if (mode) mode.hidden = step !== 'mode';
    if (chess) chess.hidden = step !== 'chess';
    gamesUpdateHeading();
    gamesUpdateNavLabel();
    if (step === 'chess') gamesUpdatePlayHelp();
}

function gamesClampMode(mode) {
    mode = normalizeCommentMode(mode);
    if (mode === 'xiaozhi' && !gamesXiaozhiAvailable) return 'saytext';
    if (mode === 'google_vi' && !gamesGoogleVIAvailable) return 'saytext';
    return mode;
}

function gamesRenderModeUI() {
    const mode = gamesClampMode(gamesCommentMode);
    gamesCommentMode = mode;

    const xzBtn = document.getElementById('chessModeXiaozhiBtn');
    const stBtn = document.getElementById('chessModeSayTextBtn');
    const gviBtn = document.getElementById('chessModeGoogleViBtn');
    const btns = document.querySelector('#gamesStepMode .chess-mode-btns');

    if (xzBtn) {
        xzBtn.disabled = !gamesXiaozhiAvailable;
        xzBtn.classList.toggle('active', mode === 'xiaozhi');
    }
    if (stBtn) stBtn.classList.toggle('active', mode === 'saytext');
    if (gviBtn) {
        gviBtn.hidden = !gamesGoogleVIAvailable;
        gviBtn.disabled = !gamesGoogleVIAvailable;
        gviBtn.classList.toggle('active', mode === 'google_vi');
    }
    if (btns) {
        btns.classList.toggle('has-google', !!gamesGoogleVIAvailable);
    }

    const status = document.getElementById('chessModeStatus');
    const hint = document.getElementById('chessModeHint');
    const gHint = document.getElementById('chessModeGoogleHint');
    if (gHint) gHint.hidden = gamesGoogleVIAvailable;

    if (mode === 'xiaozhi') {
        if (status) status.textContent = gT('games.mode_picked_prefix', 'Đã chọn:') + ' ' + gamesModeLabel(mode);
        if (hint) {
            hint.textContent = gT('games.hint_xz', 'Nước đi → mở/giữ phiên Xiaozhi, robot nói rồi mở mic hỏi tiếp.');
        }
    } else if (mode === 'google_vi') {
        if (status) status.textContent = gT('games.mode_picked_prefix', 'Đã chọn:') + ' ' + gamesModeLabel(mode);
        if (hint) {
            hint.textContent = gT('games.hint_gvi_on', 'Bình luận tiếng Việt bằng Google TTS. Không mở mic hội thoại.');
        }
    } else {
        if (status) status.textContent = gT('games.mode_picked_prefix', 'Đã chọn:') + ' ' + gamesModeLabel(mode);
        if (hint) {
            hint.textContent = gamesXiaozhiAvailable
                ? gT('games.hint_st', 'Robot đọc nước đi bằng giọng tiếng Anh (SayText).')
                : gT('games.hint_st_vosk', 'Xiaozhi đang tắt (Vosk) — giọng robot tiếng Anh / Google VI (nếu bật).');
        }
    }

    const gameLbl = document.getElementById('gamesPickedGameLabel');
    const gid = gamesSession.pendingGame || gamesSession.activeGame;
    if (gameLbl && gid && GAMES_META[gid]) {
        gameLbl.textContent = GAMES_META[gid].short;
    }
    gamesRenderDifficultyUI();
}

function gamesRenderDifficultyUI() {
    const d = normalizeDifficulty(gamesDifficulty);
    gamesDifficulty = d;
    document.querySelectorAll('#gamesDiffRow .games-diff-btn').forEach((btn) => {
        const level = btn.getAttribute('data-level');
        btn.classList.toggle('active', normalizeDifficulty(level) === d);
    });
    const lbl = document.getElementById('gamesDiffStatus');
    if (lbl) {
        lbl.textContent = gT('games.diff_status_prefix', 'Cấp độ:') + ' ' + gamesDifficultyLabel(d);
    }
}

async function gamesRefreshCaps() {
    try {
        const st = await chessFetchCaps('comment');
        gamesXiaozhiAvailable = !!st.xiaozhiAvailable;
        gamesGoogleVIAvailable = !!st.googleVIAvailable;
        if (gamesSession.activeGame || gamesSession.pendingGame) {
            gamesCommentMode = gamesClampMode(gamesSession.commentMode);
        } else {
            gamesCommentMode = 'saytext';
        }
        gamesRenderModeUI();
    } catch (_) {
        gamesXiaozhiAvailable = false;
        gamesGoogleVIAvailable = false;
        gamesCommentMode = 'saytext';
        gamesRenderModeUI();
    }
}

async function gamesApplyDifficulty(level) {
    level = normalizeDifficulty(level);
    gamesDifficulty = level;
    gamesSession.difficulty = level;
    saveGamesSession();
    try {
        await gamesFetch('difficulty?level=' + encodeURIComponent(level));
    } catch (_) {
        try {
            await chessFetchCaps('difficulty?level=' + encodeURIComponent(level));
        } catch (__) {}
    }
    gamesRenderDifficultyUI();
    gamesRenderModeUI();
}

async function gamesApplyCommentMode(mode) {
    mode = gamesClampMode(mode);
    gamesCommentMode = mode;
    try {
        await chessFetchCaps('comment_mode?mode=' + encodeURIComponent(mode));
    } catch (_) {}
    gamesRenderModeUI();
}

function gamesPickGame(gameId) {
    gameId = normalizeGameId(gameId);
    if (!gameId) return;
    gamesSession.pendingGame = gameId;
    gamesSession.activeGame = null;
    gamesSession.commentMode = gamesCommentMode;
    gamesSession.difficulty = normalizeDifficulty(gamesDifficulty || gamesSession.difficulty);
    gamesDifficulty = gamesSession.difficulty;
    saveGamesSession();
    gamesShowStep('mode');
    gamesRenderModeUI();
}

function gamesBackToPick() {
    gamesSession.pendingGame = null;
    gamesSession.activeGame = null;
    saveGamesSession();
    gamesShowStep('pick');
    gamesUpdateHeading();
    gamesUpdateNavLabel();
}

async function gamesStartPlay() {
    const gameId = normalizeGameId(gamesSession.pendingGame) || 'chess';
    await gamesApplyCommentMode(gamesCommentMode);
    gamesDifficulty = normalizeDifficulty(gamesDifficulty || gamesSession.difficulty);
    gamesSession.commentMode = gamesCommentMode;
    gamesSession.difficulty = gamesDifficulty;
    gamesSession.pendingGame = gameId;
    gamesSession.activeGame = gameId;
    saveGamesSession();
    gamesShowStep('chess');
    gamesRenderModeUI();
    await chessNewGame();
}

async function gamesExitToLobby() {
    const gameId = gamesSession.activeGame || gamesSession.pendingGame || 'chess';
    const apiName = (GAMES_META[gameId] && GAMES_META[gameId].api) || gameId;
    try {
        await gamesFetch('exit?game=' + encodeURIComponent(apiName), { method: 'POST' });
    } catch (e) {
        chessSetMeta(gT('games.exit_err', 'Thoát game') + ': ' + e.message);
    }
    gamesSession.activeGame = null;
    gamesSession.pendingGame = null;
    saveGamesSession();
    gamesShowStep('pick');
    chessState = null;
    chessSelected = null;
    gamesUpdateHeading();
    gamesUpdateNavLabel();
}

async function gamesEnterTab() {
    await gamesRefreshCaps();
    const active = normalizeGameId(gamesSession.activeGame);
    const pending = normalizeGameId(gamesSession.pendingGame);
    if (active) {
        gamesShowStep('chess');
        gamesRenderModeUI();
        await chessLoad();
        return;
    }
    if (pending) {
        gamesShowStep('mode');
        gamesRenderModeUI();
        return;
    }
    gamesShowStep('pick');
    gamesUpdateHeading();
    gamesUpdateNavLabel();
}

function chessIsXiangqi() {
    return gamesActiveId() === 'xiangqi';
}

function chessIsPlaceMode() {
    const id = gamesActiveId();
    return id === 'caro' || id === 'connect4' || id === 'reversi' || id === 'go9' ||
        !!(chessState && chessState.placeMode);
}

function chessIsMoveMode() {
    const id = gamesActiveId();
    return id === 'chess' || id === 'xiangqi' || id === 'checkers' ||
        !!(chessState && chessState.moveMode);
}

function gamesIsOver(st) {
    if (!st) return true;
    const s = st.status;
    return s === 'checkmate' || s === 'stalemate' || s === 'win' || s === 'won' || s === 'draw';
}

function chessRender(st) {
    chessState = st;
    const board = document.getElementById('chessBoard');
    if (!board) return;
    board.innerHTML = '';
    const id = gamesActiveId();
    board.className = 'chess-board';
    board.classList.toggle('xiangqi-board', id === 'xiangqi');
    board.classList.toggle('grid-board', chessIsPlaceMode() || id === 'checkers');
    board.classList.toggle('caro-board', id === 'caro');
    board.classList.toggle('c4-board', id === 'connect4');
    board.classList.toggle('go-board', id === 'go9');
    board.classList.toggle('checkers-board', id === 'checkers');

    const passBtn = document.getElementById('gamesPassBtn');
    if (passBtn) {
        passBtn.hidden = !(id === 'go9' || id === 'reversi');
    }

    if (id === 'xiangqi') {
        chessRenderXiangqi(board, st);
        chessRenderCoords({ files: 9, ranks: [9, 8, 7, 6, 5, 4, 3, 2, 1, 0] });
    } else if (id === 'chess') {
        chessRenderChess(board, st);
        chessRenderCoords({ files: 8, ranks: [8, 7, 6, 5, 4, 3, 2, 1] });
    } else if (id === 'checkers') {
        chessRenderCheckers(board, st);
        chessRenderCoords({ files: 8, ranks: [8, 7, 6, 5, 4, 3, 2, 1] });
    } else if (id === 'connect4') {
        chessRenderGrid(board, st, { files: 7, ranks: [5, 4, 3, 2, 1, 0], drop: true, checker: true, showHints: false, showLast: false });
        chessRenderCoords({ files: 7, ranks: [5, 4, 3, 2, 1, 0], matchBoard: true });
    } else if (id === 'caro') {
        const ranks = [];
        for (let i = 14; i >= 0; i--) ranks.push(i);
        // Flat even grid, no legal-hint/last recolor (stable lines while playing).
        chessRenderGrid(board, st, { files: 15, ranks, disc: true, checker: false, showHints: false, showLast: false });
        chessRenderCoords({ files: 15, ranks, matchBoard: true, narrow: true });
    } else if (id === 'go9') {
        const ranks = [];
        for (let i = 8; i >= 0; i--) ranks.push(i);
        chessRenderGrid(board, st, { files: 9, ranks, stone: true, checker: false, showHints: false, showLast: false });
        chessRenderCoords({ files: 9, ranks, matchBoard: true });
    } else if (id === 'reversi') {
        const ranks = [];
        for (let i = 7; i >= 0; i--) ranks.push(i);
        chessRenderGrid(board, st, { files: 8, ranks, disc: true, checker: true, showHints: false, showLast: false });
        chessRenderCoords({ files: 8, ranks, matchBoard: true });
    } else {
        chessRenderChess(board, st);
        chessRenderCoords({ files: 8, ranks: [8, 7, 6, 5, 4, 3, 2, 1] });
    }
}

function chessRenderGrid(board, st, opts) {
    const rows = st.board || [];
    const nFiles = opts.files;
    const ranks = opts.ranks;
    let last = st.lastMove || '';
    if (last === 'pass') last = '';
    // Stable layout: no checker "hints" on place boards (would recolor almost every empty cell).
    const showHints = !!opts.showHints;
    const showLast = opts.showLast !== false && !!opts.showLast;
    // grid: CSS variables — equal cells
    board.style.setProperty('--cols', String(nFiles));
    board.style.setProperty('--rows', String(ranks.length));
    for (let ri = 0; ri < ranks.length; ri++) {
        const rank = ranks[ri];
        const row = rows[ri] || '';
        for (let file = 0; file < nFiles; file++) {
            const dropName = String.fromCharCode(97 + file);
            const sqName = opts.drop ? (dropName + String(rank)) : (dropName + String(rank));
            const clickKey = opts.drop ? dropName : sqName;
            const ch = row[file] === '.' || row[file] == null ? '' : row[file];
            const sq = document.createElement('button');
            sq.type = 'button';
            sq.className = 'chess-sq grid-sq';
            if (opts.drop) sq.classList.add('c4-sq');
            // Flat cells for place games keep borders even; optional checker for reverse/c4.
            if (opts.checker) {
                if (((rank + file) % 2 === 1)) sq.classList.add('dark');
                else sq.classList.add('light');
            } else {
                sq.classList.add('flat');
            }
            sq.dataset.sq = clickKey;
            if (ch) {
                const span = document.createElement('span');
                if (opts.stone) {
                    span.className = 'chess-piece go-stone ' + (ch === 'X' || ch === 'x' ? 'go-black' : 'go-white');
                    span.textContent = ch === 'X' || ch === 'x' ? '●' : '○';
                } else if (opts.disc || ch === 'X' || ch === 'O') {
                    span.className = 'chess-piece disc ' + (ch === 'X' || ch === 'x' ? 'disc-x' : 'disc-o');
                    span.textContent = ch;
                } else {
                    span.className = 'chess-piece';
                    span.textContent = ch;
                }
                sq.appendChild(span);
            }
            if (showHints && (chessLegal.includes(clickKey) || chessLegal.includes(sqName))) {
                sq.classList.add('hint');
            }
            if (showLast && last && (last === clickKey || last === sqName || (opts.drop && last === dropName))) {
                sq.classList.add('last');
            }
            sq.onclick = () => chessClick(clickKey);
            board.appendChild(sq);
        }
    }
}

function chessRenderCheckers(board, st) {
    const rows = st.board || [];
    let lastFrom = '', lastTo = '';
    if (st.lastMove && st.lastMove.length >= 4) {
        lastFrom = st.lastMove.slice(0, 2);
        lastTo = st.lastMove.slice(2, 4);
    }
    board.style.setProperty('--cols', '8');
    board.style.setProperty('--rows', '8');
    const glyph = { w: '○', W: '◎', b: '●', B: '◉' };
    for (let rank = 8; rank >= 1; rank--) {
        const row = rows[8 - rank] || '........';
        for (let file = 0; file < 8; file++) {
            const name = String.fromCharCode(97 + file) + rank;
            const ch = row[file] === '.' ? '' : row[file];
            const sq = document.createElement('button');
            sq.type = 'button';
            sq.className = 'chess-sq ' + (((rank + file) % 2 === 1) ? 'dark' : 'light');
            sq.dataset.sq = name;
            if (ch) {
                const span = document.createElement('span');
                span.className = 'chess-piece ck-piece ' + (ch === 'w' || ch === 'W' ? 'ck-white' : 'ck-black');
                span.textContent = glyph[ch] || ch;
                sq.appendChild(span);
            }
            if (chessSelected === name) sq.classList.add('selected');
            if (chessSelected && chessLegal.some((m) => m.startsWith(chessSelected) && m.slice(2, 4) === name)) {
                sq.classList.add('hint');
            }
            if (name === lastFrom || name === lastTo) sq.classList.add('last');
            sq.onclick = () => chessClick(name);
            board.appendChild(sq);
        }
    }
}

/** File letters a… + rank numbers matching how Vector speaks squares (e.g. e7). */
function chessRenderCoords(opts) {
    const ranksEl = document.getElementById('chessRanks');
    const filesEl = document.getElementById('chessFiles');
    const stage = document.getElementById('chessStage');
    const board = document.getElementById('chessBoard');
    if (!ranksEl || !filesEl) return;
    const nFiles = opts.files || 8;
    const ranks = opts.ranks || [8, 7, 6, 5, 4, 3, 2, 1];
    const xq = !!chessIsXiangqi();
    if (stage) {
        stage.classList.toggle('xiangqi-stage', xq);
        stage.classList.toggle('grid-stage', !!opts.matchBoard && !xq);
        stage.classList.toggle('caro-stage', gamesActiveId() === 'caro');
    }

    ranksEl.innerHTML = '';
    ranksEl.classList.remove('xq-coords-ranks');
    ranksEl.style.gridTemplateRows = 'repeat(' + ranks.length + ', 1fr)';
    ranks.forEach((r) => {
        const d = document.createElement('span');
        d.className = 'chess-coord chess-coord-rank';
        d.textContent = String(r);
        ranksEl.appendChild(d);
    });

    filesEl.innerHTML = '';
    filesEl.classList.remove('xq-coords-files');
    filesEl.style.gridTemplateColumns = 'repeat(' + nFiles + ', 1fr)';
    for (let f = 0; f < nFiles; f++) {
        const d = document.createElement('span');
        d.className = 'chess-coord chess-coord-file';
        d.textContent = String.fromCharCode(97 + f);
        filesEl.appendChild(d);
    }

    // Keep rank/file labels aligned to play surface (critical for 15×15 caro).
    const sync = () => {
        if (!board || !ranksEl) return;
        if (xq) {
            const cs = window.getComputedStyle(board);
            const bt = parseFloat(cs.borderTopWidth) || 0;
            const mt = parseFloat(cs.marginTop) || 0;
            ranksEl.style.height = board.clientHeight + 'px';
            ranksEl.style.marginTop = (mt + bt) + 'px';
            return;
        }
        if (opts.matchBoard) {
            const h = board.offsetHeight;
            const w = board.offsetWidth;
            ranksEl.style.height = h + 'px';
            ranksEl.style.marginTop = '0';
            ranksEl.style.width = opts.narrow ? '18px' : '16px';
            if (filesEl) {
                filesEl.style.width = w + 'px';
                filesEl.style.maxWidth = w + 'px';
                filesEl.style.boxSizing = 'border-box';
            }
        } else {
            ranksEl.style.height = '';
            ranksEl.style.marginTop = '';
            ranksEl.style.width = '';
            if (filesEl) {
                filesEl.style.width = '';
                filesEl.style.maxWidth = '';
            }
        }
    };
    requestAnimationFrame(() => {
        sync();
        requestAnimationFrame(sync);
    });
}

function chessRenderChess(board, st) {
    const rows = st.board || [];
    let lastFrom = '', lastTo = '';
    if (st.lastMove && st.lastMove.length >= 4) {
        lastFrom = st.lastMove.slice(0, 2);
        lastTo = st.lastMove.slice(2, 4);
    }
    for (let rank = 8; rank >= 1; rank--) {
        const row = rows[8 - rank] || '........';
        for (let file = 0; file < 8; file++) {
            const name = String.fromCharCode(97 + file) + rank;
            const ch = row[file] === '.' ? '' : row[file];
            const sq = document.createElement('button');
            sq.type = 'button';
            sq.className = 'chess-sq ' + (((rank + file) % 2 === 1) ? 'dark' : 'light');
            sq.dataset.sq = name;
            if (ch) {
                const span = document.createElement('span');
                span.className = 'chess-piece ' + (ch === ch.toUpperCase() ? 'chess-piece-white' : 'chess-piece-black');
                span.textContent = CHESS_GLYPH[ch] || ch;
                sq.appendChild(span);
            }
            if (chessSelected === name) sq.classList.add('selected');
            if (chessLegal.some((m) => m.startsWith(chessSelected) && m.slice(2, 4) === name)) {
                sq.classList.add('hint');
            }
            if (name === lastFrom || name === lastTo) sq.classList.add('last');
            sq.onclick = () => chessClick(name);
            board.appendChild(sq);
        }
    }
}

function chessRenderXiangqi(board, st) {
    const rows = st.board || [];
    let lastFrom = '', lastTo = '';
    if (st.lastMove && st.lastMove.length >= 4) {
        lastFrom = st.lastMove.slice(0, 2);
        lastTo = st.lastMove.slice(2, 4);
    }
    // Pieces sit on intersections of 9 files × 10 ranks (traditional xiangqi).
    // Files 0..8 map to x = file/8; ranks 9..0 top→bottom map to y = (9-rank)/9.
    for (let rank = 9; rank >= 0; rank--) {
        const row = rows[9 - rank] || '.........';
        for (let file = 0; file < 9; file++) {
            const name = String.fromCharCode(97 + file) + rank;
            const ch = row[file] === '.' ? '' : row[file];
            const sq = document.createElement('button');
            sq.type = 'button';
            sq.className = 'chess-sq xq-sq';
            sq.dataset.sq = name;
            // Percent so centers land on grid intersections (matches board SVG).
            const xPct = (file / 8) * 100;
            const yPct = ((9 - rank) / 9) * 100;
            sq.style.left = xPct + '%';
            sq.style.top = yPct + '%';
            if (ch) {
                const span = document.createElement('span');
                const red = ch === ch.toUpperCase();
                span.className = 'chess-piece xq-piece ' + (red ? 'xq-piece-red' : 'xq-piece-black');
                span.textContent = XQ_GLYPH[ch] || ch;
                sq.appendChild(span);
            }
            if (chessSelected === name) sq.classList.add('selected');
            if (chessLegal.some((m) => m.startsWith(chessSelected) && m.slice(2, 4) === name)) {
                sq.classList.add('hint');
            }
            if (name === lastFrom || name === lastTo) sq.classList.add('last');
            sq.onclick = () => chessClick(name);
            board.appendChild(sq);
        }
    }
}

async function chessRefreshLegal() {
    try {
        const j = await gamesFetch('legal');
        chessLegal = j.moves || [];
    } catch (_) {
        chessLegal = [];
    }
}

function chessMyTurn() {
    if (!chessState || gamesIsOver(chessState)) return false;
    if (chessState.botThinking) return false;
    if (chessIsXiangqi()) return chessState.turn === 'red';
    if (gamesActiveId() === 'chess') return chessState.turn === 'white';
    if (chessState.turn === 'human') return true;
    if (chessState.youAre && chessState.turn === chessState.youAre) return true;
    return chessState.turn === 'white' || chessState.turn === 'red';
}

function chessSleep(ms) {
    return new Promise((r) => setTimeout(r, ms));
}

/** Poll board while bot thinks (human move already shown + spoken). */
async function chessWaitForBot() {
    chessSetMeta(gT('games.bot_think', 'Bot đang suy nghĩ…'));
    for (let i = 0; i < 30; i++) {
        await chessSleep(300);
        try {
            const st = await gamesFetch('state');
            chessSelected = null;
            chessRender(st);
            if (!st.botThinking) {
                if (st.message) {
                    chessSetMeta(st.message);
                } else {
                    chessSetMeta(gT('games.your_turn', 'Đến lượt bạn.'));
                }
                if (!gamesIsOver(st)) {
                    await chessRefreshLegal();
                    chessRender(st);
                }
                return;
            }
            chessSetMeta(st.message || gT('games.bot_think', 'Bot đang suy nghĩ…'));
        } catch (_) { /* keep waiting */ }
    }
    try {
        await chessLoad();
    } catch (_) {}
}

async function chessSubmitMove(uci) {
    chessSelected = null;
    try {
        const st = await gamesFetch('move', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: 'uci=' + encodeURIComponent(uci),
        });
        chessLegal = [];
        chessRender(st);
        if (st.botThinking) {
            chessSetMeta(st.message || gT('games.bot_think', 'Bot đang suy nghĩ…'));
            await chessWaitForBot();
        } else {
            if (st.message) chessSetMeta(st.message);
            if (!gamesIsOver(st)) {
                await chessRefreshLegal();
                chessRender(st);
            }
        }
    } catch (e) {
        chessSetMeta(gT('games.bad_move', 'Nước không hợp lệ') + ': ' + e.message);
        await chessLoad();
    }
}

async function chessClick(sq) {
    if (!chessState || gamesIsOver(chessState)) {
        return;
    }
    if (!chessMyTurn()) {
        chessSetMeta(gT('games.bot_think', 'Bot đang suy nghĩ…'));
        return;
    }
    // Place / drop: one click
    if (chessIsPlaceMode()) {
        if (!chessLegal.length) await chessRefreshLegal();
        let uci = chessLegal.find((m) => m === sq);
        if (!uci && gamesActiveId() === 'connect4') {
            uci = chessLegal.find((m) => m === sq || m.startsWith(sq));
        }
        if (!uci) {
            // allow clicking empty hinted through state
            if (chessLegal.includes(sq)) uci = sq;
        }
        if (!uci) {
            chessSetMeta(gT('games.bad_move', 'Nước không hợp lệ'));
            return;
        }
        await chessSubmitMove(uci);
        return;
    }
    // Move games: select then dest
    if (!chessSelected) {
        const moves = (chessLegal.length ? chessLegal : (await chessRefreshLegal(), chessLegal))
            .filter((m) => m.startsWith(sq));
        if (!moves.length) return;
        chessSelected = sq;
        chessRender(chessState);
        return;
    }
    if (chessSelected === sq) {
        chessSelected = null;
        chessRender(chessState);
        return;
    }
    const uciBase = chessSelected + sq;
    let uci = chessLegal.find((m) => m === uciBase || m.startsWith(uciBase));
    if (!uci) uci = chessLegal.find((m) => m.startsWith(uciBase));
    if (!uci) {
        const rem = chessLegal.filter((m) => m.startsWith(sq));
        if (rem.length) {
            chessSelected = sq;
            chessRender(chessState);
            return;
        }
        chessSelected = null;
        chessRender(chessState);
        return;
    }
    await chessSubmitMove(uci);
}

async function chessLoad() {
    try {
        const st = await gamesFetch('state');
        await chessRefreshLegal();
        chessSelected = null;
        if (st.difficulty) {
            gamesDifficulty = normalizeDifficulty(st.difficulty);
            gamesSession.difficulty = gamesDifficulty;
            saveGamesSession();
        }
        chessRender(st);
        if (typeof st.googleVIAvailable === 'boolean') gamesGoogleVIAvailable = st.googleVIAvailable;
        if (typeof st.xiaozhiAvailable === 'boolean') gamesXiaozhiAvailable = st.xiaozhiAvailable;
        gamesRenderModeUI();
    } catch (e) {
        chessSetMeta(gT('games.load_err', 'Lỗi tải bàn cờ') + ': ' + e.message);
    }
}

async function chessNewGame() {
    try {
        const level = normalizeDifficulty(gamesSession.difficulty || gamesDifficulty);
        gamesDifficulty = level;
        const st = await gamesFetch('new?level=' + encodeURIComponent(level), { method: 'POST' });
        await chessRefreshLegal();
        chessSelected = null;
        if (st.difficulty) gamesDifficulty = normalizeDifficulty(st.difficulty);
        chessRender(st);
        gamesRenderModeUI();
    } catch (e) {
        chessSetMeta(gT('games.new_err', 'Lỗi') + ': ' + e.message);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const newBtn = document.getElementById('chessNewBtn');
    if (newBtn) newBtn.onclick = () => chessNewGame();
    const exitBtn = document.getElementById('gamesExitBtn');
    if (exitBtn) exitBtn.onclick = () => gamesExitToLobby();
    const startBtn = document.getElementById('gamesModeStartBtn');
    if (startBtn) startBtn.onclick = () => gamesStartPlay();
    const backBtn = document.getElementById('gamesBackToPickBtn');
    if (backBtn) backBtn.onclick = () => gamesBackToPick();
    const pickChess = document.getElementById('gamesPickChess');
    if (pickChess) pickChess.onclick = () => gamesPickGame('chess');
    const pickXq = document.getElementById('gamesPickXiangqi');
    if (pickXq) pickXq.onclick = () => gamesPickGame('xiangqi');
    document.querySelectorAll('.games-pick-card[data-game]').forEach((btn) => {
        btn.addEventListener('click', () => {
            const id = btn.getAttribute('data-game');
            if (id) gamesPickGame(id);
        });
    });
    const passBtn = document.getElementById('gamesPassBtn');
    if (passBtn) {
        passBtn.onclick = async () => {
            if (!chessMyTurn()) return;
            await chessSubmitMove('pass');
        };
    }

    document.querySelectorAll('#gamesStepMode .chess-mode-btn').forEach((btn) => {
        btn.addEventListener('click', () => {
            if (btn.disabled || btn.hidden) return;
            const mode = btn.getAttribute('data-mode');
            if (mode) {
                gamesCommentMode = normalizeCommentMode(mode);
                gamesRenderModeUI();
            }
        });
    });
    document.querySelectorAll('#gamesDiffRow .games-diff-btn').forEach((btn) => {
        btn.addEventListener('click', () => {
            const level = btn.getAttribute('data-level');
            if (!level) return;
            // Only pick before start — locked once play begins.
            gamesDifficulty = normalizeDifficulty(level);
            gamesSession.difficulty = gamesDifficulty;
            saveGamesSession();
            gamesRenderDifficultyUI();
            gamesRenderModeUI();
        });
    });

    gamesDifficulty = normalizeDifficulty(gamesSession.difficulty);
    gamesUpdateNavLabel();

    document.addEventListener('wireos-i18n', () => {
        gamesUpdateHeading();
        gamesUpdateNavLabel();
        gamesUpdatePlayHelp();
        gamesRenderModeUI();
        gamesRenderDifficultyUI();
    });

    const obs = new MutationObserver(() => {
        const sec = document.getElementById('games');
        if (sec && sec.classList.contains('active')) {
            gamesEnterTab();
        }
    });
    const sec = document.getElementById('games');
    if (sec) {
        obs.observe(sec, { attributes: true, attributeFilter: ['class'] });
        if (sec.classList.contains('active')) {
            gamesEnterTab();
        }
    }
});
