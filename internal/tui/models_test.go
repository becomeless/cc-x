package tui

import (
	"testing"

	"github.com/becomeless/cc-x/internal/presets"
)

// testCatalog 构造一个与内置预设结构一致的目录（只含测试需要的字段）。
func testCatalog() []presets.Preset {
	return []presets.Preset{
		{
			Name:      "DeepSeek",
			URLs:      []presets.URL{{Label: "Anthropic 兼容", URL: "https://api.deepseek.com/anthropic"}},
			ModelsAPI: "https://api.deepseek.com/models",
			Models1M:  []string{"deepseek-v4-pro"},
		},
		{
			Name: "小米MiMo",
			URLs: []presets.URL{
				{Label: "按量付费API", URL: "https://api.xiaomimimo.com/anthropic"},
				{Label: "TokenPlan", URL: "https://token-plan-cn.xiaomimimo.com/anthropic"},
			},
			ModelsAPIMap: map[string]string{
				"https://api.xiaomimimo.com/anthropic":           "https://api.xiaomimimo.com/v1/models",
				"https://token-plan-cn.xiaomimimo.com/anthropic": "https://token-plan-cn.xiaomimimo.com/v1/models",
			},
		},
		{
			Name: "智谱GLM",
			URLs: []presets.URL{{Label: "Anthropic 兼容", URL: "https://open.bigmodel.cn/api/anthropic"}},
		},
		{
			Name: "阿里百炼",
			URLs: []presets.URL{
				{Label: "Token Plan（个人/团队）", URL: "https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic"},
				{Label: "按量付费", URL: "https://dashscope.aliyuncs.com/apps/anthropic"},
			},
			ModelsAPIMap: map[string]string{
				"https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic": "https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/models",
				"https://dashscope.aliyuncs.com/apps/anthropic":                  "https://dashscope.aliyuncs.com/compatible-mode/v1/models",
			},
		},
	}
}

// TestFindPresetByBase：按 BASE_URL 前缀命中预设（「DeepSeek 2」等改名配置 base 相同也命中）。
func TestFindPresetByBase(t *testing.T) {
	c := testCatalog()
	cases := []struct {
		base string
		want string // 期望命中预设名；空=不应命中
	}{
		{"https://api.deepseek.com/anthropic", "DeepSeek"},
		{"https://api.deepseek.com/anthropic/", "DeepSeek"}, // 尾斜杠
		{"https://api.xiaomimimo.com/anthropic", "小米MiMo"},
		{"https://token-plan-cn.xiaomimimo.com/anthropic", "小米MiMo"}, // map key 命中
		{"https://open.bigmodel.cn/api/anthropic", "智谱GLM"},
		{"https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic", "阿里百炼"},
		{"https://dashscope.aliyuncs.com/apps/anthropic", "阿里百炼"},
		{"https://custom.example.com/anthropic", ""}, // 自定义供应商
		{"", ""},
	}
	for _, cse := range cases {
		got := findPresetByBase(c, cse.base)
		if cse.want == "" {
			if got != nil {
				t.Errorf("findPresetByBase(%q) 应不命中，got %s", cse.base, got.Name)
			}
			continue
		}
		if got == nil || got.Name != cse.want {
			t.Errorf("findPresetByBase(%q) = %v, want %s", cse.base, got, cse.want)
		}
	}
}

// TestCatalogPreset：base URL 优先（改名配置），名字兜底，都匹配不到返回 nil。
func TestCatalogPreset(t *testing.T) {
	c := testCatalog()
	// base 命中：DeepSeek 2 配置（名字不同、base 相同）也能拿到 DeepSeek 预设
	w := &workCopy{name: "DeepSeek 2", base: "https://api.deepseek.com/anthropic"}
	if pp := catalogPreset(c, w); pp == nil || pp.Name != "DeepSeek" {
		t.Fatalf("base 命中失败: %+v", pp)
	}
	// base 不匹配时按名字兜底
	w = &workCopy{name: "智谱GLM", base: "https://custom.example.com/x"}
	if pp := catalogPreset(c, w); pp == nil || pp.Name != "智谱GLM" {
		t.Fatalf("名字兜底失败: %+v", pp)
	}
	// 都匹配不到
	w = &workCopy{name: "自建", base: "https://custom.example.com/x"}
	if pp := catalogPreset(c, w); pp != nil {
		t.Fatalf("应返回 nil, got %+v", pp)
	}
}

// TestResolveModelsEndpoint：map 按 base 前缀优先 > models_api > 空（推导）。
func TestResolveModelsEndpoint(t *testing.T) {
	c := testCatalog()
	base := "https://api.xiaomimimo.com/anthropic"
	if ep := resolveModelsEndpoint(findPresetByBase(c, base), base); ep != "https://api.xiaomimimo.com/v1/models" {
		t.Errorf("MiMo 按量端点: %q", ep)
	}
	base = "https://token-plan-cn.xiaomimimo.com/anthropic"
	if ep := resolveModelsEndpoint(findPresetByBase(c, base), base); ep != "https://token-plan-cn.xiaomimimo.com/v1/models" {
		t.Errorf("MiMo TokenPlan 端点: %q", ep)
	}
	base = "https://api.deepseek.com/anthropic"
	if ep := resolveModelsEndpoint(findPresetByBase(c, base), base); ep != "https://api.deepseek.com/models" {
		t.Errorf("DeepSeek 端点: %q", ep)
	}
	base = "https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic"
	if ep := resolveModelsEndpoint(findPresetByBase(c, base), base); ep != "https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/models" {
		t.Errorf("阿里百炼 Token Plan 端点: %q", ep)
	}
	base = "https://dashscope.aliyuncs.com/apps/anthropic"
	if ep := resolveModelsEndpoint(findPresetByBase(c, base), base); ep != "https://dashscope.aliyuncs.com/compatible-mode/v1/models" {
		t.Errorf("阿里百炼 按量端点: %q", ep)
	}
	base = "https://open.bigmodel.cn/api/anthropic"
	if ep := resolveModelsEndpoint(findPresetByBase(c, base), base); ep != "" {
		t.Errorf("GLM 应走推导（空）: %q", ep)
	}
	if ep := resolveModelsEndpoint(nil, "https://x"); ep != "" {
		t.Errorf("nil preset 应返回空: %q", ep)
	}
}
