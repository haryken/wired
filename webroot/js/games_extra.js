/* Extra minigame UI renderers (sudoku, 2048, wordle, …). Loaded after chess.js helpers. */
(function (global) {
    'use strict';

    const EXTRA_META = {
        tictactoe: {
            get title() { return gT('games.title_ttt', '❌ Tic-tac-toe'); },
            get nav() { return gT('games.tictactoe', 'Tic-tac-toe'); },
            get short() { return gT('games.tictactoe', 'Tic-tac-toe'); },
            api: 'TicTacToe', ico: '❌',
            helpKeys: ['games.ttt_help1', 'games.ttt_help2', 'games.ttt_help3', 'games.ttt_help4'],
            hintKey: 'games.hint_ttt',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_ttt', 'Bấm ô đặt X. Thắng 3 liên tiếp.'); },
        },
        sudoku: {
            get title() { return gT('games.title_sudoku', '🔢 Sudoku'); },
            get nav() { return gT('games.sudoku', 'Sudoku'); },
            get short() { return gT('games.sudoku', 'Sudoku'); },
            api: 'Sudoku', ico: '🔢',
            helpKeys: ['games.sudoku_help1', 'games.sudoku_help2', 'games.sudoku_help3', 'games.sudoku_help4'],
            hintKey: 'games.hint_sudoku',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_sudoku', 'Chọn ô → nhập 1-9. Gợi ý = hint.'); },
        },
        g2048: {
            get title() { return gT('games.title_2048', '🟧 2048'); },
            get nav() { return gT('games.g2048', '2048'); },
            get short() { return gT('games.g2048', '2048'); },
            api: 'G2048', ico: '🟧',
            helpKeys: ['games.g2048_help1', 'games.g2048_help2', 'games.g2048_help3', 'games.g2048_help4'],
            hintKey: 'games.hint_2048',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_2048', 'Vuốt hoặc phím mũi tên / nút UDLR.'); },
        },
        mines: {
            get title() { return gT('games.title_mines', '💣 Dò mìn'); },
            get nav() { return gT('games.mines', 'Dò mìn'); },
            get short() { return gT('games.mines', 'Dò mìn'); },
            api: 'Minesweeper', ico: '💣',
            helpKeys: ['games.mines_help1', 'games.mines_help2', 'games.mines_help3', 'games.mines_help4'],
            hintKey: 'games.hint_mines',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_mines', 'Click mở. Chuột phải / giữ = cờ.'); },
        },
        memory: {
            get title() { return gT('games.title_memory', '🃏 Memory'); },
            get nav() { return gT('games.memory', 'Memory'); },
            get short() { return gT('games.memory', 'Memory'); },
            api: 'Memory', ico: '🃏',
            helpKeys: ['games.memory_help1', 'games.memory_help2', 'games.memory_help3', 'games.memory_help4'],
            hintKey: 'games.hint_memory',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_memory', 'Lật 2 thẻ tìm cặp. Đấu bot.'); },
        },
        battleship: {
            get title() { return gT('games.title_ship', '🚢 Battleship'); },
            get nav() { return gT('games.battleship', 'Battleship'); },
            get short() { return gT('games.battleship', 'Battleship'); },
            api: 'Battleship', ico: '🚢',
            helpKeys: ['games.ship_help1', 'games.ship_help2', 'games.ship_help3', 'games.ship_help4'],
            hintKey: 'games.hint_ship',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_ship', 'Bắn ô trên lưới địch.'); },
        },
        wordle: {
            get title() { return gT('games.title_wordle', '🟩 Wordle VI'); },
            get nav() { return gT('games.wordle', 'Wordle'); },
            get short() { return gT('games.wordle', 'Wordle'); },
            api: 'Wordle', ico: '🟩',
            helpKeys: ['games.wordle_help1', 'games.wordle_help2', 'games.wordle_help3', 'games.wordle_help4'],
            hintKey: 'games.hint_wordle',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_wordle', 'Đoán từ 5 chữ không dấu.'); },
        },
        hangman: {
            get title() { return gT('games.title_hang', '🪢 Treo cổ'); },
            get nav() { return gT('games.hangman', 'Treo cổ'); },
            get short() { return gT('games.hangman', 'Treo cổ'); },
            api: 'Hangman', ico: '🪢',
            helpKeys: ['games.hang_help1', 'games.hang_help2', 'games.hang_help3', 'games.hang_help4'],
            hintKey: 'games.hint_hang',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_hang', 'Đoán từng chữ cái.'); },
        },
        trivia: {
            get title() { return gT('games.title_trivia', '❓ Trivia'); },
            get nav() { return gT('games.trivia', 'Trivia'); },
            get short() { return gT('games.trivia', 'Trivia'); },
            api: 'Trivia', ico: '❓',
            helpKeys: ['games.trivia_help1', 'games.trivia_help2', 'games.trivia_help3', 'games.trivia_help4'],
            hintKey: 'games.hint_trivia',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_trivia', 'Chọn đáp án A–D.'); },
        },
        guessnum: {
            get title() { return gT('games.title_guess', '🎯 Đoán số'); },
            get nav() { return gT('games.guessnum', 'Đoán số'); },
            get short() { return gT('games.guessnum', 'Đoán số'); },
            api: 'GuessNum', ico: '🎯',
            helpKeys: ['games.guess_help1', 'games.guess_help2', 'games.guess_help3', 'games.guess_help4'],
            hintKey: 'games.hint_guess',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_guess', 'Đoán số 1–100.'); },
        },
        simon: {
            get title() { return gT('games.title_simon', '🔔 Simon'); },
            get nav() { return gT('games.simon', 'Simon'); },
            get short() { return gT('games.simon', 'Simon'); },
            api: 'Simon', ico: '🔔',
            helpKeys: ['games.simon_help1', 'games.simon_help2', 'games.simon_help3', 'games.simon_help4'],
            hintKey: 'games.hint_simon',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_simon', 'Xem dãy rồi bấm lại.'); },
        },
        uno: {
            get title() { return gT('games.title_uno', '🎴 Uno'); },
            get nav() { return gT('games.uno', 'Uno'); },
            get short() { return gT('games.uno', 'Uno'); },
            api: 'Uno', ico: '🎴',
            helpKeys: ['games.uno_help1', 'games.uno_help2', 'games.uno_help3', 'games.uno_help4'],
            hintKey: 'games.hint_uno',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_uno', 'Đánh bài cùng màu/số hoặc rút.'); },
        },
        bjweb: {
            get title() { return gT('games.title_bj', '🂡 Blackjack'); },
            get nav() { return gT('games.bjweb', 'Blackjack'); },
            get short() { return gT('games.bjweb', 'Blackjack'); },
            api: 'BlackjackWeb', ico: '🂡',
            helpKeys: ['games.bj_help1', 'games.bj_help2', 'games.bj_help3', 'games.bj_help4'],
            hintKey: 'games.hint_bj',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_bj', 'Bắt đầu → Rút bài / Dừng rút.'); },
        },
        poker: {
            get title() { return gT('games.title_poker', '🃏 Poker 5 Lá'); },
            get nav() { return gT('games.poker', 'Poker'); },
            get short() { return gT('games.poker', 'Poker'); },
            api: 'Poker', ico: '🂫',
            helpKeys: ['games.poker_help1', 'games.poker_help2', 'games.poker_help3', 'games.poker_help4'],
            hintKey: 'games.hint_poker',
            get helpTitle() { return gT('games.guide', 'Hướng dẫn'); },
            get helpItems() { return this.helpKeys.map((k) => gT(k, '')); },
            get hint() { return gT('games.hint_poker', '5 lá: deal → đổi bài → show.'); },
        },
    };

    // Merge into GAMES_META if present
    if (typeof GAMES_META === 'object') {
        Object.keys(EXTRA_META).forEach((k) => { GAMES_META[k] = EXTRA_META[k]; });
    }
    // Restore session again now that extra game ids are registered.
    if (typeof loadGamesSession === 'function') {
        try {
            const restored = loadGamesSession();
            if (typeof gamesSession !== 'undefined') {
                gamesSession.pendingGame = restored.pendingGame;
                gamesSession.activeGame = restored.activeGame;
                gamesSession.commentMode = restored.commentMode;
                gamesSession.difficulty = restored.difficulty;
            }
        } catch (_) {}
    }

    const EXTRA_IDS = Object.keys(EXTRA_META);

    function isExtraGame(id) {
        return EXTRA_IDS.indexOf(id) >= 0;
    }

    function el(tag, cls, text) {
        const n = document.createElement(tag);
        if (cls) n.className = cls;
        if (text != null) n.textContent = text;
        return n;
    }

    function sendUCI(uci) {
        if (typeof chessSubmitMove === 'function') {
            chessSubmitMove(uci);
        }
    }

    function render2048(board, st) {
        board.classList.add('extra-board', 'board-2048');
        const score = el('div', 'g2048-score', 'Score: ' + (st.score || 0));
        board.appendChild(score);

        const tiles = st.tiles || [];
        const wrap = el('div', 'g2048-grid');
        for (let r = 0; r < 4; r++) {
            for (let c = 0; c < 4; c++) {
                const v = (tiles[r] && tiles[r][c]) || 0;
                const cell = el('div', 'g2048-cell' + (v ? ' t' + Math.min(v, 2048) : ''), v ? String(v) : '');
                wrap.appendChild(cell);
            }
        }
        board.appendChild(wrap);

        // Touch swipe on the tile grid (phones)
        let sx = 0, sy = 0, tracking = false;
        const SWIPE_MIN = 28;
        const onStart = (e) => {
            const t = e.touches && e.touches[0];
            if (!t) return;
            tracking = true;
            sx = t.clientX;
            sy = t.clientY;
        };
        const onEnd = (e) => {
            if (!tracking) return;
            tracking = false;
            const t = e.changedTouches && e.changedTouches[0];
            if (!t) return;
            const dx = t.clientX - sx;
            const dy = t.clientY - sy;
            if (Math.abs(dx) < SWIPE_MIN && Math.abs(dy) < SWIPE_MIN) return;
            e.preventDefault();
            if (Math.abs(dx) > Math.abs(dy)) sendUCI(dx > 0 ? 'r' : 'l');
            else sendUCI(dy > 0 ? 'd' : 'u');
        };
        wrap.addEventListener('touchstart', onStart, { passive: true });
        wrap.addEventListener('touchend', onEnd, { passive: false });
        wrap.addEventListener('touchcancel', () => { tracking = false; }, { passive: true });

        // Large on-screen D-pad for touch phones
        const pad = el('div', 'g2048-dpad');
        const mk = (uci, lab, cls) => {
            const b = el('button', 'g2048-dir ' + cls, lab);
            b.type = 'button';
            b.setAttribute('aria-label', uci);
            b.onclick = (e) => { e.preventDefault(); sendUCI(uci); };
            return b;
        };
        pad.appendChild(mk('u', '↑', 'up'));
        pad.appendChild(mk('l', '←', 'left'));
        pad.appendChild(mk('d', '↓', 'down'));
        pad.appendChild(mk('r', '→', 'right'));
        board.appendChild(pad);

        board._keyHandler = (e) => {
            const m = { ArrowUp: 'u', ArrowDown: 'd', ArrowLeft: 'l', ArrowRight: 'r' };
            if (m[e.key]) { e.preventDefault(); sendUCI(m[e.key]); }
        };
        window.addEventListener('keydown', board._keyHandler);
    }

    function renderSudoku(board, st) {
        board.classList.add('extra-board', 'board-sudoku');
        const rows = st.board || [];
        const given = st.given || [];
        let sel = null;
        const grid = el('div', 'sudoku-grid');
        for (let r = 8; r >= 0; r--) {
            const rowS = rows[8 - r] || '';
            const givS = given[8 - r] || '';
            for (let f = 0; f < 9; f++) {
                const ch = rowS[f] === '.' || !rowS[f] ? '' : rowS[f];
                const fixed = givS[f] === '1';
                const sq = el('button', 'sudoku-cell' + (fixed ? ' fixed' : ''), ch);
                sq.type = 'button';
                sq.dataset.sq = String.fromCharCode(97 + f) + r;
                if ((f + 1) % 3 === 0 && f < 8) sq.classList.add('box-r');
                if ((8 - r + 1) % 3 === 0 && r > 0) sq.classList.add('box-b');
                sq.onclick = () => {
                    grid.querySelectorAll('.sel').forEach((x) => x.classList.remove('sel'));
                    if (!fixed) { sq.classList.add('sel'); sel = sq.dataset.sq; }
                };
                grid.appendChild(sq);
            }
        }
        board.appendChild(grid);
        const pad = el('div', 'extra-actions');
        for (let d = 1; d <= 9; d++) {
            const b = el('button', 'extra-btn', String(d));
            b.type = 'button';
            b.onclick = () => { if (sel) sendUCI(sel + ':' + d); };
            pad.appendChild(b);
        }
        const clr = el('button', 'extra-btn', '✕');
        clr.type = 'button';
        clr.onclick = () => { if (sel) sendUCI(sel + ':0'); };
        pad.appendChild(clr);
        const hint = el('button', 'extra-btn primary', 'Hint');
        hint.type = 'button';
        hint.onclick = () => sendUCI('hint');
        pad.appendChild(hint);
        board.appendChild(pad);
    }

    function renderMines(board, st) {
        board.classList.add('extra-board', 'board-mines');
        const rows = st.board || [];
        const n = rows.length || 9;
        const grid = el('div', 'mines-grid');
        grid.style.setProperty('--n', String(n));
        for (let ri = 0; ri < n; ri++) {
            const rank = n - 1 - ri;
            const row = rows[ri] || '';
            for (let f = 0; f < n; f++) {
                const ch = row[f] || '.';
                const sq = el('button', 'mines-cell', '');
                sq.type = 'button';
                const name = String.fromCharCode(97 + f) + rank;
                sq.dataset.sq = name;
                if (ch === '.') sq.classList.add('covered');
                else if (ch === 'F') { sq.classList.add('flag'); sq.textContent = '🚩'; }
                else if (ch === '*') { sq.classList.add('mine'); sq.textContent = '💣'; }
                else {
                    sq.classList.add('open');
                    if (ch !== ' ' && ch !== '0') {
                        sq.textContent = ch;
                        sq.classList.add('n' + ch);
                    }
                }
                sq.onclick = () => sendUCI(name);
                sq.oncontextmenu = (e) => { e.preventDefault(); sendUCI('f' + name); };
                grid.appendChild(sq);
            }
        }
        board.appendChild(grid);
    }

    function renderMemory(board, st) {
        board.classList.add('extra-board', 'board-memory');
        const rows = st.board || [];
        const grid = el('div', 'memory-grid');
        for (let ri = 0; ri < 4; ri++) {
            const rank = 3 - ri;
            const row = rows[ri] || '';
            for (let f = 0; f < 4; f++) {
                const ch = row[f] || '?';
                const sq = el('button', 'memory-card' + (ch === '?' ? ' back' : ' face'), ch === '?' ? '?' : ch);
                sq.type = 'button';
                const name = String.fromCharCode(97 + f) + rank;
                sq.onclick = () => sendUCI(name);
                grid.appendChild(sq);
            }
        }
        board.appendChild(grid);
    }

    function renderBattleship(board, st) {
        board.classList.add('extra-board', 'board-ship');
        function gridOf(rows, title, clickable) {
            const box = el('div', 'ship-pane');
            box.appendChild(el('div', 'ship-title', title));
            const g = el('div', 'ship-grid');
            const n = (rows && rows.length) || 8;
            for (let ri = 0; ri < n; ri++) {
                const rank = n - 1 - ri;
                const row = (rows && rows[ri]) || '';
                for (let f = 0; f < n; f++) {
                    const ch = row[f] || '.';
                    const sq = el('button', 'ship-cell', '');
                    sq.type = 'button';
                    if (ch === 'H') { sq.classList.add('hit'); sq.textContent = '×'; }
                    else if (ch === 'X') { sq.classList.add('miss'); sq.textContent = '·'; }
                    else if (ch === 'S') { sq.classList.add('sunk'); sq.textContent = '■'; }
                    else if (ch === 'B' || ch === 'O') { sq.classList.add('boat'); sq.textContent = '■'; }
                    else sq.classList.add('water');
                    if (clickable) {
                        const name = String.fromCharCode(97 + f) + rank;
                        sq.onclick = () => sendUCI(name);
                    }
                    g.appendChild(sq);
                }
            }
            box.appendChild(g);
            return box;
        }
        board.appendChild(gridOf(st.board, gT('games.ship_enemy', 'Địch'), true));
        board.appendChild(gridOf(st.myBoard, gT('games.ship_mine', 'Bạn'), false));
    }

    function renderWordle(board, st) {
        board.classList.add('extra-board', 'board-wordle');
        const guesses = st.guesses || [];
        const feedback = st.feedback || [];
        const maxG = st.maxGuesses || 6;
        const rows = el('div', 'wordle-rows');
        for (let i = 0; i < maxG; i++) {
            const row = el('div', 'wordle-row');
            const g = (guesses[i] || '     ').toUpperCase();
            const fb = feedback[i] || '';
            for (let c = 0; c < 5; c++) {
                const cell = el('div', 'wordle-cell', g[c] === ' ' ? '' : g[c]);
                if (fb[c] === 'G') cell.classList.add('g');
                else if (fb[c] === 'Y') cell.classList.add('y');
                else if (fb[c] === 'B') cell.classList.add('b');
                row.appendChild(cell);
            }
            rows.appendChild(row);
        }
        board.appendChild(rows);
        const inp = el('input', 'wordle-input');
        inp.type = 'text';
        inp.maxLength = 5;
        inp.placeholder = '.....';
        inp.autocapitalize = 'characters';
        const go = el('button', 'extra-btn primary', 'Đoán');
        go.type = 'button';
        const submit = () => {
            const v = (inp.value || '').trim().toLowerCase();
            if (v.length === 5) { sendUCI(v); inp.value = ''; }
        };
        go.onclick = submit;
        inp.onkeydown = (e) => { if (e.key === 'Enter') submit(); };
        const pad = el('div', 'extra-actions');
        pad.appendChild(inp);
        pad.appendChild(go);
        board.appendChild(pad);
        setTimeout(() => inp.focus(), 50);
    }

    function renderHangman(board, st) {
        board.classList.add('extra-board', 'board-hang');
        board.appendChild(el('div', 'hang-mask', st.masked || ''));
        board.appendChild(el('div', 'hang-meta',
            gT('games.hang_wrong', 'Sai') + ': ' + (st.wrongCount || 0) + '/' + (st.maxWrong || 7)
            + '  ' + ((st.wrong || []).join(' '))));
        const pad = el('div', 'extra-actions hang-keys');
        for (let i = 0; i < 26; i++) {
            const L = String.fromCharCode(97 + i);
            const used = (st.wrong || []).indexOf(L) >= 0 || ((st.masked || '').toLowerCase().indexOf(L) >= 0);
            const b = el('button', 'extra-btn' + (used ? ' used' : ''), L.toUpperCase());
            b.type = 'button';
            b.disabled = !!used || st.status !== 'playing';
            b.onclick = () => sendUCI(L);
            pad.appendChild(b);
        }
        board.appendChild(pad);
    }

    function renderTrivia(board, st) {
        board.classList.add('extra-board', 'board-trivia');
        board.appendChild(el('div', 'trivia-q', (st.qIndex != null ? ((st.qIndex + 1) + '/' + (st.total || 5) + '. ') : '') + (st.question || '')));
        board.appendChild(el('div', 'trivia-score', 'Score: ' + (st.score || 0)));
        const pad = el('div', 'extra-actions col');
        (st.choices || []).forEach((c, i) => {
            const b = el('button', 'extra-btn choice', String.fromCharCode(65 + i) + '. ' + c);
            b.type = 'button';
            b.onclick = () => sendUCI(String(i));
            pad.appendChild(b);
        });
        board.appendChild(pad);
    }

    function renderGuess(board, st) {
        board.classList.add('extra-board', 'board-guess');
        const low = st.low != null ? st.low : 1;
        const high = st.high != null ? st.high : 100;
        const min = st.min != null ? st.min : 1;
        const max = st.max != null ? st.max : 100;
        const attempts = st.attempts || 0;
        const maxA = st.maxAttempts || 10;
        const rem = st.remaining != null ? st.remaining : Math.max(0, maxA - attempts);
        const playing = st.status === 'playing';
        const won = st.status === 'win';
        const lost = st.status === 'lose';

        // --- Title ---
        const hero = el('div', 'guess-hero');
        const ico = el('div', 'guess-hero-ico', '🎯');
        hero.appendChild(ico);
        hero.appendChild(el('div', 'guess-hero-title', gT('guess.title', 'Đoán Số Bí Mật')));
        hero.appendChild(el('div', 'guess-hero-sub', gT('guess.subtitle', 'Hãy tìm ra con số bí mật trong số lượt giới hạn.')));
        board.appendChild(hero);

        // --- Status cards ---
        const stats = el('div', 'guess-stats');
        const mkStat = (lab, val, cls) => {
            const c = el('div', 'guess-stat' + (cls ? ' ' + cls : ''));
            c.appendChild(el('div', 'guess-stat-lab', lab));
            c.appendChild(el('div', 'guess-stat-val', val));
            return c;
        };
        stats.appendChild(mkStat(gT('guess.range', 'Khoảng'), low + ' → ' + high, 'range'));
        stats.appendChild(mkStat(gT('guess.left', 'Lượt còn'), String(rem), rem <= 3 ? 'warn' : ''));
        stats.appendChild(mkStat(gT('guess.tried', 'Đã đoán'), attempts + '/' + maxA, ''));
        stats.appendChild(mkStat(gT('guess.diff', 'Độ khó'), st.difficultyLabel || st.difficulty || '—', ''));
        board.appendChild(stats);

        // --- Lives / progress ---
        const lives = el('div', 'guess-lives');
        lives.appendChild(el('div', 'guess-lives-lab', gT('guess.lives', 'Lượt chơi')));
        const hearts = el('div', 'guess-hearts');
        for (let i = 0; i < maxA; i++) {
            hearts.appendChild(el('span', 'guess-heart' + (i < rem ? ' on' : ' off'), i < rem ? '❤️' : '🤍'));
        }
        lives.appendChild(hearts);
        const bar = el('div', 'guess-bar');
        const fill = el('div', 'guess-bar-fill');
        fill.style.width = Math.max(0, Math.min(100, (rem / maxA) * 100)) + '%';
        bar.appendChild(fill);
        lives.appendChild(bar);
        board.appendChild(lives);

        // --- Range slider visual ---
        const trackWrap = el('div', 'guess-track-wrap');
        trackWrap.appendChild(el('div', 'guess-track-lab', gT('guess.track', 'Khoảng đoán')));
        const track = el('div', 'guess-track');
        const span = Math.max(1, max - min);
        const leftPct = ((low - min) / span) * 100;
        const widthPct = ((high - low) / span) * 100;
        const zone = el('div', 'guess-track-zone');
        zone.style.left = leftPct + '%';
        zone.style.width = Math.max(2, widthPct) + '%';
        track.appendChild(zone);
        const labels = el('div', 'guess-track-nums');
        labels.appendChild(el('span', '', String(low)));
        labels.appendChild(el('span', 'guess-track-mid', low + ' — ' + high));
        labels.appendChild(el('span', '', String(high)));
        trackWrap.appendChild(track);
        trackWrap.appendChild(labels);
        board.appendChild(trackWrap);

        // --- Result badge ---
        if (st.lastHint) {
            const badge = el('div', 'guess-badge pop');
            let cls = '', text = '';
            if (st.lastHint === 'exact' || won) {
                cls = 'ok'; text = gT('guess.badge_ok', '🎉 Chính xác');
            } else if (st.lastHint === 'lose' || lost) {
                cls = 'lose'; text = gT('guess.badge_lose', '💥 Hết lượt');
            } else if (st.lastClose) {
                cls = 'close';
                text = st.lastHint === 'higher'
                    ? gT('guess.badge_near_up', '🎯 Gần đúng · Cao hơn')
                    : gT('guess.badge_near_down', '🎯 Gần đúng · Thấp hơn');
            } else if (st.lastHint === 'higher') {
                cls = 'up'; text = gT('guess.badge_up', '⬆️ Cao hơn');
            } else if (st.lastHint === 'lower') {
                cls = 'down'; text = gT('guess.badge_down', '⬇️ Thấp hơn');
            }
            badge.className = 'guess-badge pop ' + cls;
            badge.textContent = text;
            board.appendChild(badge);
            if (st.message) {
                board.appendChild(el('div', 'guess-robot-line', st.message));
            }
        }

        // --- Input row ---
        if (playing) {
            const row = el('div', 'guess-input-row');
            const inp = el('input', 'guess-input');
            inp.type = 'number';
            inp.min = String(min);
            inp.max = String(max);
            inp.placeholder = gT('guess.placeholder', 'Nhập số từ 1 đến 100');
            inp.inputMode = 'numeric';
            const go = el('button', 'guess-btn', gT('guess.btn', 'Đoán'));
            go.type = 'button';
            const submit = () => {
                const v = String(inp.value || '').trim();
                if (!v) {
                    row.classList.add('shake');
                    setTimeout(() => row.classList.remove('shake'), 420);
                    return;
                }
                go.classList.add('loading');
                go.disabled = true;
                go.textContent = '…';
                row.classList.add('shake');
                setTimeout(() => row.classList.remove('shake'), 420);
                sendUCI(v);
            };
            go.onclick = submit;
            inp.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    submit();
                }
            });
            row.appendChild(inp);
            row.appendChild(go);
            board.appendChild(row);
            setTimeout(() => inp.focus(), 60);
        }

        // --- History chips ---
        const hist = Array.isArray(st.guessHistory) ? st.guessHistory : [];
        if (hist.length) {
            const hWrap = el('div', 'guess-hist');
            hWrap.appendChild(el('div', 'guess-hist-lab', gT('guess.history', 'Lịch sử đoán')));
            const chips = el('div', 'guess-chips');
            hist.forEach((h) => {
                let mark = '❌';
                let cls = 'miss';
                if (h.hint === 'exact') { mark = '✅'; cls = 'ok'; }
                else if (h.hint === 'higher') { mark = h.close ? '🎯⬆️' : '⬆️'; cls = h.close ? 'close' : 'up'; }
                else if (h.hint === 'lower') { mark = h.close ? '🎯⬇️' : '⬇️'; cls = h.close ? 'close' : 'down'; }
                chips.appendChild(el('span', 'guess-chip ' + cls, h.guess + ' ' + mark));
            });
            hWrap.appendChild(chips);
            board.appendChild(hWrap);
        }

        // --- Win / lose cards ---
        if (won) {
            const card = el('div', 'guess-end win');
            card.appendChild(el('div', 'guess-end-emoji', '🏆'));
            card.appendChild(el('div', 'guess-end-title', gT('guess.congrats', 'CHÚC MỪNG!')));
            card.appendChild(el('div', 'guess-end-sub', gT('guess.you_won', 'Bạn đã đoán đúng!')));
            const grid = el('div', 'guess-end-stats');
            grid.appendChild(mkStat(gT('guess.tried', 'Đã đoán'), String(attempts), ''));
            grid.appendChild(mkStat(gT('guess.time', 'Thời gian'), (st.elapsedSec != null ? st.elapsedSec : '—') + 's', ''));
            grid.appendChild(mkStat(gT('guess.points', 'Điểm'), String(st.score != null ? st.score : '—'), ''));
            grid.appendChild(mkStat(gT('guess.secret', 'Số bí mật'), String(st.secret != null ? st.secret : '—'), ''));
            card.appendChild(grid);
            const again = el('button', 'guess-btn again', gT('guess.again', 'Chơi lại'));
            again.type = 'button';
            again.onclick = () => {
                if (typeof gamesNewGame === 'function') gamesNewGame();
                else if (typeof chessNewGame === 'function') chessNewGame();
            };
            card.appendChild(again);
            board.appendChild(card);
            if (!board._guessConfetti) {
                board._guessConfetti = true;
                requestAnimationFrame(() => {
                    const layer = el('div', 'uno-confetti-layer');
                    const colors = ['#b794f4', '#9f7aea', '#68d391', '#f6e05e', '#fc8181', '#90cdf4', '#fff'];
                    for (let i = 0; i < 48; i++) {
                        const p = el('div', 'uno-confetti');
                        p.style.left = (Math.random() * 100) + '%';
                        p.style.background = colors[i % colors.length];
                        p.style.animationDelay = (Math.random() * 0.8) + 's';
                        p.style.animationDuration = (1.5 + Math.random() * 1.4) + 's';
                        layer.appendChild(p);
                    }
                    board.appendChild(layer);
                    setTimeout(() => { if (layer.parentNode) layer.parentNode.removeChild(layer); }, 4200);
                });
            }
        } else if (lost) {
            const card = el('div', 'guess-end lose');
            card.appendChild(el('div', 'guess-end-emoji', '😅'));
            card.appendChild(el('div', 'guess-end-title', gT('guess.lost_title', 'HẾT LƯỢT')));
            card.appendChild(el('div', 'guess-end-sub',
                gT('guess.secret_was', 'Số bí mật là') + ' ' + (st.secret != null ? st.secret : '?')));
            const again = el('button', 'guess-btn again', gT('guess.again', 'Chơi lại'));
            again.type = 'button';
            again.onclick = () => {
                if (typeof gamesNewGame === 'function') gamesNewGame();
                else if (typeof chessNewGame === 'function') chessNewGame();
            };
            card.appendChild(again);
            board.appendChild(card);
        }
    }

    function renderSimon(board, st) {
        board.classList.add('extra-board', 'board-simon');
        board.appendChild(el('div', 'simon-level', 'Level ' + (st.level || 1)));
        const pad = el('div', 'simon-pads');
        const colors = ['#e74c3c', '#2ecc71', '#3498db', '#f1c40f'];
        for (let i = 0; i < 4; i++) {
            const b = el('button', 'simon-pad', String(i));
            b.type = 'button';
            b.style.background = colors[i];
            b.onclick = () => sendUCI(String(i));
            pad.appendChild(b);
        }
        board.appendChild(pad);
        if (st.lastMove === 'playback' && Array.isArray(st.sequence)) {
            const seq = st.sequence.slice();
            let i = 0;
            const pads = pad.querySelectorAll('.simon-pad');
            const tick = () => {
                pads.forEach((p) => p.classList.remove('lit'));
                if (i >= seq.length) return;
                const p = pads[seq[i]];
                if (p) p.classList.add('lit');
                i++;
                setTimeout(() => {
                    pads.forEach((x) => x.classList.remove('lit'));
                    setTimeout(tick, 220);
                }, 450);
            };
            setTimeout(tick, 400);
        }
    }

    function unoParseCard(code) {
        const c = String(code || '').toUpperCase();
        if (c === 'W' || c === 'W4') {
            return {
                color: 'W',
                rank: c === 'W4' ? '+4' : gT('uno.rank_wild', 'MÀU'),
                kind: c === 'W4' ? 'w4' : 'wild',
                code: c,
            };
        }
        if (c.length < 2) return { color: 'X', rank: c || '?', kind: 'num', code: c };
        const color = c[0];
        const rankRaw = c.slice(1);
        let rank = rankRaw, kind = 'num';
        if (rankRaw === 'S') { rank = gT('uno.rank_skip', 'BỎ'); kind = 'skip'; }
        else if (rankRaw === 'D') { rank = '+2'; kind = 'draw2'; }
        else if (rankRaw === 'V') { rank = gT('uno.rank_rev', 'ĐẢO'); kind = 'rev'; }
        return { color, rank, kind, code: c };
    }

    function unoColorName(col) {
        return ({
            R: gT('uno.color_r', 'Đỏ'),
            G: gT('uno.color_g', 'Xanh lá'),
            B: gT('uno.color_b', 'Xanh dương'),
            Y: gT('uno.color_y', 'Vàng'),
            W: gT('uno.color_w', 'Wild'),
        })[col] || col || '?';
    }

    function unoCardLabel(code) {
        const p = unoParseCard(code);
        const col = unoColorName(p.color).toLowerCase();
        if (p.kind === 'wild') return gT('uno.lbl_wild', 'Đổi màu');
        if (p.kind === 'w4') return gT('uno.lbl_w4', 'Cộng 4');
        if (p.kind === 'skip') return gT('uno.lbl_skip', 'Bỏ lượt') + ' ' + col;
        if (p.kind === 'draw2') return '+2 ' + col;
        if (p.kind === 'rev') return gT('uno.lbl_rev', 'Đảo chiều') + ' ' + col;
        return p.rank + ' ' + col;
    }

    function unoCornerLabel(p) {
        if (p.kind === 'w4') return '+4';
        if (p.kind === 'wild') return 'W';
        if (p.kind === 'draw2') return '+2';
        if (p.kind === 'skip') return '⊘';
        if (p.kind === 'rev') return '⇄';
        return String(p.rank || '?');
    }

    function unoMakeCardEl(code, opts) {
        opts = opts || {};
        const p = unoParseCard(code);
        const b = el('button', 'uno-card color-' + p.color + ' kind-' + p.kind
            + (opts.playable ? ' playable' : '')
            + (opts.big ? ' big' : '')
            + (opts.mini ? ' mini' : '')
            + (opts.slam ? ' slam' : '')
            + (opts.justDrawn ? ' just-drawn' : ''), '');
        b.type = 'button';
        b.dataset.card = p.code;
        if (opts.disabled) b.disabled = true;
        // Corner rank — readable when cards are stacked.
        if (!opts.mini) {
            const corner = el('div', 'uno-corner', unoCornerLabel(p));
            b.appendChild(corner);
        }
        const face = el('div', 'uno-face');
        const main = el('div', 'uno-rank', p.rank);
        if (p.kind === 'wild' || p.kind === 'w4') main.classList.add('wild-label');
        face.appendChild(main);
        if (p.kind === 'skip') face.appendChild(el('div', 'uno-sub', '⊘'));
        else if (p.kind === 'draw2') face.appendChild(el('div', 'uno-sub', gT('uno.sub_draw', 'RÚT 2')));
        else if (p.kind === 'rev') face.appendChild(el('div', 'uno-sub', '⇄'));
        else if (p.kind === 'wild') face.appendChild(el('div', 'uno-sub', gT('uno.sub_wild', 'Đổi màu')));
        else if (p.kind === 'w4') face.appendChild(el('div', 'uno-sub', gT('uno.sub_w4', 'Rút 4')));
        b.appendChild(face);
        if (opts.onclick) b.onclick = opts.onclick;
        return b;
    }

    function unoMakeBackEl() {
        const b = el('div', 'uno-card uno-back', '');
        const face = el('div', 'uno-face');
        face.appendChild(el('div', 'uno-rank', 'UNO'));
        b.appendChild(face);
        return b;
    }

    function unoColorChip(color, label) {
        const wrap = el('div', 'uno-color-chip color-' + (color || 'X'));
        wrap.appendChild(el('span', 'uno-color-dot', ''));
        wrap.appendChild(el('span', 'uno-color-text', label || unoColorName(color)));
        return wrap;
    }

    let unoLastFlyKey = '';
    let unoFlyBusy = false;
    let unoCelebrateKey = '';
    let unoFlyFromRect = null;
    let unoSelectedCard = null;
    let unoLastDrawKey = '';
    let unoDrawLock = false;
    let unoLastFxKey = '';

    function unoRememberFlyFrom(elNode) {
        if (!elNode || !elNode.getBoundingClientRect) {
            unoFlyFromRect = null;
            return;
        }
        const r = elNode.getBoundingClientRect();
        if (!r.width || !r.height) {
            unoFlyFromRect = null;
            return;
        }
        unoFlyFromRect = { left: r.left, top: r.top, width: r.width, height: r.height };
    }

    function unoSpawnConfetti(host) {
        const layer = el('div', 'uno-confetti-layer');
        const colors = ['#f6e05e', '#68d391', '#63b3ed', '#fc8181', '#b794f4', '#fbd38d', '#fff'];
        for (let i = 0; i < 48; i++) {
            const p = el('div', 'uno-confetti');
            p.style.left = (Math.random() * 100) + '%';
            p.style.background = colors[i % colors.length];
            p.style.animationDelay = (Math.random() * 0.8) + 's';
            p.style.animationDuration = (1.6 + Math.random() * 1.4) + 's';
            p.style.transform = 'rotate(' + (Math.random() * 360) + 'deg)';
            layer.appendChild(p);
        }
        host.appendChild(layer);
        setTimeout(() => { if (layer.parentNode) layer.parentNode.removeChild(layer); }, 4200);
    }

    function unoHidePileCards() {
        document.querySelectorAll('.uno-arena .uno-discard .uno-card, .uno-pile .uno-card').forEach((c) => {
            c.style.visibility = 'hidden';
        });
    }

    function unoShowPileCards() {
        document.querySelectorAll('.uno-arena .uno-discard .uno-card, .uno-pile .uno-card').forEach((c) => {
            c.style.visibility = '';
        });
    }

    function unoSleep(ms) {
        return new Promise((r) => setTimeout(r, ms));
    }

    function unoFlyToPile(board, code, from) {
        const pileCard = board.querySelector('.uno-discard .uno-card.big')
            || board.querySelector('.uno-pile .uno-card.big')
            || board.querySelector('.uno-discard .uno-card');
        if (!pileCard || !code) return Promise.resolve();
        document.querySelectorAll('.uno-flyer').forEach((n) => { if (n.parentNode) n.parentNode.removeChild(n); });
        const dest = pileCard.getBoundingClientRect();
        let x0, y0, w0, h0;
        if (from === 'you' && unoFlyFromRect) {
            x0 = unoFlyFromRect.left; y0 = unoFlyFromRect.top;
            w0 = unoFlyFromRect.width; h0 = unoFlyFromRect.height;
            unoFlyFromRect = null;
        } else {
            const startEl = from === 'you'
                ? (board.querySelector('.uno-zone-you .uno-hand') || board.querySelector('.uno-zone-you'))
                : (board.querySelector('.uno-bot-hand') || board.querySelector('.uno-zone-bot'));
            if (!startEl) return Promise.resolve();
            const start = startEl.getBoundingClientRect();
            w0 = Math.min(64, dest.width); h0 = Math.min(88, dest.height);
            x0 = start.left + start.width / 2 - w0 / 2;
            y0 = from === 'you' ? (start.top + Math.max(0, start.height - h0 - 4)) : (start.top + 8);
        }
        unoFlyBusy = true;
        unoHidePileCards();
        const flyer = unoMakeCardEl(code, { disabled: true, big: true });
        flyer.classList.add('uno-flyer');
        flyer.style.width = w0 + 'px';
        flyer.style.height = h0 + 'px';
        flyer.style.transform = 'translate(' + x0 + 'px,' + y0 + 'px) scale(1.08) rotate(' + (from === 'you' ? '12deg' : '-12deg') + ')';
        document.body.appendChild(flyer);
        void flyer.offsetWidth;
        flyer.style.width = dest.width + 'px';
        flyer.style.height = dest.height + 'px';
        flyer.style.transform = 'translate(' + dest.left + 'px,' + dest.top + 'px) scale(1) rotate(0deg)';
        return new Promise((resolve) => {
            setTimeout(() => {
                if (flyer.parentNode) flyer.parentNode.removeChild(flyer);
                unoFlyBusy = false;
                unoShowPileCards();
                resolve();
            }, 420);
        });
    }

    async function unoAnimateForcedDraw(st) {
        const n = st && st.drawAnimN ? Number(st.drawAnimN) : 0;
        const to = st && st.drawAnimTo ? String(st.drawAnimTo) : '';
        if (!n || (to !== 'human' && to !== 'bot')) return;
        const key = String(st.lastMove || '') + '|' + to + '|' + n + '|' + (st.botForceCard || '') + '|' + (st.hand || []).length + '|' + (st.botCount || 0);
        if (key === unoLastDrawKey) return;
        unoLastDrawKey = key;

        const board = document.getElementById('chessBoard');
        if (!board) return;
        const pile = board.querySelector('.uno-draw-deck') || board.querySelector('.uno-discard') || board.querySelector('.uno-table-center');
        const destEl = to === 'human'
            ? (board.querySelector('.uno-zone-you .uno-hand') || board.querySelector('.uno-zone-you'))
            : (board.querySelector('.uno-bot-hand') || board.querySelector('.uno-zone-bot'));
        if (!pile || !destEl) return;

        unoDrawLock = true;
        try {
            if (n > 1 || (st.botForceCard && String(st.botForceCard).indexOf('D') >= 0) || st.botForceCard === 'W4') {
                await unoSleep(280);
            }
            for (let i = 0; i < n; i++) {
                const start = pile.getBoundingClientRect();
                const dest = destEl.getBoundingClientRect();
                const flyer = unoMakeBackEl();
                flyer.classList.add('uno-flyer', 'uno-draw-flyer');
                flyer.style.width = '48px';
                flyer.style.height = '68px';
                const x0 = start.left + start.width / 2 - 24;
                const y0 = start.top + start.height / 2 - 34;
                const x1 = dest.left + dest.width / 2 - 24 + (i * 5);
                const y1 = dest.top + dest.height / 2 - 34;
                flyer.style.transform = 'translate(' + x0 + 'px,' + y0 + 'px) scale(0.9)';
                document.body.appendChild(flyer);
                void flyer.offsetWidth;
                flyer.style.transform = 'translate(' + x1 + 'px,' + y1 + 'px) scale(1) rotate(' + (to === 'human' ? '8deg' : '-8deg') + ')';
                await unoSleep(130);
                if (flyer.parentNode) flyer.parentNode.removeChild(flyer);
                await unoSleep(40);
            }
        } finally {
            unoDrawLock = false;
        }
    }
    global.unoAnimateForcedDraw = unoAnimateForcedDraw;

    function unoShowFx(board, code) {
        const p = unoParseCard(code);
        const host = board.querySelector('.uno-fx-layer');
        if (!host) return;
        host.innerHTML = '';
        let kind = p.kind;
        let label = '';
        if (kind === 'draw2') label = '+2';
        else if (kind === 'w4') label = '+4';
        else if (kind === 'skip') label = '⊘';
        else if (kind === 'rev') label = '⇄';
        else if (kind === 'wild') label = 'WILD';
        else return;
        const fx = el('div', 'uno-fx uno-fx-' + kind, label);
        host.appendChild(fx);
        setTimeout(() => { if (fx.parentNode) fx.parentNode.removeChild(fx); }, 480);
    }

    function unoCanPlayCard(st, code) {
        if (!code) return false;
        const top = String(st.top || '').toUpperCase();
        const color = String(st.color || '').toUpperCase();
        const c = String(code).toUpperCase();
        if (c === 'W' || c === 'W4') return true;
        if (c.length < 2) return false;
        const col = c[0];
        const rank = c.slice(1);
        let topRank = '';
        if (top.length > 1 && top[0] !== 'W') topRank = top.slice(1);
        return col === color || (topRank && rank === topRank);
    }

    function renderUno(board, st) {
        board.classList.add('extra-board', 'board-uno');
        const arena = el('div', 'uno-arena');
        const botN = st.botCount || 0;
        const botPlayed = st.botLastCard || (String(st.lastMove || '').indexOf('bot:') === 0 ? String(st.lastMove).slice(4).toUpperCase() : '');
        const humanPlayed = st.humanLastCard || '';
        const botThinking = !!st.botThinking;
        const slamBot = !!st.botJustPlayed;
        const slamYou = !!st.humanJustPlayed;
        const pickColor = !!st.pendingColor;
        const playing = st.status === 'playing';
        const yourTurn = playing && !botThinking && (st.turn === 'human' || pickColor);
        const robotTurn = playing && !yourTurn;
        const handCards = st.hand || [];
        const canSelect = !pickColor && !unoDrawLock && playing && st.turn === 'human' && !botThinking;
        const dir = String(st.direction || 'cw');

        // Keep selection valid
        if (!canSelect || (unoSelectedCard && handCards.indexOf(unoSelectedCard) < 0)) {
            unoSelectedCard = null;
        } else if (unoSelectedCard && !unoCanPlayCard(st, unoSelectedCard)) {
            // keep selected for visual but play button disabled
        }

        // ========== 1. ROBOT ZONE ==========
        const botZ = el('div', 'uno-zone uno-zone-bot'
            + (robotTurn ? ' active-turn' : '')
            + (botThinking ? ' thinking' : '')
            + ((st.winner === 'bot') ? ' winner' : ''));

        // speech bubble overlay
        const bubble = el('div', 'uno-speech' + (st.message ? ' show' : ''));
        const av = el('div', 'uno-speech-av' + (botThinking || robotTurn ? ' talking' : ''), '🤖');
        bubble.appendChild(av);
        const wave = el('div', 'uno-speech-wave');
        wave.appendChild(el('span')); wave.appendChild(el('span')); wave.appendChild(el('span'));
        bubble.appendChild(wave);
        bubble.appendChild(el('div', 'uno-speech-text', st.message || '…'));
        botZ.appendChild(bubble);

        const botHead = el('div', 'uno-zone-head');
        botHead.appendChild(el('div', 'uno-zone-title', gT('uno.robot', '🤖 Robot')));
        botHead.appendChild(el('div', 'uno-count-badge', botN + ' ' + gT('uno.cards', 'lá')));
        botZ.appendChild(botHead);

        const botBody = el('div', 'uno-zone-body');
        const botHand = el('div', 'uno-bot-hand');
        const showN = Math.min(botN, 14);
        for (let i = 0; i < showN; i++) botHand.appendChild(unoMakeBackEl());
        if (botN > 14) botHand.appendChild(el('div', 'uno-more', '+' + (botN - 14)));
        if (botN === 0) botHand.appendChild(el('div', 'uno-empty', gT('uno.empty', 'Hết bài')));
        botBody.appendChild(botHand);

        const thinkOverlay = el('div', 'uno-think-overlay' + (botThinking ? ' show' : ''));
        thinkOverlay.appendChild(el('div', 'uno-think-text', gT('uno.thinking_dots', '🤔 Đang suy nghĩ…')));
        botBody.appendChild(thinkOverlay);
        botZ.appendChild(botBody);
        arena.appendChild(botZ);

        // ========== 2. TABLE ZONE ==========
        const tableZ = el('div', 'uno-zone uno-zone-table');
        const tableCard = el('div', 'uno-table-card');
        const stacks = el('div', 'uno-stacks');

        // Draw deck
        const drawDeck = el('div', 'uno-draw-deck');
        drawDeck.appendChild(unoMakeBackEl());
        drawDeck.appendChild(el('div', 'uno-stack-lab', '🂠 ' + (st.deckCount != null ? st.deckCount : '?')));
        stacks.appendChild(drawDeck);

        // Discard
        const discard = el('div', 'uno-discard uno-pile');
        const topCard = unoMakeCardEl(st.top || 'R0', { big: true, disabled: true });
        if (unoFlyBusy) topCard.style.visibility = 'hidden';
        discard.appendChild(topCard);
        stacks.appendChild(discard);
        tableCard.appendChild(stacks);

        const meta = el('div', 'uno-table-meta');
        const turnLbl = robotTurn
            ? gT('uno.turn_bot', 'Lượt Robot…')
            : (yourTurn ? gT('uno.turn_you', 'Đến lượt bạn') : gT('uno.turn_wait', 'Chờ…'));
        meta.appendChild(el('div', 'uno-turn-badge'
            + (robotTurn ? ' bot' : '')
            + (yourTurn ? ' you' : '')
            + (yourTurn || robotTurn ? ' live' : ''), turnLbl));

        const chips = el('div', 'uno-meta-chips');
        chips.appendChild(unoColorChip(st.color, gT('uno.color_short', 'Màu') + ': ' + unoColorName(st.color)));
        chips.appendChild(el('div', 'uno-dir-chip', (dir === 'ccw' ? '↺' : '↻') + ' ' + gT('uno.dir', 'Chiều')));
        meta.appendChild(chips);

        // Fixed-height notice slot (never grows layout)
        const notice = el('div', 'uno-notice');
        const drawing = (st.drawAnimN | 0) > 0;
        const drawTo = String(st.drawAnimTo || '');
        if (drawing && drawTo === 'human') {
            notice.textContent = gT('uno.forced_draw', 'Bạn bị rút') + ' +' + st.drawAnimN;
            notice.classList.add('alert');
        } else if (drawing && drawTo === 'bot') {
            notice.textContent = gT('uno.bot_drawing', 'Robot đang rút') + ' +' + st.drawAnimN;
            notice.classList.add('alert');
        } else if (st.message) {
            notice.textContent = st.message;
        } else {
            notice.innerHTML = '&nbsp;';
        }
        meta.appendChild(notice);
        tableCard.appendChild(meta);

        // FX layer (absolute, no layout impact)
        const fxLayer = el('div', 'uno-fx-layer');
        tableCard.appendChild(fxLayer);
        tableZ.appendChild(tableCard);
        arena.appendChild(tableZ);

        // ========== 3. PLAYER ZONE ==========
        const youZ = el('div', 'uno-zone uno-zone-you'
            + (yourTurn && !pickColor ? ' active-turn' : '')
            + ((st.winner === 'human') ? ' winner' : ''));
        const youHead = el('div', 'uno-zone-head');
        youHead.appendChild(el('div', 'uno-zone-title', gT('uno.you', 'Bạn')));
        youHead.appendChild(el('div', 'uno-turn-pill you' + (yourTurn ? '' : ' ghost'), gT('uno.turn_you_short', 'LƯỢT CỦA BẠN')));
        youHead.appendChild(el('div', 'uno-count-badge', handCards.length + ' ' + gT('uno.cards', 'lá')));
        youZ.appendChild(youHead);

        const youBody = el('div', 'uno-zone-body');
        const hand = el('div', 'uno-hand stacked' + (canSelect ? '' : ' locked'));
        const n = handCards.length;
        const overlap = n <= 6 ? 18 : n <= 10 ? 28 : n <= 14 ? 36 : n <= 20 ? 44 : 50;
        hand.style.setProperty('--uno-overlap', '-' + overlap + 'px');
        const justDrew = String(st.lastMove || '') === 'human:draw';

        handCards.forEach((c, i) => {
            const playable = canSelect && unoCanPlayCard(st, c);
            const selected = unoSelectedCard === c;
            const b = unoMakeCardEl(c, {
                playable: playable,
                justDrawn: justDrew && i === handCards.length - 1,
                onclick: () => {
                    if (!canSelect) return;
                    if (unoSelectedCard === c) {
                        unoSelectedCard = null;
                        b.classList.remove('sel', 'lifted');
                    } else {
                        hand.querySelectorAll('.uno-card.sel').forEach((x) => x.classList.remove('sel', 'lifted'));
                        unoSelectedCard = c;
                        b.classList.add('sel', 'lifted');
                    }
                    const playBtn = arena.querySelector('.uno-act-play');
                    if (playBtn) {
                        const ok = !!(unoSelectedCard && unoCanPlayCard(st, unoSelectedCard) && canSelect);
                        playBtn.disabled = !ok;
                    }
                },
            });
            if (selected) b.classList.add('sel', 'lifted');
            if (!canSelect) b.disabled = true;
            b.style.zIndex = String(i + 1);
            hand.appendChild(b);
        });
        youBody.appendChild(hand);
        if (justDrew) {
            requestAnimationFrame(() => { hand.scrollLeft = hand.scrollWidth; });
        }
        youZ.appendChild(youBody);

        // Action bar (fixed in player zone)
        const bar = el('div', 'uno-action-bar');
        const mkAct = (cls, label, enabled, fn) => {
            const b = el('button', 'uno-act ' + cls, label);
            b.type = 'button';
            b.disabled = !enabled;
            b.onclick = () => { if (b.disabled) return; fn(); };
            return b;
        };

        const canDraw = canSelect && !pickColor;
        // pass when just drew and have no playable / or always as long-press alternative via double: Rút only
        const canPlayBtn = canSelect && unoSelectedCard && unoCanPlayCard(st, unoSelectedCard);
        const canUno = canSelect && handCards.length === 1 && !st.saidUno;

        bar.appendChild(mkAct('draw uno-act-draw', '🂠 ' + gT('uno.draw', 'RÚT BÀI'), canDraw, () => {
            unoSelectedCard = null;
            sendUCI('draw');
        }));
        bar.appendChild(mkAct('play uno-act-play', '🃏 ' + gT('uno.play', 'ĐÁNH BÀI'), canPlayBtn, () => {
            if (!unoSelectedCard) return;
            const card = unoSelectedCard;
            const selEl = hand.querySelector('.uno-card.sel');
            if (selEl) unoRememberFlyFrom(selEl);
            unoSelectedCard = null;
            sendUCI('play:' + card);
        }));
        // UNO button; when just drew with no playable cards allow pass via long-press? Use UNO slot disabled and show pass when needed
        const needPass = canSelect && !!st.justDrew
            && !handCards.some((c) => unoCanPlayCard(st, c));
        if (needPass) {
            bar.appendChild(mkAct('pass uno-act-pass', '⏭ ' + gT('uno.pass', 'BỎ LƯỢT'), true, () => {
                unoSelectedCard = null;
                sendUCI('pass');
            }));
        } else {
            bar.appendChild(mkAct('uno uno-act-uno', 'UNO', canUno, () => sendUCI('uno')));
        }
        // Always offer pass when your turn & can't play any card (even without justDrew)
        if (canSelect && !needPass && !handCards.some((c) => unoCanPlayCard(st, c))) {
            // allow pass only after draw per classic — still enable draw; keep UNO disabled
        }
        youZ.appendChild(bar);
        arena.appendChild(youZ);

        // Color modal OVERLAY
        if (pickColor) {
            const modal = el('div', 'uno-modal uno-color-modal');
            const panel = el('div', 'uno-modal-panel');
            panel.appendChild(el('div', 'uno-modal-title', '🎨 ' + gT('uno.pick_color_title', 'CHỌN MÀU')));
            panel.appendChild(el('div', 'uno-modal-sub',
                st.pendingDraw4
                    ? gT('uno.pick_color_w4', 'Bạn đánh +4 — chọn màu để đánh tiếp')
                    : gT('uno.pick_color', 'Bạn đổi màu — chọn màu để đánh tiếp')));
            const row = el('div', 'uno-color-orbs');
            [['R', gT('uno.color_r', 'Đỏ')], ['Y', gT('uno.color_y', 'Vàng')],
             ['G', gT('uno.color_g', 'Xanh lá')], ['B', gT('uno.color_b', 'Xanh dương')]].forEach(([col, lab]) => {
                const b = el('button', 'uno-color-orb color-' + col, '');
                b.type = 'button';
                b.title = lab;
                b.setAttribute('aria-label', lab);
                b.appendChild(el('span', 'uno-color-orb-lab', lab));
                b.onclick = () => sendUCI('color:' + col);
                row.appendChild(b);
            });
            panel.appendChild(row);
            modal.appendChild(panel);
            arena.appendChild(modal);
        }

        // Win / lose modal OVERLAY
        if (st.status === 'win' || st.winner === 'human' || st.winner === 'bot') {
            const youWin = st.winner === 'human' || (st.status === 'win' && st.winner !== 'bot');
            const modal = el('div', 'uno-modal uno-end-modal');
            const panel = el('div', 'uno-modal-panel ' + (youWin ? 'win' : 'lose'));
            panel.appendChild(el('div', 'uno-modal-emoji', youWin ? '🏆' : '😭'));
            panel.appendChild(el('div', 'uno-modal-title',
                youWin ? gT('uno.congrats', 'CHÚC MỪNG!') : gT('uno.robot_wins_title', 'ROBOT THẮNG')));
            panel.appendChild(el('div', 'uno-modal-sub',
                youWin ? gT('uno.you_win', 'Bạn đã chiến thắng!') : gT('uno.try_again', 'Bạn đã thua. Ván sau nhé.')));
            const acts = el('div', 'uno-modal-acts');
            const again = el('button', 'uno-act play', gT('uno.again', 'Chơi lại'));
            again.type = 'button';
            again.onclick = () => {
                if (typeof gamesNewGame === 'function') gamesNewGame();
                else if (typeof chessNewGame === 'function') chessNewGame();
            };
            const close = el('button', 'uno-act', gT('uno.close', 'Đóng'));
            close.type = 'button';
            close.onclick = () => { if (modal.parentNode) modal.parentNode.removeChild(modal); };
            acts.appendChild(again);
            acts.appendChild(close);
            panel.appendChild(acts);
            modal.appendChild(panel);
            arena.appendChild(modal);

            const cKey = String(st.lastMove || '') + '|' + st.winner + '|' + st.status;
            if (cKey !== unoCelebrateKey) {
                unoCelebrateKey = cKey;
                if (youWin) requestAnimationFrame(() => unoSpawnConfetti(arena));
            }
        }

        board.appendChild(arena);

        // Animations
        if ((slamYou || slamBot) && st.top) {
            const from = slamYou ? 'you' : 'bot';
            const card = slamYou ? (humanPlayed || st.top) : (botPlayed || st.top);
            const flyKey = String(st.lastMove || '') + '|' + st.top + '|' + from;
            if (flyKey !== unoLastFlyKey) {
                unoLastFlyKey = flyKey;
                requestAnimationFrame(() => {
                    unoFlyToPile(board, card, from).then(() => {
                        const fxKey = flyKey + '|fx';
                        if (fxKey !== unoLastFxKey) {
                            unoLastFxKey = fxKey;
                            unoShowFx(board, card);
                        }
                    });
                });
            }
        }

        // Draw anim (forced + optional human draw)
        if ((st.drawAnimN | 0) > 0) {
            requestAnimationFrame(() => { unoAnimateForcedDraw(st); });
        }
    }

    function renderCards(board, st, kind) {
        board.classList.add('extra-board', 'board-cards');
        if (kind === 'bjweb' || st.game === 'blackjack') {
            renderBlackjack(board, st);
            return;
        }
        renderPoker(board, st);
    }

    function pkParse(code) {
        const c = String(code || '').toUpperCase();
        if (!c || c === '??' || c === '?' || c === 'XX') {
            return { back: true, rank: '', suit: '', red: false, code: '??' };
        }
        let rank = c.slice(0, -1);
        let suit = c.slice(-1);
        if (c.length === 3 && c[0] === '1' && c[1] === '0') {
            rank = '10';
            suit = c[2];
        }
        const suitGlyph = { S: '♠', H: '♥', D: '♦', C: '♣' }[suit] || suit;
        const red = suit === 'H' || suit === 'D';
        return { back: false, rank, suit, suitGlyph, red, code: c };
    }

    function pkMakeCard(code, opts) {
        opts = opts || {};
        const p = pkParse(code);
        const b = el('div', 'pcard'
            + (p.back ? ' back' : (p.red ? ' red' : ' black'))
            + (opts.big ? ' big' : '')
            + (opts.compact ? ' compact' : ''), '');
        if (p.back) {
            b.appendChild(el('div', 'pcard-back-logo', '♠'));
            return b;
        }
        const corner = el('div', 'pcard-corner');
        corner.appendChild(el('div', 'pcard-rank', p.rank));
        corner.appendChild(el('div', 'pcard-suit', p.suitGlyph));
        b.appendChild(corner);
        const center = (p.rank === 'J' || p.rank === 'Q' || p.rank === 'K' || p.rank === 'A')
            ? el('div', 'pcard-center face', p.rank)
            : el('div', 'pcard-center', p.suitGlyph);
        b.appendChild(center);
        const corner2 = el('div', 'pcard-corner bottom');
        corner2.appendChild(el('div', 'pcard-rank', p.rank));
        corner2.appendChild(el('div', 'pcard-suit', p.suitGlyph));
        b.appendChild(corner2);
        return b;
    }

    function pkHandRow(cards, opts) {
        opts = opts || {};
        const row = el('div', 'pcard-hand' + (opts.cls ? ' ' + opts.cls : ''));
        (cards || []).forEach((c) => row.appendChild(pkMakeCard(c)));
        if (!cards || !cards.length) row.appendChild(el('div', 'pcard-empty', '—'));
        return row;
    }

    let bjLastFlyKey = '';
    let bjCelebrateKey = '';

    function bjSleep(ms) {
        return new Promise((r) => setTimeout(r, ms));
    }

    async function bjAnimateDraw(st) {
        const to = st && st.drawAnimTo ? String(st.drawAnimTo) : '';
        const card = st && st.lastCard ? String(st.lastCard) : '';
        if (!to || !card || card === '??') return;
        const key = String(st.lastMove || '') + '|' + to + '|' + card + '|' + (st.playerHand || []).length + '|' + (st.dealerHand || []).length;
        if (key === bjLastFlyKey) return;
        bjLastFlyKey = key;

        const board = document.getElementById('chessBoard');
        if (!board) return;
        const destSel = to === 'dealer' ? '.bj-dealer-sec .pcard-hand' : '.bj-you-sec .pcard-hand';
        const destEl = board.querySelector(destSel) || board.querySelector(to === 'dealer' ? '.bj-dealer-sec' : '.bj-you-sec');
        const startEl = board.querySelector('.bj-table') || board;
        if (!destEl) return;

        const start = startEl.getBoundingClientRect();
        const dest = destEl.getBoundingClientRect();
        document.querySelectorAll('.bj-flyer').forEach((n) => { if (n.parentNode) n.parentNode.removeChild(n); });

        const flyer = pkMakeCard(card);
        flyer.classList.add('bj-flyer');
        const w = 64, h = 90;
        flyer.style.width = w + 'px';
        flyer.style.height = h + 'px';
        const x0 = start.left + start.width / 2 - w / 2;
        const y0 = start.top + start.height / 2 - h / 2;
        const x1 = dest.left + Math.max(0, dest.width - w - 8);
        const y1 = dest.top + dest.height / 2 - h / 2;
        flyer.style.transform = 'translate(' + x0 + 'px,' + y0 + 'px) scale(0.85) rotate(-8deg)';
        document.body.appendChild(flyer);
        void flyer.offsetWidth;
        flyer.style.transform = 'translate(' + x1 + 'px,' + y1 + 'px) scale(1) rotate(0deg)';
        await bjSleep(480);
        if (flyer.parentNode) flyer.parentNode.removeChild(flyer);
    }

    global.bjAnimateDraw = bjAnimateDraw;

    function bjSpawnConfetti(host) {
        const layer = el('div', 'uno-confetti-layer');
        const colors = ['#f6e05e', '#68d391', '#63b3ed', '#fc8181', '#b794f4', '#fbd38d', '#fff'];
        for (let i = 0; i < 40; i++) {
            const p = el('div', 'uno-confetti');
            p.style.left = (Math.random() * 100) + '%';
            p.style.background = colors[i % colors.length];
            p.style.animationDelay = (Math.random() * 0.7) + 's';
            p.style.animationDuration = (1.5 + Math.random() * 1.3) + 's';
            layer.appendChild(p);
        }
        host.appendChild(layer);
        setTimeout(() => { if (layer.parentNode) layer.parentNode.removeChild(layer); }, 4000);
    }

    function renderBlackjack(board, st) {
        board.classList.add('board-bj');
        const bank = st.bank != null ? st.bank : 100;
        const dealt = !!st.dealt;
        const botThinking = !!st.botThinking;
        const playing = st.status === 'playing' && dealt;
        const yourTurn = playing && !botThinking && st.turn === 'human';
        const dealerTurn = playing && (botThinking || st.turn === 'bot' || st.dealerRevealed);

        const top = el('div', 'bj-topbar');
        top.appendChild(el('div', 'bj-bank', gT('bj.bank', 'Tiền') + ': ' + bank));
        if (st.message) top.appendChild(el('div', 'bj-msg', st.message));
        board.appendChild(top);

        // Dealer
        const dealer = el('div', 'uno-section bj-dealer-sec' + (dealerTurn ? ' active-turn' : ''));
        const dHead = el('div', 'uno-section-head');
        dHead.appendChild(el('div', 'uno-section-title', gT('bj.dealer', '🤖 Nhà cái (Robot)')));
        if (dealerTurn && playing) {
            dHead.appendChild(el('div', 'uno-turn-pill bot', gT('bj.turn_dealer', 'LƯỢT NHÀ CÁI')));
        }
        const dVal = (playing && !st.dealerRevealed)
            ? ((st.dealerValue != null ? st.dealerValue : '?') + '+')
            : String(st.dealerValue != null ? st.dealerValue : '—');
        dHead.appendChild(el('div', 'uno-count-badge', gT('bj.score', 'Điểm') + ': ' + dVal));
        dealer.appendChild(dHead);
        dealer.appendChild(pkHandRow(st.dealerHand || [], { cls: 'dealer-hand' }));
        if (botThinking) {
            dealer.appendChild(el('div', 'bj-thinking', gT('bj.thinking', 'Nhà cái đang rút…')));
        }
        board.appendChild(dealer);

        // Table center status
        const mid = el('div', 'bj-table');
        if (!dealt) {
            mid.appendChild(el('div', 'bj-table-hint', gT('bj.hint_deal', 'Bấm Bắt đầu để chia bài')));
        } else if (st.status === 'win') {
            mid.appendChild(el('div', 'bj-result win', gT('bj.win', 'Bạn thắng!')));
            mid.appendChild(el('div', 'bj-result-sub', gT('bj.congrats', 'Chúc mừng!')));
        } else if (st.status === 'lose') {
            mid.appendChild(el('div', 'bj-result lose', gT('bj.lose', 'Bạn thua')));
        } else if (st.status === 'draw') {
            mid.appendChild(el('div', 'bj-result draw', gT('bj.draw', 'Hoà')));
        } else if (botThinking) {
            mid.appendChild(el('div', 'bj-table-hint', gT('bj.hint_dealer', 'Nhà cái đang rút từng lá…')));
        } else if (yourTurn) {
            mid.appendChild(el('div', 'bj-table-hint', gT('bj.hint_play', 'Rút bài thêm hoặc Dừng rút')));
        } else {
            mid.appendChild(el('div', 'bj-table-hint', '…'));
        }
        board.appendChild(mid);

        // Player
        const you = el('div', 'uno-section bj-you-sec' + (yourTurn ? ' active-turn' : '') + (st.status === 'win' ? ' winner' : ''));
        const yHead = el('div', 'uno-section-head');
        yHead.appendChild(el('div', 'uno-section-title', gT('bj.you', 'Bạn')));
        if (yourTurn) {
            yHead.appendChild(el('div', 'uno-turn-pill you', gT('bj.turn_you', 'LƯỢT CỦA BẠN')));
        }
        yHead.appendChild(el('div', 'uno-count-badge', gT('bj.score', 'Điểm') + ': ' + (st.playerValue != null ? st.playerValue : '—')));
        you.appendChild(yHead);
        you.appendChild(pkHandRow(st.playerHand || [], { cls: 'you-hand' }));
        board.appendChild(you);

        const pad = el('div', 'extra-actions bj-actions');
        const canAct = yourTurn && !botThinking;
        const mk = (uci, lab, primary, enabled) => {
            const b = el('button', 'extra-btn' + (primary ? ' primary' : ''), lab);
            b.type = 'button';
            b.disabled = !enabled;
            b.onclick = () => { if (enabled) sendUCI(uci); };
            return b;
        };
        pad.appendChild(mk('deal', gT('bj.deal', 'Bắt đầu'), true, !playing && !botThinking));
        pad.appendChild(mk('hit', gT('bj.hit', 'Rút bài'), false, canAct));
        pad.appendChild(mk('stand', gT('bj.stand', 'Dừng rút'), false, canAct));
        board.appendChild(pad);

        if (st.drawAnimTo && st.lastCard) {
            requestAnimationFrame(() => { bjAnimateDraw(st); });
        }
        if (st.status === 'win') {
            const cKey = 'win|' + (st.playerValue || '') + '|' + (st.dealerValue || '') + '|' + bank;
            if (cKey !== bjCelebrateKey) {
                bjCelebrateKey = cKey;
                requestAnimationFrame(() => bjSpawnConfetti(board));
            }
        }
    }

    let pokerSelected = new Set();
    let pokerDealAnimKey = '';
    let pokerWinKey = '';

    function pokerPhaseLabel(st) {
        if (st.status === 'win' || st.status === 'lose' || st.status === 'draw') {
            return gT('poker.phase_show', 'Đã mở bài');
        }
        if (!st.dealt) return gT('poker.phase_idle', 'Chờ chia bài');
        if (!st.drawn) return gT('poker.phase_select', 'Đang chọn bài');
        return gT('poker.phase_ready', 'Sẵn sàng mở bài');
    }

    function pokerSleep(ms) {
        return new Promise((r) => setTimeout(r, ms));
    }

    async function pokerSoftComment(kind) {
        try {
            const st = await gamesFetch('move', {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: 'uci=' + encodeURIComponent('comment:' + kind),
            });
            const bubble = document.querySelector('.poker-speech-text');
            if (bubble && st.message) bubble.textContent = st.message;
            if (typeof chessSetMeta === 'function' && st.message) chessSetMeta(st.message);
        } catch (_) { /* ignore */ }
    }

    function renderPoker(board, st) {
        board.classList.add('extra-board', 'board-poker');
        const dealt = !!st.dealt || (st.playerHand && st.playerHand.length === 5);
        const drawn = !!st.drawn;
        const playing = st.status === 'playing';
        const canSelect = dealt && playing && !drawn;
        const canDeal = !playing || !dealt;
        const canShow = dealt && playing;
        const canDrawBtn = canSelect;
        const hand = st.playerHand || st.hand || [];
        const botHand = st.botHand || [];

        // Drop stale selection when hand changes size / after draw/show
        if (!canSelect) pokerSelected.clear();
        pokerSelected.forEach((i) => {
            if (i < 0 || i >= hand.length) pokerSelected.delete(i);
        });

        // Hero
        const hero = el('div', 'poker-hero');
        hero.appendChild(el('div', 'poker-hero-ico', '🃏'));
        hero.appendChild(el('div', 'poker-hero-title', gT('poker.title', 'Poker 5 Lá')));
        hero.appendChild(el('div', 'poker-hero-sub', gT('poker.subtitle', 'Chia bài, đổi bài và tạo ra bộ bài mạnh nhất.')));
        board.appendChild(hero);

        // Speech bubble
        const speech = el('div', 'poker-speech');
        speech.appendChild(el('div', 'poker-speech-ico', '🤖'));
        const wave = el('div', 'poker-speech-wave');
        wave.appendChild(el('span', '', ''));
        wave.appendChild(el('span', '', ''));
        wave.appendChild(el('span', '', ''));
        speech.appendChild(wave);
        speech.appendChild(el('div', 'poker-speech-text', st.message || gT('poker.ready', 'Sẵn sàng chơi.')));
        board.appendChild(speech);

        // Player info card
        const info = el('div', 'poker-player-card');
        info.appendChild(el('div', 'poker-player-name', gT('poker.you', 'Bạn')));
        const meta = el('div', 'poker-player-meta');
        meta.appendChild(el('span', '', (hand.length || 0) + ' ' + gT('poker.cards', 'lá')));
        meta.appendChild(el('span', 'dot', '·'));
        meta.appendChild(el('span', '', pokerPhaseLabel(st)));
        info.appendChild(meta);
        const selCount = pokerSelected.size;
        const selLine = el('div', 'poker-select-line');
        if (canSelect) {
            selLine.textContent = gT('poker.selected', 'Đã chọn') + ' ' + selCount + '/5 ' + gT('poker.to_swap', 'lá để đổi');
        } else if (st.playerRank) {
            selLine.textContent = gT('poker.hand', 'Bộ bài') + ': ' + st.playerRank;
        } else {
            selLine.textContent = ' ';
        }
        info.appendChild(selLine);
        board.appendChild(info);

        // Felt table
        const table = el('div', 'poker-felt');
        if (st.status === 'win' || st.status === 'lose' || st.status === 'draw') {
            const badge = el('div', 'poker-hand-badge ' + (st.status === 'win' ? 'win' : (st.status === 'lose' ? 'lose' : 'push')));
            badge.appendChild(el('div', 'poker-hand-badge-title', '🏅 ' + (st.playerRank || '—')));
            if (st.botRank) {
                badge.appendChild(el('div', 'poker-hand-badge-sub',
                    gT('poker.vs', 'Đối thủ') + ': ' + st.botRank));
            }
            table.appendChild(badge);
        }

        // Bot row (backs or revealed)
        if (botHand.length) {
            const botRow = el('div', 'poker-bot-row');
            botRow.appendChild(el('div', 'poker-bot-lab', gT('poker.robot', '🤖 Robot')));
            const botCards = el('div', 'poker-hand bot');
            botHand.forEach((c) => botCards.appendChild(pkMakeCard(c, { compact: true })));
            botRow.appendChild(botCards);
            table.appendChild(botRow);
        }

        // Player hand on table
        const handRow = el('div', 'poker-hand you');
        const dealKey = String(st.lastMove || '') + '|' + hand.join(',');
        const animateDeal = st.lastMove === 'deal' && dealKey !== pokerDealAnimKey;
        if (animateDeal) pokerDealAnimKey = dealKey;

        hand.forEach((c, i) => {
            const card = pkMakeCard(c, { compact: true });
            card.classList.add('clickable');
            if (pokerSelected.has(i)) card.classList.add('sel', 'lifted');
            if (animateDeal) {
                card.classList.add('deal-in');
                card.style.animationDelay = (i * 0.12) + 's';
            }
            if (canSelect) {
                card.onclick = () => {
                    if (pokerSelected.has(i)) {
                        pokerSelected.delete(i);
                        card.classList.remove('sel', 'lifted');
                        pokerSoftComment('unpick');
                    } else {
                        pokerSelected.add(i);
                        card.classList.add('sel', 'lifted');
                        pokerSoftComment('pick');
                    }
                    const line = board.querySelector('.poker-select-line');
                    if (line) {
                        line.textContent = gT('poker.selected', 'Đã chọn') + ' ' + pokerSelected.size + '/5 ' + gT('poker.to_swap', 'lá để đổi');
                    }
                    const drawBtn = board.querySelector('.poker-btn.draw');
                    if (drawBtn) drawBtn.disabled = pokerSelected.size === 0;
                };
            }
            handRow.appendChild(card);
        });
        if (!hand.length) {
            // Placeholder backs before deal
            for (let i = 0; i < 5; i++) {
                const back = pkMakeCard('??', { compact: true });
                back.classList.add('ghost');
                handRow.appendChild(back);
            }
        }
        table.appendChild(handRow);
        board.appendChild(table);

        // Actions
        const pad = el('div', 'poker-actions');
        // Note: onclick must re-check .disabled, not a closed-over `enabled` flag —
        // draw is enabled later when the player selects cards without re-rendering.
        const mk = (cls, label, enabled, fn) => {
            const b = el('button', 'poker-btn ' + cls, label);
            b.type = 'button';
            b.disabled = !enabled;
            b.onclick = () => { if (b.disabled) return; fn(); };
            return b;
        };
        pad.appendChild(mk('deal', '🂠 ' + gT('poker.deal', 'Chia bài'), canDeal, () => {
            pokerSelected.clear();
            sendUCI('deal');
        }));
        pad.appendChild(mk('show', '👁 ' + gT('poker.show', 'Mở bài'), canShow, () => sendUCI('show')));
        const drawBtn = mk('draw', '🔄 ' + gT('poker.draw', 'Đổi bài đã chọn'), canDrawBtn && selCount > 0, () => {
            if (!canDrawBtn) return;
            const idxs = Array.from(pokerSelected).sort((a, b) => a - b);
            if (!idxs.length) return;
            pokerSelected.clear();
            sendUCI('draw:' + idxs.join(','));
        });
        pad.appendChild(drawBtn);
        board.appendChild(pad);

        // History
        const hist = Array.isArray(st.history) ? st.history : [];
        if (hist.length) {
            const hWrap = el('div', 'poker-hist');
            hWrap.appendChild(el('div', 'poker-hist-lab', gT('poker.history', 'Lịch sử ván gần đây')));
            const chips = el('div', 'poker-hist-chips');
            hist.slice().reverse().forEach((h) => {
                const res = h.result === 'win' ? gT('poker.res_win', 'Thắng')
                    : (h.result === 'lose' ? gT('poker.res_lose', 'Thua') : gT('poker.res_draw', 'Hoà'));
                chips.appendChild(el('span', 'poker-chip ' + (h.result || ''),
                    gT('poker.round', 'Ván') + ' ' + h.round + ' — ' + (h.hand || '?') + ' — ' + res));
            });
            hWrap.appendChild(chips);
            board.appendChild(hWrap);
        }

        // Win / lose card
        if (st.status === 'win' || st.status === 'lose' || st.status === 'draw') {
            const end = el('div', 'poker-end ' + st.status);
            if (st.status === 'win') {
                end.appendChild(el('div', 'poker-end-emoji', '🏆'));
                end.appendChild(el('div', 'poker-end-title', gT('poker.congrats', 'CHÚC MỪNG!')));
                end.appendChild(el('div', 'poker-end-sub', gT('poker.you_won', 'Bạn đã chiến thắng!')));
            } else if (st.status === 'lose') {
                end.appendChild(el('div', 'poker-end-emoji', '😅'));
                end.appendChild(el('div', 'poker-end-title', gT('poker.lost', 'CHƯA THẮNG')));
                end.appendChild(el('div', 'poker-end-sub', gT('poker.try_again', 'Thử lại ván mới nhé')));
            } else {
                end.appendChild(el('div', 'poker-end-emoji', '🤝'));
                end.appendChild(el('div', 'poker-end-title', gT('poker.push', 'HOÀ')));
                end.appendChild(el('div', 'poker-end-sub', gT('poker.push_sub', 'Cân sức')));
            }
            const grid = el('div', 'poker-end-stats');
            const mkStat = (lab, val) => {
                const c = el('div', 'poker-stat');
                c.appendChild(el('div', 'poker-stat-lab', lab));
                c.appendChild(el('div', 'poker-stat-val', val));
                return c;
            };
            grid.appendChild(mkStat(gT('poker.hand', 'Bộ bài'), st.playerRank || '—'));
            grid.appendChild(mkStat(gT('poker.points', 'Điểm'), String(st.score != null ? st.score : '—')));
            grid.appendChild(mkStat(gT('poker.swaps', 'Số lần đổi'), String(st.drawCount != null ? st.drawCount : 0)));
            end.appendChild(grid);
            const again = el('button', 'poker-btn deal again', gT('poker.again', 'Chơi lại'));
            again.type = 'button';
            again.onclick = () => {
                if (typeof chessNewGame === 'function') chessNewGame();
            };
            end.appendChild(again);
            board.appendChild(end);

            if (st.status === 'win') {
                const key = 'win|' + (st.playerRank || '') + '|' + (st.score || '');
                if (key !== pokerWinKey) {
                    pokerWinKey = key;
                    requestAnimationFrame(() => {
                        const layer = el('div', 'uno-confetti-layer');
                        const colors = ['#f6e05e', '#b794f4', '#68d391', '#fc8181', '#fff', '#fbd38d'];
                        for (let i = 0; i < 46; i++) {
                            const p = el('div', 'uno-confetti');
                            p.style.left = (Math.random() * 100) + '%';
                            p.style.background = colors[i % colors.length];
                            p.style.animationDelay = (Math.random() * 0.8) + 's';
                            p.style.animationDuration = (1.5 + Math.random() * 1.3) + 's';
                            layer.appendChild(p);
                        }
                        board.appendChild(layer);
                        setTimeout(() => { if (layer.parentNode) layer.parentNode.removeChild(layer); }, 4200);
                    });
                }
            }
        }
    }

    function renderTicTacToe(board, st) {
        if (typeof chessRenderGrid === 'function') {
            board.classList.add('grid-board', 'ttt-board');
            chessRenderGrid(board, st, {
                files: 3, ranks: [2, 1, 0], disc: true, checker: true, showHints: false, showLast: true,
            });
            if (typeof chessRenderCoords === 'function') {
                chessRenderCoords({ files: 3, ranks: [2, 1, 0], matchBoard: true });
            }
        }
    }

    function gamesRenderExtra(board, st, id) {
        if (board._keyHandler) {
            window.removeEventListener('keydown', board._keyHandler);
            board._keyHandler = null;
        }
        const mode = (st && st.uiMode) || id;
        board.innerHTML = '';
        board.className = 'chess-board extra-host';
        switch (mode) {
        case 'tictactoe': return renderTicTacToe(board, st);
        case 'sudoku': return renderSudoku(board, st);
        case 'tiles2048': case 'g2048': return render2048(board, st);
        case 'mines': return renderMines(board, st);
        case 'memory': return renderMemory(board, st);
        case 'battleship': return renderBattleship(board, st);
        case 'wordle': return renderWordle(board, st);
        case 'hangman': return renderHangman(board, st);
        case 'trivia': return renderTrivia(board, st);
        case 'guess': case 'guessnum': return renderGuess(board, st);
        case 'simon': return renderSimon(board, st);
        case 'uno': return renderUno(board, st);
        case 'cards':
            return renderCards(board, st, id);
        default:
            board.appendChild(el('div', '', 'Unknown uiMode: ' + mode));
        }
    }

    global.EXTRA_META = EXTRA_META;
    global.gamesIsExtraGame = isExtraGame;
    global.gamesRenderExtra = gamesRenderExtra;
    global.GAMES_EXTRA_IDS = EXTRA_IDS;
})(window);
