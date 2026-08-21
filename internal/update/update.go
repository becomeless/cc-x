// Package update 是「检查新版本」的轻量实现：不走 GitHub API（仅用 releases/latest 的 302 重定向
// 抠版本号，无速率限制），结果缓存在 ~/.cc-mini/update-check.json，每 24h 才真去网络。
//
// 设计：显示永远读缓存（瞬时、不阻塞），过期时后台异步刷新——所以新版本「下次打开」才提示，
// 与用户预期一致。离线/失败一律静默。只写工具自己的 ~/.cc-mini/，不碰任何 Claude Code 配置（铁律）。
//
// 网络请求通过 curl 子进程完成（exec.CommandContext）。注意「省 TLS 栈」不是理由——net/http
// 已被 presets.FetchModels 引入二进制；这里保留 curl 是因为 302 抠版本号（--max-redirs 0 +
// %{redirect_url}）、超时与失败静默的语义已在此固化并有测试对齐，迁移无体积收益却要重对齐这些行为。
// curl 在所有目标平台均可用：Windows 10 1803+ 内置 curl.exe；macOS/Linux 系统自带。
package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Mode 是更新检查模式。空 = 关闭。
const (
	ModeOff    = ""
	ModeNotify = "notify"
)

const (
	latestURL    = "https://github.com/becomeless/cc-x/releases/latest"
	cacheMaxAge  = 24 * time.Hour
	fetchTimeout = 5 * time.Second
	cacheFile    = "update-check.json"
	winUpgrade   = "irm https://github.com/becomeless/cc-x/releases/latest/download/install.ps1 | iex"
	unixUpgrade  = "curl -fsSL https://github.com/becomeless/cc-x/releases/latest/download/install.sh | sh"
)

type cache struct {
	CheckedAt int64  `json:"checkedAt"`
	Latest    string `json:"latest"`
}

// tagRe 从 .../releases/tag/v0.4.3 抠出 0.4.3。
var tagRe = regexp.MustCompile(`/tag/v?(\d+\.\d+\.\d+)`)

// Banner 读缓存：若已知有比 current 更新的版本，返回 (最新版本号, true)。不联网。
func Banner(storeDir, current string) (string, bool) {
	c, err := readCache(storeDir)
	if err != nil || c.Latest == "" {
		return "", false
	}
	if isNewer(c.Latest, current) {
		return c.Latest, true
	}
	return "", false
}

// MaybeRefresh 缓存过期（或不存在）时后台异步联网刷新一次；不阻塞调用方。
func MaybeRefresh(storeDir string) {
	c, err := readCache(storeDir)
	if err == nil && time.Since(time.Unix(c.CheckedAt, 0)) < cacheMaxAge {
		return // 仍新鲜
	}
	go func() {
		latest := fetchLatest()
		if latest == "" {
			return // 失败静默；不动缓存
		}
		writeCache(storeDir, cache{CheckedAt: time.Now().Unix(), Latest: latest})
	}()
}

// UpgradeCommand 返回当前版本的升级方式：内置自更新（>= minSelfUpdate）返回
// `xx update`；旧版二进制没有该命令，仍给安装器一行命令（防版本错配）。
func UpgradeCommand(current string) string {
	if shouldUseSelfUpdate(current) {
		return selfUpdateCmd
	}
	if runtime.GOOS == "windows" {
		return winUpgrade
	}
	return unixUpgrade
}

func cachePath(storeDir string) string { return filepath.Join(storeDir, cacheFile) }

func readCache(storeDir string) (cache, error) {
	var c cache
	data, err := os.ReadFile(cachePath(storeDir))
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return cache{}, err
	}
	return c, nil
}

// writeCache 原子写（temp + rename），避免后台 goroutine 被进程退出打断写出半截文件。
// 写或 rename 失败都清理 tmp（对齐 claudecfg.atomicWrite 的清理语义，不留孤儿文件）。
func writeCache(storeDir string, c cache) {
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		return
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	tmp := cachePath(storeDir) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		_ = os.Remove(tmp) // 写失败也可能留下半截 tmp
		return
	}
	if err := os.Rename(tmp, cachePath(storeDir)); err != nil {
		_ = os.Remove(tmp)
	}
}

// fetchLatest 用 curl 子进程获取 releases/latest 的 302 Location 并抠出版本号。
// --max-redirs 0 禁止跟随重定向；-w "%{redirect_url}" 直接输出目标 URL，无需解析响应头。
// curl 不可用或网络失败均静默返回 ""（横幅后台刷新语义；显式 update 走 latestVersion 报错）。
func fetchLatest() string {
	v, err := latestVersion()
	if err != nil {
		return ""
	}
	return v
}

// isNewer 报告 latest 是否严格新于 current（"a.b.c"，忽略前导 v 与后缀）。解析失败一律 false（不误报）。
func isNewer(latest, current string) bool {
	lp, ok1 := parseSemver(latest)
	cp, ok2 := parseSemver(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if lp[i] != cp[i] {
			return lp[i] > cp[i]
		}
	}
	return false
}

// parseSemver 解析 "v1.2.3" / "1.2.3-rc1" 的前三段数字。
func parseSemver(s string) ([3]int, bool) {
	var out [3]int
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	// 砍掉 -rc1 / +build 之类后缀
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
