// Package cmd provides the command-line interface for mcp-template.
//
// This package implements a Cobra-based CLI with multiple subcommands:
//   - serve: Starts the MCP server (default behavior when no subcommand is provided)
//   - version: Displays the application version, commit and build time
//   - self-update: Updates the binary to the latest release from GitHub after
//     verifying its cosign Sigstore bundle
//
// Command Structure:
//
//	mcp-template [flags]                 # Starts the MCP server (default)
//	mcp-template serve [flags]           # Explicitly starts the MCP server
//	mcp-template version                 # Shows version information
//	mcp-template self-update             # Updates to the latest signed release
//	mcp-template help [command]          # Shows help information
//
// The serve command supports multiple transport options:
//   - streamable-http: Streamable HTTP transport (default) - for HTTP-based integration
//   - sse: Server-Sent Events over HTTP - for web-based clients
//   - stdio: Standard input/output - for command-line integration (bypasses OAuth)
//
// Transport Configuration Examples:
//
//	mcp-template serve --transport stdio
//	mcp-template serve --transport sse --mcp-addr :8080
//	mcp-template serve --transport streamable-http --mcp-addr :8080 --metrics-addr :9091
//
// The version every command reports is rootCmd.Version, which main sets from
// pkg/project (stamped at link time by the generated Makefile and the
// architect go-build job). Do not add version variables elsewhere.
package cmd
