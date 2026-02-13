package tokens

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/locky/auth/core"
	"github.com/locky/auth/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations
type mockKeyStore struct {
	keys map[string]*core.SigningKey
}

func newMockKeyStore() *mockKeyStore {
	return &mockKeyStore{keys: make(map[string]*core.SigningKey)}
}

func (m *mockKeyStore) Create(ctx context.Context, key *core.SigningKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *mockKeyStore) GetActive(ctx context.Context, tenantID string) (*core.SigningKey, error) {
	for _, key := range m.keys {
		if key.TenantID == tenantID && key.Status == "active" {
			return key, nil
		}
	}
	return nil, errors.New("no active key")
}

func (m *mockKeyStore) GetByKID(ctx context.Context, tenantID, kid string) (*core.SigningKey, error) {
	for _, key := range m.keys {
		if key.TenantID == tenantID && key.KID == kid {
			return key, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockKeyStore) ListActive(ctx context.Context, tenantID string) ([]*core.SigningKey, error) {
	var result []*core.SigningKey
	for _, key := range m.keys {
		if key.TenantID == tenantID && (key.Status == "active" || key.Status == "inactive") {
			result = append(result, key)
		}
	}
	return result, nil
}

func (m *mockKeyStore) MarkInactive(ctx context.Context, tenantID, id string) error {
	if key, ok := m.keys[id]; ok {
		key.Status = "inactive"
	}
	return nil
}

func (m *mockKeyStore) MarkRetired(ctx context.Context, tenantID, id string) error {
	if key, ok := m.keys[id]; ok {
		key.Status = "retired"
	}
	return nil
}

type mockOAuthCodeStore struct {
	codes map[string]*core.OAuthCode
}

func newMockOAuthCodeStore() *mockOAuthCodeStore {
	return &mockOAuthCodeStore{codes: make(map[string]*core.OAuthCode)}
}

func (m *mockOAuthCodeStore) Create(ctx context.Context, code *core.OAuthCode) error {
	m.codes[code.CodeHash] = code
	return nil
}

func (m *mockOAuthCodeStore) GetAndConsume(ctx context.Context, tenantID, codeHash string) (*core.OAuthCode, error) {
	if code, ok := m.codes[codeHash]; ok {
		if code.UsedAt != nil {
			return nil, errors.New("code already used")
		}
		if time.Now().After(code.ExpiresAt) {
			return nil, errors.New("code expired")
		}
		now := time.Now()
		code.UsedAt = &now
		return code, nil
	}
	return nil, errors.New("code not found")
}

func (m *mockOAuthCodeStore) DeleteExpired(ctx context.Context, before time.Time) error {
	for k, code := range m.codes {
		if time.Now().After(code.ExpiresAt) || code.UsedAt != nil {
			delete(m.codes, k)
		}
	}
	return nil
}

type mockRefreshTokenStore struct {
	tokens map[string]*core.RefreshToken
}

func newMockRefreshTokenStore() *mockRefreshTokenStore {
	return &mockRefreshTokenStore{tokens: make(map[string]*core.RefreshToken)}
}

func (m *mockRefreshTokenStore) Create(ctx context.Context, token *core.RefreshToken) error {
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockRefreshTokenStore) GetByHash(ctx context.Context, tenantID, hash string) (*core.RefreshToken, error) {
	if token, ok := m.tokens[hash]; ok && token.TenantID == tenantID {
		return token, nil
	}
	return nil, errors.New("token not found")
}

func (m *mockRefreshTokenStore) Revoke(ctx context.Context, tenantID, hash string) error {
	if token, ok := m.tokens[hash]; ok && token.TenantID == tenantID {
		now := time.Now()
		token.RevokedAt = &now
		return nil
	}
	return errors.New("token not found")
}

func (m *mockRefreshTokenStore) DeleteExpired(ctx context.Context, before time.Time) error {
	for k, token := range m.tokens {
		if time.Now().After(token.ExpiresAt) || token.RevokedAt != nil {
			delete(m.tokens, k)
		}
	}
	return nil
}

type mockSessionStore struct {
	sessions map[string]*core.Session
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]*core.Session)}
}

