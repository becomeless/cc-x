// `xx update` 自更新的实现：实时查最新版 → 下载当前平台资产 → 校验 sha256 → 解压 →
// 原地替换自身二进制（Windows 走 install.ps1 的三步 rename 大法，Unix 直接原子 rename）。
//
// 与包内既有代码同风格：网络走 curl 子进程（curl 的 -f/-sSL/退出码语义已有测试对齐；net/http 虽已被
// presets.FetchModels 引入二进制，但迁移无体积收益、却要重对齐这些行为，故维持现状）；只写临时目录与
// 二进制所在目录，不碰任何 Claude Code 配置（铁律）。下载不写 Zone.Identifier，
// 新二进制首次运行不会触发 SmartScreen 网络来源拦截。
package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/becomeless/cc-x/internal/i18n"
)

const (
	// minSelfUpdate 是引入 `xx update` 命令的版本：旧二进制没有该命令，菜单横幅按
	// 当前版本是否 >= 它来决定提示 `xx update` 还是安装器命令（防版本错配）。
	minSelfUpdate = "0.4.25"
	selfUpdateCmd = "xx update"
	// downloadTimeout 是资产下载的硬超时；fetchLatest 的 5s 只适合重定向探测。
	downloadTimeout = 120 * time.Second
	baseReleaseURL  = "https://github.com/becomeless/cc-x/releases/download"
	// npmUpgrade 是原生版不支持的平台（发布矩阵外）与 npm 版的升级兜底。
	npmUpgrade = "npm i -g @cc-x/cc-x@latest"
)

// assetName 返回发布资产名（对齐 build-release.ps1 的命名与平台矩阵）。平台不在
// 矩阵内返回 error。
func assetName(version, goos, goarch string) (string, error) {
	switch goos {
	case "windows":
		if goarch != "amd64" {
			return "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
		}
		return fmt.Sprintf("ccx_%s_windows_amd64.zip", version), nil
	case "darwin", "linux":
		if goarch != "amd64" && goarch != "arm64" {
			return "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
		}
		return fmt.Sprintf("ccx_%s_%s_%s.tar.gz", version, goos, goarch), nil
	}
	return "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
}

// parseChecksums 解析 checksums.txt（"小写hex  文件名" 每行一个，对齐 install.sh 的
// awk $2 取值），返回 文件名 -> hex。
func parseChecksums(data []byte) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		m[fields[1]] = strings.ToLower(fields[0])
	}
	return m
}

// shouldUseSelfUpdate 报告当前版本是否内置自更新命令（正式版本号 >= minSelfUpdate）。
// 解析失败（dev 等）一律 false。
func shouldUseSelfUpdate(current string) bool {
	v, ok := parseSemver(current)
	if !ok {
		return false
	}
	min, ok := parseSemver(minSelfUpdate)
	if !ok {
		return false
	}
	for i := 0; i < 3; i++ {
		if v[i] != min[i] {
			return v[i] > min[i]
		}
	}
	return true
}

// latestVersion 实时获取最新版本号（显式 update 不走 24h 缓存），失败返回 error。
func latestVersion() (string, error) {
	nullDev := "/dev/null"
	if runtime.GOOS == "windows" {
		nullDev = "NUL"
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "curl", "-s", "--max-redirs", "0",
		"-o", nullDev, "-w", "%{redirect_url}", latestURL).Output()
	if err != nil {
		return "", fmt.Errorf("curl: %w", err)
	}
	m := tagRe.FindStringSubmatch(strings.TrimSpace(string(out)))
	if m == nil {
		return "", fmt.Errorf("无法从 %s 解析最新版本号", latestURL)
	}
	return m[1], nil
}

// downloadFile 用 curl 下载到 dest；HTTP 错误（-f）与超时都会失败返回。
func downloadFile(url, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "curl", "-fsSL",
		"--max-time", strconv.Itoa(int(downloadTimeout.Seconds())), "-o", dest, url).CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return fmt.Errorf("curl: %s", strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("curl: %w", err)
	}
	return nil
}

