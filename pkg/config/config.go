package config

import (
	"os"
	"regexp"

	"github.com/spf13/viper"
)

// envPattern 匹配完整值为 ${VAR_NAME:default_value} 或 ${VAR_NAME} 的配置项
// 例: ${AUTH_GRPC_ADDR:localhost:9001}  →  varName=AUTH_GRPC_ADDR, default=localhost:9001
var envPattern = regexp.MustCompile(`^\$\{([^:}]+)(?::([^}]*))?\}$`)

// ExpandEnvInConfig 遍历 viper 中所有字符串值，将 ${VAR:default} 替换为实际环境变量值。
// 若环境变量未设置则使用 default 部分。
// 应在 viper.ReadInConfig() 之后调用。
func ExpandEnvInConfig() {
	for _, key := range viper.AllKeys() {
		val := viper.GetString(key)
		if expanded, ok := expandValue(val); ok {
			viper.Set(key, expanded)
		}
	}
}

// expandValue 尝试展开单个值中的 ${VAR:default} 语法。
// 返回 (展开后的值, true) 或 ("", false)（值未变化时）。
func expandValue(s string) (string, bool) {
	m := envPattern.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	varName := m[1]
	defaultVal := ""
	if len(m) >= 3 {
		defaultVal = m[2]
	}
	if val, ok := os.LookupEnv(varName); ok {
		return val, true
	}
	return defaultVal, true
}
