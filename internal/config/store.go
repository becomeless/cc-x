package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// StorePaths 是 store 目录与文件的解析结果。
type StorePaths struct {
	Dir  string
	File string
}

// ResolveStorePaths 解析存储路径。storeDir 来自 --store-dir（测试用），默认 ~/.cc-mini。
func ResolveStorePaths(storeDir string) StorePaths {
	dir := storeDir
	if strings.TrimSpace(dir) == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cc-mini")
	}
	return StorePaths{Dir: dir, File: filepath.Join(dir, "providers.json")}
}

func nonEmpty(v string) bool { return strings.TrimSpace(v) != "" }

// DefaultStore 返回默认配置：官方 + DeepSeek + 智谱GLM + 小米MiMo（密钥空）。
// 官方档带 builtin="official"（评审①），其它供应商是专有名词、不翻译。
func DefaultStore() *Store {
	return &Store{
		Current: "官方",
		Lang:    LangZH,
		Providers: []Provider{
			{Name: "官方", Note: "", Builtin: "official", Env: map[string]string{}},
			{Name: "DeepSeek", Note: "", Env: map[string]string{
				"ANTHROPIC_BASE_URL":             "https://api.deepseek.com/anthropic",
				"ANTHROPIC_DEFAULT_OPUS_MODEL":   "deepseek-v4-pro",
				"ANTHROPIC_DEFAULT_SONNET_MODEL": "deepseek-v4-pro",
				"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "deepseek-v4-flash",
				"CLAUDE_CODE_EFFORT_LEVEL":       "max",
			}},
			{Name: "智谱GLM", Note: "", Env: map[string]string{
				"ANTHROPIC_BASE_URL":                       "https://open.bigmodel.cn/api/anthropic",
				"ANTHROPIC_DEFAULT_OPUS_MODEL":             "GLM-4.7",
				"ANTHROPIC_DEFAULT_SONNET_MODEL":           "GLM-4.7",
				"ANTHROPIC_DEFAULT_HAIKU_MODEL":            "glm-4.5-air",
				"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
			}},
			{Name: "小米MiMo", Note: "", Env: map[string]string{
				"ANTHROPIC_BASE_URL":             "https://api.xiaomimimo.com/anthropic",
				"ANTHROPIC_DEFAULT_OPUS_MODEL":   "mimo-v2.5-pro",
				"ANTHROPIC_DEFAULT_SONNET_MODEL": "mimo-v2.5-pro",
				"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "mimo-v2.5-pro",
			}},
		},
	}
}

// StoreErrorKind 区分配置文件不可用的原因。
type StoreErrorKind string

const (
	// ErrRead 读文件失败（权限/磁盘/同名目录等）。
	ErrRead StoreErrorKind = "read"
	// ErrParse JSON 语法损坏。
	ErrParse StoreErrorKind = "parse"
	// ErrFormat JSON 语法合法但结构损坏（顶层非对象 / providers 非数组等）。
	ErrFormat StoreErrorKind = "format"
)

// StoreError 在配置文件存在但不可用时返回。调用方据此友好提示并退出，
// 绝不静默重建/覆盖——那会清掉用户的明文密钥（违背「不碰用户数据」初心）。
type StoreError struct {
	Kind StoreErrorKind
	File string
}

func (e *StoreError) Error() string {
	return fmt.Sprintf("store %s error: %s", e.Kind, e.File)
}

// Load 读配置；文件不存在则生成默认并落盘后返回；文件不可用则返回 *StoreError（绝不覆盖）。
func Load(p StorePaths) (*Store, error) {
	data, err := os.ReadFile(p.File)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s := DefaultStore()
			if werr := Save(p, s); werr != nil {
				return nil, werr
			}
			return s, nil
		}
		return nil, &StoreError{Kind: ErrRead, File: p.File}
	}
	var raw any
	if jerr := json.Unmarshal(data, &raw); jerr != nil {
		return nil, &StoreError{Kind: ErrParse, File: p.File}
	}
	return normalizeStore(raw, p.File)
}