// fetchChecksums 下载校验和文件；失败返回 (nil, false)，调用方降级为跳过校验
// （对齐 install.sh 的语义：校验不可得不拦更新）。
func fetchChecksums(url string) ([]byte, bool) {
	tmp, err := os.CreateTemp("", "ccx-checksums-*")
	if err != nil {
		return nil, false
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)
	if err := downloadFile(url, name); err != nil {
		return nil, false
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, false
	}
	return data, true
}

// verifySHA256 校验文件的 sha256 是否匹配期望值（忽略大小写）。
func verifySHA256(file, expectedHex string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expectedHex) {
		return fmt.Errorf("sha256 不匹配：got %s want %s", actual, expectedHex)
	}
	return nil
}

// extractArchive 按扩展名分派解压到 destDir。
func extractArchive(archive, destDir string) error {
	switch {
	case strings.HasSuffix(archive, ".zip"):
		return extractZip(archive, destDir)
	case strings.HasSuffix(archive, ".tar.gz"):
		return extractTarGz(archive, destDir)
	}
	return fmt.Errorf("未知压缩格式：%s", archive)
}

// safeJoin 校验归档条目不会逃出 destDir（路径穿越防护），返回拼接后的目标路径。
func safeJoin(destDir, name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if !filepath.IsLocal(clean) {
		return "", fmt.Errorf("归档条目越界：%s", name)
	}
	return filepath.Join(destDir, clean), nil
}

