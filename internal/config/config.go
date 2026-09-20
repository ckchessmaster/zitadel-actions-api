package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration settings for the service.
type Config struct {
	// Port is the HTTP server listening port. Default is 8080.
	Port string

	// LogLevel is the logging level (debug, info, warn, error). Default is info.
	LogLevel string

	// ClaimName is the default token claim name to inject. Default is "groups".
	ClaimName string

	// RoleFormat specifies role format: "bare" (e.g. "admin") or "prefixed" (e.g. "projectId:admin").
	RoleFormat string

	// LowercaseRoles specifies whether role strings are normalized to lowercase. Default is true.
	LowercaseRoles bool

	// FilterProjectID optionally limits role extraction to a specific ZITADEL project ID.
	FilterProjectID string

	// ZitadelSigningKey is the secret signing key issued by ZITADEL for the Action Target.
	// If non-empty, incoming webhooks must include a valid HMAC signature in the Zitadel-Signature header.
	ZitadelSigningKey string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:              getEnvOrDefault("PORT", "8080"),
		LogLevel:          strings.ToLower(getEnvOrDefault("LOG_LEVEL", "info")),
		ClaimName:         getEnvOrDefault("CLAIM_NAME", "groups"),
		RoleFormat:        strings.ToLower(getEnvOrDefault("ROLE_FORMAT", "bare")),
		LowercaseRoles:    getEnvAsBoolOrDefault("LOWERCASE_ROLES", true),
		FilterProjectID:   getEnvOrDefault("FILTER_PROJECT_ID", ""),
		ZitadelSigningKey: getEnvOrDefault("ZITADEL_SIGNING_KEY", ""),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvAsBoolOrDefault(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		b, err := strconv.ParseBool(strings.TrimSpace(val))
		if err == nil {
			return b
		}
	}
	return defaultVal
}
