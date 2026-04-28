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
//
// OAuth-specific knobs live in the OAUTH_* environment variables consumed
// directly by mcp-oauth's oauthconfig package — see NewAuth. Keeping that
// surface in one place avoids a second source of truth here.
type Config struct {
	Debug        bool
	LogFormat    string
	OAuthEnabled bool
}

// LoadConfig reads the process-level env vars (DEBUG, LOG_FORMAT,
// OAUTH_ENABLED) and returns a populated Config. Validation of the OAUTH_*
// vars happens later inside oauthconfig.FromEnv when NewAuth runs — failing
// here on missing OAuth values would duplicate that work and lock the
// template into one provider.
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
	return &Config{
		Debug:        debug,
		LogFormat:    logFormat,
		OAuthEnabled: oauthEnabled,
	}, nil
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
