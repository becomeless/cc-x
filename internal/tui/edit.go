package tui

import (
	"fmt"
	"strings"

	"github.com/becomeless/cc-x/internal/config"
	"github.com/becomeless/cc-x/internal/i18n"
	"github.com/becomeless/cc-x/internal/presets"
)

type workCopy struct {
	name, note, base, auth, token, opus, sonnet, haiku, effort string
	disableTraffic                                             string
}

func fromProvider(p config.Provider) workCopy {
	m := config.GetProviderEnvMap(p)
	usesAPIKey := strings.TrimSpace(m["ANTHROPIC_API_KEY"]) != ""
	auth := presets.AuthToken
	token := m["ANTHROPIC_AUTH_TOKEN"]
	if usesAPIKey {
		auth = presets.AuthAPIKey
		token = m["ANTHROPIC_API_KEY"]
	}
	return workCopy{
		name: p.Name, note: p.Note,
		base:           m["ANTHROPIC_BASE_URL"],
		auth:           auth,
		token:          token,
		opus:           m["ANTHROPIC_DEFAULT_OPUS_MODEL"],
		sonnet:         m["ANTHROPIC_DEFAULT_SONNET_MODEL"],
		haiku:          m["ANTHROPIC_DEFAULT_HAIKU_MODEL"],
		effort:         m["CLAUDE_CODE_EFFORT_LEVEL"],
		disableTraffic: m["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"],
	}
}

func toggleLabel(show bool) string {
	if show {
		return i18n.T("edit.toggleSecretHide")
	}
	return i18n.T("edit.toggleSecretShow")
}

// EditForm 编辑 prov（就地修改）；保存返回 true，放弃返回 false。对应 npm 版 editForm。
// 密钥行默认掩码，「👁 显示/隐藏」仅切换本表单显示、不改数据、不持久化。
// focusKey=true 时初始光标落在密钥行（#9：无 key 配置 Enter 直达填密钥的最短路径）。
func EditForm(t *Terminal, prov *config.Provider, store *config.Store, catalog []presets.Preset, focusKey bool) bool {
	w := fromProvider(*prov)
	showSecret := false
	// rows 布局固定：provider,note,base,auth,key,…（密钥行索引为 4）。
	const keyRowIndex = 4
	start := 0
	if focusKey {
		start = keyRowIndex
	}

	v := func(x string) string {
		if x == "" {
			return i18n.T("empty.paren")
		}
		return x
	}

	for {
		keyDisp := i18n.T("empty.paren")
		if w.token != "" {
			if showSecret {
				keyDisp = w.token
			} else {
				keyDisp = "********"
			}
		}
		type row struct{ action, label string }
		rows := []row{
			{"provider", i18n.T("edit.field.provider") + ": " + v(w.name)},
			{"note", i18n.T("edit.field.note") + ": " + v(w.note)},
			{"base", i18n.T("edit.field.base") + ": " + v(w.base)},
			{"auth", i18n.T("edit.field.auth") + ": " + w.auth},
			{"key", i18n.T("edit.field.key") + ": " + keyDisp},
			{"opus", i18n.T("edit.field.opus") + ": " + v(w.opus)},
			{"sonnet", i18n.T("edit.field.sonnet") + ": " + v(w.sonnet)},
			{"haiku", i18n.T("edit.field.haiku") + ": " + v(w.haiku)},
			{"effort", i18n.T("edit.field.effort") + ": " + v(w.effort)},
			{"disableTraffic", i18n.T("edit.field.disableTraffic") + ": " + v(w.disableTraffic)},
			{"sep", ""},
			{"toggle", toggleLabel(showSecret)},
			{"sep", ""},
			{"save", i18n.T("edit.save")},
			{"discard", i18n.T("edit.discard")},
		}
		items := make([]string, len(rows))
		for i, r := range rows {
			items[i] = r.label
		}

		sel := SelectMenu(t, SelectOptions{Title: i18n.T("edit.title"), Items: items, Start: start, Hint: i18n.T("edit.hint"), NoNumber: true})
		if sel < 0 {
			return false // Esc / q = 放弃
		}
		start = sel

		switch rows[sel].action {
		case "provider":
			pp, custom := PickProvider(t, catalog, w.name)
			if custom {
				if name, ok := ReadText(t, "  "+i18n.T("edit.customName")); ok && strings.TrimSpace(name) != "" {
					w.name = strings.TrimSpace(name)
				}
			} else if pp != nil {
				w.name = pp.Name
				w.auth = pp.Auth
				w.base = PickProviderURL(t, pp, w.base)
				if pp.Models.Opus != "" {
					w.opus = pp.Models.Opus
				}
				if pp.Models.Sonnet != "" {
					w.sonnet = pp.Models.Sonnet
				}
				if pp.Models.Haiku != "" {
					w.haiku = pp.Models.Haiku
				}
				if pp.Effort != "" {
					w.effort = pp.Effort
				}
				if val, ok := pp.Env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"]; ok {
					w.disableTraffic = val
				}
			}
		case "note":
			if note, ok := ReadText(t, "  "+i18n.T("edit.noteInput")); ok {
				if note == "-" {
					w.note = ""
				} else if strings.TrimSpace(note) != "" {
					w.note = strings.TrimSpace(note)
				}
			}
		case "base":
			w.base = PickBaseURL(t, w.base, store, catalog)
		case "auth":
			w.auth = PickAuth(t, w.auth)
		case "key":
			if ch, val := ReadValue(t, strings.TrimSpace(i18n.T("edit.field.key")), w.token, true); ch {
				w.token = val
			}
		case "opus":
			if ch, val := ReadValue(t, strings.TrimSpace(i18n.T("edit.field.opus")), w.opus, false); ch {
				w.opus = val
			}
		case "sonnet":
			if ch, val := ReadValue(t, strings.TrimSpace(i18n.T("edit.field.sonnet")), w.sonnet, false); ch {
				w.sonnet = val
			}
		case "haiku":
			if ch, val := ReadValue(t, strings.TrimSpace(i18n.T("edit.field.haiku")), w.haiku, false); ch {
				w.haiku = val
			}
		case "effort":
			w.effort = PickEffort(t, w.effort)
		case "disableTraffic":
			w.disableTraffic = PickDisableTraffic(t, w.disableTraffic)
		case "toggle":
			showSecret = !showSecret
		case "save":
			if strings.TrimSpace(w.name) == "" {
				fmt.Printf("  %s\n", i18n.T("edit.nameEmpty"))
				continue
			}
			fields := map[string]string{
				"ANTHROPIC_BASE_URL":                       w.base,
				"ANTHROPIC_DEFAULT_OPUS_MODEL":             w.opus,
				"ANTHROPIC_DEFAULT_SONNET_MODEL":           w.sonnet,
				"ANTHROPIC_DEFAULT_HAIKU_MODEL":            w.haiku,
				"CLAUDE_CODE_EFFORT_LEVEL":                 w.effort,
				"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": w.disableTraffic,
			}
			if w.auth == presets.AuthAPIKey {
				fields["ANTHROPIC_API_KEY"] = w.token
			} else {
				fields["ANTHROPIC_AUTH_TOKEN"] = w.token
			}
			prov.Name = config.ResolveUniqueName(store, strings.TrimSpace(w.name), prov)
			prov.Env = config.BuildProviderEnv(fields)
			prov.Note = w.note
			config.ReconcileBuiltin(prov) // 官方档被配成第三方后清掉 builtin 身份
			return true
		case "discard":
			return false
		}
	}
}
