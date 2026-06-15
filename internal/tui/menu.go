package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/becomeless/cc-x/internal/check"
	"github.com/becomeless/cc-x/internal/config"
	"github.com/becomeless/cc-x/internal/defaults"
	"github.com/becomeless/cc-x/internal/display"
	"github.com/becomeless/cc-x/internal/env"
	"github.com/becomeless/cc-x/internal/i18n"
	"github.com/becomeless/cc-x/internal/launch"
	"github.com/becomeless/cc-x/internal/presets"
	"github.com/becomeless/cc-x/internal/runtimeinfo"
	"github.com/becomeless/cc-x/internal/update"
)

// OpenMenu 一级 · 主菜单。布局：[profiles…] ” 新增 语言 ” 退出。对应 npm 版 openMenu。
func OpenMenu(t *Terminal, paths config.StorePaths, store *config.Store, scope defaults.Scope, version string, catalog []presets.Preset) {
	sel := 0
	refreshed := false
	flash := ""
	warnFlash := ""
	for {
		n := len(store.Providers)
		// 更新检查（仅 notify 模式）：首轮触发一次后台刷新；横幅永远读缓存（瞬时、不阻塞）。
		notices := []string{runtimeinfo.CurrentTerminalLine(store)}
		if needsFirstRunHint(store) {
			notices = append(notices, i18n.T("menu.firstRunHint"))
		}
		if warnFlash != "" {
			notices = append(notices, warnFlash)
		}
		if store.Update == update.ModeNotify {
			if !refreshed {
				update.MaybeRefresh(paths.Dir)
				refreshed = true
			}
			if latest, ok := update.Banner(paths.Dir, version); ok {
				notices = append(notices, i18n.T("menu.updateAvailable", latest, update.UpgradeCommand()))
			}
		}
		updLabel := i18n.T("menu.updateOff")
		if store.Update == update.ModeNotify {
			updLabel = i18n.T("menu.updateNotify")
		}
		buildItems := func() []string {
			items := append([]string{}, ProfileRows(store.Providers, store.Current)...)
			return append(items, "", i18n.T("menu.newProfile"), i18n.T("menu.language"), updLabel, "", i18n.T("menu.exit"))
		}
		moveWarn := ""
		onMove := func(from, to int) []string {
			ps := store.Providers
			if from >= 0 && from < len(ps) && to >= 0 && to < len(ps) {
				ps[from], ps[to] = ps[to], ps[from]
				moveWarn = saveWarning(paths, store)
			}
			return buildItems()
		}

		defaultName := defaultDisplayName(store)
		shortcut := rune(0)
		sel = SelectMenu(t, SelectOptions{
			Title:        i18n.T("menu.mainTitle", version, defaultName),
			Notice:       strings.Join(notices, "\n"),
			Status:       flash,
			Items:        buildItems(),
			Colors:       map[int]Color{n + 1: ColorYellow},
			Start:        sel,
			MovableCount: n,
			OnMove:       onMove,
			OnKey: func(r rune, idx int) int {
				if idx >= n {
					return -1
				}
				switch r {
				case 'e', 's', 'd':
					shortcut = r
					return idx
				}
				return -1
			},
			Hint:     i18n.T("menu.mainHint"),
			NoNumber: true,
		})
		flash = ""
		warnFlash = moveWarn

		if shortcut != 0 && sel >= 0 && sel < n {
			p := &store.Providers[sel]
			switch shortcut {
			case 'e':
				old := p.Name
				if EditForm(t, p, store, catalog, false) {
					warnFlash, flash = saveEditedProfile(paths, store, p, old, scope)
				}
			case 's':
				tuiLaunchSession(*p)
			case 'd':
				warnFlash, flash = applyDefault(paths, store, p, scope)
			}
			continue
		}

		switch {
		case sel < 0 || sel == n+5: // 退出 / Esc / q
			return
		case sel == n+1: // 新增配置
			prov := config.Provider{Env: map[string]string{}}
			if EditForm(t, &prov, store, catalog, false) {
				store.Providers = append(store.Providers, prov)
				warnFlash = saveWarning(paths, store)
				sel = len(store.Providers) - 1
			}
		case sel == n+2: // 语言切换：即时切并写回 store.lang
			next := config.LangEN
			if i18n.GetLang() == config.LangEN {
				next = config.LangZH
			}
			i18n.SetLang(next)
			store.Lang = next
			warnFlash = saveWarning(paths, store)
		case sel == n+3: // 更新检查开关：关闭 <-> 提醒
			if store.Update == update.ModeNotify {
				store.Update = update.ModeOff
			} else {
				store.Update = update.ModeNotify
			}
			warnFlash = saveWarning(paths, store)
		case sel < n:
			p := &store.Providers[sel]
			if !config.IsOfficial(*p) && config.GetProviderState(*p).Key == config.KeyNone {
				// #9：无密钥的第三方配置，Enter 直达编辑并聚焦密钥行（铺平首次成功路径）。
				old := p.Name
				if EditForm(t, p, store, catalog, true) {
					warnFlash, flash = saveEditedProfile(paths, store, p, old, scope)
				}
			} else {
				if warn := actionMenu(t, paths, store, p, scope, catalog); warn != "" {
					warnFlash = warn
				}
			}
			if sel >= len(store.Providers) {
				sel = max(0, len(store.Providers)-1) // 删除后夹取
			}
		}
	}
}

