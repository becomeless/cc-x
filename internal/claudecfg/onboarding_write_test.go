package claudecfg

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeCase 表驱动：输入文件内容 -> 期望输出（整文件逐字节比对）。
type writeCase struct {
	name string
	in   string
	want string // 期望结果；不设则期望报错且字节不变
}

func runWriteCases(t *testing.T, cases []writeCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, ".claude.json")
			if err := os.WriteFile(p, []byte(tc.in), 0o644); err != nil {
				t.Fatal(err)
			}
			err := MarkOnboardingDoneIn(p)
			got, rerr := os.ReadFile(p)
			if rerr != nil {
				t.Fatalf("read back: %v", rerr)
			}
			if tc.want == "" {
				if err == nil {
					t.Fatalf("期望报错，却成功；结果: %q", got)
				}
				if string(got) != tc.in {
					t.Fatalf("报错时文件被改动:\n  原: %q\n  现: %q", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("写入失败: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("结果不符:\n  期望: %q\n  实际: %q", tc.want, got)
			}
		})
	}
}

func TestMarkOnboardingDone(t *testing.T) {
	runWriteCases(t, []writeCase{
		{
			name: "替换 false",
			in:   `{"a":1,"hasCompletedOnboarding":false,"b":2}`,
			want: `{"a":1,"hasCompletedOnboarding":true,"b":2}`,
		},
		{
			name: "键与值之间空白保留",
			in:   "{ \"hasCompletedOnboarding\" : false , \"x\" : 1 }",
			want: "{ \"hasCompletedOnboarding\" : true , \"x\" : 1 }",
		},
		{
			name: "值替换全谱：null",
			in:   `{"hasCompletedOnboarding":null,"a":1}`,
			want: `{"hasCompletedOnboarding":true,"a":1}`,
		},
		{
			name: "值替换全谱：字符串",
			in:   `{"hasCompletedOnboarding":"pending","a":1}`,
			want: `{"hasCompletedOnboarding":true,"a":1}`,
		},
		{
			name: "值替换全谱：数字",
			in:   `{"hasCompletedOnboarding":123,"a":1}`,
			want: `{"hasCompletedOnboarding":true,"a":1}`,
		},
		{
			name: "值替换全谱：负数指数",
			in:   `{"hasCompletedOnboarding":-1.5e3,"a":1}`,
			want: `{"hasCompletedOnboarding":true,"a":1}`,
		},
		{
			name: "值替换全谱：对象",
			in:   `{"hasCompletedOnboarding":{"x":1},"a":1}`,
			want: `{"hasCompletedOnboarding":true,"a":1}`,
		},
		{
			name: "值替换全谱：数组",
			in:   `{"hasCompletedOnboarding":[1,2],"a":1}`,
			want: `{"hasCompletedOnboarding":true,"a":1}`,
		},
		{
			name: "空对象插入不带尾逗号",
			in:   `{}`,
			want: `{"hasCompletedOnboarding": true}`,
		},
		{
			// 插入点紧跟顶层 {，原空白保留在插入文本之后——最小改动原则
			name: "空对象带空白",
			in:   "{  }",
			want: `{"hasCompletedOnboarding": true  }`,
		},
		{
			name: "顶层无键插入到首个键前",
			in:   `{"a":1}`,
			want: `{"hasCompletedOnboarding": true,"a":1}`,
		},
		{
			name: "插入不破坏原格式",
			in:   "{ \"a\" : 1 }",
			want: `{"hasCompletedOnboarding": true, "a" : 1 }`,
		},
		{
			name: "字符串值内同名文本不误命中",
			in:   `{"note":"hasCompletedOnboarding","a":1}`,
			want: `{"hasCompletedOnboarding": true,"note":"hasCompletedOnboarding","a":1}`,
		},
		{
			name: "嵌套对象同名键不动",
			in:   `{"mcp":{"hasCompletedOnboarding":false},"a":1}`,
			want: `{"hasCompletedOnboarding": true,"mcp":{"hasCompletedOnboarding":false},"a":1}`,
		},
		{
			name: "重复顶层键取最后一个",
			in:   `{"hasCompletedOnboarding":false,"hasCompletedOnboarding":false,"a":1}`,
			want: `{"hasCompletedOnboarding":false,"hasCompletedOnboarding":true,"a":1}`,
		},
		{
			name: "中文与 emoji 环绕",
			in:   "{\"项目\":\"计划 😀\",\"hasCompletedOnboarding\":false,\"备注\":\"含 UTF-8 中文\"}",
			want: "{\"项目\":\"计划 😀\",\"hasCompletedOnboarding\":true,\"备注\":\"含 UTF-8 中文\"}",
		},
		{
			name: "键在文件尾部（大前置）",
			in:   "{" + strings.Repeat(`"filler":"x",`, 50000) + `"hasCompletedOnboarding":false}`,
			want: "{" + strings.Repeat(`"filler":"x",`, 50000) + `"hasCompletedOnboarding":true}`,
		},
		{
			name: "已是 true 原样返回",
			in:   `{"hasCompletedOnboarding":true,"a":1}`,
			want: `{"hasCompletedOnboarding":true,"a":1}`,
		},
	})
}

