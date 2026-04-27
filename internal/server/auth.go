package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	oauth "github.com/giantswarm/mcp-oauth"
	"github.com/giantswarm/mcp-oauth/providers"
	"github.com/giantswarm/mcp-oauth/providers/dex"
	"github.com/giantswarm/mcp-oauth/storage"
	"github.com/giantswarm/mcp-oauth/storage/memory"
	"github.com/giantswarm/mcp-oauth/storage/valkey"
)

// Auth bundles the mcp-oauth Server + Handler with their teardown. nil Auth
// means OAuth is disabled — callers branch on cfg.OAuth.Enabled before
// constructing.
//
// The Handler / Server fields are exposed so a server author can reach for
// any oauth.* method we don't pre-wire (token revocation customisations,
// custom encryptor, etc.) without forking this package.
type Auth struct {
	Server  *oauth.Server
	Handler *oauth.Handler
	close   func()
	issuer  string
}

// NewAuth wires the dex provider, the storage backend, and the mcp-oauth
// server, and returns an *Auth ready to mount on the MCP mux.
func NewAuth(_ context.Context, cfg OAuthConfig, logger *slog.Logger) (*Auth, error) {
	dexProvider, err := dex.NewProvider(&dex.Config{
		IssuerURL:    cfg.DexIssuerURL,
		ClientID:     cfg.DexClientID,
		ClientSecret: cfg.DexClientSecret,
		RedirectURL:  cfg.OAuthRedirectURL,
	})
	if err != nil {
		return nil, fmt.Errorf("dex provider: %w", err)
	}

	tokenStore, clientStore, flowStore, storeClose, err := newOAuthStore(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("oauth store: %w", err)
	}

	if (cfg.OAuthStorage == "" || cfg.OAuthStorage == "memory") && os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		logger.Warn("OAUTH_STORAGE=memory in a Kubernetes deployment — OAuth state is lost on pod restart and NOT shared across replicas; use OAUTH_STORAGE=valkey for production")
	}

	srv, err := oauth.NewServer(
		dexProvider,
		tokenStore, clientStore, flowStore,
		&oauth.ServerConfig{
			Issuer:                           cfg.OAuthIssuer,
			AllowInsecureHTTP:                cfg.OAuthAllowInsecureHTTP,
			AllowPublicClientRegistration:    cfg.OAuthAllowPublicClientRegistration,
			AllowLocalhostRedirectURIs:       true,
			TrustedAudiences:                 cfg.OAuthTrustedAudiences,
			TrustedPublicRegistrationSchemes: cfg.OAuthTrustedRedirectSchemes,
		},
		logger,
	)
	if err != nil {
		storeClose()
		return nil, fmt.Errorf("oauth server: %w", err)
	}
	return &Auth{
		Server:  srv,
		Handler: oauth.NewHandler(srv, logger),
		close:   storeClose,
		issuer:  cfg.DexIssuerURL,
	}, nil
}

// Shutdown closes the storage backend.
func (a *Auth) Shutdown(_ context.Context) error {
	if a == nil || a.close == nil {
		return nil
	}
	a.close()
	return nil
}

// IssuerHealthURL returns the upstream Dex issuer URL — the readiness probe
// can hit `<issuer>/.well-known/openid-configuration` to confirm the
// upstream is reachable, but for a generic check we just GET the issuer
// root.
func (a *Auth) IssuerHealthURL() string {
	return a.issuer + "/.well-known/openid-configuration"
}

func newOAuthStore(cfg OAuthConfig, logger *slog.Logger) (
	storage.TokenStore, storage.ClientStore, storage.FlowStore, func(), error,
) {
	switch cfg.OAuthStorage {
	case "", "memory":
		s := memory.New()
		return s, s, s, func() { s.Stop() }, nil
	case "valkey":
		vcfg := valkey.Config{
			Address:  cfg.ValkeyAddr,
			Password: cfg.ValkeyPassword,
			Logger:   logger,
		}
		if cfg.ValkeyTLS {
			vcfg.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
		}
		s, err := valkey.New(vcfg)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("valkey: %w", err)
		}
		return s, s, s, func() { s.Close() }, nil
	default:
		return nil, nil, nil, nil, fmt.Errorf("unknown OAUTH_STORAGE=%q (want memory|valkey)", cfg.OAuthStorage)
	}
}

// PromoteOAuthCaller lifts the UserInfo attached by mcp-oauth's
// ValidateToken middleware onto the context mcp-go passes to tool handlers.
// Pass it to mcpsrv.WithHTTPContextFunc / WithSSEContextFunc so tool
// handlers can resolve the caller via CallerFromContext.
//
// This is the bridge between mcp-oauth's HTTP layer and mcp-go's tool
// handler context. If/when mcp-oauth ships an upstream bridge package,
// swap this function for a re-export.
func PromoteOAuthCaller(ctx context.Context, r *http.Request) context.Context {
	if ui, ok := oauth.UserInfoFromContext(r.Context()); ok {
		return context.WithValue(ctx, callerKey{}, ui)
	}
	return ctx
}

// Caller is the identity tools see. Subject is the OIDC sub claim (stable,
// non-spoofable); Email is the human-facing handle. The raw UserInfo is
// available for advanced use.
type Caller struct {
	Subject string
	Email   string
	Groups  []string
	Raw     *providers.UserInfo
}

// Empty reports whether no identifying fields were set.
func (c Caller) Empty() bool { return c.Subject == "" && c.Email == "" }

// CallerFromContext extracts the caller from a tool handler's context.
// Returns (zero, false) when no caller is attached (e.g., stdio transport
// or pre-auth middleware).
func CallerFromContext(ctx context.Context) (Caller, bool) {
	ui, ok := ctx.Value(callerKey{}).(*providers.UserInfo)
	if !ok || ui == nil {
		return Caller{}, false
	}
	return Caller{
		Subject: ui.ID,
		Email:   ui.Email,
		Groups:  ui.Groups,
		Raw:     ui,
	}, true
}

// callerKey is unexported so external packages cannot overwrite the value.
type callerKey struct{}
