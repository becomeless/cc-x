package tui

import (
	"os"
	"strings"
	"unicode/utf8"

	"github.com/becomeless/cc-x/internal/display"
	"github.com/becomeless/cc-x/internal/i18n"
)

// ReadText 在 cooked 模式读一行（兼容中文输入法）。调用前终端应处于 cooked；ReadLine 内部会确保已 Restore。
// 返回 (line, ok)；EOF/中止且无内容时 ok=false。对应 npm 版 src/ui/text.ts 的 readText。
// 语义（空=不改、"-"=清空等）由调用方处理。
func ReadText(t *Terminal, prompt string) (string, bool) {
	return t.ReadLine(prompt)
}

// ReadValue raw 逐键读 ASCII 字段：回车空=不改、"-"=清空、其它=替换、Esc=取消、Ctrl+C 退出（130）。
// secret=true 时回显 *。对应 npm 版 readValue。
//
// 支持光标编辑：←/→ 移动光标、光标处插入、退格删光标前字符、Delete 删光标处字符
// （回显按字符显示宽度重绘光标后的内容，全角字符 2 列不会留残影）。
// 一次 Read 可能带回多个按键（快速输入/按住退格/粘贴/输入法多字节字后紧跟下一个键），
// 因此整段字节必须**逐个按键**处理——只认第一个字节会把其余事件整段丢掉。见 processChunk。
func ReadValue(t *Terminal, label, current string, secret bool) (changed bool, value string) {
	cur := current
	switch {
	case cur == "":
		cur = i18n.T("empty.paren")
	case secret:
		cur = "********"
	}
	t.Write("\n  " + label + "  [" + i18n.T("edit.current", cur) + "]  " + i18n.T("edit.inputHint") + "\n  > ")

	if !t.IsTTY() {
		return cookedFallback(t, current)
	}
	if err := t.MakeRaw(); err != nil {
		return cookedFallback(t, current)
	}

	var buf []rune
	var pos int
	firstRead := true
	for {
		n, err := t.In.Read(t.buf[:])
		if n == 0 && err != nil {
			t.Restore()
			t.Write("\n")
			return false, current
		}
		chunk := t.buf[:n]
		if firstRead {
			firstRead = false
			chunk = dropStaleFirstLineFeed(chunk)
			if len(chunk) == 0 {
				continue
			}
		}
		out, npos, echo, act := processChunk(buf, pos, chunk, secret)
		buf, pos = out, npos
		t.Write(echo)
		switch act {
		case textExit:
			t.Restore()
			t.Write("\n")
			os.Exit(130)
		case textCommit:
			t.Restore()
			t.Write("\n")
			s := string(buf)
			if s == "" {
				return false, current
			}
			if s == "-" {
				return true, ""
			}
			return true, s
		case textCancel:
			t.Restore()
			t.Write("\n")
			return false, current
		}
	}
}

// textAction 是一次输入块处理产生的终止动作。
type textAction int

const (
	textContinue textAction = iota
	textCommit              // 回车：提交当前缓冲
	textCancel              // Esc：取消
	textExit                // Ctrl+C：退出（130）
)

// processChunk 把一次 read 的整段字节按序解释成按键事件，更新行缓冲与光标并生成回显。
// 一个 chunk 可能含多个按键：快速输入时字符与退格同段到达（"a\x7f" 会把退格丢掉、
// "\x7fa" 会把 a 丢掉）、按住退格（"\x7f\x7f" 只删一个字）、粘贴多字符、输入法多字节字后紧跟的键。
// 逐字节推进：退格/回车/Esc/Ctrl+C 按事件处理，ESC 序列整段解析（←/→/Delete，其余方向键忽略），
// 普通字节按 UTF-8 解码为完整字符后在光标处插入。
func processChunk(buf []rune, pos int, chunk []byte, secret bool) (out []rune, npos int, echo string, act textAction) {
	out = buf
	npos = pos
	var sb strings.Builder
	i := 0
	for i < len(chunk) {
		b := chunk[i]
		switch {
		case b == 0x03: // Ctrl+C
			return out, npos, sb.String(), textExit
		case b == 0x0d || b == 0x0a: // Enter
			return out, npos, sb.String(), textCommit
		case b == 0x7f || b == 0x08: // Backspace：删光标前一个字符
			if npos > 0 {
				removed := out[npos-1]
				out = append(out[:npos-1], out[npos:]...)
				npos--
				w := dispWidthOf(removed, secret)
				tail := out[npos:]
				sb.WriteString(strings.Repeat("\b", w))                              // 退到被删字符起始列
				sb.WriteString(dispText(tail, secret))                              // 重绘光标后内容（整体左移 w 列）
				sb.WriteString(strings.Repeat(" ", w))                              // 清掉行尾多出的 w 列
				sb.WriteString(strings.Repeat("\b", dispTextWidth(tail, secret)+w)) // 回到光标位
			}
			i++
		case b == 0x1b: // Esc：孤立 Esc / Alt+字符 → 取消；ESC[ / ESC O 序列整段解析
			if i+1 < len(chunk) && (chunk[i+1] == '[' || chunk[i+1] == 'O') {
				j := i + 2
				for j < len(chunk) && !(chunk[j] >= 0x40 && chunk[j] <= 0x7e) { // 终止字节
					j++
				}
				if j >= len(chunk) { // 终止字节不在本次 read：序列被拆到两次 read 边界，整段丢弃防越界
					return out, npos, sb.String(), textContinue
				}
				out, npos = applySeqKey(out, npos, chunk[i:j+1], secret, &sb)
				i = j + 1
				continue
			}
			return out, npos, sb.String(), textCancel
		default:
			r, size := utf8.DecodeRune(chunk[i:])
			if r == utf8.RuneError && size == 1 { // 无效字节（输入法多字节字被拆段），丢弃
				i++
				continue
			}
			if r >= 0x20 && r != 0x7f {
				out = insertRune(out, npos, r)
				npos++
				tail := out[npos:]
				sb.WriteString(dispRune(r, secret))                              // 新字符
				sb.WriteString(dispText(tail, secret))                           // 重绘光标后内容（被推右）
				sb.WriteString(strings.Repeat("\b", dispTextWidth(tail, secret))) // 回到新字符之后
			}
			i += size
		}
	}
	return out, npos, sb.String(), textContinue
}

