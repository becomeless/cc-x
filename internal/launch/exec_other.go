//go:build !windows

package launch

import (
	"os"
	"syscall"
)

// LaunchSessionExec 用 execve(2) 把当前进程替换为 claude（Unix/macOS；Windows 无此机制，见 exec_windows.go）。
// 调用方须已打印完所有提示并 env.ApplyManaged（环境随 exec 保留给 claude）。
// 成功则永不返回——进程即 claude 本体，退出码天然透传，常驻开销归零。
// 失败返回错误（本进程仍存活）。
//
// 注意：仅限 CLI `xx <name> -s` 路径使用。菜单内启动（tuiLaunchSession）必须继续用
// LaunchSession 的子进程方式——claude 退出后要回到菜单，exec 会丢掉菜单上下文。
func LaunchSessionExec(bin string) error {
	// argv[0] 用解析到的完整路径；npm 安装的 claude 是 shebang 脚本，内核直接处理。
	return syscall.Exec(bin, []string{bin}, os.Environ())
}
