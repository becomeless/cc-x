package tui

import (
	"strings"

	"github.com/becomeless/cc-x/internal/config"
	"github.com/becomeless/cc-x/internal/display"
	"github.com/becomeless/cc-x/internal/i18n"
	"github.com/becomeless/cc-x/internal/presets"
)

func orEmpty(s string) string {
	if s == "" {
		return i18n.T("empty.paren")
	}
	return s
}

// PickProvider 选供应商：返回 (选中供应商, custom)。两者皆零值表示不改。
func PickProvider(t *Terminal, catalog []presets.Preset, current string) (sel *presets.Preset, custom bool) {
	names := make([]string, len(catalog))
	for i := range catalog {
		names[i] = catalog[i].Name
	}
	items := append(append([]string{}, names...), i18n.T("pick.provider.custom"), i18n.T("pick.noChange"))
	cur := current
	if cur == "" {
		cur = i18n.T("pick.provider.none")
	}
	s := SelectMenu(t, SelectOptions{Title: i18n.T("pick.provider.title", cur), Items: items, Hint: i18n.T("pick.hint"), NoNumber: true})
	if s < 0 || s == len(items)-1 {
		return nil, false
	}
	if s == len(names) {
		return nil, true
	}
	return &catalog[s], false
}

// PickProviderURL 供应商有多个地址时让用户选一个；只有一个直接用，无地址保持原值。
func PickProviderURL(t *Terminal, preset *presets.Preset, current string) string {
	urls := preset.URLs
	if len(urls) == 0 {
		return current
	}
	if len(urls) == 1 {
		return urls[0].URL
	}
	labels := make([]string, len(urls))
	for i, u := range urls {
		labels[i] = display.Pad(u.Label, 12) + " " + orEmpty(u.URL)
	}
	items := append(labels, i18n.T("pick.noChange"))
	s := SelectMenu(t, SelectOptions{Title: i18n.T("pick.providerUrl.title", preset.Name), Items: items, Hint: i18n.T("pick.hint"), NoNumber: true})
	if s < 0 || s == len(items)-1 {
		return current
	}
	return urls[s].URL
}

// PickBaseURL 选 API 地址：目录所有 url + 已有配置用过的 url + 手动输入 + 不修改。
func PickBaseURL(t *Terminal, current string, store *config.Store, catalog []presets.Preset) string {
	type entry struct{ label, url string }
	var entries []entry
	seen := map[string]bool{}
	for i := range catalog {
		p := catalog[i]
		for _, u := range p.URLs {
			tag := p.Name
			if len(p.URLs) > 1 {
				tag = p.Name + "/" + u.Label
			}
			entries = append(entries, entry{label: display.Pad(tag, 20) + " " + orEmpty(u.URL), url: u.URL})
			seen[u.URL] = true
		}
	}
	for i := range store.Providers {
		u := config.GetProviderEnvMap(store.Providers[i])["ANTHROPIC_BASE_URL"]
		if u != "" && !seen[u] {
			seen[u] = true
			entries = append(entries, entry{label: display.Pad(i18n.T("pick.base.existing", store.Providers[i].Name), 20) + " " + u, url: u})
		}
	}
	items := make([]string, 0, len(entries)+2)
	for _, e := range entries {
		items = append(items, e.label)
	}
	items = append(items, i18n.T("pick.manual"), i18n.T("pick.noChange"))
	s := SelectMenu(t, SelectOptions{Title: i18n.T("pick.base.title", orEmpty(current)), Items: items, Hint: i18n.T("pick.hint"), NoNumber: true})
	if s < 0 || s == len(items)-1 {
		return current
	}
	if s < len(entries) {
		return entries[s].url
	}
	v, ok := ReadText(t, "  "+i18n.T("pick.base.manualInput"))
	if !ok || v == "" {
		return current
	}
	if v == "-" {
		return ""
	}
	return strings.TrimSpace(v)
}

// PickAuth 选认证字段：AUTH_TOKEN / API_KEY / 不改。
func PickAuth(t *Terminal, current string) string {
	items := []string{i18n.T("pick.auth.token"), i18n.T("pick.auth.apikey"), i18n.T("pick.noChange")}
	s := SelectMenu(t, SelectOptions{Title: i18n.T("pick.auth.title", current), Items: items, Hint: i18n.T("pick.hint"), NoNumber: true})
	if s == 0 {
		return presets.AuthToken
	}
	if s == 1 {
		return presets.AuthAPIKey
	}
	return current
}

var effortOpts = []string{"low", "medium", "high", "xhigh", "max", "auto"}

// PickEffort 选 effort 思考档：low…auto / 留空 / 不改。
func PickEffort(t *Terminal, current string) string {
	items := append(append([]string{}, effortOpts...), i18n.T("pick.effort.empty"), i18n.T("pick.noChange"))
	s := SelectMenu(t, SelectOptions{Title: i18n.T("pick.effort.title", orEmpty(current)), Items: items, Hint: i18n.T("pick.effort.hint"), NoNumber: true})
	if s < 0 || s == len(items)-1 {
		return current
	}
	if s == len(effortOpts) {
		return ""
	}
	return effortOpts[s]
}

// PickDisableTraffic 选是否禁用非核心流量：是（"1"）/ 否（""）/ 不改。
func PickDisableTraffic(t *Terminal, current string) string {
	items := []string{i18n.T("pick.disableTraffic.yes"), i18n.T("pick.disableTraffic.no"), i18n.T("pick.noChange")}
	s := SelectMenu(t, SelectOptions{Title: i18n.T("pick.disableTraffic.title", orEmpty(current)), Items: items, Hint: i18n.T("pick.hint"), NoNumber: true})
	if s < 0 || s == 2 {
		return current
	}
	if s == 0 {
		return "1"
	}
	return ""
}