func extractZip(archive, destDir string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		target, err := safeJoin(destDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		err = writeFileFrom(src, target, f.Mode())
		src.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(archive, destDir string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// 拒绝绝对路径；发布包只含普通文件与目录，symlink/hardlink 一律拒绝。
		if filepath.IsAbs(hdr.Name) {
			return fmt.Errorf("归档条目为绝对路径：%s", hdr.Name)
		}
		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := writeFileFrom(tr, target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("归档条目类型不支持：%s", hdr.Name)
		}
	}
}

// writeFileFrom 以指定 mode 落盘。
func writeFileFrom(r io.Reader, path string, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// findBinaryIn 在 dir 下找 xx / xx.exe（对齐 install.ps1 的递归查找）。
func findBinaryIn(dir string) (string, error) {
	var found string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "xx" || d.Name() == "xx.exe" {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("压缩包内未找到 xx 二进制：%s", dir)
	}
	return found, nil
}

// executablePath 返回自身二进制解析 symlink 后的真实路径（macOS symlink 安装场景）。
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// copyToStage 复制到目标目录内的 stage 文件：io.Copy 而非 rename，规避 %TEMP%
// 与目标目录不同卷时的 EXDEV。
func copyToStage(src, stage string) error {
	return copyFile(src, stage)
}

// copyFile 复制文件，目标统一 0755（可执行位）。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return writeFileFrom(in, dst, 0o755)
}

// replaceBinaryUnix：Unix 上 os.Rename 原子覆盖，补执行位。
func replaceBinaryUnix(stage, dest string) error {
	if err := os.Rename(stage, dest); err != nil {
		return err
	}
	return os.Chmod(dest, 0o755)
}

// replaceBinaryWindows 对齐 install.ps1 的三步大法：Windows 上运行中的 exe 可
// rename 不可覆盖。1) 清理历史 .old 备份（锁定中删除失败静默，留待下次升级）；
// 2) 快路径直接 rename 覆盖（xx 未运行时成功）；3) 慢路径 rename 旧→.old、
// rename 新→dest，失败回滚。
func replaceBinaryWindows(stage, dest string) error {
	dir := filepath.Dir(dest)
	base := filepath.Base(dest)
	// 1. 清理历史备份
	olds, _ := filepath.Glob(filepath.Join(dir, base+".*.old"))
	for _, old := range olds {
		_ = os.Remove(old)
	}
	// 2. 快路径：本进程未占用 dest 时可直接覆盖
	if err := os.Rename(stage, dest); err == nil {
		return nil
	}
	// 3. 慢路径：rename 运行中的旧二进制 -> .old，再把新二进制 rename 到位
	backup := filepath.Join(dir, fmt.Sprintf("%s.%d.old", base, os.Getpid()))
	if err := os.Rename(dest, backup); err != nil {
		return fmt.Errorf("旧二进制改名失败：%w", err)
	}
	if err := os.Rename(stage, dest); err != nil {
		// 回滚：dest 槽空则还原备份
		if _, statErr := os.Stat(dest); os.IsNotExist(statErr) {
			_ = os.Rename(backup, dest)
		}
		return fmt.Errorf("新二进制就位失败：%w", err)
	}
	_ = os.Remove(backup) // 本进程退出前必然锁定，失败忽略
	return nil
}

// copySidecars 把解压目录里的辅助文件（presets.json/LICENSE/README*）复制到二进制
// 所在目录（存在才拷，对齐安装器）。辅助文件失败不影响主二进制，仅跳过。
func copySidecars(srcDir, destDir string) {
	for _, name := range []string{"presets.json", "LICENSE", "README.md", "README.en.md"} {
		src := filepath.Join(srcDir, name)
		if fi, err := os.Stat(src); err != nil || fi.IsDir() {
			continue
		}
		_ = copyFile(src, filepath.Join(destDir, name))
	}
}

// RunSelfUpdate 执行 `xx update` 主流程，返回进程退出码。文案走 i18n。
func RunSelfUpdate(current string) int {
	fmt.Println("")
	// dev 构建无法自更新，先短路（不联网，省一次无谓请求）
	if _, ok := parseSemver(current); !ok {
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("update.dev", UpgradeCommand(current)))
		return 1
	}
	fmt.Printf("  %s\n", i18n.T("update.checking"))
	latest, err := latestVersion()
	if err != nil {
		return failSelfUpdate(err, current)
	}
	if !isNewer(latest, current) {
		fmt.Printf("  %s\n", i18n.T("update.latest", latest))
		return 0
	}
	fmt.Printf("  %s\n", i18n.T("update.found", latest, current))

	exe, err := executablePath()
	if err != nil {
		return failSelfUpdate(err, current)
	}
	dir := filepath.Dir(exe)
	dest := exe

	tmp, err := os.MkdirTemp("", "ccx-update-*")
	if err != nil {
		return failSelfUpdate(err, current)
	}
	defer os.RemoveAll(tmp)

	arch, err := assetName(latest, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("update.failed", err.Error()))
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("update.npmHint", npmUpgrade))
		return 1
	}
	fmt.Printf("  %s\n", i18n.T("update.downloading", arch))
	archive := filepath.Join(tmp, arch)
	if err := downloadFile(baseReleaseURL+"/v"+latest+"/"+arch, archive); err != nil {
		return failSelfUpdate(err, current)
	}
	fmt.Printf("  %s\n", i18n.T("update.verify"))
	if data, ok := fetchChecksums(baseReleaseURL + "/v" + latest + "/checksums.txt"); ok {
		expected := parseChecksums(data)[arch]
		if expected == "" {
			fmt.Printf("  %s\n", i18n.T("update.verifySkip"))
		} else if err := verifySHA256(archive, expected); err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("update.verifyMismatch"))
			return 1
		}
	} else {
		fmt.Printf("  %s\n", i18n.T("update.verifySkip"))
	}

	if err := extractArchive(archive, tmp); err != nil {
		return failSelfUpdate(err, current)
	}
	src, err := findBinaryIn(tmp)
	if err != nil {
		return failSelfUpdate(err, current)
	}
	stage := filepath.Join(dir, filepath.Base(dest)+".new")
	if err := copyToStage(src, stage); err != nil {
		return failSelfUpdate(err, current)
	}
	defer os.Remove(stage) // 替换成功后 stage 已不存在；失败时清理残留

	var replaceErr error
	if runtime.GOOS == "windows" {
		replaceErr = replaceBinaryWindows(stage, dest)
	} else {
		replaceErr = replaceBinaryUnix(stage, dest)
	}
	if replaceErr != nil {
		return failSelfUpdate(replaceErr, current)
	}
	copySidecars(filepath.Dir(src), dir)
	fmt.Printf("  %s\n", i18n.T("update.done", latest))
	return 0
}

// failSelfUpdate 打印失败原因 + 安装命令兜底，返回退出码 1。
func failSelfUpdate(err error, current string) int {
	fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("update.failed", err.Error()))
	fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("update.fallback", UpgradeCommand(current)))
	return 1
}
