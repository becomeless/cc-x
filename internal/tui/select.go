package tui

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/becomeless/cc-x/internal/display"
	"github.com/becomeless/cc-x/internal/i18n"
)

// SelectOptions 配置一次 ↑↓ 选择菜单。” 项为不可选分隔空行（导航跳过）。
type SelectOptions struct {
	Title        string
	Notice       string // 标题下方黄字横幅（如「有新版本」），常驻显示
	Items        []string
	Hint         string
	Status       string        // 顶部绿色 toast
	Start        int           // 初始选中（记忆选中）
	Colors       map[int]Color // 按索引上色
	MovableCount int           // 顶部可排序区项数
	OnMove       func(from, to int) []string
	OnKey        func(r rune, idx int) int // 单键快捷键：返回新索引并确认，-1=忽略
	NoNumber     bool                      // 关闭行首序号（默认显示，与数字直选一致；编辑表单项多于 9 个时关闭）
}

// SelectMenu 自绘 ↑↓ 选择菜单，返回选中索引；取消（q/Esc/非法）返回 -1；Ctrl+C 恢复终端后以 130 退出。
// 自管 raw 进出（与 ReadText 的 cooked 互斥，同一时刻只跑一套）。非 TTY 回退到打印列表 + 读序号。
func SelectMenu(t *Terminal, opts SelectOptions) int {
	if !t.IsTTY() {
		return fallbackSelect(t, opts)
	}
	items := append([]string(nil), opts.Items...)
	nextSel := func(i, d int) int {
		n := len(items)
		for {
			i = ((i+d)%n + n) % n
			if items[i] != "" {
				return i
			}
		}
	}
	idx := opts.Start
	if idx < 0 || idx >= len(items) || items[idx] == "" {
		c := idx
		if c < 0 {
			c = 0
		}
		if c > len(items)-1 {
			c = len(items) - 1
		}
		idx = nextSel(c, 1)
	}

	if err := t.MakeRaw(); err != nil {
		return fallbackSelect(t, opts)
	}
	t.Write(ClearScreen + HideCursor)

	prevLines := 0
	top := 0 // 滚动窗口首项下标：保证选中项 idx 始终落在窗口内，随上下移动滑窗
	render := func() {
		// 头部（标题/横幅/toast）固定，随整棵菜单一起重绘，但不计入滚动窗口。
		lines := []string{""}
		if opts.Title != "" {
			lines = append(lines, "  "+Paint(opts.Title, ColorCyan), "")
		}
		if opts.Notice != "" {
			for _, line := range splitNonEmptyLines(opts.Notice) {
				lines = append(lines, "  "+Paint(line, ColorYellow))
			}
			lines = append(lines, "")
		}
		if opts.Status != "" {
			for _, line := range splitNonEmptyLines(opts.Status) {
				lines = append(lines, "  "+Paint(line, ColorGreen))
			}
			lines = append(lines, "")
		}
		headerN := len(lines)

		// 项窗口行数 = 终端可视高度 - 头部 - 尾部；预留 1 行，避免光标落在底行时回滚抖动。
		footerBase := 1
		if opts.Hint != "" {
			footerBase = 2
		}
		// 先按「未加位置指示」估视口，判断 item 数是否溢出（会话内 item 数固定，此判断稳定）。
		baseViewH := termHeight(t) - headerN - footerBase - 1
		if baseViewH < 1 {
			baseViewH = 1
		}
		scrollable := len(items) > baseViewH
		footerN := footerBase
		indicator := ""
		if scrollable {
			indicator = strconv.Itoa(idx+1) + " / " + strconv.Itoa(len(items))
			footerN++
		}
		viewH := termHeight(t) - headerN - footerN - 1
		if viewH < 1 {
			viewH = 1
		}
		start, end := scrollWindow(top, idx, viewH, len(items))
		top = start

		// 项窗口：只渲染 [start,end)，选中项随滑窗保证可见。
		//（原逻辑把整棵 items 全画出来，超过屏高后选中项滚出视口、屏上「不动」。）
		for i := start; i < end; i++ {
			it := items[i]
			if it == "" {
				lines = append(lines, "")
				continue
			}
			num := ""
			if !opts.NoNumber {
				num = strconv.Itoa(i+1) + ". " // 行首序号，与数字键直选对应
			}
			if i == idx {
				lines = append(lines, Paint("   ▶ "+num+it, ColorGreen))
			} else {
				col := ColorNone
				if c, ok := opts.Colors[i]; ok {
					col = c
				}
				lines = append(lines, Paint("     "+num+it, col))
			}
		}
		lines = append(lines, "")
		if indicator != "" {
			lines = append(lines, "  "+Paint(indicator, ColorDim))
		}
		if opts.Hint != "" {
			lines = append(lines, "  "+Paint(opts.Hint, ColorDim))
		}

		cols := termWidth(t)
		var b strings.Builder
		if prevLines > 0 {
			b.WriteString(CursorUp(prevLines-1) + CR + ClearDown)
		}
		for i, l := range lines {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(display.Truncate(l, cols-1))
		}
		t.Write(b.String())
		prevLines = len(lines)
	}
	cleanup := func() {
		t.Write(ShowCursor + "\n")
		t.Restore()
	}

	render()
	for {
		k := t.ReadKey()
		switch k.Type {
		case KeyCtrlC:
			cleanup()
			os.Exit(130)
		case KeyUp:
			idx = nextSel(idx, -1)
			render()
		case KeyDown:
			idx = nextSel(idx, 1)
			render()
		case KeyShiftUp, KeyPgUp:
			if opts.OnMove != nil && idx > 0 && idx < opts.MovableCount {
				items = opts.OnMove(idx, idx-1)
				idx--
				render()
			}
		case KeyShiftDown, KeyPgDn:
			if opts.OnMove != nil && idx < opts.MovableCount-1 {
				items = opts.OnMove(idx, idx+1)
				idx++
				render()
			}
		case KeyEnter:
			cleanup()
			return idx
		case KeyEsc:
			cleanup()
			return -1
		case KeyDigit:
			if !opts.NoNumber {
				n := int(k.Rune - '0')
				if n >= 1 && n <= len(items) && items[n-1] != "" {
					cleanup()
					return n - 1
				}
			}
		case KeyChar:
			if k.Rune == 'q' {
				cleanup()
				return -1
			}
			if opts.OnKey != nil {
				if r := opts.OnKey(k.Rune, idx); r >= 0 {
					cleanup()
					return r
				}
			}
		}
	}
}

