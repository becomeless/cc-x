package unix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/becomeless/cc-x/internal/env"
)

func sampleVals() env.ManagedVals {
	return env.ManagedVals{
		"ANTHROPIC_BASE_URL":             "https://api.deepseek.com/anthropic",
		"ANTHROPIC_AUTH_TOKEN":           "sk-1",
		"ANTHROPIC_API_KEY":              "",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":   "deepseek-v4-pro",
		"ANTHROPIC_DEFAULT_SONNET_MODEL": "",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "",
		// effort 不走设为默认（CLAUDE_CODE_EFFORT_LEVEL 已非受管键，改走 --effort 启动参数）
	}
}

func TestBuildBlockOrderAndContent(t *testing.T) {
	got := BuildBlock(sampleVals())
	want := strings.Join([]string{
		"# >>> xx >>>",
		"export ANTHROPIC_BASE_URL='https://api.deepseek.com/anthropic'",
		"export ANTHROPIC_AUTH_TOKEN='sk-1'",
		"export ANTHROPIC_DEFAULT_OPUS_MODEL='deepseek-v4-pro'",
		"# <<< xx <<<",
	}, "\n")
	if got != want {
		t.Fatalf("block 不符：\n got=%q\nwant=%q", got, want)
	}
}

func TestShQuoteEscapesSingleQuote(t *testing.T) {
	got := BuildBlock(env.ManagedVals{"ANTHROPIC_AUTH_TOKEN": "a'b"})
	if !strings.Contains(got, `export ANTHROPIC_AUTH_TOKEN='a'\''b'`) {
		t.Fatalf("单引号转义不符：%q", got)
	}
}

func TestWriteMarkerBlockNewFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".zshrc")
	if err := WriteMarkerBlock(f, sampleVals()); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, f)
	if got != BuildBlock(sampleVals())+"\n" {
		t.Fatalf("新文件内容不符：%q", got)
	}
}

func TestWriteMarkerBlockReplacesExisting(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".zshrc")
	orig := "export PATH=/usr/bin\n# >>> xx >>>\nexport OLD='x'\n# <<< xx <<<\nalias ll='ls -l'\n"
	if err := os.WriteFile(f, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteMarkerBlock(f, sampleVals()); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, f)
	// 块外内容保留，块内被替换，且只替换一次。
	if !strings.HasPrefix(got, "export PATH=/usr/bin\n") || !strings.HasSuffix(got, "alias ll='ls -l'\n") {
		t.Fatalf("块外内容未保留：%q", got)
	}
	if strings.Contains(got, "OLD") {
		t.Fatalf("旧块未被替换：%q", got)
	}
	if !strings.Contains(got, "export ANTHROPIC_BASE_URL='https://api.deepseek.com/anthropic'") {
		t.Fatalf("新块未写入：%q", got)
	}
}

func TestWriteMarkerBlockAppendsNoTrailingNewline(t *testing.T) {
	f := filepath.Join(t.TempDir(), ".profile")
	if err := os.WriteFile(f, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteMarkerBlock(f, sampleVals()); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, f)
	want := "abc\n\n" + BuildBlock(sampleVals()) + "\n"
	if got != want {
		t.Fatalf("追加不符：\n got=%q\nwant=%q", got, want)
	}
}

func TestRcTargetFor(t *testing.T) {
	cases := []struct {
		shell, goos, wantFile string
		wantKind              ShellKind
	}{
		{"/bin/zsh", "darwin", "/home/u/.zshrc", KindZsh},
		{"/usr/bin/bash", "darwin", "/home/u/.bash_profile", KindBash},
		{"/usr/bin/bash", "linux", "/home/u/.bashrc", KindBash},
		{"/usr/bin/fish", "linux", "/home/u/.config/fish/config.fish", KindFish},
		{"/bin/dash", "linux", "/home/u/.profile", KindSh},
		{"", "linux", "/home/u/.profile", KindSh},
	}
	for _, c := range cases {
		got := RcTargetFor(c.shell, c.goos, "/home/u")
		if got.File != c.wantFile || got.Kind != c.wantKind {
			t.Fatalf("RcTargetFor(%q,%q)=%+v，want file=%q kind=%q", c.shell, c.goos, got, c.wantFile, c.wantKind)
		}
	}
}

func readFile(t *testing.T, f string) string {
	t.Helper()
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
