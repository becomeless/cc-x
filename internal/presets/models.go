// models.go: 拉取供应商模型列表（编辑表单「获取模型列表」用）。
//
// 端点缺省推导 {base}/v1/models（Anthropic 风格），供应商可用 presets.json 的 models_api 覆盖
// （MiMo 的 models 端点是 OpenAI 风格 /v1/models，不带 /anthropic 前缀）。
// 响应兼容 Anthropic 风格（data[].id + display_name）与 OpenAI 风格（data[].id），
// 并防御「HTTP 200 但 body 是业务错误」的情况（GLM 无 key 时返回 {code,msg}）。
package presets

import (
	"encoding/json"
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

// modelsTimeout 是拉取模型列表的请求超时。
const modelsTimeout = 10 * time.Second

// FetchModels 拉取模型列表。baseURL 是 Anthropic 兼容端点，apiKey 是认证令牌。
// endpoint 为空时推导 {baseURL}/v1/models。返回失败时 error 带上可读原因（含响应里的 msg 字段）。
func FetchModels(baseURL, apiKey, endpoint string) ([]ModelInfo, error) {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/v1/models"
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("请求地址无效: %w", err)
	}
	// 双认证头都带上：Anthropic 官方认 x-api-key，国产中转一般认 Authorization: Bearer。
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("User-Agent", "ccx")

	client := &http.Client{Timeout: modelsTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络错误: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode >= 400 {
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