func termWidth(t *Terminal) int {
	w, _, err := term.GetSize(int(t.Out.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

func termHeight(t *Terminal) int {
	_, h, err := term.GetSize(int(t.Out.Fd()))
	if err != nil || h <= 0 {
		return 24
	}
	return h
}

// scrollWindow 计算滚动视图窗口 [start,end)：高 viewH 行，始终包含选中项 idx。
// 优先保持先前 top 不动，仅当 idx 越界时才滑动；窗口向 [0,n] 内收敛（n 为 item 总数，含空分隔行）。
func scrollWindow(top, idx, viewH, n int) (int, int) {
	if viewH < 1 {
		viewH = 1
	}
	if idx < top {
		top = idx
	} else if idx >= top+viewH {
		top = idx - viewH + 1
	}
	if top < 0 {
		top = 0
	}
	if top > n-viewH {
		top = n - viewH
	}
	if top < 0 {
		top = 0
	}
	end := top + viewH
	if end > n {
		end = n
	}
	return top, end
}

// fallbackSelect 非交互回退：打印列表 + 读一行序号。
func fallbackSelect(t *Terminal, opts SelectOptions) int {
	t.Write("\n")
	if opts.Title != "" {
		t.Write("  " + opts.Title + "\n\n")
	}
	if opts.Notice != "" {
		for _, line := range splitNonEmptyLines(opts.Notice) {
			t.Write("  " + line + "\n")
		}
		t.Write("\n")
	}
	if opts.Status != "" {
		for _, line := range splitNonEmptyLines(opts.Status) {
			t.Write("  " + line + "\n")
		}
		t.Write("\n")
	}
	indexMap := []int{}
	for i, it := range opts.Items {
		if it != "" {
			indexMap = append(indexMap, i)
			t.Write("   " + strconv.Itoa(len(indexMap)) + ". " + it + "\n")
		}
	}
	ans, ok := t.ReadLine("  " + i18n.T("menu.prompt"))
	if !ok {
		return -1
	}
	s := strings.TrimSpace(ans)
	if s == "q" {
		return -1
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(indexMap) {
		return indexMap[n-1]
	}
	return -1
}

func splitNonEmptyLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
