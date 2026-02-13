package tokens

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/locky/auth/core"
	"github.com/locky/auth/crypto"
)

// JWTSigner signs and verifies JWTs (implemented by *crypto.JWTManager and mocks)
type JWTSigner interface {
	Sign(ctx context.Context, tenantID, issuer string, claims map[string]interface{}, ttl time.Duration) (string, error)
	Verify(ctx context.Context, tenantID, tokenString string) (*core.TokenClaims, error)
}

// Service implements core.TokenService
type Service struct {
	keys          core.SigningKeyStore
	oauthCodes    core.OAuthCodeStore
	refreshTokens core.RefreshTokenStore
	sessions      core.SessionStore
	jwtManager    JWTSigner
	clock         core.Clock
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewService creates a new token service
func NewService(keys core.SigningKeyStore, oauthCodes core.OAuthCodeStore, refreshTokens core.RefreshTokenStore, sessions core.SessionStore, jwtManager JWTSigner, clock core.Clock, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		keys:          keys,
		oauthCodes:    oauthCodes,
		refreshTokens: refreshTokens,
		sessions:      sessions,
		jwtManager:    jwtManager,
		clock:         clock,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

// IssueAccessToken issues a new access token
func (s *Service) IssueAccessToken(ctx context.Context, tenantID, userID, clientID string, scope string, roles []string, sessionID *string) (string, error) {
	// Get tenant info for issuer
	// In a real implementation, you'd fetch the tenant's domain
	issuer := fmt.Sprintf("https://%s.auth.example.com", tenantID)

	claims := map[string]interface{}{
		"sub":   userID,
		"aud":   clientID,
		"scope": scope,
		"roles": roles,
	}
	if sessionID != nil {
		claims["sid"] = *sessionID
	}

	token, err := s.jwtManager.Sign(ctx, tenantID, issuer, claims, s.accessTTL)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return token, nil
}

// IssueRefreshToken issues a new refresh token
func (s *Service) IssueRefreshToken(ctx context.Context, tenantID, userID, clientID string, scope string) (string, error) {
	tokenValue := uuid.New().String()
	tokenHash := crypto.HashString(tokenValue)

	rt := &core.RefreshToken{
		TokenHash: tokenHash,
		TenantID:  tenantID,
		ClientID:  clientID,
		UserID:    userID,
		Scope:     scope,
		CreatedAt: s.clock.Now(),
		ExpiresAt: s.clock.Now().Add(s.refreshTTL),
	}

	if err := s.refreshTokens.Create(ctx, rt); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}

	return tokenValue, nil
}

// ValidateAccessToken validates an access token
func (s *Service) ValidateAccessToken(ctx context.Context, token string) (*core.TokenClaims, error) {
	// Extract tenant ID from token claims (without full validation first)
	// This is a simplified version - you'd parse the JWT header to get the tenant
	return nil, fmt.Errorf("not implemented")
}

// RotateRefreshToken rotates a refresh token (issues a new one, revokes the old)
func (s *Service) RotateRefreshToken(ctx context.Context, tenantID, oldToken string) (string, error) {
	oldHash := crypto.HashString(oldToken)

	// Get the old token
	rt, err := s.refreshTokens.GetByHash(ctx, tenantID, oldHash)
	if err != nil {
		return "", fmt.Errorf("get refresh token: %w", err)
	}

	// Check if revoked or expired
	if rt.RevokedAt != nil {
		return "", fmt.Errorf("refresh token revoked")
	}
	if s.clock.Now().After(rt.ExpiresAt) {
		return "", fmt.Errorf("refresh token expired")
	}

	// Revoke the old token
	if err := s.refreshTokens.Revoke(ctx, tenantID, oldHash); err != nil {
		return "", fmt.Errorf("revoke old token: %w", err)
	}

	// Issue a new token
	newToken := uuid.New().String()
	newHash := crypto.HashString(newToken)

	newRT := &core.RefreshToken{
		TokenHash:       newHash,
		TenantID:        rt.TenantID,
		ClientID:        rt.ClientID,
		UserID:          rt.UserID,
		Scope:           rt.Scope,
		CreatedAt:       s.clock.Now(),
		ExpiresAt:       s.clock.Now().Add(s.refreshTTL),
		RotatedFromHash: &oldHash,
	}

	if err := s.refreshTokens.Create(ctx, newRT); err != nil {
		return "", fmt.Errorf("store new refresh token: %w", err)
	}

	return newToken, nil
}

// ExchangeCode exchanges an authorization code for tokens
func (s *Service) ExchangeCode(ctx context.Context, tenantID, code, codeVerifier, redirectURI string) (*core.TokenResponse, error) {
	codeHash := crypto.HashString(code)

	// Get and consume the code
	oauthCode, err := s.oauthCodes.GetAndConsume(ctx, tenantID, codeHash)
	if err != nil {
		return nil, fmt.Errorf("get code: %w", err)
	}

	// Verify PKCE
	if oauthCode.PKCEChallenge != "" {
		// This is a simplified check - proper PKCE uses S256 hash
		if crypto.HashString(codeVerifier) != oauthCode.PKCEChallenge {
			return nil, fmt.Errorf("invalid code verifier")
		}
	}

	// Verify redirect URI
	if oauthCode.RedirectURI != redirectURI {
		return nil, fmt.Errorf("invalid redirect uri")
	}

	// Get user's roles (simplified - you'd fetch from RBAC)
	roles := []string{}

	// Issue access token
	accessToken, err := s.IssueAccessToken(ctx, tenantID, oauthCode.UserID, oauthCode.ClientID, oauthCode.Scope, roles, nil)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	// Issue refresh token
	refreshToken, err := s.IssueRefreshToken(ctx, tenantID, oauthCode.UserID, oauthCode.ClientID, oauthCode.Scope)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	return &core.TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTTL.Seconds()),
		RefreshToken: refreshToken,
		Scope:        oauthCode.Scope,
	}, nil
}
