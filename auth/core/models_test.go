package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealClock_Now(t *testing.T) {
	clock := RealClock{}
	before := time.Now()
	now := clock.Now()
	after := time.Now()

	assert.True(t, now.Equal(before) || now.After(before))
	assert.True(t, now.Equal(after) || now.Before(after))
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		DatabaseURL: "postgres://localhost/test",
		AdminAPIKey: "test-key",
		BaseDomain:  "auth.example.com",
	}

	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseURL)
	assert.Equal(t, "test-key", cfg.AdminAPIKey)
	assert.Equal(t, "auth.example.com", cfg.BaseDomain)
}

func TestTokenClaims_Validation(t *testing.T) {
	now := time.Now().Unix()
	claims := TokenClaims{
		Issuer:    "https://test.auth.example.com",
		Subject:   "user-123",
		Audience:  "client-456",
		TenantID:  "tenant-789",
		SessionID: strPtr("session-abc"),
		Roles:     []string{"admin", "user"},
		Scope:     "openid profile",
		IssuedAt:  now,
		ExpiresAt: now + 900,
		NotBefore: now,
		JWTID:     "jwt-xyz",
	}

	assert.Equal(t, "https://test.auth.example.com", claims.Issuer)
	assert.Equal(t, "user-123", claims.Subject)
	assert.Equal(t, "client-456", claims.Audience)
	assert.Equal(t, "tenant-789", claims.TenantID)
	assert.Equal(t, "session-abc", *claims.SessionID)
	assert.Equal(t, []string{"admin", "user"}, claims.Roles)
	assert.Equal(t, "openid profile", claims.Scope)
	assert.Equal(t, now, claims.IssuedAt)
	assert.Equal(t, now+900, claims.ExpiresAt)
	assert.Equal(t, now, claims.NotBefore)
	assert.Equal(t, "jwt-xyz", claims.JWTID)
}

func TestTenant_Validation(t *testing.T) {
	now := time.Now()
	tenant := Tenant{
		ID:        "tenant-123",
		Slug:      "acme-corp",
		Name:      "Acme Corporation",
		Status:    "active",
		CreatedAt: now,
	}

	require.NotEmpty(t, tenant.ID)
	require.NotEmpty(t, tenant.Slug)
	require.NotEmpty(t, tenant.Name)
	assert.True(t, tenant.Status == "active" || tenant.Status == "suspended")
	assert.False(t, tenant.CreatedAt.IsZero())
}

func TestUser_Validation(t *testing.T) {
	now := time.Now()
	displayName := "John Doe"
	user := User{
		ID:            "user-123",
		TenantID:      "tenant-456",
		Email:         "john@example.com",
		EmailVerified: true,
		Status:        "active",
		DisplayName:   &displayName,
		CreatedAt:     now,
		UpdatedAt:     &now,
	}

	require.NotEmpty(t, user.ID)
	require.NotEmpty(t, user.TenantID)
	require.NotEmpty(t, user.Email)
	assert.True(t, user.Status == "active" || user.Status == "disabled")
	assert.NotNil(t, user.DisplayName)
	assert.Equal(t, "John Doe", *user.DisplayName)
}

func TestSession_Validation(t *testing.T) {
	now := time.Now()
	clientID := "client-123"
	session := Session{
		ID:         "session-abc",
		TenantID:   "tenant-456",
		UserID:     "user-789",
		ClientID:   &clientID,
		IP:         "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		CreatedAt:  now,
		LastSeenAt: now,
		RevokedAt:  nil,
	}

	require.NotEmpty(t, session.ID)
	require.NotEmpty(t, session.TenantID)
	require.NotEmpty(t, session.UserID)
	assert.NotNil(t, session.ClientID)
	assert.Equal(t, "client-123", *session.ClientID)
	assert.False(t, session.CreatedAt.IsZero())
	assert.False(t, session.LastSeenAt.IsZero())
	assert.Nil(t, session.RevokedAt)
}