func TestMarkOnboardingDoneErrors(t *testing.T) {
	runWriteCases(t, []writeCase{
		{name: "残缺 JSON", in: `{`},
		{name: "空文件", in: ``},
		{name: "顶层数组", in: `[1,2,3]`},
		{name: "顶层字符串", in: `"hello"`},
		{name: "顶层数字", in: `42`},
		{name: "BOM 前缀", in: "\xef\xbb\xbf" + `{"hasCompletedOnboarding":false}`},
	})
}

func TestMarkOnboardingDoneBOMInsideAfterValidPrefix(t *testing.T) {
	// BOM（U+FEFF）在字符串值内部：json.Valid 通过，splice 必须按字节处理、不污染多字节内容
	bom := "\xef\xbb\xbf"
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	in := "{\"x\":\"" + bom + "\",\"hasCompletedOnboarding\":false}"
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MarkOnboardingDoneIn(p); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	got, _ := os.ReadFile(p)
	want := "{\"x\":\"" + bom + "\",\"hasCompletedOnboarding\":true}"
	if string(got) != want {
		t.Fatalf("结果不符:\n  期望: %q\n  实际: %q", want, got)
	}
}

func TestMarkOnboardingDoneCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	if err := MarkOnboardingDoneIn(p); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "{\"hasCompletedOnboarding\": true}\n" {
		t.Fatalf("新文件内容不符: %q", got)
	}
	if runtime.GOOS != "windows" {
		st, _ := os.Stat(p)
		if mode := st.Mode().Perm(); mode != 0o600 {
			t.Fatalf("新建文件权限应为 0600，实际 %o", mode)
		}
	}
}

// TestMarkOnboardingDonePreservesMode：已存在文件保留原权限位（CreateTemp 0600 会被 Chmod 回原权限）。
func TestMarkOnboardingDonePreservesMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("权限位语义仅 POSIX")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(p, []byte(`{"hasCompletedOnboarding":false}`), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := MarkOnboardingDoneIn(p); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(p)
	if mode := st.Mode().Perm(); mode != 0o640 {
		t.Fatalf("已存在文件应保留原权限 0640，实际 %o", mode)
	}
}

// TestMarkOnboardingDoneReadonlyWindows：Windows 上只读目标使 rename 失败 →
// 报错 + temp 清理（清理前先 Chmod 恢复可写，否则只读 temp 无法删除）+ 原文件不变。
// POSIX 上 rename 取决于父目录权限，可覆盖只读目标（见 atomicWrite 注释），该行为不测。
func TestMarkOnboardingDoneReadonlyWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("只读目标阻挡 rename 仅 Windows 语义")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(p, []byte(`{"hasCompletedOnboarding":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := MarkOnboardingDoneIn(p); err == nil {
		t.Fatal("只读目标应报错")
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".claude.json-") {
			t.Fatalf("temp 未清理: %s", e.Name())
		}
	}
	got, _ := os.ReadFile(p)
	if string(got) != `{"hasCompletedOnboarding":false}` {
		t.Fatalf("原文件被改动: %q", got)
	}
}

// TestMarkOnboardingDoneIdempotent 第二次调用不写盘：字节与 mtime 都不变。
func TestMarkOnboardingDoneIdempotent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".claude.json")
	if err := MarkOnboardingDoneIn(p); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	st1, _ := os.Stat(p)
	if err := MarkOnboardingDoneIn(p); err != nil {
		t.Fatalf("第二次调用失败: %v", err)
	}
	after, _ := os.ReadFile(p)
	st2, _ := os.Stat(p)
	if string(before) != string(after) {
		t.Fatalf("第二次调用改了文件: %q -> %q", before, after)
	}
	if !st1.ModTime().Equal(st2.ModTime()) {
		t.Fatalf("第二次调用改了 mtime: %v -> %v", st1.ModTime(), st2.ModTime())
	}
}

func TestMarkOnboardingDonePathIsDir(t *testing.T) {
	dir := t.TempDir()
	if err := MarkOnboardingDoneIn(dir); err == nil {
		t.Fatal("路径是目录应报错")
	}
}

func TestMarkOnboardingDoneMissingDir(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nope", ".claude.json")
	if err := MarkOnboardingDoneIn(p); err == nil {
		t.Fatal("不存在的目录应报错")
	}
}

// TestMarkOnboardingDoneSymlink Unix 上验证符号链接本体不被替换（temp+rename 直写会毁链接）。
func TestMarkOnboardingDoneSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink 语义仅 Unix")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	link := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(target, []byte(`{"hasCompletedOnboarding":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := MarkOnboardingDoneIn(link); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != `{"hasCompletedOnboarding":true}` {
		t.Fatalf("target 内容不符: %q", got)
	}
	// 链接本体必须还是链接
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("符号链接本体被替换成了普通文件")
	}
}
