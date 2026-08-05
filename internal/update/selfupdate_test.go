package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAssetName(t *testing.T) {
	cases := []struct {
		goos, goarch string
		want         string
		ok           bool
	}{
		{"windows", "amd64", "ccx_0.4.25_windows_amd64.zip", true},
		{"darwin", "amd64", "ccx_0.4.25_darwin_amd64.tar.gz", true},
		{"darwin", "arm64", "ccx_0.4.25_darwin_arm64.tar.gz", true},
		{"linux", "amd64", "ccx_0.4.25_linux_amd64.tar.gz", true},
		{"linux", "arm64", "ccx_0.4.25_linux_arm64.tar.gz", true},
		{"windows", "arm64", "", false}, // 发布矩阵外
		{"freebsd", "amd64", "", false},
	}
	for _, c := range cases {
		got, err := assetName("0.4.25", c.goos, c.goarch)
		if (err == nil) != c.ok {
			t.Errorf("assetName(%q,%q) err=%v want ok=%v", c.goos, c.goarch, err, c.ok)
			continue
		}
		if err == nil && got != c.want {
			t.Errorf("assetName(%q,%q)=%q want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestParseChecksums(t *testing.T) {
	data := []byte("abc123  ccx_0.4.25_windows_amd64.zip\n" +
		"  def456   ccx_0.4.25_linux_amd64.tar.gz  \n" + // 首尾空白
		"garbage-line-without-two-fields\n" +
		"")
	m := parseChecksums(data)
	if m["ccx_0.4.25_windows_amd64.zip"] != "abc123" {
		t.Errorf("zip 条目解析失败: %v", m)
	}
	if m["ccx_0.4.25_linux_amd64.tar.gz"] != "def456" {
		t.Errorf("tar.gz 条目解析失败: %v", m)
	}
	if len(m) != 2 {
		t.Errorf("垃圾行不应解析出条目: %v", m)
	}
}

func TestShouldSelfUpdate(t *testing.T) {
	cases := []struct {
		current string
		want    bool
	}{
		{"0.4.24", false},
		{"0.4.25", true},
		{"0.4.26", true},
		{"v0.4.25", true},
		{"0.4.25-rc1", true}, // rc 后缀截断后视为 0.4.25
		{"dev", false},       // 解析失败不误报
		{"", false},
	}
	for _, c := range cases {
		if got := shouldUseSelfUpdate(c.current); got != c.want {
			t.Errorf("shouldUseSelfUpdate(%q)=%v want %v", c.current, got, c.want)
		}
	}
}

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "x")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	// sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	if err := verifySHA256(file, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"); err != nil {
		t.Errorf("正确校验和不应失败: %v", err)
	}
	if err := verifySHA256(file, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Error("错误校验和应失败")
	}
}

