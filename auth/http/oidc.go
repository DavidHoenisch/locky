package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/locky/auth/core"
)

// OIDCHandlers handles OIDC/OAuth2 endpoints
type OIDCHandlers struct {
	oauthService   core.OAuthService
	keyManager     core.KeyManager
	tenantResolver core.TenantResolver
}

// NewOIDCHandlers creates new OIDC handlers
func NewOIDCHandlers(oauthService core.OAuthService, keyManager core.KeyManager, tenantResolver core.TenantResolver) *OIDCHandlers {
	return &OIDCHandlers{
		oauthService:   oauthService,
		keyManager:     keyManager,
		tenantResolver: tenantResolver,
	}
}

// DiscoveryHandler handles the OIDC discovery endpoint
func (h *OIDCHandlers) DiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := GetTenant(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_not_found", "Tenant not found")
		return
	}

	issuer := fmt.Sprintf("https://%s", r.Host)

	discovery := map[string]interface{}{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/oauth2/authorize",
		"token_endpoint":                        issuer + "/oauth2/token",
		"userinfo_endpoint":                     issuer + "/oauth2/userinfo",
		"jwks_uri":                              issuer + "/oauth2/jwks.json",
		"revocation_endpoint":                   issuer + "/oauth2/revoke",
		"introspection_endpoint":                issuer + "/oauth2/introspect",
		"end_session_endpoint":                  issuer + "/oauth2/logout",
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "client_credentials"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"ES256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "tid", "sid", "email", "email_verified", "name"},
		"code_challenge_methods_supported":      []string{"S256"},
	}

	data, _ := json.Marshal(discovery)
	writeJSON(w, http.StatusOK, data)
}

// JWKSHandler handles the JWKS endpoint
func (h *OIDCHandlers) JWKSHandler(w http.ResponseWriter, r *http.Request) {
	tenant, ok := GetTenant(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_not_found", "Tenant not found")
		return
	}

	jwks, err := h.keyManager.GetPublicJWKS(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get JWKS")
		return
	}

	data, _ := json.Marshal(jwks)
	writeJSON(w, http.StatusOK, data)
}

// AuthorizeHandler handles the OAuth2 authorize endpoint
func (h *OIDCHandlers) AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	tenant, ok := GetTenant(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_not_found", "Tenant not found")
		return
	}

	req := &core.AuthorizeRequest{
		ResponseType:        r.URL.Query().Get("response_type"),
		ClientID:            r.URL.Query().Get("client_id"),
		RedirectURI:         r.URL.Query().Get("redirect_uri"),
		Scope:               r.URL.Query().Get("scope"),
		State:               r.URL.Query().Get("state"),
		CodeChallenge:       r.URL.Query().Get("code_challenge"),
		CodeChallengeMethod: r.URL.Query().Get("code_challenge_method"),
		Nonce:               r.URL.Query().Get("nonce"),
		TenantID:            tenant.ID,
		UserID:              r.URL.Query().Get("user_id"),
	}

	// Validate required parameters
	if req.ResponseType != "code" {
		writeError(w, http.StatusBadRequest, "unsupported_response_type", "Only 'code' response type is supported")
		return
	}

	resp, err := h.oauthService.Authorize(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Redirect back to client
	redirectURL := fmt.Sprintf("%s?code=%s&state=%s", resp.RedirectURI, resp.Code, resp.State)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// TokenHandler handles the OAuth2 token endpoint
func (h *OIDCHandlers) TokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	tenant, ok := GetTenant(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_not_found", "Tenant not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	req := &core.TokenRequest{
		GrantType:    r.FormValue("grant_type"),
		Code:         r.FormValue("code"),
		RedirectURI:  r.FormValue("redirect_uri"),
		CodeVerifier: r.FormValue("code_verifier"),
		RefreshToken: r.FormValue("refresh_token"),
		ClientID:     r.FormValue("client_id"),
		ClientSecret: r.FormValue("client_secret"),
		Scope:        r.FormValue("scope"),
		TenantID:     tenant.ID,
	}

	resp, err := h.oauthService.Token(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}

	data, _ := json.Marshal(resp)
	writeJSON(w, http.StatusOK, data)
}

// UserInfoHandler handles the userinfo endpoint
func (h *OIDCHandlers) UserInfoHandler(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth == "" || len(auth) < 8 || auth[:7] != "Bearer " {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization header")
		return
	}

	token := auth[7:]
	userInfo, err := h.oauthService.UserInfo(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid token")
		return
	}

	data, _ := json.Marshal(userInfo)
	writeJSON(w, http.StatusOK, data)
}

// RevokeHandler handles the token revocation endpoint
func (h *OIDCHandlers) RevokeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	tenant, ok := GetTenant(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_not_found", "Tenant not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	token := r.FormValue("token")
	tokenType := r.FormValue("token_type_hint")

	if err := h.oauthService.Revoke(r.Context(), tenant.ID, token, tokenType); err != nil {
		// RFC 7009: Return 200 even if token was invalid
		// This prevents token scanning attacks
	}

	w.WriteHeader(http.StatusOK)
}

// IntrospectHandler handles the token introspection endpoint
func (h *OIDCHandlers) IntrospectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	tenant, ok := GetTenant(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_not_found", "Tenant not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	token := r.FormValue("token")

	resp, err := h.oauthService.Introspect(r.Context(), tenant.ID, token)
	if err != nil {
		resp = &core.IntrospectResponse{Active: false}
	}

	data, _ := json.Marshal(resp)
	writeJSON(w, http.StatusOK, data)
}

// LogoutHandler handles the RP-initiated logout endpoint
func (h *OIDCHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Simplified logout - in production you'd also clear the session
	redirectURI := r.URL.Query().Get("post_logout_redirect_uri")
	state := r.URL.Query().Get("state")

	if redirectURI == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	redirectURL := redirectURI
	if state != "" {
		redirectURL = fmt.Sprintf("%s?state=%s", redirectURI, state)
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}
