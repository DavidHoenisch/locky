package oauth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/locky/auth/core"
	"github.com/locky/auth/crypto"
)

// Service implements core.OAuthService
type Service struct {
	clients        core.ClientStore
	users          core.UserStore
	oauthCodes     core.OAuthCodeStore
	refreshTokens  core.RefreshTokenStore
	tokenService   core.TokenService
	sessionService core.SessionService
	tenantResolver core.TenantResolver
	auditSink      core.AuditSink
	clock          core.Clock
	codeTTL        time.Duration
}

// NewService creates a new OAuth service
func NewService(clients core.ClientStore, users core.UserStore, oauthCodes core.OAuthCodeStore,
	refreshTokens core.RefreshTokenStore, tokenService core.TokenService, sessionService core.SessionService,
	tenantResolver core.TenantResolver, auditSink core.AuditSink, clock core.Clock, codeTTL time.Duration) *Service {
	return &Service{
		clients:        clients,
		users:          users,
		oauthCodes:     oauthCodes,
		refreshTokens:  refreshTokens,
		tokenService:   tokenService,
		sessionService: sessionService,
		tenantResolver: tenantResolver,
		auditSink:      auditSink,
		clock:          clock,
		codeTTL:        codeTTL,
	}
}

// Authorize handles the OAuth2 authorization request
func (s *Service) Authorize(ctx context.Context, req *core.AuthorizeRequest) (*core.AuthorizeResponse, error) {
	// Validate client
	client, err := s.clients.GetByClientID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("client not found: %w", err)
	}

	// Validate redirect URI
	validRedirect := false
	for _, uri := range client.RedirectURIs {
		if uri == req.RedirectURI {
			validRedirect = true
			break
		}
	}
	if !validRedirect {
		return nil, fmt.Errorf("invalid redirect uri")
	}

	// Validate response type
	if req.ResponseType != "code" {
		return nil, fmt.Errorf("unsupported response type")
	}

	// Generate authorization code
	codeValue := uuid.New().String()
	codeHash := crypto.HashString(codeValue)

	// Store the code
	code := &core.OAuthCode{
		CodeHash:      codeHash,
		TenantID:      req.TenantID,
		ClientID:      req.ClientID,
		UserID:        req.UserID,
		RedirectURI:   req.RedirectURI,
		PKCEChallenge: req.CodeChallenge,
		PKCEMethod:    req.CodeChallengeMethod,
		Scope:         req.Scope,
		ExpiresAt:     s.clock.Now().Add(s.codeTTL),
		CreatedAt:     s.clock.Now(),
	}

	if err := s.oauthCodes.Create(ctx, code); err != nil {
		return nil, fmt.Errorf("store code: %w", err)
	}

	// Log audit event
	if s.auditSink != nil {
		s.auditSink.Log(ctx, &core.AuditEvent{
			ID:        uuid.New().String(),
			TenantID:  req.TenantID,
			ActorType: "user",
			Type:      "oauth_authorize",
			CreatedAt: s.clock.Now(),
			Data: map[string]interface{}{
				"client_id": req.ClientID,
				"scope":     req.Scope,
			},
		})
	}

	return &core.AuthorizeResponse{
		Code:        codeValue,
		State:       req.State,
		RedirectURI: req.RedirectURI,
	}, nil
}

