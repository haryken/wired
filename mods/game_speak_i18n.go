package mods

import (
	"fmt"
	"strings"
)

// gameSpeakLang is the language of spoken game comments.
// Google TTS: match the Games lobby voice (vi, zh-CN, it, …).
// SayText / Xiaozhi: English facts (Acapela is English; Xiaozhi prompt is separate).
func gameSpeakLang() string {
	if override := strings.TrimSpace(chessSpeakLangOverride); override != "" {
		return normalizeGoogleTTSLang(override)
	}
	if getChessCommentMode() == chessModeGoogleVI {
		return chessGoogleTTSLang()
	}
	return "en"
}

// chessSpeakLangOverride is for tests only.
var chessSpeakLangOverride string

// L is a phrase table keyed by Google TTS language code.
type L map[string]string

func (m L) get() string {
	if m == nil {
		return ""
	}
	lang := gameSpeakLang()
	if s := strings.TrimSpace(m[lang]); s != "" {
		return s
	}
	if s := strings.TrimSpace(m["en"]); s != "" {
		return s
	}
	return strings.TrimSpace(m["vi"])
}

func (m L) sprintf(args ...interface{}) string {
	s := m.get()
	if s == "" || len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// localizeSpeak picks vi / en / other-lang text for the current speak language.
func localizeSpeak(en, vi string) string {
	switch lang := gameSpeakLang(); lang {
	case "vi":
		return vi
	case "en":
		return en
	default:
		if t := phraseByEN[en][lang]; t != "" {
			return t
		}
		return en
	}
}

// speakNew is localizeSpeak as a (say, summary) pair for new-game lines.
func speakNew(en, vi string) (string, string) {
	s := localizeSpeak(en, vi)
	return s, s
}

// speakf localizes a printf template (English format is the catalog key).
func speakf(viFmt, enFmt string, args ...interface{}) (string, string) {
	lang := gameSpeakLang()
	tpl := enFmt
	switch lang {
	case "vi":
		tpl = viFmt
	case "en":
		tpl = enFmt
	default:
		if t := phraseByEN[enFmt][lang]; t != "" {
			tpl = t
		}
	}
	s := fmt.Sprintf(tpl, args...)
	return s, s
}

func wordL(table map[string]L, key, fallback string) string {
	if table == nil {
		return fallback
	}
	if m, ok := table[strings.ToLower(strings.TrimSpace(key))]; ok {
		if s := m.get(); s != "" {
			return s
		}
	}
	if m, ok := table[strings.TrimSpace(key)]; ok {
		if s := m.get(); s != "" {
			return s
		}
	}
	return fallback
}

// phraseByEN maps English source (or English printf format) → other Google langs.
// Vietnamese is taken from the viOrEN / speakNew first argument, not this table.
var phraseByEN = map[string]map[string]string{
	"New game. You are white. Your move.": {
		"zh-CN": "新对局。你执白棋。该你走了。",
		"it":    "Nuova partita. Sei il bianco. Tocca a te.",
		"ru":    "Новая партия. Вы играете белыми. Ваш ход.",
		"fr":    "Nouvelle partie. Vous êtes les blancs. À vous.",
		"de":    "Neue Partie. Du spielst Weiß. Du bist am Zug.",
		"es":    "Nueva partida. Eres blancas. Te toca.",
		"pt":    "Novo jogo. Você é as brancas. Sua vez.",
	},
	"New game. You are red. Your move.": {
		"zh-CN": "新对局。你执红棋。该你走了。",
		"it":    "Nuova partita. Sei il rosso. Tocca a te.",
		"ru":    "Новая партия. Вы играете красными. Ваш ход.",
		"fr":    "Nouvelle partie. Vous êtes les rouges. À vous.",
		"de":    "Neue Partie. Du spielst Rot. Du bist am Zug.",
		"es":    "Nueva partida. Eres rojas. Te toca.",
		"pt":    "Novo jogo. Você é o vermelho. Sua vez.",
	},
	"New caro game. You are X. Freestyle with double-open bans. Your move.": {
		"zh-CN": "新的五子棋。你执 X。自由开局，禁双头活三。该你走了。",
		"it":    "Nuova partita di Caro. Sei X. Freestyle con divieto di doppia apertura. Tocca a te.",
		"ru":    "Новая партия каро. Вы играете X. Свободный стиль, запрет двойного открытого ряда. Ваш ход.",
		"fr":    "Nouvelle partie de Caro. Vous êtes X. Freestyle, interdiction des doubles ouvertures. À vous.",
		"de":    "Neues Caro. Du bist X. Freestyle mit Verbot doppelter offener Reihen. Du bist am Zug.",
		"es":    "Nueva partida de Caro. Eres X. Estilo libre, prohibidas las dobles abiertas. Te toca.",
		"pt":    "Novo Caro. Você é X. Freestyle, proibido dois abertos. Sua vez.",
	},
	"New Connect Four. You drop first. Your move.": {
		"zh-CN": "新的四子棋。你先落子。该你了。",
		"it":    "Nuovo Forza 4. Inizi tu. Tocca a te.",
		"ru":    "Новая партия в «Четыре в ряд». Вы ходите первыми.",
		"fr":    "Nouveau Puissance 4. Vous jouez en premier. À vous.",
		"de":    "Neues Vier gewinnt. Du wirfst zuerst. Du bist am Zug.",
		"es":    "Nuevo Conecta 4. Tiras primero. Te toca.",
		"pt":    "Novo Liga 4. Você começa. Sua vez.",
	},
	"New Reversi. You are black. Your move.": {
		"zh-CN": "新的黑白棋。你执黑。该你走了。",
		"it":    "Nuovo Reversi. Sei il nero. Tocca a te.",
		"ru":    "Новая партия реверси. Вы играете чёрными. Ваш ход.",
		"fr":    "Nouveau Reversi. Vous êtes les noirs. À vous.",
		"de":    "Neues Reversi. Du spielst Schwarz. Du bist am Zug.",
		"es":    "Nuevo Reversi. Eres negras. Te toca.",
		"pt":    "Novo Reversi. Você é as pretas. Sua vez.",
	},
	"New checkers game. You are white. Your move.": {
		"zh-CN": "新的跳棋。你执白。该你走了。",
		"it":    "Nuova dama. Sei il bianco. Tocca a te.",
		"ru":    "Новые шашки. Вы играете белыми. Ваш ход.",
		"fr":    "Nouvelle partie de dames. Vous êtes les blancs. À vous.",
		"de":    "Neues Dame-Spiel. Du spielst Weiß. Du bist am Zug.",
		"es":    "Nuevas damas. Eres blancas. Te toca.",
		"pt":    "Novas damas. Você é as brancas. Sua vez.",
	},
	"New 9 by 9 go game. You are black. Your move.": {
		"zh-CN": "新的九路围棋。你执黑。该你走了。",
		"it":    "Nuova partita di Go 9 per 9. Sei il nero. Tocca a te.",
		"ru":    "Новое го 9×9. Вы играете чёрными. Ваш ход.",
		"fr":    "Nouveau go 9 par 9. Vous êtes les noirs. À vous.",
		"de":    "Neues Go 9 mal 9. Du spielst Schwarz. Du bist am Zug.",
		"es":    "Nuevo go 9 por 9. Eres negras. Te toca.",
		"pt":    "Novo go 9 por 9. Você é as pretas. Sua vez.",
	},
	"New battleship game. Fire at the enemy board.": {
		"zh-CN": "新的海战棋。向对方棋盘开火。",
		"it":    "Nuova battaglia navale. Fuoco sulla griglia nemica.",
		"ru":    "Новой морской бой. Стреляйте по полю противника.",
		"fr":    "Nouvelle bataille navale. Tirez sur la grille adverse.",
		"de":    "Neues Schiffe versenken. Schieß auf das gegnerische Feld.",
		"es":    "Nueva batalla naval. Dispara al tablero enemigo.",
		"pt":    "Nova batalha naval. Atire no tabuleiro inimigo.",
	},
	"New memory game. Flip two cards to find matching pairs.": {
		"zh-CN": "新的记忆翻牌。翻两张牌找对子。",
		"it":    "Nuovo memory. Gira due carte per trovare le coppie.",
		"ru":    "Новая игра на память. Откройте две карты и найдите пары.",
		"fr":    "Nouveau memory. Retournez deux cartes pour trouver les paires.",
		"de":    "Neues Memory. Decke zwei Karten auf, um Paare zu finden.",
		"es":    "Nuevo memory. Voltea dos cartas para encontrar parejas.",
		"pt":    "Novo jogo da memória. Vire duas cartas para achar os pares.",
	},
	"New minesweeper game. Reveal a cell to start.": {
		"zh-CN": "新的扫雷。先翻开一格。",
		"it":    "Nuovo campo minato. Scopri una cella per iniziare.",
		"ru":    "Новый сапёр. Откройте клетку, чтобы начать.",
		"fr":    "Nouveau démineur. Révélez une case pour commencer.",
		"de":    "Neues Minesweeper. Decke ein Feld auf, um zu starten.",
		"es":    "Nuevo buscaminas. Destapa una casilla para empezar.",
		"pt":    "Novo campo minado. Revele uma célula para começar.",
	},
	"New tic-tac-toe game. You are X and go first. Your move.": {
		"zh-CN": "新的井字棋。你执 X，先走。该你了。",
		"it":    "Nuovo tris. Sei X e inizi tu. Tocca a te.",
		"ru":    "Новые крестики-нолики. Вы играете X и ходите первыми.",
		"fr":    "Nouveau morpion. Vous êtes X et jouez en premier. À vous.",
		"de":    "Neues Tic-Tac-Toe. Du bist X und fängst an. Du bist am Zug.",
		"es":    "Nuevo tres en raya. Eres X y empiezas. Te toca.",
		"pt":    "Novo jogo da velha. Você é X e começa. Sua vez.",
	},
	"New 2048 game. Merge matching tiles to reach 2048.": {
		"zh-CN": "新的 2048。合并相同数字，凑到 2048。",
		"it":    "Nuovo 2048. Unisci le tessere uguali per arrivare a 2048.",
		"ru":    "Новая игра 2048. Складывайте одинаковые плитки, чтобы получить 2048.",
		"fr":    "Nouveau 2048. Fusionnez les tuiles identiques pour atteindre 2048.",
		"de":    "Neues 2048. Verschmelze gleiche Steine, um 2048 zu erreichen.",
		"es":    "Nuevo 2048. Junta fichas iguales para llegar a 2048.",
		"pt":    "Novo 2048. Junte peças iguais até chegar a 2048.",
	},
	"New sudoku puzzle. Fill in digits 1 to 9.": {
		"zh-CN": "新的数独。填入 1 到 9。",
		"it":    "Nuovo sudoku. Inserisci le cifre da 1 a 9.",
		"ru":    "Новое судоку. Заполните цифры от 1 до 9.",
		"fr":    "Nouveau sudoku. Remplissez les chiffres de 1 à 9.",
		"de":    "Neues Sudoku. Trage die Ziffern 1 bis 9 ein.",
		"es":    "Nuevo sudoku. Rellena los dígitos del 1 al 9.",
		"pt":    "Novo sudoku. Preencha os dígitos de 1 a 9.",
	},
	"New Blackjack round. Press Start to deal.": {
		"zh-CN": "新的二十一点。按开始发牌。",
		"it":    "Nuovo Blackjack. Premi Avvia per dare le carte.",
		"ru":    "Новый раунд блэкджека. Нажмите Старт, чтобы раздать.",
		"fr":    "Nouveau Blackjack. Appuyez sur Démarrer pour distribuer.",
		"de":    "Neue Blackjack-Runde. Drücke Start zum Geben.",
		"es":    "Nueva ronda de Blackjack. Pulsa Iniciar para repartir.",
		"pt":    "Nova rodada de Blackjack. Toque em Iniciar para dar as cartas.",
	},
	"Bust! You lose.": {
		"zh-CN": "爆牌了！你输了。",
		"it":    "Sballato! Hai perso.",
		"ru":    "Перебор! Вы проиграли.",
		"fr":    "Bust ! Vous perdez.",
		"de":    "Bust! Du verlierst.",
		"es":    "¡Te pasaste! Pierdes.",
		"pt":    "Estourou! Você perdeu.",
	},
	"Five-card Charlie! You win!": {
		"zh-CN": "五张查理！你赢了！",
		"it":    "Five-card Charlie! Hai vinto!",
		"ru":    "Пять карт Чарли! Вы победили!",
		"fr":    "Five-card Charlie ! Vous gagnez !",
		"de":    "Five-card Charlie! Du gewinnst!",
		"es":    "¡Five-card Charlie! ¡Ganaste!",
		"pt":    "Five-card Charlie! Você ganhou!",
	},
	"New Simon game. Watch the sequence, then repeat it (0-3).": {
		"zh-CN": "新的西蒙。看序列，再用 0 到 3 重复。",
		"it":    "Nuovo Simon. Guarda la sequenza e ripetila (0-3).",
		"ru":    "Новый Саймон. Смотрите последовательность и повторите (0–3).",
		"fr":    "Nouveau Simon. Regardez la séquence puis répétez-la (0-3).",
		"de":    "Neues Simon. Schau die Folge an und wiederhole sie (0-3).",
		"es":    "Nuevo Simon. Mira la secuencia y repítela (0-3).",
		"pt":    "Novo Simon. Veja a sequência e repita (0-3).",
	},
	"New Simon game. Watch the sequence, then repeat it using pads 0 to 3.": {
		"zh-CN": "新的西蒙。看序列，再用按键 0 到 3 重复。",
		"it":    "Nuovo Simon. Guarda la sequenza e ripetila con i tasti da 0 a 3.",
		"ru":    "Новый Саймон. Смотрите последовательность и повторите кнопками 0–3.",
		"fr":    "Nouveau Simon. Regardez la séquence puis répétez avec les touches 0 à 3.",
		"de":    "Neues Simon. Schau die Folge an und wiederhole sie mit den Tasten 0 bis 3.",
		"es":    "Nuevo Simon. Mira la secuencia y repítela con las teclas 0 a 3.",
		"pt":    "Novo Simon. Veja a sequência e repita com as teclas 0 a 3.",
	},
	"Keep going...": {
		"zh-CN": "继续……",
		"it":    "Continua...",
		"ru":    "Продолжайте...",
		"fr":    "Continuez...",
		"de":    "Weiter...",
		"es":    "Sigue...",
		"pt":    "Continue...",
	},
	"New hangman game. Guess a letter.": {
		"zh-CN": "新的猜词游戏。猜一个字母。",
		"it":    "Nuovo impiccato. Indovina una lettera.",
		"ru":    "Новая виселица. Назовите букву.",
		"fr":    "Nouveau pendu. Devinez une lettre.",
		"de":    "Neues Hangman. Rate einen Buchstaben.",
		"es":    "Nuevo ahorcado. Adivina una letra.",
		"pt":    "Nova forca. Adivinhe uma letra.",
	},
	"New hangman game. Guess a letter of the Vietnamese word (no diacritics).": {
		"zh-CN": "新的猜词。猜越南语单词的一个字母（无声调）。",
		"it":    "Nuovo impiccato. Indovina una lettera della parola vietnamita (senza segni).",
		"ru":    "Новая виселица. Назовите букву вьетнамского слова (без диакритики).",
		"fr":    "Nouveau pendu. Devinez une lettre du mot vietnamien (sans accents).",
		"de":    "Neues Hangman. Rate einen Buchstaben des vietnamesischen Worts (ohne Akzente).",
		"es":    "Nuevo ahorcado. Adivina una letra de la palabra vietnamita (sin tildes).",
		"pt":    "Nova forca. Adivinhe uma letra da palavra vietnamita (sem acentos).",
	},
	"Correct!": {
		"zh-CN": "对了！",
		"it":    "Giusto!",
		"ru":    "Верно!",
		"fr":    "Correct !",
		"de":    "Richtig!",
		"es":    "¡Correcto!",
		"pt":    "Correto!",
	},
	"Wrong!": {
		"zh-CN": "错了！",
		"it":    "Sbagliato!",
		"ru":    "Неверно!",
		"fr":    "Faux !",
		"de":    "Falsch!",
		"es":    "¡Incorrecto!",
		"pt":    "Errado!",
	},
	"You guessed the whole word! You win.": {
		"zh-CN": "你猜对整个词了！你赢了。",
		"it":    "Hai indovinato tutta la parola! Hai vinto.",
		"ru":    "Вы угадали всё слово! Вы победили.",
		"fr":    "Vous avez trouvé tout le mot ! Vous gagnez.",
		"de":    "Du hast das ganze Wort erraten! Du gewinnst.",
		"es":    "¡Adivinaste la palabra entera! Ganaste.",
		"pt":    "Você acertou a palavra inteira! Você ganhou.",
	},
	"Correct! You win.": {
		"zh-CN": "对了！你赢了。",
		"it":    "Giusto! Hai vinto.",
		"ru":    "Верно! Вы победили.",
		"fr":    "Correct ! Vous gagnez.",
		"de":    "Richtig! Du gewinnst.",
		"es":    "¡Correcto! Ganaste.",
		"pt":    "Correto! Você ganhou.",
	},
	"I win!": {
		"zh-CN": "我赢了！",
		"it":    "Ho vinto!",
		"ru":    "Я победил!",
		"fr":    "J'ai gagné !",
		"de":    "Ich gewinne!",
		"es":    "¡Gané!",
		"pt":    "Eu ganhei!",
	},
	"Next game.": {
		"zh-CN": "下一局。",
		"it":    "Prossima partita.",
		"ru":    "Следующая игра.",
		"fr":    "Prochaine partie.",
		"de":    "Nächstes Spiel.",
		"es":    "Siguiente partida.",
		"pt":    "Próximo jogo.",
	},
	"Better luck next time.": {
		"zh-CN": "下次好运。",
		"it":    "Sarà per la prossima.",
		"ru":    "Повезёт в следующий раз.",
		"fr":    "Plus de chance la prochaine fois.",
		"de":    "Beim nächsten Mal mehr Glück.",
		"es":    "Más suerte la próxima.",
		"pt":    "Mais sorte da próxima vez.",
	},
	"I'm out — robot wins.": {
		"zh-CN": "我出完了——机器人赢了。",
		"it":    "Sono a zero — vince il robot.",
		"ru":    "У меня пусто — победил робот.",
		"fr":    "Je n'ai plus de cartes — le robot gagne.",
		"de":    "Ich bin leer — der Roboter gewinnt.",
		"es":    "Me quedé sin cartas — gana el robot.",
		"pt":    "Acabei as cartas — o robô ganhou.",
	},
	"Luck wasn't with you.": {
		"zh-CN": "这次运气不在你这边。",
		"it":    "La fortuna non era con te.",
		"ru":    "Удача была не на вашей стороне.",
		"fr":    "La chance n'était pas avec vous.",
		"de":    "Das Glück war nicht auf deiner Seite.",
		"es":    "La suerte no te acompañó.",
		"pt":    "A sorte não estava com você.",
	},
	"You drew a %s. Your total is %d. Do you want to hit again?": {
		"zh-CN": "你抽到了%s。当前点数 %d。还要加牌吗？",
		"it":    "Hai pescato un %s. Totale %d. Vuoi un'altra carta?",
		"ru":    "Вы взяли %s. Сумма %d. Взять ещё?",
		"fr":    "Vous avez tiré un %s. Total %d. Voulez-vous une autre carte ?",
		"de":    "Du hast %s gezogen. Summe %d. Noch eine Karte?",
		"es":    "Sacaste un %s. Total %d. ¿Quieres otra carta?",
		"pt":    "Você tirou um %s. Total %d. Quer mais uma carta?",
	},
	"Your current score is %d. Do you want to hit?": {
		"zh-CN": "你现在是 %d 点。还要加牌吗？",
		"it":    "Il tuo punteggio è %d. Vuoi un'altra carta?",
		"ru":    "У вас %d. Взять карту?",
		"fr":    "Votre score est %d. Voulez-vous une carte ?",
		"de":    "Dein Stand ist %d. Noch eine Karte?",
		"es":    "Tu puntuación es %d. ¿Quieres carta?",
		"pt":    "Sua pontuação é %d. Quer carta?",
	},
	"Dealer busts with %d! You win!": {
		"zh-CN": "庄家以 %d 点爆牌！你赢了！",
		"it":    "Il banco sballa con %d! Hai vinto!",
		"ru":    "Дилер перебрал с %d! Вы победили!",
		"fr":    "Le croupier dépasse avec %d ! Vous gagnez !",
		"de":    "Der Dealer bustet mit %d! Du gewinnst!",
		"es":    "¡La banca se pasa con %d! ¡Ganaste!",
		"pt":    "A banca estourou com %d! Você ganhou!",
	},
	"You %d — dealer %d. You win!": {
		"zh-CN": "你 %d — 庄家 %d。你赢了！",
		"it":    "Tu %d — banco %d. Hai vinto!",
		"ru":    "Вы %d — дилер %d. Вы победили!",
		"fr":    "Vous %d — croupier %d. Vous gagnez !",
		"de":    "Du %d — Dealer %d. Du gewinnst!",
		"es":    "Tú %d — banca %d. ¡Ganaste!",
		"pt":    "Você %d — banca %d. Você ganhou!",
	},
	"You %d — dealer %d. You lose.": {
		"zh-CN": "你 %d — 庄家 %d。你输了。",
		"it":    "Tu %d — banco %d. Hai perso.",
		"ru":    "Вы %d — дилер %d. Вы проиграли.",
		"fr":    "Vous %d — croupier %d. Vous perdez.",
		"de":    "Du %d — Dealer %d. Du verlierst.",
		"es":    "Tú %d — banca %d. Pierdes.",
		"pt":    "Você %d — banca %d. Você perdeu.",
	},
	"Push at %d.": {
		"zh-CN": "%d 点平局。",
		"it":    "Pareggio a %d.",
		"ru":    "Ничья при %d.",
		"fr":    "Égalité à %d.",
		"de":    "Unentschieden bei %d.",
		"es":    "Empate a %d.",
		"pt":    "Empate em %d.",
	},
	"Standing — dealer's turn. Dealer shows %d.": {
		"zh-CN": "你停牌——轮到庄家。庄家亮出 %d 点。",
		"it":    "Stai — tocca al banco. Il banco mostra %d.",
		"ru":    "Стоп — ход дилера. У дилера %d.",
		"fr":    "Vous restez — au croupier. Il montre %d.",
		"de":    "Du bleibst — der Dealer ist dran. Er zeigt %d.",
		"es":    "Te plantas — turno de la banca. Muestra %d.",
		"pt":    "Você parou — vez da banca. Ela mostra %d.",
	},
	"You drew a %s — total %d. Bust! You lose.": {
		"zh-CN": "你抽到%s — 共 %d 点。爆了！你输了。",
		"it":    "Hai pescato un %s — totale %d. Sballato! Hai perso.",
		"ru":    "Вы взяли %s — сумма %d. Перебор! Вы проиграли.",
		"fr":    "Vous avez tiré un %s — total %d. Bust ! Vous perdez.",
		"de":    "Du zogst %s — Summe %d. Bust! Du verlierst.",
		"es":    "Sacaste un %s — total %d. ¡Te pasaste! Pierdes.",
		"pt":    "Você tirou um %s — total %d. Estourou! Você perdeu.",
	},
	"Dealer draws a %s. Dealer total %d — bust!": {
		"zh-CN": "庄家抽到%s。庄家共 %d 点——爆了！",
		"it":    "Il banco pesca un %s. Totale banco %d — sballa!",
		"ru":    "Дилер берёт %s. Сумма дилера %d — перебор!",
		"fr":    "Le croupier tire un %s. Total %d — bust !",
		"de":    "Der Dealer zieht %s. Summe %d — Bust!",
		"es":    "La banca saca un %s. Total %d — ¡se pasa!",
		"pt":    "A banca tira um %s. Total %d — estourou!",
	},
	"Dealer draws a %s. Dealer total %d — stands.": {
		"zh-CN": "庄家抽到%s。庄家共 %d 点——停牌。",
		"it":    "Il banco pesca un %s. Totale banco %d — sta.",
		"ru":    "Дилер берёт %s. Сумма дилера %d — стоп.",
		"fr":    "Le croupier tire un %s. Total %d — il reste.",
		"de":    "Der Dealer zieht %s. Summe %d — bleibt.",
		"es":    "La banca saca un %s. Total %d — se planta.",
		"pt":    "A banca tira um %s. Total %d — para.",
	},
	"Dealer draws a %s. Dealer total is %d — hits again.": {
		"zh-CN": "庄家抽到%s。庄家现为 %d 点——再要。",
		"it":    "Il banco pesca un %s. Totale banco %d — pesca ancora.",
		"ru":    "Дилер берёт %s. Сейчас у дилера %d — ещё карту.",
		"fr":    "Le croupier tire un %s. Total %d — il reprend.",
		"de":    "Der Dealer zieht %s. Stand %d — zieht nochmal.",
		"es":    "La banca saca un %s. Total %d — pide otra.",
		"pt":    "A banca tira um %s. Total %d — pede outra.",
	},
	"Wrong! You lost at level %d.": {
		"zh-CN": "错了！你在第 %d 关输了。",
		"it":    "Sbagliato! Hai perso al livello %d.",
		"ru":    "Неверно! Вы проиграли на уровне %d.",
		"fr":    "Faux ! Vous avez perdu au niveau %d.",
		"de":    "Falsch! Du hast auf Stufe %d verloren.",
		"es":    "¡Incorrecto! Perdiste en el nivel %d.",
		"pt":    "Errado! Você perdeu no nível %d.",
	},
	"Correct! New sequence has %d steps.": {
		"zh-CN": "对了！新序列有 %d 步。",
		"it":    "Giusto! La nuova sequenza ha %d passi.",
		"ru":    "Верно! Новая последовательность из %d шагов.",
		"fr":    "Correct ! La nouvelle séquence a %d étapes.",
		"de":    "Richtig! Die neue Folge hat %d Schritte.",
		"es":    "¡Correcto! La nueva secuencia tiene %d pasos.",
		"pt":    "Correto! A nova sequência tem %d passos.",
	},
	"You lose! The word was %s.": {
		"zh-CN": "你输了！单词是 %s。",
		"it":    "Hai perso! La parola era %s.",
		"ru":    "Вы проиграли! Слово было %s.",
		"fr":    "Vous perdez ! Le mot était %s.",
		"de":    "Du verlierst! Das Wort war %s.",
		"es":    "¡Perdiste! La palabra era %s.",
		"pt":    "Você perdeu! A palavra era %s.",
	},
	"New trivia round, %d questions. Answer with 0-3.": {
		"zh-CN": "新的问答，共 %d 题。用 0 到 3 作答。",
		"it":    "Nuovo trivia, %d domande. Rispondi con 0-3.",
		"ru":    "Новая викторина, %d вопросов. Отвечайте 0–3.",
		"fr":    "Nouveau quiz, %d questions. Répondez avec 0-3.",
		"de":    "Neues Quiz, %d Fragen. Antworte mit 0-3.",
		"es":    "Nuevo trivia, %d preguntas. Responde con 0-3.",
		"pt":    "Novo quiz, %d perguntas. Responda com 0-3.",
	},
	"New trivia round, %d questions. Answer with 0, 1, 2 or 3.": {
		"zh-CN": "新的问答，共 %d 题。用 0、1、2 或 3 作答。",
		"it":    "Nuovo trivia, %d domande. Rispondi con 0, 1, 2 o 3.",
		"ru":    "Новая викторина, %d вопросов. Отвечайте 0, 1, 2 или 3.",
		"fr":    "Nouveau quiz, %d questions. Répondez avec 0, 1, 2 ou 3.",
		"de":    "Neues Quiz, %d Fragen. Antworte mit 0, 1, 2 oder 3.",
		"es":    "Nuevo trivia, %d preguntas. Responde con 0, 1, 2 o 3.",
		"pt":    "Novo quiz, %d perguntas. Responda com 0, 1, 2 ou 3.",
	},
	"Wrong! The correct answer was: %s.": {
		"zh-CN": "错了！正确答案是：%s。",
		"it":    "Sbagliato! La risposta corretta era: %s.",
		"ru":    "Неверно! Правильный ответ: %s.",
		"fr":    "Faux ! La bonne réponse était : %s.",
		"de":    "Falsch! Die richtige Antwort war: %s.",
		"es":    "¡Incorrecto! La respuesta correcta era: %s.",
		"pt":    "Errado! A resposta certa era: %s.",
	},
	"Done! You scored %d/%d.": {
		"zh-CN": "结束！你得了 %d/%d 分。",
		"it":    "Finito! Hai fatto %d/%d.",
		"ru":    "Готово! Счёт %d/%d.",
		"fr":    "Terminé ! Score %d/%d.",
		"de":    "Fertig! Du hast %d/%d Punkte.",
		"es":    "¡Listo! Puntuaste %d/%d.",
		"pt":    "Pronto! Você fez %d/%d.",
	},
	"New Wordle. Guess the %d-letter Vietnamese word (no diacritics). You have %d guesses.": {
		"zh-CN": "新的 Wordle。猜 %d 个字母的越南语词（无声调）。你有 %d 次机会。",
		"it":    "Nuovo Wordle. Indovina la parola vietnamita di %d lettere (senza segni). Hai %d tentativi.",
		"ru":    "Новый Wordle. Угадайте вьетнамское слово из %d букв (без диакритики). У вас %d попыток.",
		"fr":    "Nouveau Wordle. Devinez le mot vietnamien de %d lettres (sans accents). Vous avez %d essais.",
		"de":    "Neues Wordle. Rate das vietnamesische Wort mit %d Buchstaben (ohne Akzente). Du hast %d Versuche.",
		"es":    "Nuevo Wordle. Adivina la palabra vietnamita de %d letras (sin tildes). Tienes %d intentos.",
		"pt":    "Novo Wordle. Adivinhe a palavra vietnamita de %d letras (sem acentos). Você tem %d tentativas.",
	},
	"New Wordle. Guess the %d-letter Vietnamese word, no diacritics. You have %d guesses.": {
		"zh-CN": "新的 Wordle。猜 %d 个字母的越南语词，无声调。你有 %d 次机会。",
		"it":    "Nuovo Wordle. Indovina la parola vietnamita di %d lettere, senza segni. Hai %d tentativi.",
		"ru":    "Новый Wordle. Угадайте вьетнамское слово из %d букв без диакритики. У вас %d попыток.",
		"fr":    "Nouveau Wordle. Devinez le mot vietnamien de %d lettres, sans accents. Vous avez %d essais.",
		"de":    "Neues Wordle. Rate das vietnamesische Wort mit %d Buchstaben, ohne Akzente. Du hast %d Versuche.",
		"es":    "Nuevo Wordle. Adivina la palabra vietnamita de %d letras, sin tildes. Tienes %d intentos.",
		"pt":    "Novo Wordle. Adivinhe a palavra vietnamita de %d letras, sem acentos. Você tem %d tentativas.",
	},
	"Out of guesses. The word was %s.": {
		"zh-CN": "次数用完。单词是 %s。",
		"it":    "Tentativi finiti. La parola era %s.",
		"ru":    "Попытки кончились. Слово было %s.",
		"fr":    "Plus d'essais. Le mot était %s.",
		"de":    "Keine Versuche mehr. Das Wort war %s.",
		"es":    "Sin intentos. La palabra era %s.",
		"pt":    "Sem tentativas. A palavra era %s.",
	},
	"I picked a number from %d to %d. Guess it!": {
		"zh-CN": "我想了一个从 %d 到 %d 的数字。来猜吧！",
		"it":    "Ho pensato un numero da %d a %d. Indovinalo!",
		"ru":    "Я загадал число от %d до %d. Угадайте!",
		"fr":    "J'ai choisi un nombre de %d à %d. Devinez-le !",
		"de":    "Ich habe eine Zahl von %d bis %d gewählt. Rate sie!",
		"es":    "Pensé un número del %d al %d. ¡Adivínalo!",
		"pt":    "Pensei num número de %d a %d. Adivinhe!",
	},
	"%s The number was %d — you won in %d guesses.": {
		"zh-CN": "%s 数字是 %d — 你用 %d 次猜中了。",
		"it":    "%s Il numero era %d — hai vinto in %d tentativi.",
		"ru":    "%s Число было %d — вы угадали за %d попыток.",
		"fr":    "%s Le nombre était %d — vous avez gagné en %d essais.",
		"de":    "%s Die Zahl war %d — du hast in %d Versuchen gewonnen.",
		"es":    "%s El número era %d — ganaste en %d intentos.",
		"pt":    "%s O número era %d — você acertou em %d tentativas.",
	},
	"Out of guesses! The number was %d.": {
		"zh-CN": "次数用完了！数字是 %d。",
		"it":    "Tentativi finiti! Il numero era %d.",
		"ru":    "Попытки кончились! Число было %d.",
		"fr":    "Plus d'essais ! Le nombre était %d.",
		"de":    "Keine Versuche mehr! Die Zahl war %d.",
		"es":    "¡Sin intentos! El número era %d.",
		"pt":    "Sem tentativas! O número era %d.",
	},
	"Brilliant! You found the secret number.": {
		"zh-CN": "太棒了！你找到了秘密数字。",
		"it":    "Brillante! Hai trovato il numero segreto.",
		"ru":    "Отлично! Вы нашли загаданное число.",
		"fr":    "Bravo ! Vous avez trouvé le nombre secret.",
		"de":    "Brillant! Du hast die geheime Zahl gefunden.",
		"es":    "¡Brillante! Encontraste el número secreto.",
		"pt":    "Brilhante! Você achou o número secreto.",
	},
	"I knew you could do it!": {
		"zh-CN": "我就知道你行！",
		"it":    "Sapevo che ce l'avresti fatta!",
		"ru":    "Я знал, что у вас получится!",
		"fr":    "Je savais que vous y arriveriez !",
		"de":    "Ich wusste, dass du es schaffst!",
		"es":    "¡Sabía que podías!",
		"pt":    "Eu sabia que você conseguia!",
	},
	"Victory!": {
		"zh-CN": "胜利！",
		"it":    "Vittoria!",
		"ru":    "Победа!",
		"fr":    "Victoire !",
		"de":    "Sieg!",
		"es":    "¡Victoria!",
		"pt":    "Vitória!",
	},
	"You're really good at this.": {
		"zh-CN": "你真的很擅长这个。",
		"it":    "Sei davvero bravo.",
		"ru":    "У вас отлично получается.",
		"fr":    "Vous êtes vraiment doué.",
		"de":    "Daran bist du wirklich gut.",
		"es":    "Se te da muy bien.",
		"pt":    "Você é muito bom nisso.",
	},
	"Let's play another round!": {
		"zh-CN": "再来一局吧！",
		"it":    "Facciamo un'altra partita!",
		"ru":    "Сыграем ещё раунд!",
		"fr":    "Encore une manche !",
		"de":    "Noch eine Runde!",
		"es":    "¡Otra ronda!",
		"pt":    "Vamos jogar outra rodada!",
	},
	"Correct! That was a sharp guess.": {
		"zh-CN": "对了！这一猜很准。",
		"it":    "Giusto! Bel tentativo.",
		"ru":    "Верно! Отличная догадка.",
		"fr":    "Correct ! Belle intuition.",
		"de":    "Richtig! Das war ein scharfer Tipp.",
		"es":    "¡Correcto! Buen tino.",
		"pt":    "Correto! Foi um palpite afiado.",
	},
	"Awesome! The secret is out.": {
		"zh-CN": "太好了！秘密揭晓了。",
		"it":    "Ottimo! Il segreto è svelato.",
		"ru":    "Супер! Секрет раскрыт.",
		"fr":    "Génial ! Le secret est levé.",
		"de":    "Super! Das Geheimnis ist raus.",
		"es":    "¡Genial! El secreto salió.",
		"pt":    "Ótimo! O segredo saiu.",
	},
	"New guess-the-number round. I picked a number from %d to %d. You have %d guesses. Good luck!": {
		"zh-CN": "新的猜数字。我想了 %d 到 %d 之间的一个数。你有 %d 次机会。祝你好运！",
		"it":    "Nuovo indovina il numero. Ho scelto un numero da %d a %d. Hai %d tentativi. Buona fortuna!",
		"ru":    "Новая игра «угадай число». Я загадал число от %d до %d. У вас %d попыток. Удачи!",
		"fr":    "Nouveau juste nombre. J'ai choisi un nombre de %d à %d. Vous avez %d essais. Bonne chance !",
		"de":    "Neues Zahlenraten. Ich habe eine Zahl von %d bis %d gewählt. Du hast %d Versuche. Viel Glück!",
		"es":    "Nueva adivina el número. Pensé un número del %d al %d. Tienes %d intentos. ¡Suerte!",
		"pt":    "Novo adivinhe o número. Pensei num número de %d a %d. Você tem %d tentativas. Boa sorte!",
	},
	"Higher.": {
		"zh-CN": "再高一点。", "it": "Più alto.", "ru": "Выше.", "fr": "Plus haut.", "de": "Höher.", "es": "Más alto.", "pt": "Mais alto.",
	},
	"Lower.": {
		"zh-CN": "再低一点。", "it": "Più basso.", "ru": "Ниже.", "fr": "Plus bas.", "de": "Niedriger.", "es": "Más bajo.", "pt": "Mais baixo.",
	},
	"So close! Go a little higher.": {
		"zh-CN": "很接近！再高一点点。", "it": "Quasi! Un po' più alto.", "ru": "Почти! Чуть выше.", "fr": "Presque ! Un peu plus haut.", "de": "Ganz nah! Etwas höher.", "es": "¡Casi! Un poco más alto.", "pt": "Quase! Um pouco mais alto.",
	},
	"So close! Go a little lower.": {
		"zh-CN": "很接近！再低一点点。", "it": "Quasi! Un po' più basso.", "ru": "Почти! Чуть ниже.", "fr": "Presque ! Un peu plus bas.", "de": "Ganz nah! Etwas niedriger.", "es": "¡Casi! Un poco más bajo.", "pt": "Quase! Um pouco mais baixo.",
	},
	"Way too low — go higher!": {
		"zh-CN": "太低了——再高！", "it": "Troppo basso — più alto!", "ru": "Слишком низко — выше!", "fr": "Beaucoup trop bas — plus haut !", "de": "Viel zu niedrig — höher!", "es": "¡Muy bajo — más alto!", "pt": "Muito baixo — mais alto!",
	},
	"Too high — try a smaller number.": {
		"zh-CN": "太高了——试试更小的数。", "it": "Troppo alto — prova un numero più piccolo.", "ru": "Слишком высоко — меньше.", "fr": "Trop haut — un plus petit nombre.", "de": "Zu hoch — eine kleinere Zahl.", "es": "Demasiado alto — un número menor.", "pt": "Alto demais — tente um menor.",
	},
	"%d guesses left.": {
		"zh-CN": "还剩 %d 次。",
		"it":    "Ancora %d tentativi.",
		"ru":    "Осталось %d попыток.",
		"fr":    "Encore %d essais.",
		"de":    "Noch %d Versuche.",
		"es":    "Quedan %d intentos.",
		"pt":    "Restam %d tentativas.",
	},
}

var (
	phraseYouMoved = L{
		"en":    "You moved %s.",
		"vi":    "Bạn đi %s.",
		"zh-CN": "你走了 %s。",
		"it":    "Hai mosso %s.",
		"ru":    "Вы походили %s.",
		"fr":    "Vous avez joué %s.",
		"de":    "Du hast %s gezogen.",
		"es":    "Moviste %s.",
		"pt":    "Você moveu %s.",
	}
	phraseIMoved = L{
		"en":    "I moved %s.",
		"vi":    "Tôi đi %s.",
		"zh-CN": "我走了 %s。",
		"it":    "Ho mosso %s.",
		"ru":    "Я походил %s.",
		"fr":    "J'ai joué %s.",
		"de":    "Ich habe %s gezogen.",
		"es":    "Moví %s.",
		"pt":    "Eu movi %s.",
	}
	phraseMoveFromTo = L{
		"en":    "%s from %s to %s",
		"vi":    "%s từ %s đến %s",
		"zh-CN": "%s从 %s 到 %s",
		"it":    "%s da %s a %s",
		"ru":    "%s с %s на %s",
		"fr":    "%s de %s vers %s",
		"de":    "%s von %s nach %s",
		"es":    "%s de %s a %s",
		"pt":    "%s de %s para %s",
	}
	phrasePromote = L{
		"en":    ", promote to %s",
		"vi":    ", phong %s",
		"zh-CN": "，升变为%s",
		"it":    ", promozione a %s",
		"ru":    ", превращение в %s",
		"fr":    ", promotion en %s",
		"de":    ", Umwandlung in %s",
		"es":    ", coronación a %s",
		"pt":    ", promoção a %s",
	}
	phraseYouPlayed = L{
		"en":    "You played %s.",
		"vi":    "Bạn đánh %s.",
		"zh-CN": "你下了 %s。",
		"it":    "Hai giocato %s.",
		"ru":    "Вы сыграли %s.",
		"fr":    "Vous avez joué %s.",
		"de":    "Du hast %s gespielt.",
		"es":    "Jugaste %s.",
		"pt":    "Você jogou %s.",
	}
	phraseIPlayed = L{
		"en":    "I played %s.",
		"vi":    "Bot đánh %s.",
		"zh-CN": "我下了 %s。",
		"it":    "Ho giocato %s.",
		"ru":    "Я сыграл %s.",
		"fr":    "J'ai joué %s.",
		"de":    "Ich habe %s gespielt.",
		"es":    "Jugué %s.",
		"pt":    "Eu joguei %s.",
	}
	phraseYouWin = L{
		"en":    "You win!",
		"vi":    "Bạn thắng!",
		"zh-CN": "你赢了！",
		"it":    "Hai vinto!",
		"ru":    "Вы победили!",
		"fr":    "Vous gagnez !",
		"de":    "Du gewinnst!",
		"es":    "¡Ganaste!",
		"pt":    "Você ganhou!",
	}
	phraseIWin = L{
		"en":    "I win.",
		"vi":    "Bot thắng.",
		"zh-CN": "我赢了。",
		"it":    "Ho vinto.",
		"ru":    "Я победил.",
		"fr":    "Je gagne.",
		"de":    "Ich gewinne.",
		"es":    "Gané.",
		"pt":    "Eu ganhei.",
	}
	phraseGameOver = L{
		"en":    "Game over.",
		"vi":    "Hết ván.",
		"zh-CN": "对局结束。",
		"it":    "Partita finita.",
		"ru":    "Игра окончена.",
		"fr":    "Partie terminée.",
		"de":    "Spiel vorbei.",
		"es":    "Fin de la partida.",
		"pt":    "Fim de jogo.",
	}
	phraseDraw = L{
		"en":    "Draw.",
		"vi":    "Hoà.",
		"zh-CN": "和棋。",
		"it":    "Patta.",
		"ru":    "Ничья.",
		"fr":    "Nulle.",
		"de":    "Remis.",
		"es":    "Tablas.",
		"pt":    "Empate.",
	}
	phraseYourTurn = L{
		"en":    "Your turn.",
		"vi":    "Đến lượt bạn.",
		"zh-CN": "该你了。",
		"it":    "Tocca a te.",
		"ru":    "Ваш ход.",
		"fr":    "À vous.",
		"de":    "Du bist am Zug.",
		"es":    "Te toca.",
		"pt":    "Sua vez.",
	}
	phraseCheck = L{
		"en":    "Check!",
		"vi":    "Chiếu!",
		"zh-CN": "将军！",
		"it":    "Scacco!",
		"ru":    "Шах!",
		"fr":    "Échec !",
		"de":    "Schach!",
		"es":    "¡Jaque!",
		"pt":    "Xeque!",
	}
	phraseCheckmate = L{
		"en":    "Checkmate.",
		"vi":    "Chiếu hết.",
		"zh-CN": "将死。",
		"it":    "Scacco matto.",
		"ru":    "Мат.",
		"fr":    "Échec et mat.",
		"de":    "Schachmatt.",
		"es":    "Jaque mate.",
		"pt":    "Xeque-mate.",
	}
	phraseCheckmateXQ = L{
		"en":    "Checkmate.",
		"vi":    "Chiếu bí.",
		"zh-CN": "将死。",
		"it":    "Scacco matto.",
		"ru":    "Мат.",
		"fr":    "Échec et mat.",
		"de":    "Schachmatt.",
		"es":    "Jaque mate.",
		"pt":    "Xeque-mate.",
	}
	phraseStalemate = L{
		"en":    "Stalemate. Draw.",
		"vi":    "Hết nước. Hòa.",
		"zh-CN": "逼和。和棋。",
		"it":    "Stallo. Patta.",
		"ru":    "Пат. Ничья.",
		"fr":    "Pat. Nulle.",
		"de":    "Patt. Remis.",
		"es":    "Ahogado. Tablas.",
		"pt":    "Afogamento. Empate.",
	}
	phraseStalemateXQ = L{
		"en":    "Stalemate. Draw.",
		"vi":    "Hết nước đi. Hòa.",
		"zh-CN": "困毙。和棋。",
		"it":    "Stallo. Patta.",
		"ru":    "Пат. Ничья.",
		"fr":    "Pat. Nulle.",
		"de":    "Patt. Remis.",
		"es":    "Ahogado. Tablas.",
		"pt":    "Afogamento. Empate.",
	}

	chessPieceL = map[string]L{
		"k": {"en": "king", "vi": "vua", "zh-CN": "王", "it": "re", "ru": "король", "fr": "roi", "de": "König", "es": "rey", "pt": "rei"},
		"q": {"en": "queen", "vi": "hậu", "zh-CN": "后", "it": "regina", "ru": "ферзь", "fr": "dame", "de": "Dame", "es": "dama", "pt": "dama"},
		"r": {"en": "rook", "vi": "xe", "zh-CN": "车", "it": "torre", "ru": "ладья", "fr": "tour", "de": "Turm", "es": "torre", "pt": "torre"},
		"b": {"en": "bishop", "vi": "tượng", "zh-CN": "象", "it": "alfiere", "ru": "слон", "fr": "fou", "de": "Läufer", "es": "alfil", "pt": "bispo"},
		"n": {"en": "knight", "vi": "mã", "zh-CN": "马", "it": "cavallo", "ru": "конь", "fr": "cavalier", "de": "Springer", "es": "caballo", "pt": "cavalo"},
		"p": {"en": "pawn", "vi": "tốt", "zh-CN": "兵", "it": "pedone", "ru": "пешка", "fr": "pion", "de": "Bauer", "es": "peón", "pt": "peão"},
	}
	xiangqiPieceL = map[string]L{
		"k": {"en": "general", "vi": "tướng", "zh-CN": "将", "it": "generale", "ru": "генерал", "fr": "général", "de": "General", "es": "general", "pt": "general"},
		"a": {"en": "advisor", "vi": "sĩ", "zh-CN": "士", "it": "consigliere", "ru": "советник", "fr": "garde", "de": "Mandarin", "es": "guardia", "pt": "conselheiro"},
		"e": {"en": "elephant", "vi": "tượng", "zh-CN": "象", "it": "elefante", "ru": "слон", "fr": "éléphant", "de": "Elefant", "es": "elefante", "pt": "elefante"},
		"h": {"en": "horse", "vi": "mã", "zh-CN": "马", "it": "cavallo", "ru": "конь", "fr": "cavalier", "de": "Pferd", "es": "caballo", "pt": "cavalo"},
		"r": {"en": "chariot", "vi": "xe", "zh-CN": "车", "it": "carro", "ru": "колесница", "fr": "char", "de": "Wagen", "es": "carro", "pt": "carro"},
		"c": {"en": "cannon", "vi": "pháo", "zh-CN": "炮", "it": "cannone", "ru": "пушка", "fr": "canon", "de": "Kanone", "es": "cañón", "pt": "canhão"},
		"p": {"en": "soldier", "vi": "tốt", "zh-CN": "兵", "it": "soldato", "ru": "солдат", "fr": "soldat", "de": "Soldat", "es": "soldado", "pt": "soldado"},
	}
	unoColorL = map[string]L{
		"r": {"en": "red", "vi": "đỏ", "zh-CN": "红", "it": "rosso", "ru": "красный", "fr": "rouge", "de": "rot", "es": "rojo", "pt": "vermelho"},
		"g": {"en": "green", "vi": "xanh lá", "zh-CN": "绿", "it": "verde", "ru": "зелёный", "fr": "vert", "de": "grün", "es": "verde", "pt": "verde"},
		"b": {"en": "blue", "vi": "xanh dương", "zh-CN": "蓝", "it": "blu", "ru": "синий", "fr": "bleu", "de": "blau", "es": "azul", "pt": "azul"},
		"y": {"en": "yellow", "vi": "vàng", "zh-CN": "黄", "it": "giallo", "ru": "жёлтый", "fr": "jaune", "de": "gelb", "es": "amarillo", "pt": "amarelo"},
	}
	unoSpecialL = map[string]L{
		"wild":     {"en": "wild", "vi": "đổi màu", "zh-CN": "变色", "it": "jolly", "ru": "цвет", "fr": "joker", "de": "Farbwahl", "es": "comodín", "pt": "coringas"},
		"wild4":    {"en": "draw four", "vi": "cộng bốn", "zh-CN": "加四", "it": "pesca quattro", "ru": "плюс четыре", "fr": "plus quatre", "de": "plus vier", "es": "roba cuatro", "pt": "compra quatro"},
		"skip":     {"en": "skip %s", "vi": "bỏ lượt %s", "zh-CN": "禁手%s", "it": "salta %s", "ru": "пропуск %s", "fr": "passe %s", "de": "Aussetzen %s", "es": "salta %s", "pt": "pula %s"},
		"draw2":    {"en": "draw two %s", "vi": "cộng hai %s", "zh-CN": "加二%s", "it": "pesca due %s", "ru": "плюс два %s", "fr": "plus deux %s", "de": "plus zwei %s", "es": "roba dos %s", "pt": "compra dois %s"},
		"reverse":  {"en": "reverse %s", "vi": "đảo chiều %s", "zh-CN": "反转%s", "it": "inverti %s", "ru": "реверс %s", "fr": "sens inverse %s", "de": "Richtungswechsel %s", "es": "reversa %s", "pt": "inverte %s"},
		"chose":    {"en": ", chose %s", "vi": ", chọn %s", "zh-CN": "，选%s", "it": ", colore %s", "ru": ", цвет %s", "fr": ", couleur %s", "de": ", Farbe %s", "es": ", color %s", "pt": ", cor %s"},
		"num":      {"en": "%s %s", "vi": "%s %s", "zh-CN": "%s%s", "it": "%s %s", "ru": "%s %s", "fr": "%s %s", "de": "%s %s", "es": "%s %s", "pt": "%s %s"},
	}
	bjRankL = map[string]L{
		"A": {"en": "Ace", "vi": "Át", "zh-CN": "A", "it": "Asso", "ru": "Туз", "fr": "As", "de": "Ass", "es": "As", "pt": "Ás"},
		"J": {"en": "Jack", "vi": "J", "zh-CN": "J", "it": "Fante", "ru": "Валет", "fr": "Valet", "de": "Bube", "es": "Jota", "pt": "Valete"},
		"Q": {"en": "Queen", "vi": "Q", "zh-CN": "Q", "it": "Donna", "ru": "Дама", "fr": "Dame", "de": "Dame", "es": "Reina", "pt": "Dama"},
		"K": {"en": "King", "vi": "K", "zh-CN": "K", "it": "Re", "ru": "Король", "fr": "Roi", "de": "König", "es": "Rey", "pt": "Rei"},
	}
	pokerHandL = map[string]L{
		"straight_flush": {"en": "STRAIGHT FLUSH", "vi": "THÙNG PHÁ SẢNH", "zh-CN": "同花顺", "it": "SCALA COLORE", "ru": "СТРИТ-ФЛЕШ", "fr": "QUINTE FLUSH", "de": "STRAIGHT FLUSH", "es": "ESCALERA DE COLOR", "pt": "SEQUÊNCIA DE COR"},
		"four":           {"en": "FOUR OF A KIND", "vi": "TỨ QUÝ", "zh-CN": "四条", "it": "POKER", "ru": "КАРЕ", "fr": "CARRÉ", "de": "VIERLING", "es": "PÓKER", "pt": "QUADRA"},
		"full_house":     {"en": "FULL HOUSE", "vi": "CÙ LŨ", "zh-CN": "葫芦", "it": "FULL", "ru": "ФУЛ-ХАУС", "fr": "FULL", "de": "FULL HOUSE", "es": "FULL", "pt": "FULL HOUSE"},
		"flush":          {"en": "FLUSH", "vi": "THÙNG", "zh-CN": "同花", "it": "COLORE", "ru": "ФЛЕШ", "fr": "COULEUR", "de": "FLUSH", "es": "COLOR", "pt": "FLUSH"},
		"straight":       {"en": "STRAIGHT", "vi": "SẢNH", "zh-CN": "顺子", "it": "SCALA", "ru": "СТРИТ", "fr": "QUINTE", "de": "STRASSE", "es": "ESCALERA", "pt": "SEQUÊNCIA"},
		"three":          {"en": "THREE OF A KIND", "vi": "BỘ BA", "zh-CN": "三条", "it": "TRIS", "ru": "ТРОЙКА", "fr": "BRELAN", "de": "DRILLING", "es": "TRÍO", "pt": "TRINCA"},
		"two_pair":       {"en": "TWO PAIR", "vi": "HAI ĐÔI", "zh-CN": "两对", "it": "DOPPIA COPPIA", "ru": "ДВЕ ПАРЫ", "fr": "DEUX PAIRES", "de": "ZWEI PAARE", "es": "DOBLES PAREJAS", "pt": "DOIS PARES"},
		"pair":           {"en": "ONE PAIR", "vi": "MỘT ĐÔI", "zh-CN": "一对", "it": "COPPIA", "ru": "ПАРА", "de": "EIN PAAR", "fr": "UNE PAIRE", "es": "PAREJA", "pt": "UM PAR"},
		"high":           {"en": "HIGH CARD", "vi": "MẬU THẦU", "zh-CN": "高牌", "it": "CARTA ALTA", "ru": "СТАРШАЯ КАРТА", "fr": "CARTE HAUTE", "de": "HOHE KARTE", "es": "CARTA ALTA", "pt": "CARTA ALTA"},
	}
)

func chessPieceSpoken(p string) string {
	s := wordL(chessPieceL, p, "")
	if s == "" {
		return L{"en": "piece", "vi": "quân", "zh-CN": "棋子", "it": "pezzo", "ru": "фигура", "fr": "pièce", "de": "Figur", "es": "pieza", "pt": "peça"}.get()
	}
	return s
}

func xiangqiPieceSpoken(p string) string {
	s := wordL(xiangqiPieceL, p, "")
	if s == "" {
		return L{"en": "piece", "vi": "quân cờ", "zh-CN": "棋子", "it": "pezzo", "ru": "фигура", "fr": "pièce", "de": "Figur", "es": "pieza", "pt": "peça"}.get()
	}
	return s
}

func speakMoveLang(piece, uci string) string {
	uci = strings.TrimSpace(strings.ToLower(uci))
	name := chessPieceSpoken(piece)
	if len(uci) < 4 {
		return name
	}
	s := phraseMoveFromTo.sprintf(name, speakSq(uci[0:2]), speakSq(uci[2:4]))
	if len(uci) >= 5 {
		s += phrasePromote.sprintf(chessPieceSpoken(string(uci[4])))
	}
	return s
}

func xiangqiSpeakMoveLang(piece, uci string) string {
	uci = strings.TrimSpace(strings.ToLower(uci))
	name := xiangqiPieceSpoken(piece)
	if len(uci) < 4 {
		return name
	}
	return phraseMoveFromTo.sprintf(name, xiangqiSpeakSq(uci[0:2]), xiangqiSpeakSq(uci[2:4]))
}

func joinSpeak(parts []string) string {
	return strings.TrimSpace(strings.Join(parts, " "))
}

func appendMate(parts []string, mate L, winnerHuman bool, winnerKnown bool) []string {
	if winnerKnown && winnerHuman {
		return append(parts, mate.get()+" "+phraseYouWin.get())
	}
	if winnerKnown && !winnerHuman {
		return append(parts, mate.get()+" "+phraseIWin.get())
	}
	return append(parts, strings.TrimSuffix(mate.get(), ".")+"!")
}

func buildChessCommentForLang(youMove, botMove, youPiece, botPiece, status, winner string) string {
	var parts []string
	if youMove != "" {
		parts = append(parts, phraseYouMoved.sprintf(speakMoveLang(youPiece, youMove)))
	}
	if botMove != "" {
		parts = append(parts, phraseIMoved.sprintf(speakMoveLang(botPiece, botMove)))
	}
	switch status {
	case "checkmate":
		parts = appendMate(parts, phraseCheckmate, winner == "white", winner == "white" || winner == "black")
	case "stalemate":
		parts = append(parts, phraseStalemate.get())
	case "check":
		parts = append(parts, phraseCheck.get())
	}
	return joinSpeak(parts)
}

func buildXiangqiCommentForLang(youMove, botMove, youPiece, botPiece, status, winner string) string {
	var parts []string
	if youMove != "" {
		parts = append(parts, phraseYouMoved.sprintf(xiangqiSpeakMoveLang(youPiece, youMove)))
	}
	if botMove != "" {
		parts = append(parts, phraseIMoved.sprintf(xiangqiSpeakMoveLang(botPiece, botMove)))
	}
	switch status {
	case "checkmate":
		parts = appendMate(parts, phraseCheckmateXQ, winner == "red", winner == "red" || winner == "black")
	case "stalemate":
		parts = append(parts, phraseStalemateXQ.get())
	case "check":
		parts = append(parts, phraseCheck.get())
	}
	return joinSpeak(parts)
}

func buildPlaceCommentForLang(youMove, botMove, status, winner string) string {
	var parts []string
	if youMove != "" {
		parts = append(parts, phraseYouPlayed.sprintf(youMove))
	}
	if botMove != "" {
		parts = append(parts, phraseIPlayed.sprintf(botMove))
	}
	switch status {
	case "win", "won", "checkmate":
		if winner == "bot" || winner == "O" {
			parts = append(parts, phraseIWin.get())
		} else if winner != "" {
			parts = append(parts, phraseYouWin.get())
		} else {
			parts = append(parts, phraseGameOver.get())
		}
	case "draw", "stalemate":
		parts = append(parts, phraseDraw.get())
	}
	if len(parts) == 0 {
		return phraseYourTurn.get()
	}
	return strings.Join(parts, " ")
}

func unoColorSpoken(col string) string {
	s := wordL(unoColorL, col, "")
	if s != "" {
		return s
	}
	return strings.ToLower(strings.TrimSpace(col))
}

func unoSpeakCardLang(card, chosenColor string) string {
	c := strings.ToUpper(strings.TrimSpace(card))
	if c == "W" {
		s := unoSpecialL["wild"].get()
		if chosenColor != "" {
			s += unoSpecialL["chose"].sprintf(unoColorSpoken(chosenColor))
		}
		return s
	}
	if c == "W4" {
		s := unoSpecialL["wild4"].get()
		if chosenColor != "" {
			s += unoSpecialL["chose"].sprintf(unoColorSpoken(chosenColor))
		}
		return s
	}
	if len(c) < 2 {
		return c
	}
	col, rank := c[:1], c[1:]
	cv := unoColorSpoken(col)
	switch rank {
	case "S":
		return unoSpecialL["skip"].sprintf(cv)
	case "D":
		return unoSpecialL["draw2"].sprintf(cv)
	case "V":
		return unoSpecialL["reverse"].sprintf(cv)
	default:
		return unoSpecialL["num"].sprintf(rank, cv)
	}
}

func bjRankSpoken(card string) string {
	r := bjCardRank(card)
	if s := wordL(bjRankL, r, ""); s != "" {
		return s
	}
	if s := bjRankL[r].get(); s != "" {
		return s
	}
	return r
}

func pokerHandSpoken(key string) string {
	if s := wordL(pokerHandL, key, ""); s != "" {
		return s
	}
	return key
}
