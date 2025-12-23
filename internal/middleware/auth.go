package middleware

import (
	"net/http"
	"strings"

	"github.com/yourname/api-gateway/internal/auth"
	"github.com/yourname/api-gateway/internal/store"
)

// AuthConfig defines how the auth middleware behaves.
type AuthConfig struct {
	Store        *store.Store
	JWTSecret    []byte
	APIKeyHeader string
	SkipPaths    map[string]struct{}
}

// Auth enforces JWT or API key authentication for protected routes.
func Auth(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := cfg.SkipPaths[r.URL.Path]; ok || strings.HasPrefix(r.URL.Path, "/healthz") {
				next.ServeHTTP(w, r)
				return
			}

			if identity, ok := authenticateJWT(r, cfg.JWTSecret); ok {
				ctx := auth.WithIdentity(r.Context(), *identity)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if identity, ok := authenticateAPIKey(r, cfg); ok {
				ctx := auth.WithIdentity(r.Context(), identity)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
	}
}

func authenticateJWT(r *http.Request, secret []byte) (*auth.Identity, bool) {
	authz := r.Header.Get("Authorization")
	if authz == "" {
		return nil, false
	}
	if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return nil, false
	}
	token := strings.TrimSpace(authz[7:])
	if token == "" {
		return nil, false
	}
	identity, err := auth.VerifyToken(token, secret)
	if err != nil {
		return nil, false
	}
	return identity, true
}

func authenticateAPIKey(r *http.Request, cfg AuthConfig) (auth.Identity, bool) {
	header := cfg.APIKeyHeader
	if header == "" {
		header = "X-API-Key"
	}
	value := r.Header.Get(header)
	if value == "" {
		return auth.Identity{}, false
	}
	identity, err := cfg.Store.LookupAPIKey(r.Context(), value)
	if err != nil {
		return auth.Identity{}, false
	}
	return auth.IdentityFromStore(identity, "api_key"), true
}