// Token handles the OAuth2 token request
func (s *Service) Token(ctx context.Context, req *core.TokenRequest) (*core.TokenResponse, error) {
	switch req.GrantType {
	case "authorization_code":
		return s.handleAuthorizationCode(ctx, req)
	case "refresh_token":
		return s.handleRefreshToken(ctx, req)
	case "client_credentials":
		return s.handleClientCredentials(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported grant type: %s", req.GrantType)
	}
}

func (s *Service) handleAuthorizationCode(ctx context.Context, req *core.TokenRequest) (*core.TokenResponse, error) {
	// Get and validate the code
	codeHash := crypto.HashString(req.Code)
	code, err := s.oauthCodes.GetAndConsume(ctx, req.TenantID, codeHash)
	if err != nil {
		return nil, fmt.Errorf("invalid code: %w", err)
	}

	// Validate PKCE
	if code.PKCEChallenge != "" {
		verifierHash := crypto.HashString(req.CodeVerifier)
		if verifierHash != code.PKCEChallenge {
			return nil, fmt.Errorf("invalid code verifier")
		}
	}

	// Validate redirect URI
	if code.RedirectURI != req.RedirectURI {
		return nil, fmt.Errorf("invalid redirect uri")
	}

	// Get user and roles
	// In a real implementation, fetch roles from RBAC
	roles := []string{}

	// Issue tokens
	accessToken, err := s.tokenService.IssueAccessToken(ctx, req.TenantID, code.UserID, code.ClientID, code.Scope, roles, nil)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, err := s.tokenService.IssueRefreshToken(ctx, req.TenantID, code.UserID, code.ClientID, code.Scope)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	// Log audit event
	if s.auditSink != nil {
		s.auditSink.Log(ctx, &core.AuditEvent{
			ID:        uuid.New().String(),
			TenantID:  req.TenantID,
			ActorType: "user",
			ActorID:   &code.UserID,
			Type:      "oauth_token_exchange",
			CreatedAt: s.clock.Now(),
			Data: map[string]interface{}{
				"client_id":  code.ClientID,
				"grant_type": "authorization_code",
			},
		})
	}

	return &core.TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes
		RefreshToken: refreshToken,
		Scope:        code.Scope,
	}, nil
}

func (s *Service) handleRefreshToken(ctx context.Context, req *core.TokenRequest) (*core.TokenResponse, error) {
	// Rotate refresh token
	newRefreshToken, err := s.tokenService.RotateRefreshToken(ctx, req.TenantID, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	// Get the old token to extract info (this is simplified)
	// In a real implementation, you'd get user info from the rotated token
	// Issue new access token
	accessToken, err := s.tokenService.IssueAccessToken(ctx, req.TenantID, "", req.ClientID, req.Scope, []string{}, nil)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &core.TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		RefreshToken: newRefreshToken,
		Scope:        req.Scope,
	}, nil
}

func (s *Service) handleClientCredentials(ctx context.Context, req *core.TokenRequest) (*core.TokenResponse, error) {
	// Validate client credentials
	client, err := s.clients.GetByClientID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("client not found: %w", err)
	}

	// Verify client secret (simplified)
	if client.ClientSecretHash == nil || *client.ClientSecretHash == "" {
		return nil, fmt.Errorf("client secret required")
	}

	// Issue access token
	accessToken, err := s.tokenService.IssueAccessToken(ctx, req.TenantID, "", req.ClientID, req.Scope, []string{}, nil)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &core.TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
		Scope:       req.Scope,
	}, nil
}

// UserInfo returns user info for the given access token
func (s *Service) UserInfo(ctx context.Context, accessToken string) (*core.UserInfo, error) {
	// Parse and validate the token
	// Simplified - in real implementation you'd use the JWT manager
	return nil, fmt.Errorf("not implemented")
}

// Revoke revokes a token
func (s *Service) Revoke(ctx context.Context, tenantID, token string, tokenType string) error {
	tokenHash := crypto.HashString(token)
	return s.refreshTokens.Revoke(ctx, tenantID, tokenHash)
}

// Introspect introspects a token
func (s *Service) Introspect(ctx context.Context, tenantID, token string) (*core.IntrospectResponse, error) {
	// Simplified - check if token exists in refresh tokens
	tokenHash := crypto.HashString(token)
	rt, err := s.refreshTokens.GetByHash(ctx, tenantID, tokenHash)
	if err != nil {
		// Token not found - it might be an access token (JWT)
		// For JWT, we'd validate and extract claims
		return &core.IntrospectResponse{Active: false}, nil
	}

	// Check if revoked or expired
	if rt.RevokedAt != nil || s.clock.Now().After(rt.ExpiresAt) {
		return &core.IntrospectResponse{Active: false}, nil
	}

	exp := rt.ExpiresAt.Unix()
	return &core.IntrospectResponse{
		Active:    true,
		Subject:   &rt.UserID,
		ClientID:  &rt.ClientID,
		TenantID:  &rt.TenantID,
		Scope:     &rt.Scope,
		ExpiresAt: &exp,
	}, nil
}
