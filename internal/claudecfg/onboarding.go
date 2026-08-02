// Package claudecfg 提供对 ~/.claude.json 的受控访问。
// 铁律：ccx 从不写 Claude Code 配置文件。唯一豁免（用户主动触发的主菜单「一键免登录」）：
// 把顶层布尔字段 hasCompletedOnboarding 写为 true——字符串级字节最小修改，绝不整文件 JSON 重排，
// 文件不合法 JSON 时拒绝写入。除此外仍只读不写。
package claudecfg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// MarkOnboardingDone 一键免登录：把 ~/.claude.json 顶层 hasCompletedOnboarding 写为 true。
// 文件不存在则创建仅含该字段的最小文件（mimo 官方文档示例格式）。返回 error 表示写入失败且文件未被修改。
func MarkOnboardingDone() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return MarkOnboardingDoneIn(filepath.Join(home, ".claude.json"))
}

// MarkOnboardingDoneIn 是显式路径实现（测试用）。
//
// 流程：文件不存在 -> 创建；存在 -> json.Valid 校验 + 顶层须为对象，然后字节级替换/插入，
// 其它字节原样保留；值已是 true 时不写盘（mtime 不动）。任何失败都保持文件原样。
// 符号链接先解析再写（temp+rename 直接写会替换链接本体）。
func MarkOnboardingDoneIn(path string) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return atomicWrite(path, []byte("{\"hasCompletedOnboarding\": true}\n"))
		}
		return err
	}
	if !json.Valid(data) {
		return fmt.Errorf("%s 不是合法 JSON，拒绝改动", path)
	}
	if !bytes.HasPrefix(bytes.TrimLeft(data, " \t\r\n"), []byte("{")) {
		return fmt.Errorf("%s 顶层不是 JSON 对象，拒绝改动", path)
	}
	next, changed, err := spliceOnboarding(data)
	if err != nil {
		return err
	}
	if !changed {
		return nil // 值已是 true：幂等，不写盘
	}
	return atomicWrite(path, next)
}

// spliceOnboarding 单遍扫描：把顶层 hasCompletedOnboarding 的值替换为 true。
// 顶层无该键时插入到第一个 { 之后（空对象 {} 不带尾逗号——{"x": true,} 是非法 JSON）。
// 重复顶层键按 JSON 语义取最后一次命中（V8/Go 都是后者胜出）。
// 字符串值内的同名文本不会误命中（键后必须紧跟 : 才算键）。调用方须保证 data 是合法 JSON 且顶层为对象。
func spliceOnboarding(data []byte) ([]byte, bool, error) {
	key := []byte(`hasCompletedOnboarding`)
	depth := 0     // 对象+数组深度；depth==1 表示在顶层对象内
	inStr := false // 当前在字符串值内（字符串内一切字节不参与键判定）
	escaped := false
	var lastStart, lastEnd int // 最后一次命中的值 span
	found := false
	braceAt := -1 // 顶层第一个 { 的位置（插入点）

	for i := 0; i < len(data); i++ {
		c := data[i]
		if inStr {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			if depth == 1 {
				// 候选键：解析到闭合引号，键后跳过空白必须紧跟 : 才算键
				end := i + 1
				es := false
				for end < len(data) {
					if es {
						es = false
					} else if data[end] == '\\' {
						es = true
					} else if data[end] == '"' {
						break
					}
					end++
				}
				if end >= len(data) { // json.Valid 已保证可达；防御
					return nil, false, errors.New("未闭合的字符串，拒绝改动")
				}
				colon := end + 1
				for colon < len(data) && isJSONSpace(data[colon]) {
					colon++
				}
				if colon < len(data) && data[colon] == ':' && bytes.Equal(data[i+1:end], key) {
					v := colon + 1
					for v < len(data) && isJSONSpace(data[v]) {
						v++
					}
					vs, ve, err := valueSpan(data, v)
					if err != nil {
						return nil, false, err
					}
					lastStart, lastEnd = vs, ve
					found = true
					i = ve - 1 // 跳过已消费的值，继续向后扫描（重复键取最后）
					continue
				}
				i = end // 非目标键：跳过该字符串
				continue
			}
			inStr = true
		case '{':
			if braceAt == -1 {
				braceAt = i
			}
			depth++
		case '}':
			depth--
		case '[':
			depth++
		case ']':
			depth--
		}
	}

	if found {
		if bytes.Equal(data[lastStart:lastEnd], []byte("true")) {
			return data, false, nil
		}
		return splice(data, lastStart, lastEnd, []byte("true")), true, nil
	}
	// 未命中：插入到顶层 { 之后
	next := braceAt + 1
	for next < len(data) && isJSONSpace(data[next]) {
		next++
	}
	ins := []byte(`"hasCompletedOnboarding": true`)
	if next < len(data) && data[next] == '}' {
		// 空对象：不加尾逗号
	} else {
		ins = append(ins, ',')
	}
	out := make([]byte, 0, len(data)+len(ins)+1)
	out = append(out, data[:braceAt+1]...)
	out = append(out, ins...)
	out = append(out, data[braceAt+1:]...)
	return out, true, nil
}

// valueSpan 定位 data[v] 起的一个 JSON 值的结束位置（不含）。v 指向值首字节。
func valueSpan(data []byte, v int) (int, int, error) {
	if v >= len(data) {
		return 0, 0, errors.New("值缺失，拒绝改动")
	}
	c := data[v]
	switch {
	case c == '"': // 字符串：转义感知到闭合引号
		es := false
		for e := v + 1; e < len(data); e++ {
			if es {
				es = false
			} else if data[e] == '\\' {
				es = true
			} else if data[e] == '"' {
				return v, e + 1, nil
			}
		}
		return 0, 0, errors.New("未闭合的字符串，拒绝改动")
	case c == '{' || c == '[': // 对象/数组：深度匹配
		d := 1
		in := false
		es := false
		for e := v + 1; e < len(data); e++ {
			if in {
				if es {
					es = false
				} else if data[e] == '\\' {
					es = true
				} else if data[e] == '"' {
					in = false
				}
				continue
			}
			switch data[e] {
			case '"':
				in = true
			case '{', '[':
				d++
			case '}', ']':
				d--
				if d == 0 {
					return v, e + 1, nil
				}
			}
		}
		return 0, 0, errors.New("未闭合的容器，拒绝改动")
	case c == 't' || c == 'f' || c == 'n': // true/false/null：字母串
		e := v
		for e < len(data) && isAlpha(data[e]) {
			e++
		}
		return v, e, nil
	case c == '-' || (c >= '0' && c <= '9'): // 数字
		e := v
		for e < len(data) && isNumChar(data[e]) {
			e++
		}
		return v, e, nil
	default:
		return 0, 0, fmt.Errorf("无法定位值，拒绝改动")
	}
}

func splice(data []byte, s, e int, repl []byte) []byte {
	out := make([]byte, 0, len(data)-(e-s)+len(repl))
	out = append(out, data[:s]...)
	out = append(out, repl...)
	out = append(out, data[e:]...)
	return out
}

func isJSONSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }
func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isNumChar(c byte) bool {
	return (c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E'
}

// atomicWrite 同目录 temp + Sync + Rename 原子写；已存在时保留原权限位。失败时清理 temp 并保证原文件不动。
func atomicWrite(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".claude.json-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if st, err := os.Stat(path); err == nil {
		_ = os.Chmod(tmpName, st.Mode().Perm())
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	return nil
}