// applySeqKey 处理一个完整的 ESC 序列：←/→ 移动光标、Home/End 跳行首行尾、Delete 删光标处字符，
// 其余（↑↓/PgUp…）忽略。复用 ParseKey 判定按键，按行编辑器语义更新缓冲、光标与回显。
func applySeqKey(out []rune, pos int, seq []byte, secret bool, sb *strings.Builder) ([]rune, int) {
	switch ParseKey(seq).Type {
	case KeyLeft:
		if pos > 0 {
			pos--
			sb.WriteString(strings.Repeat("\b", dispWidthOf(out[pos], secret)))
		}
	case KeyRight:
		if pos < len(out) {
			sb.WriteString(dispRune(out[pos], secret))
			pos++
		}
	case KeyHome:
		if pos > 0 {
			sb.WriteString(strings.Repeat("\b", dispTextWidth(out[:pos], secret))) // 退到行首
			pos = 0
		}
	case KeyEnd:
		if pos < len(out) {
			sb.WriteString(dispText(out[pos:], secret)) // 重打光标后的内容，光标随之前进到行尾
			pos = len(out)
		}
	case KeyDelete:
		if pos < len(out) {
			removed := out[pos]
			out = append(out[:pos], out[pos+1:]...)
			w := dispWidthOf(removed, secret)
			tail := out[pos:]
			sb.WriteString(dispText(tail, secret))                              // 光标后内容整体左移 w 列
			sb.WriteString(strings.Repeat(" ", w))                              // 清掉行尾多出的 w 列
			sb.WriteString(strings.Repeat("\b", dispTextWidth(tail, secret)+w)) // 回到光标位
		}
	}
	return out, pos
}

// insertRune 在 rs 的 pos 处插入 r（pos ∈ [0, len(rs)]），返回新切片。
func insertRune(rs []rune, pos int, r rune) []rune {
	rs = append(rs, 0)
	copy(rs[pos+1:], rs[pos:])
	rs[pos] = r
	return rs
}

// dispWidthOf 返回一个 rune 在屏上的列宽；secret 时每字符回显为 1 列 '*'。
// 组合附加符宽度为 0（不单独占列），与 dispTextWidth 的整串宽度保持一致：
// 光标跨过组合符时不产生位移、插入/重绘按真实列数对齐。已知残留：退格/Delete 直接删一个
// 0 宽组合符时，屏上的附标会滞留到下一次编辑重绘该列（缓冲值始终正确，且比钳成 1 列时
// 连基础字符一起擦掉、光标倒退回提示符的行为安全得多）。
func dispWidthOf(r rune, secret bool) int {
	if secret {
		return 1
	}
	return display.Width(string(r))
}

// dispRune 返回 rune 的回显文本。
func dispRune(r rune, secret bool) string {
	if secret {
		return "*"
	}
	return string(r)
}

// dispText 返回一串 rune 的回显文本。
func dispText(rs []rune, secret bool) string {
	if secret {
		return strings.Repeat("*", len(rs))
	}
	return string(rs)
}

// dispTextWidth 返回一串 rune 回显后的列宽。
func dispTextWidth(rs []rune, secret bool) int {
	if secret {
		return len(rs)
	}
	return display.Width(string(rs))
}

// dropStaleFirstLineFeed drains a lone LF that Windows terminals may leave
// after SelectMenu consumes the CR half of an Enter key reported as CRLF.
func dropStaleFirstLineFeed(chunk []byte) []byte {
	if len(chunk) > 0 && chunk[0] == '\n' {
		return chunk[1:]
	}
	return chunk
}

// cookedFallback：非 TTY / raw 失败时，cooked 读一行（语义同 ReadValue 的回车/-/替换）。
func cookedFallback(t *Terminal, current string) (bool, string) {
	line, ok := t.ReadLine("")
	if !ok || line == "" {
		return false, current
	}
	if line == "-" {
		return true, ""
	}
	return true, line
}