// actionMenu 二级 · 动作菜单（循环停留；返回/删除已确认才回一级）。
func actionMenu(t *Terminal, paths config.StorePaths, store *config.Store, p *config.Provider, scope defaults.Scope, catalog []presets.Preset) string {
	sel := 0
	flash := ""
	warnFlash := "" // 黄字警告（如缺密钥），走 Notice 与绿色 Status 区分
	for {
		dft := ""
		if p.Name == store.Current {
			dft = i18n.T("menu.default")
		}
		title := i18n.T("action.titlePrefix") + i18n.ProviderDisplayName(*p) + dft + i18n.NoteSuffix(*p) + "    [" + i18n.StateLabel(*p) + "]"
		items := []string{i18n.T("action.session"), i18n.T("action.setDefault"), i18n.T("action.check"), i18n.T("action.edit"), i18n.T("action.delete"), i18n.T("action.back")}

		opts := SelectOptions{Title: title, Items: items, Start: sel, Hint: i18n.T("action.hint"), NoNumber: true}
		if warnFlash != "" {
			opts.Notice = warnFlash
		}
		if flash != "" {
			opts.Status = flash
		}
		sel = SelectMenu(t, opts)
		flash = ""
		warnFlash = ""

		switch sel {
		case 0:
			tuiLaunchSession(*p)
		case 1:
			warnFlash, flash = applyDefault(paths, store, p, scope)
		case 2:
			r := check.Profile(*p)
			if r.OK {
				flash = r.Message
			} else {
				warnFlash = r.Message
			}
		case 3:
			old := p.Name
			if EditForm(t, p, store, catalog, false) {
				warnFlash, flash = saveEditedProfile(paths, store, p, old, scope)
			}
		case 4:
			if config.IsOfficial(*p) {
				fmt.Printf("  %s\n", i18n.T("action.deleteOfficialWarn"))
			}
			if confirmKey(t, i18n.T("action.deleteConfirm", i18n.ProviderDisplayName(*p))) {
				removeProvider(store, p)
				config.ReconcileCurrent(store)
				warnFlash = saveWarning(paths, store)
				return warnFlash
			}
		default:
			return "" // 返回 / q / Esc
		}
	}
}

func defaultDisplayName(store *config.Store) string {
	if store.Current == "" {
		return "—"
	}
	for _, p := range store.Providers {
		if p.Name == store.Current {
			return i18n.ProviderDisplayName(p)
		}
	}
	return store.Current
}

func saveWarning(paths config.StorePaths, store *config.Store) string {
	if err := config.Save(paths, store); err != nil {
		return i18n.T("error.storeSave", err)
	}
	return ""
}

func saveEditedProfile(paths config.StorePaths, store *config.Store, p *config.Provider, oldName string, scope defaults.Scope) (warn, toast string) {
	wasDefault := store.Current == oldName
	if wasDefault {
		store.Current = p.Name // 改名/供应商时同步默认指向
	}
	if warn = saveWarning(paths, store); warn != "" {
		return warn, ""
	}
	if wasDefault {
		return syncDefaultEnv(p, scope)
	}
	return "", ""
}

func defaultWarning(p *config.Provider) string {
	if config.GetProviderState(*p).Key == config.KeyNone {
		return i18n.T("default.noKey", i18n.ProviderDisplayName(*p))
	}
	return ""
}

func defaultResultMessage(warn, name string, r defaults.Result) (string, string) {
	appendWarn := func(extra string) {
		if extra == "" {
			return
		}
		if warn == "" {
			warn = extra
			return
		}
		warn += "\n" + extra
	}
	if r.StoreErr != "" {
		appendWarn(i18n.T("error.storeSave", r.StoreErr))
	}
	switch {
	case r.DryRun:
		return warn, i18n.T("default.done", name) + "  " + i18n.T("default.dryRun")
	case r.WinOK != nil && !*r.WinOK:
		return warn, i18n.T("default.failed", r.WinErr)
	case r.Unix != nil && r.Unix.Unsupported:
		return warn, i18n.T("default.fishUnsupported")
	default:
		return warn, i18n.T("default.done", name)
	}
}

func syncDefaultEnv(p *config.Provider, scope defaults.Scope) (warn, toast string) {
	name := i18n.ProviderDisplayName(*p)
	return defaultResultMessage(defaultWarning(p), name, defaults.PersistEnv(*p, scope))
}

