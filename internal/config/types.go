// Package config 是数据层：读写 ~/.cc-mini/providers.json、供应商预设、默认配置与校验。
//
// 铁律：这里只写工具自己的数据文件（providers.json），绝不碰 ~/.claude/*。
// 数据格式必须与 npm/PowerShell 版完全兼容（老用户的 providers.json 能直接读）。
package config

import (
	"bytes"
	"encoding/json"
	"sort"
)

// knownKeys 是受管的 9 个环境变量（工具只动这些，其它一律不碰），顺序即写盘顺序。
//
// 故意私有：这 9 个键是项目铁律的一部分，若导出且可变，任何包都能 append / 改元素，
// 导致 env 清理范围和 JSON 键顺序悄悄漂。对外只经 ManagedKeys() 暴露副本。
var knownKeys = []string{
	"ANTHROPIC_BASE_URL",
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"CLAUDE_CODE_EFFORT_LEVEL",
	"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
	"CLAUDE_CODE_AUTO_COMPACT_WINDOW",
}

// ManagedKeys 返回受管环境变量名的副本（按写盘顺序）。调用方修改返回值不影响内部常量。
func ManagedKeys() []string {
	out := make([]string, len(knownKeys))
	copy(out, knownKeys)
	return out
}

// Lang 是界面语言。
type Lang string

const (
	LangZH Lang = "zh"
	LangEN Lang = "en"
)

// Provider 是一个「配置」（profile）。Name 是唯一键（current / xx <name> / 删除都靠它）。
//
// Builtin 是与显示名解耦的稳定内部标识（评审①）：官方档 = "official"。
// 界面切英文时显示名可译成 "Official"，而代码判断仍认 Builtin、数据主键 Name 不变。
// 老文件没有该字段：仅用 Name=="官方" && env 为空 兜底判定（见 IsOfficial）。
type Provider struct {
	Name    string
	Note    string
	Builtin string
	Env     map[string]string
}

// MarshalJSON 手写以保证字段顺序（name, note, builtin?, env）与 env 内键顺序（KnownKeys 序）
// 与 npm 版的 JSON.stringify 输出一致——否则两版交替保存时 providers.json 会无谓地反复变动。
func (p Provider) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	writeField(&b, "name", p.Name, true)
	writeField(&b, "note", p.Note, false)
	if p.Builtin != "" {
		writeField(&b, "builtin", p.Builtin, false)
	}
	b.WriteString(`,"env":{`)
	for i, k := range orderedEnvKeys(p.Env) {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := marshalString(k)
		b.Write(kb)
		b.WriteByte(':')
		vb, _ := marshalString(p.Env[k])
		b.Write(vb)
	}
	b.WriteString("}}")
	return b.Bytes(), nil
}

// writeField 写 `,"key":value`（first=true 时省略前导逗号）。value 为字符串。
func writeField(b *bytes.Buffer, key, val string, first bool) {
	if !first {
		b.WriteByte(',')
	}
	kb, _ := marshalString(key)
	b.Write(kb)
	b.WriteByte(':')
	vb, _ := marshalString(val)
	b.Write(vb)
}

// marshalString 以不转义 HTML 的方式编码一个字符串（对齐 npm 版不转义 <>& 的行为）。
func marshalString(s string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// orderedEnvKeys 返回 env 的键：先按 KnownKeys 顺序排已有的受管键，再把其余键按字母序附后。
func orderedEnvKeys(env map[string]string) []string {
	seen := make(map[string]bool, len(env))
	keys := make([]string, 0, len(env))
	for _, k := range knownKeys {
		if _, ok := env[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	extra := make([]string, 0)
	for k := range env {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	return append(keys, extra...)
}

// Store 是 ~/.cc-mini/providers.json 的顶层结构。Lang 为新增字段，旧文件缺省视为 zh。
// 字段顺序（current, lang?, providers, update?）与 npm 版输出一致。
// Update 为更新检查模式："notify"=提醒；空=关闭（默认，省略不写）。预留 "auto" 给将来 Go 版自动升级。
type Store struct {
	Current   string     `json:"current"`
	Lang      Lang       `json:"lang,omitempty"`
	Providers []Provider `json:"providers"`
	Update    string     `json:"update,omitempty"`
}

// 注：供应商目录（presets）的类型与加载放在 internal/presets 包（见 go-rewrite-plan §8）。