func (m *mockSessionStore) Create(ctx context.Context, session *core.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionStore) GetByID(ctx context.Context, tenantID, id string) (*core.Session, error) {
	if session, ok := m.sessions[id]; ok && session.TenantID == tenantID {
		return session, nil
	}
	return nil, errors.New("session not found")
}

func (m *mockSessionStore) Update(ctx context.Context, session *core.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionStore) Revoke(ctx context.Context, tenantID, id string) error {
	if session, ok := m.sessions[id]; ok && session.TenantID == tenantID {
		now := time.Now()
		session.RevokedAt = &now
		return nil
	}
	return errors.New("session not found")
}

func (m *mockSessionStore) List(ctx context.Context, tenantID string, userID, clientID *string, activeOnly bool, limit int, cursor string) ([]*core.Session, string, error) {
	var result []*core.Session
	for _, session := range m.sessions {
		if session.TenantID == tenantID {
			if activeOnly && session.RevokedAt != nil {
				continue
			}
			result = append(result, session)
		}
	}
	return result, "", nil
}

func (m *mockSessionStore) DeleteExpired(ctx context.Context, before time.Time) error {
	for k, session := range m.sessions {
		if session.RevokedAt != nil {
			delete(m.sessions, k)
		}
	}
	return nil
}

type mockJWTManager struct {
	shouldFail bool
}

func (m *mockJWTManager) Sign(ctx context.Context, tenantID, issuer string, claims map[string]interface{}, ttl time.Duration) (string, error) {
	if m.shouldFail {
		return "", errors.New("signing failed")
	}
	return "mock-jwt-token", nil
}

func (m *mockJWTManager) Verify(ctx context.Context, tenantID, tokenString string) (*core.TokenClaims, error) {
	if m.shouldFail {
		return nil, errors.New("verification failed")
	}
	return &core.TokenClaims{}, nil
}

type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

func setupTokenService() (*Service, *mockOAuthCodeStore, *mockRefreshTokenStore, *mockClock) {
	keyStore := newMockKeyStore()
	oauthCodeStore := newMockOAuthCodeStore()
	refreshTokenStore := newMockRefreshTokenStore()
	sessionStore := newMockSessionStore()
	jwtManager := &mockJWTManager{}
	clock := &mockClock{now: time.Now()}

	service := NewService(
		keyStore,
		oauthCodeStore,
		refreshTokenStore,
		sessionStore,
		jwtManager,
		clock,
		15*time.Minute,
		14*24*time.Hour,
	)

	return service, oauthCodeStore, refreshTokenStore, clock
}

func TestService_IssueAccessToken(t *testing.T) {
	service, _, _, _ := setupTokenService()
	ctx := context.Background()

	tests := []struct {
		name      string
		tenantID  string
		userID    string
		clientID  string
		scope     string
		roles     []string
		sessionID *string
		wantErr   bool
	}{
		{
			name:      "valid_token",
			tenantID:  "tenant-123",
			userID:    "user-456",
			clientID:  "client-789",
			scope:     "openid profile",
			roles:     []string{"admin"},
			sessionID: nil,
			wantErr:   false,
		},
		{
			name:      "token_with_session",
			tenantID:  "tenant-123",
			userID:    "user-456",
			clientID:  "client-789",
			scope:     "openid",
			roles:     []string{},
			sessionID: strPtr("session-abc"),
			wantErr:   false,
		},
		{
			name:      "empty_roles",
			tenantID:  "tenant-123",
			userID:    "user-456",
			clientID:  "client-789",
			scope:     "openid",
			roles:     []string{},
			sessionID: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.IssueAccessToken(ctx, tt.tenantID, tt.userID, tt.clientID, tt.scope, tt.roles, tt.sessionID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, token)
		})
	}
}

