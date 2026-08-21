// Package launch 负责定位 claude 并以继承终端的方式启动它（本次启用）。
package launch

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ResolveClaude 在 PATH 中定位 claude；找不到返回 ok=false。
// Windows 下 exec.LookPath 会按 PATHEXT 解析到 claude.exe 或 claude.cmd。
func ResolveClaude() (string, bool) {
	p, err := exec.LookPath("claude")
	if err != nil {
		return "", false
	}
	return p, true
}

// LaunchSession 启动 claude（路径 bin），继承当前进程的 stdin/stdout/stderr 与环境，阻塞至退出，返回其退出码。
// args 追加到 claude 的命令行（如 --effort high，见 env.EffortArgs）。
// 调用方须先 env.ApplyManaged 设好受管环境（子进程继承）。
//
// Windows 注意：.cmd / .bat 不是 PE 可执行文件，不能直接 CreateProcess，必须经 cmd.exe /c 启动
// （npm 安装的 claude 常是 claude.cmd；原生安装是 claude.exe，可直接 exec）。详见 plan §2.5.3 / §12。
func LaunchSession(bin string, args ...string) (int, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" && isCmdScript(bin) {
		comspec := os.Getenv("ComSpec")
		if comspec == "" {
			comspec = "cmd.exe"
		}
		cmd = exec.Command(comspec, append([]string{"/d", "/s", "/c", bin}, args...)...)
	} else {
		cmd = exec.Command(bin, args...)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if asExit(err, &ee) {
			return ee.ExitCode(), nil // claude 自身非零退出：透传退出码，不算 spawn 失败
		}
		return 1, err // spawn 本身失败（找不到/无法启动）
	}
	return 0, nil
}

func isCmdScript(p string) bool {
	s := strings.ToLower(p)
	return strings.HasSuffix(s, ".cmd") || strings.HasSuffix(s, ".bat")
}

func asExit(err error, target **exec.ExitError) bool {
	ee, ok := err.(*exec.ExitError)
	if ok {
		*target = ee
	}
	return ok
}