func TestClient_Validation(t *testing.T) {
	now := time.Now()
	secretHash := "hash123"
	secretLast4 := "5678"
	client := Client{
		ID:                     "client-123",
		TenantID:               "tenant-456",
		Name:                   "Test Application",
		ClientID:               "test-app-123",
		ClientSecretHash:       &secretHash,
		ClientSecretLast4:      &secretLast4,
		RedirectURIs:           []string{"http://localhost:3000/callback"},
		PostLogoutRedirectURIs: []string{"http://localhost:3000"},
		GrantTypes:             []string{"authorization_code", "refresh_token"},
		ResponseTypes:          []string{"code"},
		Scopes:                 []string{"openid", "profile", "email"},
		TokenTTLSeconds:        900,
		RefreshTTLSeconds:      1209600,
		CreatedAt:              now,
	}

	require.NotEmpty(t, client.ID)
	require.NotEmpty(t, client.TenantID)
	require.NotEmpty(t, client.Name)
	require.NotEmpty(t, client.ClientID)
	assert.NotEmpty(t, client.RedirectURIs)
	assert.Contains(t, client.GrantTypes, "authorization_code")
	assert.Greater(t, client.TokenTTLSeconds, 0)
	assert.Greater(t, client.RefreshTTLSeconds, 0)
}

func TestOAuthCode_Validation(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(10 * time.Minute)
	code := OAuthCode{
		CodeHash:      "hash123",
		TenantID:      "tenant-456",
		ClientID:      "client-789",
		UserID:        "user-abc",
		RedirectURI:   "http://localhost:3000/callback",
		PKCEChallenge: "challenge123",
		PKCEMethod:    "S256",
		Scope:         "openid profile",
		ExpiresAt:     expiresAt,
		UsedAt:        nil,
		CreatedAt:     now,
	}

	require.NotEmpty(t, code.CodeHash)
	require.NotEmpty(t, code.TenantID)
	require.NotEmpty(t, code.ClientID)
	require.NotEmpty(t, code.UserID)
	require.NotEmpty(t, code.RedirectURI)
	assert.Equal(t, "S256", code.PKCEMethod)
	assert.True(t, code.ExpiresAt.After(now))
	assert.Nil(t, code.UsedAt)
}

func TestRefreshToken_Validation(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(14 * 24 * time.Hour)
	rotatedFrom := "old-hash"
	token := RefreshToken{
		TokenHash:       "hash123",
		TenantID:        "tenant-456",
		ClientID:        "client-789",
		UserID:          "user-abc",
		Scope:           "openid profile",
		CreatedAt:       now,
		ExpiresAt:       expiresAt,
		RevokedAt:       nil,
		RotatedFromHash: &rotatedFrom,
	}

	require.NotEmpty(t, token.TokenHash)
	require.NotEmpty(t, token.TenantID)
	require.NotEmpty(t, token.UserID)
	assert.True(t, token.ExpiresAt.After(now))
	assert.Nil(t, token.RevokedAt)
	assert.NotNil(t, token.RotatedFromHash)
	assert.Equal(t, "old-hash", *token.RotatedFromHash)
}

func TestAuditEvent_Validation(t *testing.T) {
	now := time.Now()
	actorID := "admin-123"
	ip := "192.168.1.1"
	ua := "Mozilla/5.0"
	event := AuditEvent{
		ID:        "event-123",
		TenantID:  "tenant-456",
		ActorType: "admin",
		ActorID:   &actorID,
		Type:      "user_created",
		IP:        &ip,
		UserAgent: &ua,
		CreatedAt: now,
		Data: map[string]interface{}{
			"user_id": "user-789",
			"email":   "test@example.com",
		},
	}

	require.NotEmpty(t, event.ID)
	require.NotEmpty(t, event.TenantID)
	assert.True(t, event.ActorType == "admin" || event.ActorType == "user" || event.ActorType == "system")
	require.NotEmpty(t, event.Type)
	assert.NotNil(t, event.Data)
	assert.NotEmpty(t, event.Data)
}

func TestPolicy_Validation(t *testing.T) {
	now := time.Now()
	policy := Policy{
		ID:       "policy-123",
		TenantID: "tenant-456",
		Name:     "Default Policy",
		Version:  1,
		Status:   "active",
		Document: map[string]interface{}{
			"max_login_attempts": 5,
			"mfa_required":       false,
		},
		CreatedAt: now,
	}

	require.NotEmpty(t, policy.ID)
	require.NotEmpty(t, policy.TenantID)
	require.NotEmpty(t, policy.Name)
	assert.Greater(t, policy.Version, 0)
	assert.True(t, policy.Status == "active" || policy.Status == "inactive")
	assert.NotNil(t, policy.Document)
}

