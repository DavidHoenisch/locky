package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/locky/auth/core"
)

// contextKey is a type for context keys
type contextKey string

const (
	// ContextKeyTenant stores the tenant in the request context
	ContextKeyTenant contextKey = "tenant"
	// ContextKeySession stores the session in the request context
	ContextKeySession contextKey = "session"
	// ContextKeyUser stores the user in the request context
	ContextKeyUser contextKey = "user"
)

// TenantMiddleware resolves the tenant from the host header
type TenantMiddleware struct {
	resolver core.TenantResolver
}

// NewTenantMiddleware creates a new tenant middleware
func NewTenantMiddleware(resolver core.TenantResolver) *TenantMiddleware {
	return &TenantMiddleware{resolver: resolver}
}

// Handler wraps an http.Handler with tenant resolution
func (m *TenantMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		tenant, err := m.resolver.ResolveTenant(r.Context(), host)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "tenant_not_found", "message": %q}`, err.Error()), http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), ContextKeyTenant, tenant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTenant retrieves the tenant from the request context
func GetTenant(ctx context.Context) (*core.Tenant, bool) {
	tenant, ok := ctx.Value(ContextKeyTenant).(*core.Tenant)
	return tenant, ok
}

// AdminAuthMiddleware validates admin API keys
type AdminAuthMiddleware struct {
	keys      core.AdminKeyStore
	configKey string
}

// NewAdminAuthMiddleware creates a new admin auth middleware
func NewAdminAuthMiddleware(keys core.AdminKeyStore, configKey string) *AdminAuthMiddleware {
	return &AdminAuthMiddleware{keys: keys, configKey: configKey}
}

// Handler wraps an http.Handler with admin key validation
func (m *AdminAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-Admin-Key")
		if apiKey == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Missing API key")
			return
		}

		// Check against configured bootstrap key first
		if m.configKey != "" && apiKey == m.configKey {
			next.ServeHTTP(w, r)
			return
		}

		// Check against database keys
		keyHash := hashString(apiKey)
		_, err := m.keys.GetByHash(r.Context(), keyHash)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SessionMiddleware validates session cookies
type SessionMiddleware struct {
	sessions   core.SessionService
	cookieName string
}

// NewSessionMiddleware creates a new session middleware
func NewSessionMiddleware(sessions core.SessionService, cookieName string) *SessionMiddleware {
	return &SessionMiddleware{
		sessions:   sessions,
		cookieName: cookieName,
	}
}

// Handler wraps an http.Handler with session validation
func (m *SessionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(m.cookieName)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		tenant, ok := GetTenant(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		session, err := m.sessions.Validate(r.Context(), tenant.ID, cookie.Value)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), ContextKeySession, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetSession retrieves the session from the request context
func GetSession(ctx context.Context) (*core.Session, bool) {
	session, ok := ctx.Value(ContextKeySession).(*core.Session)
	return session, ok
}

// LoggingMiddleware logs HTTP requests
type LoggingMiddleware struct{}

// Handler wraps an http.Handler with request logging
func (m *LoggingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple logging - in production use structured logging
		next.ServeHTTP(w, r)
	})
}

// CORS middleware handles CORS headers
type CORSMiddleware struct {
	allowedOrigins []string
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(allowedOrigins []string) *CORSMiddleware {
	return &CORSMiddleware{allowedOrigins: allowedOrigins}
}

// Handler wraps an http.Handler with CORS headers
func (m *CORSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range m.allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Key")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Helper functions

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error": %q, "message": %q}`, code, message)
}

func writeJSON(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

func hashString(s string) string {
	// Simple hash for now - use proper crypto in production
	// This is just for API key comparison
	return s
}
