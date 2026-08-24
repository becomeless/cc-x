package tui

import "testing"

// TestScrollWindow：滚动窗口始终包含选中项，且向 [0,n] 内收敛不越界。
func TestScrollWindow(t *testing.T) {
	cases := []struct {
		name               string
		top, idx, viewH, n int
		wantStart, wantEnd int
	}{
		{"全部能装下：不滚动", 0, 0, 5, 3, 0, 3},
		{"选中项在顶部窗口内", 0, 3, 5, 20, 0, 5},
		{"中部下移滑窗", 0, 12, 5, 20, 8, 13},
		{"底部钳制", 0, 19, 5, 20, 15, 20},
		{"上移超过顶界", 10, 2, 5, 20, 2, 7},
		{"viewH 最小为 1", 0, 5, 1, 20, 5, 6},
		{"n 小于 viewH：全量", 0, 0, 10, 4, 0, 4},
	}
	for _, c := range cases {
		start, end := scrollWindow(c.top, c.idx, c.viewH, c.n)
		if start != c.wantStart || end != c.wantEnd {
			t.Errorf("%s: scrollWindow(%d,%d,%d,%d) = (%d,%d), want (%d,%d)",
				c.name, c.top, c.idx, c.viewH, c.n, start, end, c.wantStart, c.wantEnd)
		}
		// 不变量：选中项必须可见。
		if c.n > 0 && c.idx >= 0 && c.idx < c.n && (c.idx < start || c.idx >= end) {
			t.Errorf("%s: idx=%d 不可见 in [%d,%d)", c.name, c.idx, start, end)
		}
	}
}

// 穷举不变量：任意合法输入（含越界的 top）下选中项必可见，窗口不越界（对齐 TS 版用例）。
func TestScrollWindowInvariant(t *testing.T) {
	for _, n := range []int{4, 20, 100} {
		for _, viewH := range []int{1, 5, 25} {
			for idx := 0; idx < n; idx++ {
				start, end := scrollWindow(50, idx, viewH, n)
				if idx < start || idx >= end {
					t.Errorf("n=%d viewH=%d idx=%d → [%d,%d) 未包含选中项", n, viewH, idx, start, end)
				}
				if start < 0 || end > n || start > end {
					t.Errorf("n=%d viewH=%d idx=%d → 窗口 [%d,%d) 越界", n, viewH, idx, start, end)
				}
			}
		}
	}
}