// normalizeStore 把任意解析结果规整为合法 Store。
//
// 字段级宽松容错（缺 lang/note/builtin/env 都不报错）保持兼容；
// 结构级严格校验（顶层须对象、providers 须数组、每条 name/env 结构须合法）否则返回 ErrFormat。
// 堵住「语法合法但结构损坏的 JSON 被静默规整成空 providers、用户一保存就覆盖丢数据」的坑。
func normalizeStore(raw any, file string) (*Store, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, &StoreError{Kind: ErrFormat, File: file}
	}
	provsRaw, ok := obj["providers"].([]any)
	if !ok {
		return nil, &StoreError{Kind: ErrFormat, File: file}
	}
	providers := make([]Provider, 0, len(provsRaw))
	for _, pr := range provsRaw {
		p, err := normalizeProvider(pr, file)
		if err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	var lang Lang
	if l, ok := obj["lang"].(string); ok {
		switch l {
		case "en":
			lang = LangEN
		case "zh":
			lang = LangZH
		}
	}
	current := ""
	if c, ok := obj["current"].(string); ok {
		current = c
	} else if len(providers) > 0 {
		current = providers[0].Name
	}
	update := ""
	if u, ok := obj["update"].(string); ok && (u == "notify" || u == "auto") {
		update = u // 只认已知模式，未知值视为关闭（不回写）
	}
	return &Store{Current: current, Lang: lang, Providers: providers, Update: update}, nil
}

func normalizeProvider(raw any, file string) (Provider, error) {
	formatErr := func() (Provider, error) { return Provider{}, &StoreError{Kind: ErrFormat, File: file} }
	p, ok := raw.(map[string]any)
	if !ok {
		return formatErr()
	}
	name, ok := p["name"].(string)
	if !ok {
		return formatErr()
	}
	note := ""
	if nv, exists := p["note"]; exists {
		s, ok := nv.(string)
		if !ok {
			return formatErr()
		}
		note = s
	}
	builtin := ""
	if bv, exists := p["builtin"]; exists {
		s, ok := bv.(string)
		if !ok {
			return formatErr()
		}
		builtin = s
	}
	env := map[string]string{}
	if ev, exists := p["env"]; exists {
		m, ok := ev.(map[string]any)
		if !ok {
			return formatErr()
		}
		for k, vv := range m {
			s, ok := vv.(string)
			if !ok {
				return formatErr()
			}
			env[k] = s
		}
	}
	return Provider{Name: name, Note: note, Builtin: builtin, Env: env}, nil
}

