//go:build windows

package launch

import "errors"

// LaunchSessionExec Windows 无 execve。main.go 以 runtime.GOOS 分支保证永不调用，
// 此 stub 仅为让 Windows 构建有符号可链接（返回错误兜底）。
func LaunchSessionExec(bin string) error {
	return errors.New("exec not supported on windows")
}
