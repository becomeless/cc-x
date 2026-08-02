// Command xx 是 ccx 的 Go 原生入口：解析参数 -> 分派到 CLI 路径或交互菜单。
//
// 铁律：绝不写 Claude Code 配置文件；API 切换只动 10 个受管环境变量。
package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/becomeless/cc-x/internal/config"
	"github.com/becomeless/cc-x/internal/defaults"
	"github.com/becomeless/cc-x/internal/env"
	"github.com/becomeless/cc-x/internal/i18n"
	"github.com/becomeless/cc-x/internal/launch"
	"github.com/becomeless/cc-x/internal/presets"
	"github.com/becomeless/cc-x/internal/runtimeinfo"
	"github.com/becomeless/cc-x/internal/tui"
)

// version 在发布时通过 `-ldflags "-X main.version=x.y.z"` 注入；dev 下为占位。
var version = "dev"

type options struct {
	name         string
	session      bool
	list         bool
	showVersion  bool
	showHelp     bool
	storeDir     string
	defaultScope string
	lang         string
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", err.Error())
		os.Exit(1)
	}

	// help/version 要在产出任何文案前定语言：--lang > providers.json 的 lang（只读探测，不建文件）> 环境 > zh。
	earlyStoreLang := config.PeekStoreLang(config.ResolveStorePaths(opts.storeDir))
	i18n.SetLang(i18n.ResolveLang(config.Lang(opts.lang), earlyStoreLang))

	if opts.showVersion {
		fmt.Println(version)
		return
	}
	if opts.showHelp {
		printHelp()
		return
	}
	os.Exit(dispatch(opts))
}

// parseArgs 手写解析（Go 无 commander）：支持长短选项、`--opt val` 与 `--opt=val`，首个非选项参数为 name。
func parseArgs(argv []string) (options, error) {
	o := options{defaultScope: "user"}
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		takeVal := func(flag string) (string, error) {
			if i+1 >= len(argv) {
				return "", fmt.Errorf("%s requires a value", flag)
			}
			i++
			return argv[i], nil
		}
		switch {
		case a == "-s" || a == "--session":
			o.session = true
		case a == "-l" || a == "--list":
			o.list = true
		case a == "-v" || a == "--version":
			o.showVersion = true
		case a == "-h" || a == "--help":
			o.showHelp = true
		case a == "--store-dir":
			v, err := takeVal(a)
			if err != nil {
				return o, err
			}
			o.storeDir = v
		case strings.HasPrefix(a, "--store-dir="):
			o.storeDir = strings.TrimPrefix(a, "--store-dir=")
		case a == "--default-scope":
			v, err := takeVal(a)
			if err != nil {
				return o, err
			}
			o.defaultScope = v
		case strings.HasPrefix(a, "--default-scope="):
			o.defaultScope = strings.TrimPrefix(a, "--default-scope=")
		case a == "--lang":
			v, err := takeVal(a)
			if err != nil {
				return o, err
			}
			o.lang = v
		case strings.HasPrefix(a, "--lang="):
			o.lang = strings.TrimPrefix(a, "--lang=")
		case strings.HasPrefix(a, "-") && a != "-":
			return o, fmt.Errorf("unknown option: %s", a)
		default:
			if o.name == "" {
				o.name = a
			}
		}
	}
	if o.defaultScope != "user" && o.defaultScope != "process" {
		return o, fmt.Errorf("--default-scope must be user or process")
	}
	if o.lang != "" && o.lang != "zh" && o.lang != "en" {
		return o, fmt.Errorf("--lang must be zh or en")
	}
	return o, nil
}

func dispatch(o options) int {
	paths := config.ResolveStorePaths(o.storeDir)
	store, err := config.Load(paths)
	if err != nil {
		var se *config.StoreError
		if errors.As(err, &se) {
			// store 不可用，按 --lang/环境定语言（拿不到 store.lang）。绝不重建、不动用户数据。
			i18n.SetLang(i18n.ResolveLang(config.Lang(o.lang), ""))
			head := "error.storeCorrupt"
			switch se.Kind {
			case config.ErrRead:
				head = "error.storeRead"
			case config.ErrFormat:
				head = "error.storeFormat"
			}
			fmt.Fprintf(os.Stderr, "  %s\n", i18n.T(head, se.File))
			fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("error.storeCorruptHint"))
			if se.Kind == config.ErrParse || se.Kind == config.ErrFormat {
				fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("error.storeBackupHint", backupCommand(se.File)))
			}
			return 1
		}
		fmt.Fprintf(os.Stderr, "  %s\n", err.Error())
		return 1
	}
	// 语言：--lang > providers.json lang > 环境 > zh。在产出任何文案前先定好。
	i18n.SetLang(i18n.ResolveLang(config.Lang(o.lang), store.Lang))

	if o.list {
		runList(store)
		return 0
	}
	if o.name != "" {
		p := findProviderForCLI(store, o.name)
		if p == nil {
			names := make([]string, len(store.Providers))
			for i, pp := range store.Providers {
				names[i] = i18n.ProviderDisplayName(pp)
			}
			fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("error.notFound", o.name))
			fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("error.existing", strings.Join(names, ", ")))
			fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("error.notFoundHint"))
			return 1
		}
		if o.session {
			return launchSession(*p)
		}
		return runDefault(paths, store, p, defaults.Scope(o.defaultScope))
	}
	// 无参：打开交互菜单。
	tui.OpenMenu(tui.New(), paths, store, defaults.Scope(o.defaultScope), version, presets.Load(o.storeDir))
	return 0
}

