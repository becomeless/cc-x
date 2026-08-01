// Package env 是受管环境变量的纯计算 + 进程级套用（本次启用用）。
// 不做任何持久化副作用（那是 internal/platform/* 与 internal/defaults 的事）。
package env

import (
	"os"
	"strings"

	"github.com/becomeless/cc-x/internal/config"
)

// ManagedVals 是受管键 -> 值；空字符串表示「清除该键」。始终包含全部 10 个受管键。
type ManagedVals map[string]string

// ComputeManagedVals 给定配置，算出每个受管键的「设值或清除」。
// 对齐 npm 版 computeManagedVals：有非空值的写原值，否则记空（=清除）。
func ComputeManagedVals(p config.Provider) ManagedVals {
	m := config.GetProviderEnvMap(p)
	vals := make(ManagedVals, len(config.ManagedKeys()))
	for _, k := range config.ManagedKeys() {
		v := m[k]
		if strings.TrimSpace(v) != "" {
			vals[k] = v
		} else {
			vals[k] = ""
		}
	}
	return vals
}

// ApplyManaged 把目标配置的受管变量套到当前进程（有值 Setenv、没值 Unsetenv，只动这 10 个）。
// 本次启用用：之后 exec 出的 claude 子进程会继承当前进程环境。对齐 npm 版 applyManagedEnv。
func ApplyManaged(p config.Provider) {
	m := config.GetProviderEnvMap(p)
	for _, k := range config.ManagedKeys() {
		v := m[k]
		if strings.TrimSpace(v) != "" {
			_ = os.Setenv(k, v)
		} else {
			_ = os.Unsetenv(k)
		}
	}
}
