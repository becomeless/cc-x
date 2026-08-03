// models.go: 拉取供应商模型列表（编辑表单「获取模型列表」用）。
//
// 端点缺省推导 {base}/v1/models（Anthropic 风格），供应商可用 presets.json 的 models_api 覆盖
// （MiMo 的 models 端点是 OpenAI 风格 /v1/models，不带 /anthropic 前缀）。
// 响应兼容 Anthropic 风格（data[].id + display_name）与 OpenAI 风格（data[].id），
// 并防御「HTTP 200 但 body 是业务错误」的情况（GLM 无 key 时返回 {code,msg}）。
package presets

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ModelInfo 是模型列表里的一项。
type ModelInfo struct {
	ID          string
	DisplayName string // 可能为空（OpenAI 风格响应没有）
}

// Supports1M 判断模型是否命中供应商的 1M 支持表（前缀匹配，先剥掉已有的 [1m] 后缀，大小写不敏感——
// API 返回的模型 ID 大小写可能与表不一致，如 glm-5.2 vs GLM-5.2）。
// 用于「获取模型列表」时给支持 1M 的模型默认推荐 [1m] 后缀。
func Supports1M(p *Preset, modelID string) bool {
	if p == nil {
		return false
	}
	id := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(modelID), "[1m]"))
	for _, item := range p.Models1M {
		if strings.HasPrefix(id, strings.ToLower(item)) {
			return true
		}
	}
	return false
}

// CanAttach1M 档位是否允许自动附加 [1m] 后缀：Claude Code 支持通过 [1m] 标记扩展上下文，
// 并在当前运行时中识别 opus/sonnet/fable；第三方映射到 fable 档的模型需要该标记声明 1M 能力
// （无标记按 200K 管理）。haiku 没有对应的文档和运行时依据，不自动附加。
// 是否真正附加仍由供应商 models_1m 支持表决定。与 TS 版对齐。
func CanAttach1M(slot string) bool {
	return slot == "opus" || slot == "sonnet" || slot == "fable"
}

// ApplyModelSelection 从模型列表选中一个模型后的落值：允许附加且命中 1M 表时加 [1m] 后缀
// （已带后缀则幂等不重复）。纯函数，便于矩阵单测（对齐 TS 版 applyModelSelection）。
func ApplyModelSelection(slot, modelID string, supports1M bool) string {
	if !CanAttach1M(slot) || !supports1M {
		return modelID
	}
	return strings.TrimSuffix(modelID, "[1m]") + "[1m]"
}

// modelsTimeout 是拉取模型列表的请求超时。
const modelsTimeout = 10 * time.Second

// maxModelsBody 模型列表响应上限（对齐 TS 版；防御异常/恶意端点的大响应）。
const maxModelsBody = 1 << 20

// errRedirectRefused 模型列表端点发生重定向时 CheckRedirect 返回的错误（拒绝跟随）。
// 模型列表端点不应重定向；跨主机重定向会泄露认证头——Go 的敏感头剥离列表（Authorization 等）
// 不含 x-api-key，跨域跟随会把 x-api-key 原样转发到第三方主机。
var errRedirectRefused = errors.New("模型列表端点发生重定向，已拒绝")

// FetchModels 拉取模型列表。baseURL 是 Anthropic 兼容端点，token 是认证凭据（Bearer token 或 API key），
// auth 决定携带哪个认证头（AuthToken → Authorization: Bearer；AuthAPIKey → x-api-key）——
// 与编辑表单的 auth 选择一致，也即 Claude Code 实际调用时使用的头，不再双头齐发。
// endpoint 为空时推导 {baseURL}/v1/models。3xx 重定向一律拒绝。返回失败时 error 带上可读原因。
func FetchModels(baseURL, token, auth, endpoint string) ([]ModelInfo, error) {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/v1/models"
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("请求地址无效: %w", err)
	}
	req.Header.Set("User-Agent", "ccx")
	req.Header.Set("anthropic-version", "2023-06-01")
	if auth == AuthAPIKey {
		req.Header.Set("x-api-key", token)
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{
		Timeout: modelsTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("%w：%s", errRedirectRefused, req.URL)
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, errRedirectRefused) {
			return nil, err
		}
		return nil, fmt.Errorf("网络错误: %w", err)
	}
	defer resp.Body.Close()
	// 多读 1 字节检测是否超限：chunked 响应没有 Content-Length，必须按实际字节数判断。
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxModelsBody+1))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if len(body) > maxModelsBody {
		return nil, errors.New("响应超过 1 MiB，已拒绝")
	}
	// 300-399 一律拒绝：跟随类 3xx 已被 CheckRedirect 拦截，这里兜底非跟随类（300/304 等），
	// 与 TS 版 redirect:'manual' 显式检查所有 3xx 的行为一致。
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s（%s）", resp.StatusCode, extractErrMsg(body), endpoint)
	}
	return parseModels(body)
}

// parseModels 解析 models 响应，兼容 Anthropic / OpenAI 两种 data 格式。
func parseModels(body []byte) ([]ModelInfo, error) {
	var raw struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("响应不是 JSON: %v", err)
	}
	if raw.Data == nil {
		// 200 但无 data 数组：业务错误体（如 GLM 的 {code,msg}）。
		return nil, fmt.Errorf("%s", extractErrMsg(body))
	}
	out := make([]ModelInfo, 0, len(raw.Data))
	seen := make(map[string]bool, len(raw.Data))
	for _, d := range raw.Data {
		id := strings.TrimSpace(d.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, ModelInfo{ID: id, DisplayName: strings.TrimSpace(d.DisplayName)})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("模型列表为空")
	}
	return out, nil
}

// extractErrMsg 尽力从错误响应体提取可读信息（error.message / msg / message / detail）。
func extractErrMsg(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "空响应"
	}
	var obj map[string]any
	if json.Unmarshal(body, &obj) != nil {
		// 非 JSON（HTML 网关页等）：截断展示
		if len(trimmed) > 120 {
			trimmed = trimmed[:120]
		}
		return trimmed
	}
	for _, k := range []string{"msg", "message", "error", "detail"} {
		if v, ok := obj[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		// error 可能是对象 {"message": "..."}
		if m, ok := obj[k].(map[string]any); ok {
			if s, ok := m["message"].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return trimmed
}
