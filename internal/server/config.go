// Package server contains the wiring an MCP server built from this template
// needs: config loading, logging, OTEL bootstrap, health probes, OAuth
// integration, and HTTP mux composition. Keep it boring and visible —
// engineers should be able to read cmd/mcp-template/serve.go and trace every
// import here without surprises.
package server

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LogFormat values accepted by NewLogger.
const (
	LogFormatJSON = "json"
	LogFormatText = "text"
)

// Config is the resolved process configuration. Fed by LoadConfig at startup.
type Config struct {
	Debug     bool
	LogFormat string

	OAuth OAuthConfig
}

// OAuthConfig collects every knob NewAuth needs. Enabled=false leaves the
// rest unused.
type OAuthConfig struct {
	Enabled                            bool
	DexIssuerURL                       string
	DexClientID                        string
	DexClientSecret                    string
	OAuthIssuer                        string
	OAuthRedirectURL                   string
	OAuthAllowInsecureHTTP             bool
	OAuthAllowPublicClientRegistration bool
	OAuthStorage                       string // "memory" | "valkey"
	OAuthTrustedAudiences              []string
	OAuthTrustedRedirectSchemes        []string
	ValkeyAddr                         string
	ValkeyPassword                     string
	ValkeyTLS                          bool
}

// LoadConfig reads every env var, validates them, and returns a populated
// Config. Fails fast on missing required vars or unparseable values.
func LoadConfig() (*Config, error) {
	debug, err := EnvBool("DEBUG", false)
	if err != nil {
		return nil, err
	}
	logFormat, err := resolveLogFormat()
	if err != nil {
		return nil, err
	}

	oauthEnabled, err := EnvBool("OAUTH_ENABLED", false)
	if err != nil {
		return nil, err
	}
	c := &Config{Debug: debug, LogFormat: logFormat}
	if !oauthEnabled {
		return c, nil
	}

	allowInsecureHTTP, err := EnvBool("OAUTH_ALLOW_INSECURE_HTTP", false)
	if err != nil {
		return nil, err
	}
	allowPublicClientReg, err := EnvBool("OAUTH_ALLOW_PUBLIC_CLIENT_REGISTRATION", false)
	if err != nil {
		return nil, err
	}
	valkeyTLS, err := EnvBool("VALKEY_TLS", false)
	if err != nil {
		return nil, err
	}

	c.OAuth = OAuthConfig{
		Enabled:                            true,
		DexIssuerURL:                       os.Getenv("DEX_ISSUER_URL"),
		DexClientID:                        os.Getenv("DEX_CLIENT_ID"),
		DexClientSecret:                    os.Getenv("DEX_CLIENT_SECRET"),
		OAuthIssuer:                        os.Getenv("OAUTH_ISSUER"),
		OAuthRedirectURL:                   os.Getenv("OAUTH_REDIRECT_URL"),
		OAuthAllowInsecureHTTP:             allowInsecureHTTP,
		OAuthAllowPublicClientRegistration: allowPublicClientReg,
		OAuthStorage:                       strings.ToLower(EnvOr("OAUTH_STORAGE", "memory")),
		OAuthTrustedAudiences:              EnvCSV("OAUTH_TRUSTED_AUDIENCES"),
		OAuthTrustedRedirectSchemes:        EnvCSV("OAUTH_TRUSTED_REDIRECT_SCHEMES"),
		ValkeyAddr:                         os.Getenv("VALKEY_ADDR"),
		ValkeyPassword:                     os.Getenv("VALKEY_PASSWORD"),
		ValkeyTLS:                          valkeyTLS,
	}

	var missing []string
	for k, v := range map[string]string{
		"DEX_ISSUER_URL":    c.OAuth.DexIssuerURL,
		"DEX_CLIENT_ID":     c.OAuth.DexClientID,
		"DEX_CLIENT_SECRET": c.OAuth.DexClientSecret,
		"OAUTH_ISSUER":      c.OAuth.OAuthIssuer,
	} {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if c.OAuth.OAuthStorage == "valkey" && c.OAuth.ValkeyAddr == "" {
		missing = append(missing, "VALKEY_ADDR (required when OAUTH_STORAGE=valkey)")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %v", missing)
	}
	if c.OAuth.OAuthRedirectURL == "" {
		c.OAuth.OAuthRedirectURL = c.OAuth.OAuthIssuer + "/oauth/callback"
	}
	return c, nil
}

func resolveLogFormat() (string, error) {
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		switch strings.ToLower(v) {
		case LogFormatJSON:
			return LogFormatJSON, nil
		case LogFormatText:
			return LogFormatText, nil
		default:
			return "", fmt.Errorf("LOG_FORMAT=%q: want %q or %q", v, LogFormatJSON, LogFormatText)
		}
	}
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return LogFormatJSON, nil
	}
	return LogFormatText, nil
}

// EnvOr returns os.Getenv(k) when non-empty, otherwise def.
func EnvOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// EnvBool reads a bool env var. Empty -> def. Unparseable -> error so a typo
// like DEBUG=yes fails startup instead of silently becoming false.
func EnvBool(k string, def bool) (bool, error) {
	v := os.Getenv(k)
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s=%q: not a bool (want true|false|1|0)", k, v)
	}
	return b, nil
}

// EnvDuration reads a time.Duration env var. "" -> def. "0"/"0s" -> 0
// (conventional disable marker). Unparseable -> hard error.
func EnvDuration(k string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(k)
	if v == "" {
		return def, nil
	}
	if v == "0" || v == "0s" {
		return 0, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: not a duration (%w)", k, v, err)
	}
	return d, nil
}

// EnvInt reads an int env var. "" -> def. Unparseable -> error.
func EnvInt(k string, def int) (int, error) {
	v := os.Getenv(k)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: not an integer (%w)", k, v, err)
	}
	return n, nil
}

// EnvCSV reads a comma-separated env var, trims whitespace, drops empties.
func EnvCSV(k string) []string {
	v := os.Getenv(k)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
