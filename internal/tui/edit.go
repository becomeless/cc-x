package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/becomeless/cc-x/internal/config"
	"github.com/becomeless/cc-x/internal/display"
	"github.com/becomeless/cc-x/internal/i18n"
	"github.com/becomeless/cc-x/internal/presets"
)

// 模型档位（与 TS 版 ModelSlot 对齐）：决定「从模型列表选择」是否自动附加 [1m] 后缀。
const (
	slotOpus   = "opus"
	slotSonnet = "sonnet"
	slotHaiku  = "haiku"
	slotFable  = "fable"
)

type workCopy struct {
	name, note, base, auth, token, opus, sonnet, haiku, fable, subagent, effort string
	disableTraffic                                                              string
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
		fable:          m["ANTHROPIC_DEFAULT_FABLE_MODEL"],
		subagent:       m["CLAUDE_CODE_SUBAGENT_MODEL"],
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

// subagentLabel 子代理行为空时显示「默认」（不强制覆盖；未另行指定时继承主模型），不显示 (空)。
func subagentLabel(v string) string {
	if strings.TrimSpace(v) == "" {
		return i18n.T("edit.default")
	}
	return v
}

// findPreset 按显示名在目录里找供应商；自定义名找不到返回 nil（仍可手敲）。
func findPreset(catalog []presets.Preset, name string) *presets.Preset {
	for i := range catalog {
		if catalog[i].Name == name {
			return &catalog[i]
		}
	}
	return nil
}

// findPresetByBase 按 BASE_URL 前缀在目录里找供应商。
// 配置名与预设名不同（如「DeepSeek 2」）也能命中：端点/1M 表是供应商的属性，跟着 base 走才对。
func findPresetByBase(catalog []presets.Preset, baseURL string) *presets.Preset {
	for i := range catalog {
		for _, u := range catalog[i].URLs {
			if u.URL != "" && strings.HasPrefix(baseURL, u.URL) {
				return &catalog[i]
			}
		}
		for k := range catalog[i].ModelsAPIMap {
			if strings.HasPrefix(baseURL, k) {
				return &catalog[i]
			}
		}
	}
	return nil
}

// resolveModelsEndpoint 解析模型列表端点：models_api_map 按 base 前缀匹配 > models_api > 空（推导）。
func resolveModelsEndpoint(pp *presets.Preset, baseURL string) string {
	if pp == nil {
		return ""
	}
	if len(pp.ModelsAPIMap) > 0 {
		for base, ep := range pp.ModelsAPIMap {
			if strings.HasPrefix(baseURL, base) {
				return ep
			}
		}
	}
	return pp.ModelsAPI
}

// catalogPreset 定位当前表单的供应商预设：先按 base URL（同名不同配置也能命中），再按配置名兜底。
func catalogPreset(catalog []presets.Preset, w *workCopy) *presets.Preset {
	if pp := findPresetByBase(catalog, w.base); pp != nil {
		return pp
	}
	return findPreset(catalog, w.name)
}

// buildModelItems 构建模型列表菜单条目与 1M 标记（opus/sonnet/fable 档命中供应商 1M 表才标 [1M]；haiku 不标）。
// 有 display_name 且与 ID 不同（如 GLM 返回 "GLM-5.2" + id "glm-5.2"）时显示「友好名 (实际ID)」——
// 主标签可读，括号里是选中后真正填入的值。
func buildModelItems(models []presets.ModelInfo, pp *presets.Preset, slot string) (items []string, is1M []bool) {
	items = make([]string, 0, len(models))
	is1M = make([]bool, len(models))
	attach := presets.CanAttach1M(slot)
	for i, m := range models {
		label := m.ID
		if m.DisplayName != "" && m.DisplayName != m.ID {
			label = m.DisplayName + " (" + m.ID + ")"
		}
		if attach && presets.Supports1M(pp, m.ID) {
			label += "  [1M]"
			is1M[i] = true
		}
		items = append(items, label)
	}
	return items, is1M
}

// pickFromList 从已拉取的模型列表选一个；opus/sonnet/fable 档命中 1M 表自动附加 [1m] 后缀
// （想用 200K 可在表单行手动删）；haiku 档不附加。取消返回 false。
func pickFromList(t *Terminal, models []presets.ModelInfo, items []string, is1M []bool, slot, title, hint string) (string, bool) {
	sel := SelectMenu(t, SelectOptions{Title: title, Items: items, Start: 0, Hint: hint, NoNumber: true})
	if sel < 0 {
		return "", false
	}
	return presets.ApplyModelSelection(slot, models[sel].ID, is1M[sel]), true
}

// fetchAndPickModel 从供应商 API 拉取模型列表并让用户选一个。
// 返回选中模型 ID；用户取消返回 ("", nil)；失败返回 ("", err)（由表单 Status 展示）。
func fetchAndPickModel(t *Terminal, w *workCopy, catalog []presets.Preset, slot, title string) (string, error) {
	if strings.TrimSpace(w.base) == "" || strings.TrimSpace(w.token) == "" {
		return "", errors.New(i18n.T("models.needBaseKey"))
	}
	pp := catalogPreset(catalog, w)
	var endpoint string
	if pp != nil {
		endpoint = resolveModelsEndpoint(pp, w.base)
	}
	fmt.Printf("  %s\n", i18n.T("models.fetching"))
	models, err := presets.FetchModels(w.base, w.token, w.auth, endpoint)
	if err != nil {
		return "", err
	}
	items, is1M := buildModelItems(models, pp, slot)
	hint := i18n.T("models.hint")
	if !presets.CanAttach1M(slot) {
		hint = i18n.T("models.hintNoSuffix")
	}
	model, ok := pickFromList(t, models, items, is1M, slot, title, hint)
	if !ok {
		return "", nil
	}
	return model, nil
}

// pickSlotModel 档位行的编辑入口：手动输入 / 从模型列表选择。
// 返回是否变更与新值；失败返回 err（由表单 Status 展示，错误不再被清屏吞掉）。
func pickSlotModel(t *Terminal, w *workCopy, catalog []presets.Preset, slot, label, current string) (bool, string, error) {
	cur := current
	if cur == "" {
		cur = i18n.T("empty.paren")
	}
	opts := []string{i18n.T("models.pickManual"), i18n.T("models.pickFromList")}
	sel := SelectMenu(t, SelectOptions{Title: fmt.Sprintf("%s（当前：%s）", label, cur), Items: opts, Start: 0, Hint: i18n.T("pick.hint"), NoNumber: true})
	if sel < 0 {
		return false, current, nil
	}
	if sel == 0 {
		ch, val := ReadValue(t, strings.TrimSpace(label), current, false)
		return ch, val, nil
	}
	model, err := fetchAndPickModel(t, w, catalog, slot, i18n.T("models.pickTitle"))
	if err != nil {
		return false, current, err
	}
	if model == "" {
		return false, current, nil // 用户取消
	}
	return true, model, nil
}

// EditForm 编辑 prov（就地修改）；保存返回 true，放弃返回 false。对应 npm 版 editForm。
// 密钥行默认掩码，「👁 显示/隐藏」仅切换本表单显示、不改数据、不持久化。
// focusKey=true 时初始光标落在密钥行（#9：无 key 配置 Enter 直达填密钥的最短路径）。
func EditForm(t *Terminal, prov *config.Provider, store *config.Store, catalog []presets.Preset, focusKey bool) bool {
	w := fromProvider(*prov)
	showSecret := false
	status := "" // 表单顶部绿色提示条：最近一次模型操作的结果/错误（SelectMenu 清屏不会吞掉它）
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
		// 12 个字段行（顺序即行序，密钥行索引为 4）。标签按显示宽度统一补齐后再拼冒号——
		// 中英混排 + 全角/半角字符时手写尾部空格数不可靠（→ 宽度还随终端字体变化），对齐在渲染时计算。
		fieldRows := []struct{ action, label, value string }{
			{"provider", i18n.T("edit.field.provider"), v(w.name)},
			{"note", i18n.T("edit.field.note"), v(w.note)},
			{"base", i18n.T("edit.field.base"), v(w.base)},
			{"auth", i18n.T("edit.field.auth"), w.auth},
			{"key", i18n.T("edit.field.key"), keyDisp},
			{"fable", i18n.T("edit.field.fable"), v(w.fable)},
			{"opus", i18n.T("edit.field.opus"), v(w.opus)},
			{"sonnet", i18n.T("edit.field.sonnet"), v(w.sonnet)},
			{"haiku", i18n.T("edit.field.haiku"), v(w.haiku)},
			{"subagent", i18n.T("edit.field.subagent"), subagentLabel(w.subagent)},
			{"effort", i18n.T("edit.field.effort"), v(w.effort)},
			{"disableTraffic", i18n.T("edit.field.disableTraffic"), v(w.disableTraffic)},
		}
		labelW := 0
		for _, f := range fieldRows {
			if w := display.Width(f.label); w > labelW {
				labelW = w
			}
		}
		type row struct{ action, label string }
		rows := make([]row, 0, len(fieldRows)+5)
		for _, f := range fieldRows {
			rows = append(rows, row{f.action, display.Pad(f.label, labelW) + ": " + f.value})
		}
		rows = append(rows,
			row{"sep", ""},
			row{"toggle", toggleLabel(showSecret)},
			row{"sep", ""},
			row{"save", i18n.T("edit.save")},
			row{"discard", i18n.T("edit.discard")},
		)
		items := make([]string, len(rows))
		for i, r := range rows {
			items[i] = r.label
		}

		sel := SelectMenu(t, SelectOptions{Title: i18n.T("edit.title"), Items: items, Start: start, Hint: i18n.T("edit.hint"), Status: status, NoNumber: true})
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
				if pp.Models.Fable != "" {
					w.fable = pp.Models.Fable
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
			if ch, val, err := pickSlotModel(t, &w, catalog, slotOpus, strings.TrimSpace(i18n.T("edit.field.opus")), w.opus); err != nil {
				status = i18n.T("models.fail", err)
			} else if ch {
				w.opus = val
				status = ""
			}
		case "sonnet":
			if ch, val, err := pickSlotModel(t, &w, catalog, slotSonnet, strings.TrimSpace(i18n.T("edit.field.sonnet")), w.sonnet); err != nil {
				status = i18n.T("models.fail", err)
			} else if ch {
				w.sonnet = val
				status = ""
			}
		case "haiku":
			if ch, val, err := pickSlotModel(t, &w, catalog, slotHaiku, strings.TrimSpace(i18n.T("edit.field.haiku")), w.haiku); err != nil {
				status = i18n.T("models.fail", err)
			} else if ch {
				w.haiku = val
				status = ""
			}
		case "fable":
			if ch, val, err := pickSlotModel(t, &w, catalog, slotFable, strings.TrimSpace(i18n.T("edit.field.fable")), w.fable); err != nil {
				status = i18n.T("models.fail", err)
			} else if ch {
				w.fable = val
				status = ""
			}
		case "subagent":
			if ch, val := ReadValue(t, strings.TrimSpace(i18n.T("edit.field.subagent")), w.subagent, false); ch {
				w.subagent = val
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
				"ANTHROPIC_DEFAULT_FABLE_MODEL":            w.fable,
				"CLAUDE_CODE_SUBAGENT_MODEL":               w.subagent,
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
