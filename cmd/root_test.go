package cmd

import (
	"strings"
	"testing"
)

func TestRootCmdProperties(t *testing.T) {
	if rootCmd.Use != "mcp-template" {
		t.Errorf("Use = %q, want mcp-template", rootCmd.Use)
	}
	if rootCmd.Short == "" {
		t.Error("Short must describe the server (scripts/init.sh fills the placeholder)")
	}
	for _, want := range []string{"Model Context Protocol", "mcp-template serve"} {
		if !strings.Contains(rootCmd.Long, want) {
			t.Errorf("Long should mention %q, got:\n%s", want, rootCmd.Long)
		}
	}
	if !rootCmd.SilenceUsage {
		t.Error("SilenceUsage should be set so a handled error does not print the usage text")
	}
}

func TestSetVersion(t *testing.T) {
	original := rootCmd.Version
	t.Cleanup(func() { rootCmd.Version = original })

	SetVersion("v1.2.3-test")
	if rootCmd.Version != "v1.2.3-test" {
		t.Errorf("rootCmd.Version = %q, want v1.2.3-test", rootCmd.Version)
	}
}

func TestRootCommandHasSubcommands(t *testing.T) {
	found := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		found[c.Use] = true
	}
	for _, want := range []string{"serve", "version", "self-update"} {
		if !found[want] {
			t.Errorf("subcommand %q is missing; have %v", want, found)
		}
	}
}