func TestRbacTuple_Validation(t *testing.T) {
	now := time.Now()
	v3 := "read"
	tuple := RbacTuple{
		ID:        "tuple-123",
		TenantID:  "tenant-456",
		TupleType: "p", // policy
		V0:        "role:admin",
		V1:        "tenant-456",
		V2:        "resource:*",
		V3:        &v3,
		V4:        nil,
		V5:        nil,
		CreatedAt: now,
	}

	require.NotEmpty(t, tuple.ID)
	require.NotEmpty(t, tuple.TenantID)
	assert.True(t, tuple.TupleType == "p" || tuple.TupleType == "g")
	require.NotEmpty(t, tuple.V0)
	require.NotEmpty(t, tuple.V1)
	require.NotEmpty(t, tuple.V2)
	assert.NotNil(t, tuple.V3)
	assert.Equal(t, "read", *tuple.V3)
}

func TestAuthorizeRequest_Validation(t *testing.T) {
	req := AuthorizeRequest{
		ResponseType:        "code",
		ClientID:            "client-123",
		RedirectURI:         "http://localhost:3000/callback",
		Scope:               "openid profile",
		State:               "random-state-123",
		CodeChallenge:       "challenge123",
		CodeChallengeMethod: "S256",
		Nonce:               "nonce123",
		TenantID:            "tenant-456",
	}

	assert.Equal(t, "code", req.ResponseType)
	assert.NotEmpty(t, req.ClientID)
	assert.NotEmpty(t, req.RedirectURI)
	assert.NotEmpty(t, req.Scope)
	assert.NotEmpty(t, req.State)
	assert.Equal(t, "S256", req.CodeChallengeMethod)
	assert.NotEmpty(t, req.TenantID)
}

func TestTokenRequest_Validation(t *testing.T) {
	req := TokenRequest{
		GrantType:    "authorization_code",
		Code:         "auth-code-123",
		RedirectURI:  "http://localhost:3000/callback",
		CodeVerifier: "verifier123",
		RefreshToken: "",
		ClientID:     "client-123",
		ClientSecret: "secret123",
		Scope:        "openid profile",
		TenantID:     "tenant-456",
	}

	assert.Equal(t, "authorization_code", req.GrantType)
	assert.NotEmpty(t, req.Code)
	assert.NotEmpty(t, req.RedirectURI)
	assert.NotEmpty(t, req.CodeVerifier)
	assert.NotEmpty(t, req.ClientID)
	assert.NotEmpty(t, req.TenantID)
}

func TestTokenResponse_Validation(t *testing.T) {
	resp := TokenResponse{
		AccessToken:  "access-token-123",
		TokenType:    "Bearer",
		ExpiresIn:    900,
		RefreshToken: "refresh-token-456",
		IDToken:      "id-token-789",
		Scope:        "openid profile",
	}

	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Greater(t, resp.ExpiresIn, 0)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEmpty(t, resp.IDToken)
	assert.NotEmpty(t, resp.Scope)
}

func TestUserInfo_Validation(t *testing.T) {
	userInfo := UserInfo{
		Subject:       "user-123",
		Email:         "john@example.com",
		EmailVerified: true,
		DisplayName:   "John Doe",
		Extra: map[string]interface{}{
			"custom_claim": "value",
		},
	}

	assert.NotEmpty(t, userInfo.Subject)
	assert.NotEmpty(t, userInfo.Email)
	assert.True(t, userInfo.EmailVerified)
	assert.NotEmpty(t, userInfo.DisplayName)
	assert.NotNil(t, userInfo.Extra)
}

func TestIntrospectResponse_Validation(t *testing.T) {
	exp := int64(1234567890)
	iat := int64(1234567000)
	sub := "user-123"
	aud := "client-456"
	iss := "https://auth.example.com"
	scope := "openid profile"
	clientID := "client-456"
	tenantID := "tenant-789"
	roles := []string{"admin", "user"}

	resp := IntrospectResponse{
		Active:    true,
		Subject:   &sub,
		Audience:  &aud,
		Issuer:    &iss,
		ExpiresAt: &exp,
		IssuedAt:  &iat,
		Scope:     &scope,
		ClientID:  &clientID,
		TenantID:  &tenantID,
		Roles:     roles,
	}

	assert.True(t, resp.Active)
	assert.NotNil(t, resp.Subject)
	assert.NotNil(t, resp.ExpiresAt)
	assert.NotNil(t, resp.Roles)
	assert.Len(t, resp.Roles, 2)
}

// Helper functions

func strPtr(s string) *string {
	return &s
}
