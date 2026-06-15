// Package presets 是供应商目录（presets）加载层：用户覆盖文件 > 二进制旁路文件 > 内置兜底。
//
// 术语：一个保存的条目叫「配置」(profile，见 internal/config)，这里的目录条目叫「供应商」(provider)。
// 与 internal/config 解耦：presets 可以依赖 config（取存储目录），但 config 不依赖 presets。
//
// 实现说明：计划 §4.2 提到 go:embed，但 Go embed 无法引用包外的 ../../presets.json（npm 发布用的根文件）。
// 为避免在 internal/presets 再放一份同名副本造成漂移，这里改用与 TS 一致的「字面量 BuiltinPresets +
// 对拍测试（presets_test.go 断言它等于根 presets.json）」方案，根目录仍是唯一可编辑源。
package presets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/becomeless/cc-x/internal/config"
)

// 认证字段取值。
const (
	AuthToken  = "AUTH_TOKEN"
	AuthAPIKey = "API_KEY"
)

// URL 是某供应商的一个 API 地址（可多个，多个时让用户选）。
type URL struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// Models 是三档模型映射（可部分为空）。
type Models struct {
	Opus   string `json:"opus,omitempty"`
	Sonnet string `json:"sonnet,omitempty"`
	Haiku  string `json:"haiku,omitempty"`
}

// Preset 是一个「供应商」目录条目。
type Preset struct {
	Name   string            `json:"name"`
	Auth   string            `json:"auth"` // AUTH_TOKEN | API_KEY
	URLs   []URL             `json:"urls"`
	Models Models            `json:"models"`
	Effort string            `json:"effort,omitempty"`
	Env    map[string]string `json:"env,omitempty"`
}

// BuiltinPresets 是内置兜底目录，镜像仓库根 presets.json（由 presets_test.go 对拍保证不漂）。
// 第三方供应商绝不预置 `[1m]`（见 plan §3.1.1）。
var BuiltinPresets = []Preset{
	{
		Name:   "DeepSeek",
		Auth:   AuthToken,
		Effort: "max",
		URLs:   []URL{{Label: "Anthropic 兼容", URL: "https://api.deepseek.com/anthropic"}},
		Models: Models{Opus: "deepseek-v4-pro", Sonnet: "deepseek-v4-pro", Haiku: "deepseek-v4-flash"},
	},
	{
		Name:   "智谱GLM",
		Auth:   AuthToken,
		URLs:   []URL{{Label: "Anthropic 兼容", URL: "https://open.bigmodel.cn/api/anthropic"}},
		Models: Models{Opus: "GLM-4.7", Sonnet: "GLM-4.7", Haiku: "glm-4.5-air"},
		Env: map[string]string{
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
			"CLAUDE_CODE_AUTO_COMPACT_WINDOW":          "1000000",
		},
	},
	{
		Name: "小米MiMo",
		Auth: AuthToken,
		URLs: []URL{
			{Label: "按量付费API", URL: "https://api.xiaomimimo.com/anthropic"},
			{Label: "TokenPlan", URL: "https://token-plan-cn.xiaomimimo.com/anthropic"},
		},
		Models: Models{Opus: "mimo-v2.5-pro", Sonnet: "mimo-v2.5-pro", Haiku: "mimo-v2.5-pro"},
	},
	{
		Name:   "官方Anthropic",
		Auth:   AuthAPIKey,
		URLs:   []URL{{Label: "(留空，用登录态)", URL: ""}},
		Models: Models{},
	},
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func normalizeURL(raw any) URL {
	m, _ := raw.(map[string]any)
	return URL{Label: asString(m["label"]), URL: asString(m["url"])}
}

func normalizeModels(raw any) Models {
	m, _ := raw.(map[string]any)
	return Models{Opus: asString(m["opus"]), Sonnet: asString(m["sonnet"]), Haiku: asString(m["haiku"])}
}

func normalizeEnv(raw any) map[string]string {
	m, _ := raw.(map[string]any)
	allowed := []string{"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", "CLAUDE_CODE_AUTO_COMPACT_WINDOW"}
	out := map[string]string{}
	for _, key := range allowed {
		if v, ok := m[key].(string); ok {
			out[key] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// normalizePreset 宽松规整一条；无名条目返回 ok=false 由调用方丢弃。
func normalizePreset(raw any) (Preset, bool) {
	m, _ := raw.(map[string]any)
	name := strings.TrimSpace(asString(m["name"]))
	if name == "" {
		return Preset{}, false
	}
	auth := AuthToken
	if asString(m["auth"]) == AuthAPIKey {
		auth = AuthAPIKey
	}
	urls := []URL{}
	if arr, ok := m["urls"].([]any); ok {
		for _, u := range arr {
			urls = append(urls, normalizeURL(u))
		}
	}
	p := Preset{Name: name, Auth: auth, URLs: urls, Models: normalizeModels(m["models"]), Env: normalizeEnv(m["env"])}
	if e := strings.TrimSpace(asString(m["effort"])); e != "" {
		p.Effort = e
	}
	return p, true
}

// normalizePresets 把任意解析结果规整为 []Preset；非数组或全空返回 nil（让调用方跌落兜底）。
func normalizePresets(raw any) []Preset {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]Preset, 0, len(arr))
	for _, item := range arr {
		if p, ok := normalizePreset(item); ok {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// tryLoadFile 尝试读并解析一个 presets.json；任何问题都安静返回 nil。
func tryLoadFile(file string) []Preset {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var raw any
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	return normalizePresets(raw)
}

// Load 加载供应商目录。优先级：用户 <storeDir>/presets.json > 二进制旁路 presets.json > 内置 BuiltinPresets。
// 任一步缺失/损坏都安静跌落下一步，绝不中断启动。
func Load(storeDir string) []Preset {
	userFile := filepath.Join(config.ResolveStorePaths(storeDir).Dir, "presets.json")
	if p := tryLoadFile(userFile); p != nil {
		return p
	}
	if exe, err := os.Executable(); err == nil {
		if p := tryLoadFile(filepath.Join(filepath.Dir(exe), "presets.json")); p != nil {
			return p
		}
	}
	return BuiltinPresets
}