// applyDefault 设为默认，返回 (warn, toast)：warn 为黄字警告（缺密钥），toast 为绿色结果。
// 分开返回让调用方各自上色，避免警告被染成「成功」绿。对应 npm 版 applyDefault。
func applyDefault(paths config.StorePaths, store *config.Store, p *config.Provider, scope defaults.Scope) (warn, toast string) {
	name := i18n.ProviderDisplayName(*p)
	return defaultResultMessage(defaultWarning(p), name, defaults.SetDefault(paths, store, *p, scope))
}

// tuiLaunchSession 菜单内「本次启用」：提示 + banner + 套环境启动 claude，退出后回到动作菜单。
func tuiLaunchSession(p config.Provider) {
	if config.GetProviderState(p).Key == config.KeyNone {
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("session.noKey", i18n.ProviderDisplayName(p)))
	}
	fmt.Println("")
	fmt.Printf("  %s\n", i18n.T("session.launch", i18n.ProviderDisplayName(p)))
	fmt.Printf("  %s\n", i18n.T("session.starting"))
	fmt.Println("")
	bin, ok := launch.ResolveClaude()
	if !ok {
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("session.noClaude"))
		fmt.Fprintf(os.Stderr, "  %s\n", i18n.T("session.noClaudeHint"))
		return
	}
	env.ApplyManaged(p)
	_, _ = launch.LaunchSession(bin)
}

const rowNameW = 13 // 名字列宽（显示宽度，CJK 计 2）；状态/备注列宽按内容动态算

// ProfileRows 把所有配置格式化成定宽分栏的菜单行：名字为主信息（默认项加粗），
// 状态/备注/host 一律 dim 退为次要信息，让备注、host 各自对齐成竖列。
// 状态、备注列宽按当前配置实际内容动态取最大值——自动兜住任何语言/文案。
// TUI 主菜单与 xx --list 共用此函数，保证两处呈现一致。
func ProfileRows(providers []config.Provider, current string) []string {
	stateW, noteW := 0, 0
	for i := range providers {
		if w := display.Width(i18n.StateLabel(providers[i])); w > stateW {
			stateW = w
		}
		if w := display.Width(providers[i].Note); w > noteW {
			noteW = w
		}
	}
	stateW += 2 // 状态列尾留出与备注列的间距（最宽行否则会与备注贴死）
	if noteW > 0 {
		noteW += 2 // 有备注时留出与 host 的列间距
	}
	rows := make([]string, len(providers))
	for i := range providers {
		rows[i] = profileRow(providers[i], providers[i].Name == current, stateW, noteW)
	}
	return rows
}

// profileRow 组成一行：定宽分栏 + 次要信息 dim + 默认项加粗。
// 加粗用 1m/22m（reset 不动颜色），不会破坏 SelectMenu 选中行的外层绿色。
func profileRow(p config.Provider, isDefault bool, stateW, noteW int) string {
	name := display.Pad(i18n.ProviderDisplayName(p), rowNameW)
	marker := "  "
	if isDefault {
		marker = Paint("● ", ColorBold)
		name = Paint(name, ColorBold)
	}
	state := Paint(" · "+display.Pad(i18n.StateLabel(p), stateW), ColorDim)
	note := display.Pad(p.Note, noteW) // 空备注=定宽空白，撑出 host 前的列间距
	if strings.TrimSpace(p.Note) != "" {
		note = Paint(note, ColorDim)
	}
	return marker + name + state + note + hostCol(p)
}

// hostCol 返回行尾 dim 的 host（无 base 返回空）。超宽时由 SelectMenu 的 ANSI-aware 截断裁掉。
func hostCol(p config.Provider) string {
	base := strings.TrimSpace(config.GetProviderEnvMap(p)["ANTHROPIC_BASE_URL"])
	if base == "" {
		return ""
	}
	return Paint(runtimeinfo.HostOf(base), ColorDim)
}

func needsFirstRunHint(store *config.Store) bool {
	hasThirdParty := false
	for _, p := range store.Providers {
		if config.IsOfficial(p) {
			continue
		}
		hasThirdParty = true
		if config.GetProviderState(p).Key != config.KeyNone {
			return false
		}
	}
	return hasThirdParty
}

// confirmKey 在 raw 模式下读一个按键确认（y/Y=是，其余任意键=否），与菜单 raw 体验一致、无需回车。
// 进 raw 失败（非 TTY 等）时回退到 cooked 读行。
func confirmKey(t *Terminal, prompt string) bool {
	if err := t.MakeRaw(); err != nil {
		ans, _ := t.ReadLine("  " + prompt) // 回退：cooked 读行
		return ans == "y" || ans == "Y"
	}
	t.Write("  " + prompt)
	k := t.ReadKey()
	t.Restore()
	t.Write("\n")
	return k.Rune == 'y' || k.Rune == 'Y'
}

// removeProvider 从 store.Providers 删除指针 p 指向的元素。
func removeProvider(store *config.Store, p *config.Provider) {
	for i := range store.Providers {
		if &store.Providers[i] == p {
			store.Providers = append(store.Providers[:i], store.Providers[i+1:]...)
			return
		}
	}
}
