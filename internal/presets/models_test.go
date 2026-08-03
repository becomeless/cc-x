package presets

import (
	"net/http"
	"net/http/httptest"
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
		{"GLM-5.2", true},     // 大小写不敏感（API 可能返回大写）
		{"GLM-5.2[1m]", true}, // 大写 + 后缀
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

// TestCanAttach1M：opus/sonnet/fable 档允许自动附加 [1m]，haiku 不允许。
func TestCanAttach1M(t *testing.T) {
	cases := map[string]bool{
		"opus":   true,
		"sonnet": true,
		"fable":  true,
		"haiku":  false,
	}
	for slot, want := range cases {
		if got := CanAttach1M(slot); got != want {
			t.Errorf("CanAttach1M(%q) = %v, want %v", slot, got, want)
		}
	}
}

// TestApplyModelSelection：4 档 × 支持/不支持 1M + 已带后缀幂等。
func TestApplyModelSelection(t *testing.T) {
	cases := []struct {
		name       string
		slot       string
		modelID    string
		supports1M bool
		want       string
	}{
		{"opus 命中", "opus", "glm-5.2", true, "glm-5.2[1m]"},
		{"opus 未命中", "opus", "glm-5.2-air", false, "glm-5.2-air"},
		{"sonnet 命中", "sonnet", "deepseek-v4-pro", true, "deepseek-v4-pro[1m]"},
		{"sonnet 未命中", "sonnet", "deepseek-v4-flash", false, "deepseek-v4-flash"},
		{"haiku 命中也不附加", "haiku", "glm-5.2", true, "glm-5.2"},
		{"fable 命中附加", "fable", "glm-5.2", true, "glm-5.2[1m]"},
		{"fable 未命中不附加", "fable", "glm-5.2-air", false, "glm-5.2-air"},
		{"已带后缀幂等", "opus", "glm-5.2[1m]", true, "glm-5.2[1m]"},
	}
	for _, c := range cases {
		if got := ApplyModelSelection(c.slot, c.modelID, c.supports1M); got != c.want {
			t.Errorf("%s: ApplyModelSelection(%q, %q, %v) = %q, want %q", c.name, c.slot, c.modelID, c.supports1M, got, c.want)
		}
	}
}

// TestFetchModelsAuthHeaderAndRedirect：只发配置对应的一种认证头（带 anthropic-version），
// 3xx 重定向一律拒绝。
func TestFetchModelsAuthHeaderAndRedirect(t *testing.T) {
	t.Run("AUTH_TOKEN 只发 Authorization", func(t *testing.T) {
		var gotAuth, gotKey, gotVer string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotKey = r.Header.Get("x-api-key")
			gotVer = r.Header.Get("anthropic-version")
			w.Write([]byte(`{"data":[{"id":"m"}]}`))
		}))
		defer srv.Close()
		if _, err := FetchModels(srv.URL, "t", AuthToken, ""); err != nil {
			t.Fatalf("FetchModels: %v", err)
		}
		if gotAuth != "Bearer t" || gotKey != "" {
			t.Fatalf("AUTH_TOKEN 应只发 Authorization: Bearer t，实际 Authorization=%q x-api-key=%q", gotAuth, gotKey)
		}
		if gotVer == "" {
			t.Fatal("应携带 anthropic-version 头")
		}
	})

	t.Run("API_KEY 只发 x-api-key", func(t *testing.T) {
		var gotAuth, gotKey string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotKey = r.Header.Get("x-api-key")
			w.Write([]byte(`{"data":[{"id":"m"}]}`))
		}))
		defer srv.Close()
		if _, err := FetchModels(srv.URL, "k", AuthAPIKey, ""); err != nil {
			t.Fatalf("FetchModels: %v", err)
		}
		if gotKey != "k" || gotAuth != "" {
			t.Fatalf("API_KEY 应只发 x-api-key: k，实际 x-api-key=%q Authorization=%q", gotKey, gotAuth)
		}
	})

	t.Run("302 重定向拒绝", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://example.com/other", http.StatusFound)
		}))
		defer srv.Close()
		_, err := FetchModels(srv.URL, "k", AuthAPIKey, "")
		if err == nil {
			t.Fatal("重定向应报错")
		}
		if !strings.Contains(err.Error(), "重定向") || !strings.Contains(err.Error(), "example.com") {
			t.Fatalf("错误应含重定向与目标地址: %v", err)
		}
	})

	t.Run("响应超过 1 MiB 拒绝", func(t *testing.T) {
		big := make([]byte, 2<<20)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(big)
		}))
		defer srv.Close()
		_, err := FetchModels(srv.URL, "k", AuthAPIKey, "")
		if err == nil {
			t.Fatal("大响应应报错")
		}
		if !strings.Contains(err.Error(), "超过 1 MiB") {
			t.Fatalf("错误应含超限信息: %v", err)
		}
	})
}