// TestExtractZipSlipGuard：含 ../evil 条目的 zip 必须被拒绝。
func TestExtractZipSlipGuard(t *testing.T) {
	dir := t.TempDir()
	zpath := filepath.Join(dir, "evil.zip")
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	w, err := zw.Create("../evil")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	zw.Close()
	if err := os.WriteFile(zpath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := extractArchive(zpath, filepath.Join(dir, "out")); err == nil {
		t.Fatal("含越界条目的 zip 应被拒绝")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil")); !os.IsNotExist(err) {
		t.Fatal("越界条目不应落盘")
	}
}

// TestExtractZipRoundtrip：标准发布布局（顶层目录 + xx.exe + presets.json）解压后
// findBinaryIn 能找到二进制。
func TestExtractZipRoundtrip(t *testing.T) {
	dir := t.TempDir()
	zpath := filepath.Join(dir, "ccx_0.4.25_windows_amd64.zip")
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for _, name := range []string{
		"ccx_0.4.25_windows_amd64/xx.exe",
		"ccx_0.4.25_windows_amd64/presets.json",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("payload")); err != nil {
			t.Fatal(err)
		}
	}
	zw.Close()
	if err := os.WriteFile(zpath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := extractArchive(zpath, out); err != nil {
		t.Fatalf("解压失败: %v", err)
	}
	bin, err := findBinaryIn(out)
	if err != nil {
		t.Fatalf("findBinaryIn: %v", err)
	}
	if !strings.HasSuffix(bin, "xx.exe") {
		t.Errorf("应找到 xx.exe，got %s", bin)
	}
}

// TestExtractTarGzRoundtrip：tar.gz 布局 + 普通文件 mode 保留。
func TestExtractTarGzRoundtrip(t *testing.T) {
	dir := t.TempDir()
	apath := filepath.Join(dir, "ccx_0.4.25_linux_amd64.tar.gz")
	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)
	body := []byte("#!/bin/fake\n")
	if err := tw.WriteHeader(&tar.Header{Name: "ccx_0.4.25_linux_amd64/xx", Mode: 0o755, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	if err := os.WriteFile(apath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := extractArchive(apath, out); err != nil {
		t.Fatalf("解压失败: %v", err)
	}
	bin, err := findBinaryIn(out)
	if err != nil {
		t.Fatalf("findBinaryIn: %v", err)
	}
	fi, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	// Windows 无 Unix 权限位（统一显示 -rw-rw-rw-），执行位断言仅对类 Unix 生效。
	if runtime.GOOS != "windows" && fi.Mode()&0o111 == 0 {
		t.Errorf("二进制应保留执行位: %v", fi.Mode())
	}
}

// TestExtractTarAbsPathGuard：绝对路径条目必须被拒绝。
func TestExtractTarAbsPathGuard(t *testing.T) {
	dir := t.TempDir()
	apath := filepath.Join(dir, "evil.tar.gz")
	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)
	body := []byte("x")
	if err := tw.WriteHeader(&tar.Header{Name: "/tmp/evil", Mode: 0o644, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	if err := os.WriteFile(apath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := extractArchive(apath, filepath.Join(dir, "out")); err == nil {
		t.Fatal("绝对路径条目应被拒绝")
	}
}

// TestReplaceWindowsRunningExe 验证 Windows 慢路径：目标 exe 被运行中的进程锁定时，
// 三步 rename 大法仍能把新二进制就位、旧二进制落到 .old 备份。
// 用子进程（helper 模式）持有目标 exe 的锁，父测试进程做替换。
func TestReplaceWindowsRunningExe(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows：运行中 exe 的 rename 语义")
	}
	dir := t.TempDir()
	// 子进程 = 测试二进制的一份副本（不动 go test 自己的二进制）
	helper := filepath.Join(dir, "xx.exe")
	self, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(helper, self, 0o755); err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(dir, "ready")
	cmd := exec.Command(helper, "-test.run=TestHelperHoldLock", "-test.v=false")
	cmd.Env = append(os.Environ(), "CCX_HELPER=1", "CCX_READY_FILE="+ready)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()
	// 等子进程跑起来并持有 exe 锁
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("子进程未在超时内就绪")
		}
		time.Sleep(20 * time.Millisecond)
	}

	stage := filepath.Join(dir, "xx.exe.new")
	if err := os.WriteFile(stage, []byte("NEW BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := replaceBinaryWindows(stage, helper); err != nil {
		t.Fatalf("replaceBinaryWindows: %v", err)
	}
	// 新内容就位；stage 已被 rename 走
	got, err := os.ReadFile(helper)
	if err != nil || string(got) != "NEW BINARY" {
		t.Fatalf("新二进制未就位: err=%v content=%q", err, got)
	}
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatal("stage 应已被 rename 走")
	}
	// 旧二进制在 .old 备份里（子进程仍持有锁，删除必然失败——设计如此）
	olds, _ := filepath.Glob(filepath.Join(dir, "xx.exe.*.old"))
	if len(olds) != 1 {
		t.Fatalf("应恰有一个 .old 备份，got %v", olds)
	}
	oldData, err := os.ReadFile(olds[0])
	if err != nil || len(oldData) == 0 || string(oldData) == "NEW BINARY" {
		t.Fatalf(".old 应为旧二进制内容: err=%v", err)
	}
	// 子进程退出后，.old 可删（下次升级的清理语义）
	cmd.Process.Kill()
	cmd.Wait()
	if err := os.Remove(olds[0]); err != nil {
		t.Fatalf("子进程退出后 .old 应可删除: %v", err)
	}
}

// TestHelperHoldLock 只作为子进程载体：持有自身 exe 的锁直到被杀。父进程
// 通过 CCX_READY_FILE 感知它已开始运行。
func TestHelperHoldLock(t *testing.T) {
	if os.Getenv("CCX_HELPER") != "1" {
		t.Skip("helper 模式")
	}
	if f, err := os.Create(os.Getenv("CCX_READY_FILE")); err == nil {
		f.Close()
	}
	time.Sleep(time.Hour)
}
