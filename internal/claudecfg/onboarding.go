// Package claudecfg 提供对 ~/.claude/ 配置文件的只读访问。
// 铁律：只读不写，任何写操作都在 ccx 范围之外。
package claudecfg

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// HasCompletedOnboarding 只读检测 ~/.claude.json 的 hasCompletedOnboarding 字段。
// 文件不存在、解析失败或字段缺失，均返回 false（视为未完成）。
func HasCompletedOnboarding() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return false
	}
	var cfg struct {
		HasCompletedOnboarding bool `json:"hasCompletedOnboarding"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return false
	}
	return cfg.HasCompletedOnboarding
}
