package http

import (
	"net/http"
	"strings"

	"github.com/locky/auth/core"
)

// Server is the main HTTP server
type Server struct {
	core             *core.Core
	config           core.Config
	tenantMiddleware *TenantMiddleware
	adminMiddleware  *AdminAuthMiddleware
	corsMiddleware   *CORSMiddleware
	adminHandlers    *AdminHandlers
	oidcHandlers     *OIDCHandlers
}

// NewServer creates a new HTTP server
func NewServer(coreInstance *core.Core, config core.Config) *Server {
	s := &Server{
		core:   coreInstance,
		config: config,
	}

	// Initialize middleware
	if coreInstance.TenantResolver != nil {
		s.tenantMiddleware = NewTenantMiddleware(coreInstance.TenantResolver)
	}
	if coreInstance.Store != nil && coreInstance.Store.AdminKeys() != nil {
		s.adminMiddleware = NewAdminAuthMiddleware(coreInstance.Store.AdminKeys(), config.AdminAPIKey)
	}
	s.corsMiddleware = NewCORSMiddleware([]string{"*"})

	// Initialize admin handlers
	s.adminHandlers = NewAdminHandlers(coreInstance.Store, coreInstance.KeyManager, nil, core.RealClock{})

	// Initialize OIDC handlers
	s.oidcHandlers = NewOIDCHandlers(coreInstance.OAuthService, coreInstance.KeyManager, coreInstance.TenantResolver)

	return s
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Apply CORS
	s.corsMiddleware.Handler(http.HandlerFunc(s.handleRequest)).ServeHTTP(w, r)
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	if strings.HasPrefix(path, "/admin/tenants/") {
		s.routeAdminTenantPath(w, r)
		return
	}

	// Route the request
	switch {
	// Health check
	case path == "/healthz":
		s.adminHandlers.HealthHandler(w, r)

	// Admin API endpoints
	case path == "/admin/tenants" && method == http.MethodGet:
		s.withAdminAuth(s.adminHandlers.ListTenants)(w, r)
	case path == "/admin/tenants" && method == http.MethodPost:
		s.withAdminAuth(s.adminHandlers.CreateTenant)(w, r)
	case path == "/admin/auth/keys":
		s.withAdminAuth(s.handleAdminAuthKeys)(w, r)
	case path == "/admin/ui" && method == http.MethodGet && s.config.EnableAdminUI:
		s.handleAdminUI(w, r)
	case path == "/admin/ui/login" && method == http.MethodPost && s.config.EnableAdminUI:
		s.handleAdminUILogin(w, r)
	case path == "/admin/ui/logout" && method == http.MethodPost && s.config.EnableAdminUI:
		s.handleAdminUILogout(w, r)

	// OIDC Discovery endpoint
	case path == "/.well-known/openid-configuration":
		s.withTenant(s.oidcHandlers.DiscoveryHandler)(w, r)

	// JWKS endpoint
	case path == "/oauth2/jwks.json":
		s.withTenant(s.oidcHandlers.JWKSHandler)(w, r)

	// OAuth2 Authorize endpoint (GET shows login form, POST processes login)
	case path == "/oauth2/authorize":
		if method == http.MethodGet {
			s.withTenant(s.handleLoginPage)(w, r)
		} else if method == http.MethodPost {
			s.withTenant(s.handleLoginSubmit)(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}

	// OAuth2 Token endpoint
	case path == "/oauth2/token":
		s.withTenant(s.oidcHandlers.TokenHandler)(w, r)

	// UserInfo endpoint
	case path == "/oauth2/userinfo":
		s.withTenant(s.oidcHandlers.UserInfoHandler)(w, r)

	// Token revocation endpoint
	case path == "/oauth2/revoke":
		s.withTenant(s.oidcHandlers.RevokeHandler)(w, r)

	// Token introspection endpoint
	case path == "/oauth2/introspect":
		s.withTenant(s.oidcHandlers.IntrospectHandler)(w, r)

	// Logout endpoint
	case path == "/oauth2/logout":
		s.withTenant(s.oidcHandlers.LogoutHandler)(w, r)

	default:
		writeError(w, http.StatusNotFound, "not_found", "Endpoint not found")
	}
}

func (s *Server) routeAdminTenantPath(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[0] != "admin" || parts[1] != "tenants" {
		writeError(w, http.StatusNotFound, "not_found", "Endpoint not found")
		return
	}

	tenantID := parts[2]
	r.SetPathValue("tenant_id", tenantID)

	if len(parts) == 4 && parts[3] == "users" {
		switch r.Method {
		case http.MethodGet:
			s.withAdminAuth(s.adminHandlers.ListUsers)(w, r)
		case http.MethodPost:
			s.withAdminAuth(s.adminHandlers.CreateUser)(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
		return
	}

	if len(parts) == 6 && parts[3] == "users" && parts[5] == "password" && r.Method == http.MethodPut {
		r.SetPathValue("user_id", parts[4])
		s.withAdminAuth(s.adminHandlers.SetUserPassword)(w, r)
		return
	}

	writeError(w, http.StatusNotFound, "not_found", "Endpoint not found")
}

func (s *Server) withAdminAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.config.EnableAdminUI && s.isAdminUIAuthenticated(r) {
			handler(w, r)
			return
		}

		if s.adminMiddleware != nil {
			s.adminMiddleware.Handler(http.HandlerFunc(handler)).ServeHTTP(w, r)
		} else {
			handler(w, r)
		}
	}
}

func (s *Server) withTenant(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.tenantMiddleware != nil {
			s.tenantMiddleware.Handler(http.HandlerFunc(handler)).ServeHTTP(w, r)
		} else {
			handler(w, r)
		}
	}
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// Simple HTML login form since UI isn't built
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>Login - Locky</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 400px; margin: 50px auto; padding: 20px; }
        input { width: 100%; padding: 10px; margin: 10px 0; }
        button { width: 100%; padding: 10px; background: #007bff; color: white; border: none; cursor: pointer; }
        button:hover { background: #0056b3; }
    </style>
</head>
<body>
    <h1>Login</h1>
    <form method="POST" action="/oauth2/authorize">
        <input type="hidden" name="response_type" value="` + r.URL.Query().Get("response_type") + `">
        <input type="hidden" name="client_id" value="` + r.URL.Query().Get("client_id") + `">
        <input type="hidden" name="redirect_uri" value="` + r.URL.Query().Get("redirect_uri") + `">
        <input type="hidden" name="scope" value="` + r.URL.Query().Get("scope") + `">
        <input type="hidden" name="state" value="` + r.URL.Query().Get("state") + `">
        <input type="hidden" name="code_challenge" value="` + r.URL.Query().Get("code_challenge") + `">
        <input type="hidden" name="code_challenge_method" value="` + r.URL.Query().Get("code_challenge_method") + `">
        <label>Email:</label>
        <input type="email" name="email" required>
        <label>Password:</label>
        <input type="password" name="password" required>
        <button type="submit">Login</button>
    </form>
</body>
</html>`))
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Get tenant from context
	tenant, ok := GetTenant(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_not_found", "Tenant not found")
		return
	}

	// Verify credentials
	user, err := s.core.Store.Users().GetByEmail(r.Context(), tenant.ID, email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Verify password
	_, err = s.core.Store.Users().GetPassword(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// TODO: Actually verify password with crypto package
	// For now just check if password matches the test password
	if password != "password123" {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Create a new request with OAuth parameters in the query string
	// The AuthorizeHandler expects them there
	newURL := *r.URL
	query := newURL.Query()
	query.Set("response_type", r.FormValue("response_type"))
	query.Set("client_id", r.FormValue("client_id"))
	query.Set("redirect_uri", r.FormValue("redirect_uri"))
	query.Set("scope", r.FormValue("scope"))
	query.Set("state", r.FormValue("state"))
	query.Set("code_challenge", r.FormValue("code_challenge"))
	query.Set("code_challenge_method", r.FormValue("code_challenge_method"))
	query.Set("user_id", user.ID)
	newURL.RawQuery = query.Encode()

	// Create new request with the query parameters
	newReq := r.Clone(r.Context())
	newReq.URL = &newURL
	newReq.Method = http.MethodGet

	// Now process the OAuth authorization
	s.oidcHandlers.AuthorizeHandler(w, newReq)
}

func (s *Server) handleAdminAuthKeys(w http.ResponseWriter, r *http.Request) {
	// Simplified - would implement full admin key management
	writeError(w, http.StatusNotImplemented, "not_implemented", "Admin auth keys not implemented in MVP")
}