func findProviderForCLI(store *config.Store, name string) *config.Provider {
	if p := config.FindProvider(store, name); p != nil {
		return p
	}
	for i := range store.Providers {
		if i18n.ProviderDisplayName(store.Providers[i]) == name {
			return &store.Providers[i]
		}
	}
	return nil
}

// warnIfNoKey：非官方且未填密钥时给黄字提示（到 stderr，对齐 npm 版）。
func warnIfNoKey(p config.Provider) {
	if config.GetProviderState(p).Key == config.KeyNone {
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("session.noKey", i18n.ProviderDisplayName(p)))
	}
}

func warnIfNoKeyForDefault(p config.Provider) {
	if config.GetProviderState(p).Key == config.KeyNone {
		fmt.Printf("  %s\n", i18n.T("default.noKey", i18n.ProviderDisplayName(p)))
	}
}

// runDefault：设为默认（写用户环境变量或 dry-run）+ 更新 store.current。逐行对齐 npm 版 runDefault。
func runDefault(paths config.StorePaths, store *config.Store, p *config.Provider, scope defaults.Scope) int {
	warnIfNoKeyForDefault(*p)
	name := i18n.ProviderDisplayName(*p)
	r := defaults.SetDefault(paths, store, *p, scope)

	if r.DryRun {
		fmt.Printf("  %s\n", i18n.T("default.done", name))
		fmt.Printf("  %s\n", i18n.T("default.dryRun"))
		fmt.Printf("  %s\n", i18n.T("default.hintSession", quoteArg(p.Name)))
		return 0
	}
	if r.WinOK != nil && !*r.WinOK {
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("default.failed", r.WinErr))
		return 1
	}
	if r.Unix != nil && r.Unix.Unsupported {
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("default.fishUnsupported"))
		return 0
	}
	fmt.Printf("  %s\n", i18n.T("default.done", name))
	if r.Unix != nil {
		fmt.Printf("  %s\n", i18n.T("default.unixWrote", r.Unix.File))
	}
	fmt.Printf("  %s\n", i18n.T("default.hintSession", quoteArg(p.Name)))
	return 0
}

// launchSession：本次启用 —— 提示 + banner + 套环境启动 claude，阻塞至其退出。对齐 npm 版 launchSession/sessionLaunch。
// Unix/macOS：env 套好后直接 syscall.Exec 进程替换（常驻归零、退出码透传）；Windows 无 exec-replace，保持子进程等待。
func launchSession(p config.Provider) int {
	warnIfNoKey(p)
	fmt.Println("")
	fmt.Printf("  %s\n", i18n.T("session.launch", i18n.ProviderDisplayName(p)))
	fmt.Printf("  %s\n", i18n.T("session.starting"))
	fmt.Println("")
	bin, ok := launch.ResolveClaude()
	if !ok {
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("session.noClaude"))
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("session.noClaudeHint"))
		return 1
	}
	env.ApplyManaged(p)
	if runtime.GOOS != "windows" {
		// banner 已打印（os.Stdout 直写无缓冲）；exec 成功则永不返回，退出码由 claude 本体透传。
		if err := launch.LaunchSessionExec(bin); err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", err.Error())
			return 1
		}
		return 0 // 不可达：exec 成功则进程已被替换
	}
	code, err := launch.LaunchSession(bin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", err.Error())
		return 1
	}
	return code
}

// runList 列出所有配置及状态。官方档显示名走 i18n（评审①），其余原样。逐行对齐 npm 版 runList。
func runList(store *config.Store) {
	var cur *config.Provider
	for i := range store.Providers {
		if store.Providers[i].Name == store.Current {
			cur = &store.Providers[i]
			break
		}
	}
	fmt.Println("")
	curName := store.Current
	if cur != nil {
		curName = i18n.ProviderDisplayName(*cur)
	}
	fmt.Printf("  %s\n", i18n.T("list.default", curName))
	fmt.Printf("  %s\n", runtimeinfo.CurrentTerminalLine(store))
	for _, line := range tui.ProfileRows(store.Providers, store.Current) {
		fmt.Printf("  %s\n", line)
	}
	fmt.Println("")
}

func printHelp() {
	fmt.Println("Usage: xx [options] [name]")
	fmt.Println("")
	fmt.Println("  " + i18n.T("cli.desc"))
	fmt.Println("")
	fmt.Println("Arguments:")
	fmt.Printf("  name                       %s\n", i18n.T("cli.arg.name"))
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Printf("  -s, --session              %s\n", i18n.T("cli.opt.session"))
	fmt.Printf("  -l, --list                 %s\n", i18n.T("cli.opt.list"))
	fmt.Printf("      --store-dir <dir>      %s\n", i18n.T("cli.opt.storeDir"))
	fmt.Printf("      --default-scope <s>    %s\n", i18n.T("cli.opt.defaultScope"))
	fmt.Printf("      --lang <zh|en>         %s\n", i18n.T("cli.opt.lang"))
	fmt.Printf("  -v, --version              %s\n", i18n.T("cli.opt.version"))
	fmt.Printf("  -h, --help                 %s\n", i18n.T("cli.opt.help"))
}

func backupCommand(file string) string {
	dst := file + ".bak"
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("Copy-Item -LiteralPath %s -Destination %s", psQuote(file), psQuote(dst))
	}
	return fmt.Sprintf("cp %s %s", shQuote(file), shQuote(dst))
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func quoteArg(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