// PeekStoreLang 只读探测 lang，不生成文件（用于 --help/--version 在 parse 前定语言，避免副作用）。
// 文件不存在/解析失败都返回空。
func PeekStoreLang(p StorePaths) Lang {
	data, err := os.ReadFile(p.File)
	if err != nil {
		return ""
	}
	var raw struct {
		Lang string `json:"lang"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return ""
	}
	switch raw.Lang {
	case "en":
		return LangEN
	case "zh":
		return LangZH
	}
	return ""
}

// Save 写配置：UTF-8 无 BOM、2 空格缩进、不转义 HTML、尾随换行（与 npm 版一致）。
func Save(p StorePaths, store *Store) error {
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return &StoreError{Kind: ErrRead, File: p.File}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(store); err != nil {
		return err
	}
	// enc.Encode 已追加一个换行，与 npm 版 `JSON.stringify(...)+"\n"` 一致。
	if err := os.WriteFile(p.File, buf.Bytes(), 0o644); err != nil {
		return &StoreError{Kind: ErrRead, File: p.File}
	}
	return nil
}

// IsOfficial 判断是否官方档。优先认稳定标识 builtin=="official"（评审①）；
// 老文件没有 builtin 时，仅将「中文名为官方 + env 为空」视为官方档。
func IsOfficial(p Provider) bool {
	if p.Builtin != "" {
		return p.Builtin == "official"
	}
	return p.Name == "官方" && len(p.Env) == 0
}

// ReconcileBuiltin 编辑保存后修正身份：官方档（builtin=official、空 env）一旦被配成真实第三方
// （env 非空），就清掉 builtin —— 否则会继续被当登录态、跳过缺密钥警告。
func ReconcileBuiltin(p *Provider) {
	if p.Builtin == "official" && len(p.Env) > 0 {
		p.Builtin = ""
	}
}

// ReconcileCurrent 删除等操作后修正默认指向：优先剩余官方档，其次第一项；没有配置则置空。
func ReconcileCurrent(store *Store) {
	for _, p := range store.Providers {
		if p.Name == store.Current {
			return
		}
	}
	for _, p := range store.Providers {
		if IsOfficial(p) {
			store.Current = p.Name
			return
		}
	}
	if len(store.Providers) > 0 {
		store.Current = store.Providers[0].Name
	} else {
		store.Current = ""
	}
}

// GetProviderEnvMap 取配置的 env map（保证非 nil）。
func GetProviderEnvMap(p Provider) map[string]string {
	if p.Env == nil {
		return map[string]string{}
	}
	return p.Env
}

// BuildProviderEnv 由一组字段构造 provider.env：按 KnownKeys 顺序、丢弃空白值（trim 后存）。
// `[1m]` 等后缀属于自由文本，原样保留（见 plan §3.1.1）。
func BuildProviderEnv(fields map[string]string) map[string]string {
	env := map[string]string{}
	for _, key := range knownKeys {
		v := fields[key]
		if nonEmpty(v) {
			env[key] = strings.TrimSpace(v)
		}
	}
	return env
}

// KeyState 是配置的密钥状态（语义枚举，不含界面文案；翻译交给 i18n 层，评审①）。
type KeyState string

const (
	KeyOfficial KeyState = "official"
	KeyNone     KeyState = "noKey"
	KeyAPIKey   KeyState = "apiKey"
	KeyToken    KeyState = "hasToken"
)

// ProviderState 是配置的运行状态：密钥种类 + effort。
type ProviderState struct {
	Key    KeyState
	Effort string
}

// GetProviderState 计算配置状态，对齐 npm 版 getProviderState。
func GetProviderState(p Provider) ProviderState {
	m := GetProviderEnvMap(p)
	effort := ""
	if nonEmpty(m["CLAUDE_CODE_EFFORT_LEVEL"]) {
		effort = m["CLAUDE_CODE_EFFORT_LEVEL"]
	}
	if IsOfficial(p) {
		return ProviderState{Key: KeyOfficial, Effort: effort}
	}
	hasTok := nonEmpty(m["ANTHROPIC_AUTH_TOKEN"])
	hasKey := nonEmpty(m["ANTHROPIC_API_KEY"])
	key := KeyToken
	switch {
	case !hasTok && !hasKey:
		key = KeyNone
	case hasKey:
		key = KeyAPIKey
	}
	return ProviderState{Key: key, Effort: effort}
}

// FindProvider 按 name 找配置；找不到返回 nil。
func FindProvider(store *Store, name string) *Provider {
	for i := range store.Providers {
		if store.Providers[i].Name == name {
			return &store.Providers[i]
		}
	}
	return nil
}

// ResolveUniqueName 名称去重：同名被【其它】配置占用时追加 " 2/3/…"。
// exclude 是正在编辑的本条（按指针排除自身，传 nil 表示不排除）。对齐 npm 版 resolveUniqueName。
func ResolveUniqueName(store *Store, name string, exclude *Provider) string {
	existing := make(map[string]bool)
	for i := range store.Providers {
		if exclude != nil && &store.Providers[i] == exclude {
			continue
		}
		existing[store.Providers[i].Name] = true
	}
	if !existing[name] {
		return name
	}
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s %d", name, i)
		if !existing[cand] {
			return cand
		}
	}
}

// GetLang 取语言：store.lang 字段，缺省视为 zh。
func GetLang(store *Store) Lang {
	if store.Lang == LangEN {
		return LangEN
	}
	return LangZH
}

// SetLang 设置语言。
func SetLang(store *Store, lang Lang) {
	store.Lang = lang
}
