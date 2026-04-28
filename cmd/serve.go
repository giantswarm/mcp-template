package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcpsrv "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"github.com/giantswarm/mcp-template/internal/example"
	"github.com/giantswarm/mcp-template/internal/server"
	"github.com/giantswarm/mcp-template/internal/tools"
)

const (
	transportStdio          = "stdio"
	transportSSE            = "sse"
	transportStreamableHTTP = "streamable-http"
)

var (
	flagTransport   string
	flagMCPAddr     string
	flagMetricsAddr string
	flagDebug       bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the MCP server",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().StringVar(&flagTransport, "transport", server.EnvOr("MCP_TRANSPORT", transportStreamableHTTP),
		transportStdio+" | "+transportSSE+" | "+transportStreamableHTTP)
	serveCmd.Flags().StringVar(&flagMCPAddr, "mcp-addr", server.EnvOr("MCP_ADDR", ":8080"),
		"listen address for MCP HTTP transport")
	serveCmd.Flags().StringVar(&flagMetricsAddr, "metrics-addr", server.EnvOr("METRICS_ADDR", ":9091"),
		"listen address for /metrics, /healthz, /readyz")
	serveCmd.Flags().BoolVar(&flagDebug, "debug", false, "enable debug logging (overrides DEBUG env)")
}

func runServe(_ *cobra.Command, _ []string) error {
	if err := validateTransport(flagTransport); err != nil {
		return err
	}

	shutdownCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := server.LoadConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := server.NewLogger(cfg.Debug || flagDebug, cfg.LogFormat)

	shutdownOTEL, err := server.InitTracing(shutdownCtx, "mcp-template", version)
	if err != nil {
		logger.Warn("otel init failed; continuing without tracing", "error", err)
	} else {
		defer shutdownWithTimeout(shutdownOTEL)
	}

	exClient := example.NewFakeClient()

	mcp := mcpsrv.NewMCPServer(
		"mcp-template", version,
		mcpsrv.WithToolCapabilities(false),
		mcpsrv.WithRecovery(),
	)
	tools.Register(mcp, tools.Deps{Client: exClient, Log: logger})

	if flagTransport == transportStdio {
		logger.Info("MCP serving on stdio", "transport", transportStdio)
		logger.Warn("stdio transport bypasses OAuth — tool calls hit authz errors unless the session installs a caller identity")
		return mcpsrv.ServeStdio(mcp)
	}

	var auth *server.Auth
	if cfg.OAuthEnabled {
		auth, err = server.NewAuth(shutdownCtx, logger)
		if err != nil {
			return fmt.Errorf("oauth: %w", err)
		}
		defer func() { _ = auth.Shutdown(context.Background()) }()
	} else {
		logger.Warn("OAuth is DISABLED — set OAUTH_ENABLED=true plus OAUTH_ISSUER, OAUTH_PROVIDER, and the provider-specific OAUTH_* vars for production")
	}

	mcpMux := server.BuildMCPMux(flagTransport, mcp, auth)

	obsMux := http.NewServeMux()
	hc := server.NewHealthChecker(version, 2*time.Second)
	if auth != nil {
		if u := auth.IssuerHealthURL(); u != "" {
			hc.Register("oauth-issuer", server.HTTPProbe(nil, u))
		}
	}
	hc.RegisterHandlers(obsMux)
	obsMux.Handle("/metrics", server.MetricsHandler())

	mcpServer := &http.Server{
		Addr:              flagMCPAddr,
		Handler:           mcpMux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	obsServer := &http.Server{
		Addr:              flagMetricsAddr,
		Handler:           obsMux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	runListenAndServe(mcpServer, "MCP", logger, cancel)
	runListenAndServe(obsServer, "observability", logger, cancel)

	<-shutdownCtx.Done()
	logger.Info("shutdown requested")
	twoPhaseShutdown(logger, mcpServer, obsServer)
	return nil
}

func validateTransport(t string) error {
	switch t {
	case transportStdio, transportSSE, transportStreamableHTTP:
		return nil
	default:
		return fmt.Errorf("transport %q is not supported (want one of: %s, %s, %s)",
			t, transportStdio, transportSSE, transportStreamableHTTP)
	}
}

func shutdownWithTimeout(fn func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = fn(ctx)
}

func runListenAndServe(srv *http.Server, label string, logger *slog.Logger, cancel context.CancelFunc) {
	go func() {
		logger.Info(label+" listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(label+" server failed", "error", err)
			cancel()
		}
	}()
}

func twoPhaseShutdown(log *slog.Logger, mcpServer, obsServer *http.Server) {
	mcpDrainCtx, mcpCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mcpCancel()
	if err := mcpServer.Shutdown(mcpDrainCtx); err != nil {
		log.Error("mcp server drain returned error", "error", err)
	}
	obsDrainCtx, obsCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer obsCancel()
	if err := obsServer.Shutdown(obsDrainCtx); err != nil {
		log.Error("observability server drain returned error", "error", err)
	}
}
