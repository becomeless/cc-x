package presets

import (
	"reflect"
	"strings"
	"testing"
)

// TestParseModelsAnthropicStyle：Anthropic 风格响应（data[].id + display_name）。
func TestParseModelsAnthropicStyle(t *testing.T) {
	body := `{"data":[{"id":"claude-sonnet-4-5","display_name":"Claude Sonnet 4.5","created_at":"2025-06-20","type":"model"},{"id":"claude-opus-4-8","display_name":"Claude Opus 4.8"}]}`
	got, err := parseModels([]byte(body))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	want := []ModelInfo{{ID: "claude-sonnet-4-5", DisplayName: "Claude Sonnet 4.5"}, {ID: "claude-opus-4-8", DisplayName: "Claude Opus 4.8"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestParseModelsOpenAIStyle：OpenAI 风格响应（data[].id + object，无 display_name）。
func TestParseModelsOpenAIStyle(t *testing.T) {
	body := `{"object":"list","data":[{"id":"mimo-v2.5-pro","object":"model","owned_by":"xiaomi"},{"id":"deepseek-chat","object":"model"}]}`
	got, err := parseModels([]byte(body))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	want := []ModelInfo{{ID: "mimo-v2.5-pro"}, {ID: "deepseek-chat"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestParseModelsDedupAndBlank：重复与空 ID 被过滤。
func TestParseModelsDedupAndBlank(t *testing.T) {
	body := `{"data":[{"id":"a"},{"id":"a"},{"id":""},{"id":"b","display_name":"B"}]}`
	got, err := parseModels([]byte(body))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("去重/过滤失效: %+v", got)
	}
}

// TestParseModelsBusinessError：HTTP 200 但 body 是业务错误（GLM 无 key 时返回 {code,msg}）。
func TestParseModelsBusinessError(t *testing.T) {
	body := `{"code":1001,"msg":"Header中未收到Authorization参数，无法进行身份验证。","success":false}`
	_, err := parseModels([]byte(body))
	if err == nil {
		t.Fatal("期望报错，got nil")
	}
	if !strings.Contains(err.Error(), "Authorization") {
		t.Fatalf("错误信息应包含 msg 内容: %v", err)
	}
}

// TestParseModelsEmptyList：data 数组为空报错（不能给用户一个空菜单）。
func TestParseModelsEmptyList(t *testing.T) {
	_, err := parseModels([]byte(`{"data":[]}`))
	if err == nil {
		t.Fatal("期望报错，got nil")
	}
}

// TestExtractErrMsg：error 对象嵌套 message 的提取。
func TestExtractErrMsg(t *testing.T) {
	got := extractErrMsg([]byte(`{"error":{"message":"invalid api key"}}`))
	if got != "invalid api key" {
		t.Fatalf("got %q", got)
	}
	got = extractErrMsg([]byte(`<html>gateway error page</html>`))
	if !strings.Contains(got, "gateway error") {
		t.Fatalf("非 JSON 应截断展示: %q", got)
	}
}

// TestSupports1M：前缀匹配、剥 [1m] 后缀、nil preset 安全。
func TestSupports1M(t *testing.T) {
	p := &Preset{Models1M: []string{"deepseek-v4-pro", "glm-5.2"}}
	cases := []struct {
		id  string
		got bool
	}{
		{"deepseek-v4-pro", true},
		{"deepseek-v4-pro[1m]", true},
		{"deepseek-v4-flash", false},
		{"glm-5.2", true},
		{"glm-5.2-air", true}, // 前缀匹配
		{"mimo-v2.5-pro", false},
	}
	for _, c := range cases {
		if got := Supports1M(p, c.id); got != c.got {
			t.Errorf("Supports1M(%q) = %v, want %v", c.id, got, c.got)
		}
	}
	if Supports1M(nil, "deepseek-v4-pro") {
		t.Error("nil preset 应返回 false")
	}
}
