package config

import (
	"os"
	"regexp"

	"github.com/spf13/viper"
)

// envPattern matches config items with full value ${VAR_NAME:default_value} or ${VAR_NAME}
// Example: ${AUTH_GRPC_ADDR:localhost:9001}  →  varName=AUTH_GRPC_ADDR, default=localhost:9001
var envPattern = regexp.MustCompile(`^\$\{([^:}]+)(?::([^}]*))?\}$`)

// ExpandEnvInConfig iterates through all string values in viper and replaces ${VAR:default} with actual environment variable values.
// If environment variable is not set, the default part is used.
// Should be called after viper.ReadInConfig().
func ExpandEnvInConfig() {
	for _, key := range viper.AllKeys() {
		val := viper.GetString(key)
		if expanded, ok := expandValue(val); ok {
			viper.Set(key, expanded)
		}
	}
}

// expandValue tries to expand ${VAR:default} syntax in a single value.
// Returns (expanded value, true) or ("", false) when value is unchanged.
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
