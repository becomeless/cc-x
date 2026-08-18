package tui

import "testing"

func TestParseKey(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want KeyType
		rune rune
	}{
		{"up", []byte{0x1b, '[', 'A'}, KeyUp, 0},
		{"down", []byte{0x1b, '[', 'B'}, KeyDown, 0},
		{"right", []byte{0x1b, '[', 'C'}, KeyRight, 0},
		{"left", []byte{0x1b, '[', 'D'}, KeyLeft, 0},
		{"home-csi", []byte{0x1b, '[', 'H'}, KeyHome, 0},
		{"end-csi", []byte{0x1b, '[', 'F'}, KeyEnd, 0},
		{"home-ss3", []byte{0x1b, 'O', 'H'}, KeyHome, 0},
		{"end-ss3", []byte{0x1b, 'O', 'F'}, KeyEnd, 0},
		{"home-vt100", []byte{0x1b, '[', '1', '~'}, KeyHome, 0},
		{"end-vt100", []byte{0x1b, '[', '4', '~'}, KeyEnd, 0},
		{"home-rxvt", []byte{0x1b, '[', '7', '~'}, KeyHome, 0},
		{"end-rxvt", []byte{0x1b, '[', '8', '~'}, KeyEnd, 0},
		{"delete", []byte{0x1b, '[', '3', '~'}, KeyDelete, 0},
		{"pgup", []byte{0x1b, '[', '5', '~'}, KeyPgUp, 0},
		{"pgdn", []byte{0x1b, '[', '6', '~'}, KeyPgDn, 0},
		{"shift-up", []byte{0x1b, '[', '1', ';', '2', 'A'}, KeyShiftUp, 0},
		{"shift-down", []byte{0x1b, '[', '1', ';', '2', 'B'}, KeyShiftDown, 0},
		{"lone-esc", []byte{0x1b}, KeyEsc, 0},
		{"enter-cr", []byte{0x0d}, KeyEnter, 0},
		{"enter-lf", []byte{0x0a}, KeyEnter, 0},
		{"ctrl-c", []byte{0x03}, KeyCtrlC, 0},
		{"backspace-del", []byte{0x7f}, KeyBackspace, 0},
		{"backspace-bs", []byte{0x08}, KeyBackspace, 0},
		{"digit", []byte{'5'}, KeyDigit, '5'},
		{"char-q", []byte{'q'}, KeyChar, 'q'},
		{"cjk", []byte{0xe4, 0xb8, 0xad}, KeyChar, '中'}, // “中” 的 UTF-8
		{"empty", []byte{}, KeyUnknown, 0},
	}
	for _, c := range cases {
		got := ParseKey(c.in)
		if got.Type != c.want {
			t.Errorf("%s: type=%v want %v", c.name, got.Type, c.want)
		}
		if c.rune != 0 && got.Rune != c.rune {
			t.Errorf("%s: rune=%q want %q", c.name, got.Rune, c.rune)
		}
	}
}
