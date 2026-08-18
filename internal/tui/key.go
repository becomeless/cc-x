package tui

import (
	"strings"
	"unicode/utf8"
)

// KeyType 是 ccx 用到的按键种类（只覆盖菜单/输入所需，不为不存在的复杂编辑扩展）。
type KeyType int

const (
	KeyUnknown KeyType = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEsc
	KeyBackspace
	KeyDelete
	KeyHome
	KeyEnd
	KeyCtrlC
	KeyPgUp
	KeyPgDn
	KeyShiftUp
	KeyShiftDown
	KeyDigit // Rune 为 '0'..'9'
	KeyChar  // Rune 为可打印字符
)

// Key 是一次按键事件。
type Key struct {
	Type KeyType
	Rune rune
}

// ParseKey 把一次终端读到的字节块解析成一个按键。
//
// 关键前提：raw 模式下终端会把一个转义序列（如方向键 ESC[A）在单次 read 里原子投递，
// 所以「单次 Read 的整段字节」即一个按键事件。这样 lone-ESC（len==1）与 ESC 序列天然可区分，
// 不需要超时判定（Windows 控制台无法对 stdin 设读超时，这点尤其重要）。
func ParseKey(b []byte) Key {
	if len(b) == 0 {
		return Key{Type: KeyUnknown}
	}
	switch b[0] {
	case 0x03:
		return Key{Type: KeyCtrlC}
	case 0x0d, 0x0a:
		return Key{Type: KeyEnter}
	case 0x7f, 0x08:
		return Key{Type: KeyBackspace}
	case 0x1b:
		if len(b) == 1 {
			return Key{Type: KeyEsc}
		}
		return parseEsc(b)
	}
	r, _ := utf8.DecodeRune(b)
	if r >= '0' && r <= '9' {
		return Key{Type: KeyDigit, Rune: r}
	}
	return Key{Type: KeyChar, Rune: r}
}

// parseEsc 解析 CSI（ESC [ …）/ SS3（ESC O …）序列。只识别菜单需要的键。
func parseEsc(b []byte) Key {
	if len(b) < 3 || (b[1] != '[' && b[1] != 'O') {
		return Key{Type: KeyEsc}
	}
	final := b[len(b)-1]
	body := string(b[2 : len(b)-1]) // '[' 与 final 之间的参数，如 "1;2"
	shift := strings.Contains(body, ";2")
	switch final {
	case 'A':
		if shift {
			return Key{Type: KeyShiftUp}
		}
		return Key{Type: KeyUp}
	case 'B':
		if shift {
			return Key{Type: KeyShiftDown}
		}
		return Key{Type: KeyDown}
	case 'C':
		return Key{Type: KeyRight}
	case 'D':
		return Key{Type: KeyLeft}
	case 'H':
		return Key{Type: KeyHome} // ESC[H / ESC OH
	case 'F':
		return Key{Type: KeyEnd} // ESC[F / ESC OF
	case '~':
		switch body {
		case "1":
			return Key{Type: KeyHome} // VT100 风格 ESC[1~
		case "3":
			return Key{Type: KeyDelete}
		case "4":
			return Key{Type: KeyEnd} // VT100 风格 ESC[4~
		case "5":
			return Key{Type: KeyPgUp}
		case "6":
			return Key{Type: KeyPgDn}
		case "7":
			return Key{Type: KeyHome} // rxvt 风格 ESC[7~
		case "8":
			return Key{Type: KeyEnd} // rxvt 风格 ESC[8~
		}
	}
	return Key{Type: KeyUnknown}
}
