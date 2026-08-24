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

// Models 是四档模型映射（可部分为空）。
type Models struct {
	Opus   string `json:"opus,omitempty"`
	Sonnet string `json:"sonnet,omitempty"`
	Haiku  string `json:"haiku,omitempty"`
	Fable  string `json:"fable,omitempty"`
}

// Preset 是一个「供应商」目录条目。
type Preset struct {
	Name         string            `json:"name"`
	Auth         string            `json:"auth"` // AUTH_TOKEN | API_KEY
	URLs         []URL             `json:"urls"`
	Models       Models            `json:"models"`
	Effort       string            `json:"effort,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	ModelsAPI    string            `json:"models_api,omitempty"`     // 模型列表端点；缺省推导 {base}/v1/models（MiMo 路径不同需显式）
	ModelsAPIMap map[string]string `json:"models_api_map,omitempty"` // 按 BASE_URL 前缀匹配的模型列表端点（多地址供应商：按量/TokenPlan 各自端点）
	Models1M     []string          `json:"models_1m,omitempty"`      // 支持 1M 上下文的模型 ID 前缀表（[1m] 后缀默认开关）
}

// BuiltinPresets 是内置兜底目录，镜像仓库根 presets.json（由 presets_test.go 对拍保证不漂）。
// 第三方供应商绝不预置 `[1m]`（见 plan §3.1.1）。
var BuiltinPresets = []Preset{
	{
		Name:      "DeepSeek",
		Auth:      AuthToken,
		Effort:    "max",
		URLs:      []URL{{Label: "Anthropic 兼容", URL: "https://api.deepseek.com/anthropic"}},
		Models:    Models{Opus: "deepseek-v4-pro", Sonnet: "deepseek-v4-pro", Haiku: "deepseek-v4-flash"},
		ModelsAPI: "https://api.deepseek.com/models", // Anthropic 兼容端点未实现 /v1/models，模型列表在 OpenAI 风格端点
		Models1M:  []string{"deepseek-v4-pro"},
	},
	{
		Name:   "智谱GLM",
		Auth:   AuthToken,
		URLs:   []URL{{Label: "Anthropic 兼容", URL: "https://open.bigmodel.cn/api/anthropic"}},
		Models: Models{Opus: "GLM-4.7", Sonnet: "GLM-4.7", Haiku: "glm-4.5-air"},
		Env: map[string]string{
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
		},
		ModelsAPI: "https://open.bigmodel.cn/api/anthropic/v1/models",
		Models1M:  []string{"glm-5.2"},
	},
	{
		Name: "小米MiMo",
		Auth: AuthToken,
		URLs: []URL{
			{Label: "按量付费API", URL: "https://api.xiaomimimo.com/anthropic"},
			{Label: "TokenPlan", URL: "https://token-plan-cn.xiaomimimo.com/anthropic"},
		},
		Models: Models{Opus: "mimo-v2.5-pro", Sonnet: "mimo-v2.5-pro", Haiku: "mimo-v2.5-pro"},
		// MiMo 的 models 端点是 OpenAI 风格 /v1/models（不带 /anthropic 前缀），按 base 前缀匹配各自端点
		ModelsAPIMap: map[string]string{
			"https://api.xiaomimimo.com/anthropic":           "https://api.xiaomimimo.com/v1/models",
			"https://token-plan-cn.xiaomimimo.com/anthropic": "https://token-plan-cn.xiaomimimo.com/v1/models",
		},
		Models1M: []string{"mimo-v2.5-pro"},
	},
	{
		Name:     "Kimi (月之暗面)",
		Auth:     AuthToken,
		Effort:   "max",
		URLs:     []URL{{Label: "Anthropic 兼容", URL: "https://api.moonshot.cn/anthropic"}},
		Models:   Models{Opus: "kimi-k3", Sonnet: "kimi-k3", Haiku: "kimi-k3", Fable: "kimi-k3"},
		Models1M: []string{"kimi-k3"},
	},
	{
		Name: "MiniMax",
		Auth: AuthToken,
		URLs: []URL{
			{Label: "国内站", URL: "https://api.minimaxi.com/anthropic"},
			{Label: "国际站", URL: "https://api.minimax.io/anthropic"},
		},
		Models:   Models{Opus: "MiniMax-M3", Sonnet: "MiniMax-M3", Haiku: "MiniMax-M3", Fable: "MiniMax-M3"},
		Models1M: []string{"MiniMax-M3"},
	},
	{
		Name:   "OpenRouter",
		Auth:   AuthAPIKey,
		URLs:   []URL{{Label: "Anthropic 兼容", URL: "https://openrouter.ai/api"}},
		Models: Models{Opus: "anthropic/claude-opus-5", Sonnet: "anthropic/claude-sonnet-5", Haiku: "anthropic/claude-haiku-4-5", Fable: "anthropic/claude-fable-5"},
	},
	{
		Name:   "百度千帆",
		Auth:   AuthToken,
		URLs:   []URL{{Label: "Coding Plan", URL: "https://qianfan.baidubce.com/anthropic/coding"}},
		Models: Models{Opus: "qianfan-code-latest", Sonnet: "qianfan-code-latest", Haiku: "qianfan-code-latest", Fable: "qianfan-code-latest"},
		Env: map[string]string{
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
		},
	},
	{
		Name: "阿里百炼",
		Auth: AuthToken,
		URLs: []URL{
			{Label: "Token Plan（个人/团队）", URL: "https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic"},
			{Label: "按量付费", URL: "https://dashscope.aliyuncs.com/apps/anthropic"},
		},
		Models:   Models{Opus: "qwen3.8-max", Sonnet: "qwen3.8-max", Haiku: "qwen3.6-flash", Fable: "qwen3.8-max"},
		Models1M: []string{"qwen3.8-max", "qwen3.7-max", "qwen3.7-plus"},
	},
	{
		Name:   "火山方舟",
		Auth:   AuthToken,
		URLs:   []URL{{Label: "Coding Plan", URL: "https://ark.cn-beijing.volces.com/api/coding"}},
		Models: Models{Opus: "doubao-seed-2.0-code", Sonnet: "doubao-seed-2.0-code", Haiku: "doubao-seed-2.0-code", Fable: "doubao-seed-2.0-code"},
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
	return Models{Opus: asString(m["opus"]), Sonnet: asString(m["sonnet"]), Haiku: asString(m["haiku"]), Fable: asString(m["fable"])}
}

func normalizeEnv(raw any) map[string]string {
	m, _ := raw.(map[string]any)
	allowed := []string{"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", "CLAUDE_CODE_SUBAGENT_MODEL"}
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
	p.ModelsAPI = strings.TrimSpace(asString(m["models_api"]))
	if mm, ok := m["models_api_map"].(map[string]any); ok && len(mm) > 0 {
		p.ModelsAPIMap = map[string]string{}
		for k, v := range mm {
			if s := strings.TrimSpace(asString(v)); s != "" {
				p.ModelsAPIMap[strings.TrimSpace(k)] = s
			}
		}
		if len(p.ModelsAPIMap) == 0 {
			p.ModelsAPIMap = nil
		}
	}
	if arr, ok := m["models_1m"].([]any); ok {
		for _, item := range arr {
			if s := strings.TrimSpace(asString(item)); s != "" {
				p.Models1M = append(p.Models1M, s)
			}
		}
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