func TestService_IssueRefreshToken(t *testing.T) {
	service, _, refreshTokenStore, clock := setupTokenService()
	ctx := context.Background()

	tenantID := "tenant-123"
	userID := "user-456"
	clientID := "client-789"
	scope := "openid profile"

	token, err := service.IssueRefreshToken(ctx, tenantID, userID, clientID, scope)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token was stored
	tokenHash := crypto.HashString(token)
	storedToken, err := refreshTokenStore.GetByHash(ctx, tenantID, tokenHash)
	require.NoError(t, err)
	assert.Equal(t, tenantID, storedToken.TenantID)
	assert.Equal(t, userID, storedToken.UserID)
	assert.Equal(t, clientID, storedToken.ClientID)
	assert.Equal(t, scope, storedToken.Scope)
	assert.True(t, storedToken.ExpiresAt.After(clock.Now()))
	assert.Nil(t, storedToken.RevokedAt)
}

func TestService_RotateRefreshToken(t *testing.T) {
	service, _, refreshTokenStore, clock := setupTokenService()
	ctx := context.Background()

	tenantID := "tenant-123"
	userID := "user-456"
	clientID := "client-789"
	scope := "openid"

	// Create initial token
	oldToken := "old-token-value"
	oldHash := crypto.HashString(oldToken)
	oldRefreshToken := &core.RefreshToken{
		TokenHash: oldHash,
		TenantID:  tenantID,
		UserID:    userID,
		ClientID:  clientID,
		Scope:     scope,
		CreatedAt: clock.Now(),
		ExpiresAt: clock.Now().Add(14 * 24 * time.Hour),
	}
	require.NoError(t, refreshTokenStore.Create(ctx, oldRefreshToken))

	// Rotate token
	newToken, err := service.RotateRefreshToken(ctx, tenantID, oldToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, oldToken, newToken)

	// Verify old token is revoked
	oldStored, err := refreshTokenStore.GetByHash(ctx, tenantID, oldHash)
	require.NoError(t, err)
	assert.NotNil(t, oldStored.RevokedAt)

	// Verify new token exists
	newHash := crypto.HashString(newToken)
	newStored, err := refreshTokenStore.GetByHash(ctx, tenantID, newHash)
	require.NoError(t, err)
	assert.Equal(t, tenantID, newStored.TenantID)
	assert.Equal(t, userID, newStored.UserID)
	assert.Equal(t, oldHash, *newStored.RotatedFromHash)
}

