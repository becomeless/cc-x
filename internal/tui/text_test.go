package tui

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDropStaleFirstLineFeed(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want []byte
	}{
		{"empty", []byte{}, []byte{}},
		{"lone-lf", []byte{'\n'}, []byte{}},
		{"lf-before-typed-value", []byte("\nmimo-v2.5[1m]"), []byte("mimo-v2.5[1m]")},
		{"cr-enter-untouched", []byte{'\r'}, []byte{'\r'}},
		{"esc-untouched", []byte{0x1b}, []byte{0x1b}},
		{"ordinary-value-untouched", []byte("mimo-v2.5[1m]"), []byte("mimo-v2.5[1m]")},
	}

	for _, c := range cases {
		got := dropStaleFirstLineFeed(c.in)
		if !bytes.Equal(got, c.want) {
			t.Fatalf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

// TestProcessChunk 回归：一次 read 的 chunk 可能含多个按键（快速输入/按住退格/粘贴/输入法多字节字后紧跟的键）。
// 早先只按第一个字节解析整段，导致 "\x7f\x7f" 只删一个字、"a\x7f" 丢退格、"\x7fa" 丢字符
// ——即用户报告的「保存后少一两个字母 / 退格删不干净」。同时覆盖光标编辑（←/→/Delete/中插）。
func TestProcessChunk(t *testing.T) {
	escD := []byte{0x1b, '[', 'D'}
	escC := []byte{0x1b, '[', 'C'}
	escDel := []byte{0x1b, '[', '3', '~'}
	escHome := []byte{0x1b, '[', 'H'}
	escEnd := []byte{0x1b, '[', 'F'}
	cases := []struct {
		name     string
		start    []rune
		pos      int
		chunk    []byte
		secret   bool
		want     []rune
		wantPos  int
		wantEcho string
		act      textAction
	}{
		{"paste-multi-char", nil, 0, []byte("abc"), false, []rune("abc"), 3, "abc", textContinue},
		{"char-then-backspace-keeps-backspace", []rune("deepseek"), 8, []byte("x\x7f"), false, []rune("deepseek"), 8, "x\b \b", textContinue},
		{"backspace-then-char-keeps-char", []rune("deepseek"), 8, []byte("\x7fa"), false, []rune("deepseea"), 8, "\b \ba", textContinue},
		{"double-backspace-deletes-two", []rune("ab"), 2, []byte("\x7f\x7f"), false, []rune{}, 0, "\b \b\b \b", textContinue},
		{"backspace-then-enter-commits", []rune("ab"), 2, []byte("\x7f\r"), false, []rune("a"), 1, "\b \b", textCommit},
		{"enter-commits", []rune("ab"), 2, []byte{'\r'}, false, []rune("ab"), 2, "", textCommit},
		{"esc-cancels", []rune("ab"), 2, []byte{0x1b}, false, []rune("ab"), 2, "", textCancel},
		{"alt-char-cancels", []rune("ab"), 2, []byte{0x1b, 'x'}, false, []rune("ab"), 2, "", textCancel},
		{"ctrl-c-exits", []rune("ab"), 2, []byte{0x03}, false, []rune("ab"), 2, "", textExit},
		{"ctrl-chars-filtered", nil, 0, []byte{'a', 0x09, 'b'}, false, []rune("ab"), 2, "ab", textContinue},
		{"invalid-utf8-dropped", nil, 0, []byte{0xe3, 0x80}, false, nil, 0, "", textContinue},
		{"empty-chunk-noop", []rune("ab"), 2, []byte{}, false, []rune("ab"), 2, "", textContinue},
		// —— 光标编辑 ——
		{"insert-middle", []rune("AB"), 1, []byte("C"), false, []rune("ACB"), 2, "CB\b", textContinue},
		{"insert-fullwidth-middle", []rune("AB"), 1, []byte("【"), false, []rune("A【B"), 2, "【B\b", textContinue},
		{"insert-at-end-append", []rune("AB"), 2, []byte("C"), false, []rune("ABC"), 3, "C", textContinue},
		{"backspace-middle", []rune("ACB"), 2, []byte{'\x7f'}, false, []rune("AB"), 1, "\bB \b\b", textContinue},
		{"backspace-fullwidth-tail", []rune("AC【"), 2, []byte{'\x7f'}, false, []rune("A【"), 1, "\b【 \b\b\b", textContinue},
		{"backspace-at-pos0-noop", []rune("AB"), 0, []byte{'\x7f'}, false, []rune("AB"), 0, "", textContinue},
		{"delete-middle", []rune("ACB"), 1, escDel, false, []rune("AB"), 1, "B \b\b", textContinue},
		{"delete-at-end-noop", []rune("AB"), 2, escDel, false, []rune("AB"), 2, "", textContinue},
		{"left-moves-cursor", []rune("A【"), 2, escD, false, []rune("A【"), 1, "\b\b", textContinue},
		{"left-at-pos0-noop", []rune("A"), 0, escD, false, []rune("A"), 0, "", textContinue},
		{"right-moves-cursor", []rune("A【"), 1, escC, false, []rune("A【"), 2, "【", textContinue},
		{"right-at-end-noop", []rune("A"), 1, escC, false, []rune("A"), 1, "", textContinue},
		{"left-then-insert", []rune("AB"), 2, append(append([]byte{}, escD...), 'C'), false, []rune("ACB"), 2, "\bCB\b", textContinue},
		{"home-jumps-to-start", []rune("A【"), 2, escHome, false, []rune("A【"), 0, "\b\b\b", textContinue},
		{"home-at-start-noop", []rune("AB"), 0, escHome, false, []rune("AB"), 0, "", textContinue},
		{"end-jumps-to-end", []rune("A【"), 0, escEnd, false, []rune("A【"), 2, "A【", textContinue},
		{"end-at-end-noop", []rune("AB"), 2, escEnd, false, []rune("AB"), 2, "", textContinue},
		{"home-rxvt-jumps-to-start", []rune("A【"), 2, []byte{0x1b, '[', '7', '~'}, false, []rune("A【"), 0, "\b\b\b", textContinue},
		{"end-rxvt-jumps-to-end", []rune("A【"), 0, []byte{0x1b, '[', '8', '~'}, false, []rune("A【"), 2, "A【", textContinue},
		{"partial-rxvt-home-dropped", []rune("ab"), 2, []byte{0x1b, '[', '7'}, false, []rune("ab"), 2, "", textContinue},
		{"home-then-type", []rune("AB"), 2, append(append([]byte{}, escHome...), 'C'), false, []rune("CAB"), 1, "\b\bCAB\b\b", textContinue},
		{"secret-home-end", []rune("A【"), 2, append(append([]byte{}, escHome...), escEnd...), true, []rune("A【"), 2, "\b\b**", textContinue},
		{"up-arrow-ignored", []rune("ab"), 2, []byte{0x1b, '[', 'A'}, false, []rune("ab"), 2, "", textContinue},
		// —— ESC 序列被拆到两次 read 的边界：整段丢弃，不 panic、不误插字符 ——
		{"partial-esc-bracket-dropped", []rune("ab"), 2, []byte{0x1b, '['}, false, []rune("ab"), 2, "", textContinue},
		{"partial-delete-seq-dropped", []rune("ab"), 2, []byte{0x1b, '[', '3'}, false, []rune("ab"), 2, "", textContinue},
		{"partial-ss3-dropped", []rune("ab"), 2, []byte{0x1b, 'O'}, false, []rune("ab"), 2, "", textContinue},
		{"seq-then-partial-in-one-chunk", []rune("AB"), 2, []byte{0x1b, '[', 'D', 0x1b, '[', '3'}, false, []rune("AB"), 1, "\b", textContinue},
		// —— 组合附加符（0 列）：单字符宽度与整串宽度一致，不做钳制 ——
		{"left-across-combining-mark", []rune("e\u0301"), 2, escD, false, []rune("e\u0301"), 1, "", textContinue},
		{"backspace-combining-mark-value-only", []rune("e\u0301"), 2, []byte{'\x7f'}, false, []rune("e"), 1, "", textContinue},
		{"insert-between-base-and-mark", []rune("e\u0301"), 1, []byte("x"), false, []rune("ex\u0301"), 2, "x\u0301", textContinue},
		{"secret-combining-mark-1col", []rune("e\u0301"), 2, []byte{'\x7f'}, true, []rune("e"), 1, "\b \b", textContinue},
		// —— secret 回显 ——
		{"secret-insert-middle", []rune("AB"), 1, []byte("C"), true, []rune("ACB"), 2, "**\b", textContinue},
		{"secret-left-right", []rune("A【"), 2, append(append([]byte{}, escD...), escC...), true, []rune("A【"), 2, "\b*", textContinue},
		{"secret-backspace-middle", []rune("ACB"), 2, []byte{'\x7f'}, true, []rune("AB"), 1, "\b* \b\b", textContinue},
	}

	for _, c := range cases {
		got, gotPos, echo, act := processChunk(c.start, c.pos, c.chunk, c.secret)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%s: buf got %q want %q", c.name, got, c.want)
		}
		if gotPos != c.wantPos {
			t.Fatalf("%s: pos got %d want %d", c.name, gotPos, c.wantPos)
		}
		if echo != c.wantEcho {
			t.Fatalf("%s: echo got %q want %q", c.name, echo, c.wantEcho)
		}
		if act != c.act {
			t.Fatalf("%s: act got %v want %v", c.name, act, c.act)
		}
	}
}

func TestInsertRune(t *testing.T) {
	cases := []struct {
		name string
		in   []rune
		pos  int
		r    rune
		want []rune
	}{
		{"middle", []rune("AB"), 1, 'C', []rune("ACB")},
		{"front", []rune("AB"), 0, 'C', []rune("CAB")},
		{"end", []rune("AB"), 2, 'C', []rune("ABC")},
		{"empty", nil, 0, 'C', []rune("C")},
	}
	for _, c := range cases {
		got := insertRune(c.in, c.pos, c.r)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}
