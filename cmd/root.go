// Package cmd defines the cobra root and subcommands. main.go calls
// Execute(); tests can drive rootCmd.ExecuteContext directly.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is injected at build time via
// -ldflags="-X github.com/giantswarm/mcp-template/cmd.version=<ver>".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "mcp-template",
	Short: "MCP server (template — replace this description)",
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
}
