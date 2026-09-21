package config

import (
	"errors"
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
	// This is strictly required for HMAC signature verification.
	ZitadelSigningKey string
}

// Load reads configuration from environment variables with sensible defaults.
// It returns an error if the mandatory ZITADEL_SIGNING_KEY is missing or empty.
func Load() (*Config, error) {
	key := getEnvOrDefault("ZITADEL_SIGNING_KEY", "")
	if key == "" {
		return nil, errors.New("ZITADEL_SIGNING_KEY is required but not set")
	}

	return &Config{
		Port:              getEnvOrDefault("PORT", "8080"),
		LogLevel:          strings.ToLower(getEnvOrDefault("LOG_LEVEL", "info")),
		ClaimName:         getEnvOrDefault("CLAIM_NAME", "groups"),
		RoleFormat:        strings.ToLower(getEnvOrDefault("ROLE_FORMAT", "bare")),
		LowercaseRoles:    getEnvAsBoolOrDefault("LOWERCASE_ROLES", true),
		FilterProjectID:   getEnvOrDefault("FILTER_PROJECT_ID", ""),
		ZitadelSigningKey: key,
	}, nil
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