func TestService_RotateRefreshToken_Expired(t *testing.T) {
	service, _, refreshTokenStore, clock := setupTokenService()
	ctx := context.Background()

	tenantID := "tenant-123"

	// Create expired token
	oldToken := "expired-token"
	oldHash := crypto.HashString(oldToken)
	expiredToken := &core.RefreshToken{
		TokenHash: oldHash,
		TenantID:  tenantID,
		UserID:    "user-456",
		ClientID:  "client-789",
		Scope:     "openid",
		CreatedAt: clock.Now().Add(-30 * 24 * time.Hour),
		ExpiresAt: clock.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, refreshTokenStore.Create(ctx, expiredToken))

	// Rotation should fail
	_, err := service.RotateRefreshToken(ctx, tenantID, oldToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestService_RotateRefreshToken_Revoked(t *testing.T) {
	service, _, refreshTokenStore, clock := setupTokenService()
	ctx := context.Background()

	tenantID := "tenant-123"

	// Create revoked token
	oldToken := "revoked-token"
	oldHash := crypto.HashString(oldToken)
	now := clock.Now()
	revokedToken := &core.RefreshToken{
		TokenHash: oldHash,
		TenantID:  tenantID,
		UserID:    "user-456",
		ClientID:  "client-789",
		Scope:     "openid",
		CreatedAt: clock.Now(),
		ExpiresAt: clock.Now().Add(14 * 24 * time.Hour),
		RevokedAt: &now,
	}
	require.NoError(t, refreshTokenStore.Create(ctx, revokedToken))

	// Rotation should fail
	_, err := service.RotateRefreshToken(ctx, tenantID, oldToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}

func TestService_ExchangeCode(t *testing.T) {
	service, oauthCodeStore, _, clock := setupTokenService()
	ctx := context.Background()

	tenantID := "tenant-123"
	codeValue := "test-code"
	codeHash := crypto.HashString(codeValue)
	pkceVerifier := "test-verifier"
	pkceChallenge := crypto.HashString(pkceVerifier)

	// Create authorization code
	authCode := &core.OAuthCode{
		CodeHash:      codeHash,
		TenantID:      tenantID,
		ClientID:      "client-789",
		UserID:        "user-456",
		RedirectURI:   "http://localhost:3000/callback",
		PKCEChallenge: pkceChallenge,
		PKCEMethod:    "S256",
		Scope:         "openid profile",
		ExpiresAt:     clock.Now().Add(10 * time.Minute),
		CreatedAt:     clock.Now(),
	}
	require.NoError(t, oauthCodeStore.Create(ctx, authCode))

	// Exchange code
	resp, err := service.ExchangeCode(ctx, tenantID, codeValue, pkceVerifier, "http://localhost:3000/callback")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Greater(t, resp.ExpiresIn, 0)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "openid profile", resp.Scope)
}

func TestService_ExchangeCode_InvalidPKCE(t *testing.T) {
	service, oauthCodeStore, _, clock := setupTokenService()
	ctx := context.Background()

	tenantID := "tenant-123"
	codeValue := "test-code"
	codeHash := crypto.HashString(codeValue)

	// Create authorization code with PKCE
	authCode := &core.OAuthCode{
		CodeHash:      codeHash,
		TenantID:      tenantID,
		ClientID:      "client-789",
		UserID:        "user-456",
		RedirectURI:   "http://localhost:3000/callback",
		PKCEChallenge: "correct-challenge",
		PKCEMethod:    "S256",
		Scope:         "openid",
		ExpiresAt:     clock.Now().Add(10 * time.Minute),
		CreatedAt:     clock.Now(),
	}
	require.NoError(t, oauthCodeStore.Create(ctx, authCode))

	// Try with wrong verifier
	_, err := service.ExchangeCode(ctx, tenantID, codeValue, "wrong-verifier", "http://localhost:3000/callback")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code verifier")
}

func TestService_ExchangeCode_WrongRedirectURI(t *testing.T) {
	service, oauthCodeStore, _, clock := setupTokenService()
	ctx := context.Background()

	tenantID := "tenant-123"
	codeValue := "test-code"
	codeHash := crypto.HashString(codeValue)
	pkceVerifier := "test-verifier"
	pkceChallenge := crypto.HashString(pkceVerifier)

	// Create authorization code
	authCode := &core.OAuthCode{
		CodeHash:      codeHash,
		TenantID:      tenantID,
		ClientID:      "client-789",
		UserID:        "user-456",
		RedirectURI:   "http://localhost:3000/callback",
		PKCEChallenge: pkceChallenge,
		PKCEMethod:    "S256",
		Scope:         "openid",
		ExpiresAt:     clock.Now().Add(10 * time.Minute),
		CreatedAt:     clock.Now(),
	}
	require.NoError(t, oauthCodeStore.Create(ctx, authCode))

	// Try with wrong redirect URI
	_, err := service.ExchangeCode(ctx, tenantID, codeValue, pkceVerifier, "http://evil.com/callback")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redirect uri")
}

func TestService_ExchangeCode_AlreadyUsed(t *testing.T) {
	service, oauthCodeStore, _, clock := setupTokenService()
	ctx := context.Background()

	tenantID := "tenant-123"
	codeValue := "test-code"
	codeHash := crypto.HashString(codeValue)
	pkceVerifier := "test-verifier"
	pkceChallenge := crypto.HashString(pkceVerifier)
	now := clock.Now()

	// Create already-used authorization code
	authCode := &core.OAuthCode{
		CodeHash:      codeHash,
		TenantID:      tenantID,
		ClientID:      "client-789",
		UserID:        "user-456",
		RedirectURI:   "http://localhost:3000/callback",
		PKCEChallenge: pkceChallenge,
		PKCEMethod:    "S256",
		Scope:         "openid",
		ExpiresAt:     clock.Now().Add(10 * time.Minute),
		UsedAt:        &now,
		CreatedAt:     clock.Now(),
	}
	require.NoError(t, oauthCodeStore.Create(ctx, authCode))

	// Exchange should fail
	_, err := service.ExchangeCode(ctx, tenantID, codeValue, pkceVerifier, "http://localhost:3000/callback")
	assert.Error(t, err)
}

func strPtr(s string) *string {
	return &s
}
