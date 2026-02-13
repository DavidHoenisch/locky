package sessions

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/locky/auth/core"
)

// Service implements core.SessionService
type Service struct {
	sessions core.SessionStore
	clock    core.Clock
	ttl      time.Duration
}

// NewService creates a new session service
func NewService(sessions core.SessionStore, clock core.Clock, ttl time.Duration) *Service {
	return &Service{
		sessions: sessions,
		clock:    clock,
		ttl:      ttl,
	}
}

// Create creates a new session
func (s *Service) Create(ctx context.Context, tenantID, userID, clientID string, ip, userAgent string) (*core.Session, error) {
	now := s.clock.Now()
	session := &core.Session{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		UserID:     userID,
		ClientID:   &clientID,
		IP:         ip,
		UserAgent:  userAgent,
		CreatedAt:  now,
		LastSeenAt: now,
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}

// Validate validates a session
func (s *Service) Validate(ctx context.Context, tenantID, sessionID string) (*core.Session, error) {
	session, err := s.sessions.GetByID(ctx, tenantID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Check if revoked
	if session.RevokedAt != nil {
		return nil, fmt.Errorf("session revoked")
	}

	// Check expiration
	if s.clock.Now().After(session.CreatedAt.Add(s.ttl)) {
		return nil, fmt.Errorf("session expired")
	}

	// Update last seen
	session.LastSeenAt = s.clock.Now()
	if err := s.sessions.Update(ctx, session); err != nil {
		// Log error but don't fail validation
		_ = err
	}

	return session, nil
}

// Revoke revokes a session
func (s *Service) Revoke(ctx context.Context, tenantID, sessionID string) error {
	return s.sessions.Revoke(ctx, tenantID, sessionID)
}
