package sessions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/locky/auth/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock SessionStore
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
	if _, ok := m.sessions[session.ID]; ok {
		m.sessions[session.ID] = session
		return nil
	}
	return errors.New("session not found")
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
		if session.TenantID != tenantID {
			continue
		}
		if activeOnly && session.RevokedAt != nil {
			continue
		}
		if userID != nil && session.UserID != *userID {
			continue
		}
		if clientID != nil && (session.ClientID == nil || *session.ClientID != *clientID) {
			continue
		}
		result = append(result, session)
	}
	return result, "", nil
}

func (m *mockSessionStore) DeleteExpired(ctx context.Context, before time.Time) error {
	for k, session := range m.sessions {
		if session.RevokedAt != nil || session.CreatedAt.Before(before) {
			delete(m.sessions, k)
		}
	}
	return nil
}

// Mock Clock
type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

func setupSessionService() (*Service, *mockSessionStore, *mockClock) {
	store := newMockSessionStore()
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	service := NewService(store, clock, 30*24*time.Hour)
	return service, store, clock
}

func TestService_Create(t *testing.T) {
	service, store, clock := setupSessionService()
	ctx := context.Background()

	tests := []struct {
		name      string
		tenantID  string
		userID    string
		clientID  string
		ip        string
		userAgent string
		wantErr   bool
	}{
		{
			name:      "valid_session",
			tenantID:  "tenant-123",
			userID:    "user-456",
			clientID:  "client-789",
			ip:        "192.168.1.1",
			userAgent: "Mozilla/5.0",
			wantErr:   false,
		},
		{
			name:      "minimal_session",
			tenantID:  "tenant-123",
			userID:    "user-456",
			clientID:  "",
			ip:        "",
			userAgent: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := service.Create(ctx, tt.tenantID, tt.userID, tt.clientID, tt.ip, tt.userAgent)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, session)

			// Verify session properties
			assert.NotEmpty(t, session.ID)
			assert.Equal(t, tt.tenantID, session.TenantID)
			assert.Equal(t, tt.userID, session.UserID)
			assert.Equal(t, clock.Now(), session.CreatedAt)
			assert.Equal(t, clock.Now(), session.LastSeenAt)
			assert.Nil(t, session.RevokedAt)

			// Verify session was stored
			stored, err := store.GetByID(ctx, tt.tenantID, session.ID)
			require.NoError(t, err)
			assert.Equal(t, session.ID, stored.ID)
		})
	}
}

func TestService_Validate(t *testing.T) {
	service, store, clock := setupSessionService()
	ctx := context.Background()

	tenantID := "tenant-123"
	userID := "user-456"

	tests := []struct {
		name       string
		setupFunc  func() string
		wantErr    bool
		errContain string
	}{
		{
			name: "valid_session",
			setupFunc: func() string {
				session := &core.Session{
					ID:         "session-1",
					TenantID:   tenantID,
					UserID:     userID,
					ClientID:   strPtr("client-789"),
					IP:         "192.168.1.1",
					UserAgent:  "Mozilla/5.0",
					CreatedAt:  clock.Now(),
					LastSeenAt: clock.Now(),
					RevokedAt:  nil,
				}
				store.Create(ctx, session)
				return session.ID
			},
			wantErr: false,
		},
		{
			name: "revoked_session",
			setupFunc: func() string {
				now := clock.Now()
				session := &core.Session{
					ID:         "session-2",
					TenantID:   tenantID,
					UserID:     userID,
					CreatedAt:  clock.Now(),
					LastSeenAt: clock.Now(),
					RevokedAt:  &now,
				}
				store.Create(ctx, session)
				return session.ID
			},
			wantErr:    true,
			errContain: "revoked",
		},
		{
			name: "expired_session",
			setupFunc: func() string {
				session := &core.Session{
					ID:         "session-3",
					TenantID:   tenantID,
					UserID:     userID,
					CreatedAt:  clock.Now().Add(-40 * 24 * time.Hour), // Created 40 days ago
					LastSeenAt: clock.Now().Add(-40 * 24 * time.Hour),
					RevokedAt:  nil,
				}
				store.Create(ctx, session)
				return session.ID
			},
			wantErr:    true,
			errContain: "expired",
		},
		{
			name: "nonexistent_session",
			setupFunc: func() string {
				return "nonexistent-id"
			},
			wantErr:    true,
			errContain: "get session",
		},
		{
			name: "wrong_tenant",
			setupFunc: func() string {
				session := &core.Session{
					ID:         "session-4",
					TenantID:   "different-tenant",
					UserID:     userID,
					CreatedAt:  clock.Now(),
					LastSeenAt: clock.Now(),
				}
				store.Create(ctx, session)
				return session.ID
			},
			wantErr:    true,
			errContain: "get session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID := tt.setupFunc()
			session, err := service.Validate(ctx, tenantID, sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, session)
			assert.Equal(t, sessionID, session.ID)
		})
	}
}

func TestService_Validate_UpdatesLastSeen(t *testing.T) {
	service, store, clock := setupSessionService()
	ctx := context.Background()

	tenantID := "tenant-123"
	session := &core.Session{
		ID:         "session-1",
		TenantID:   tenantID,
		UserID:     "user-456",
		CreatedAt:  clock.Now(),
		LastSeenAt: clock.Now().Add(-1 * time.Hour), // Last seen 1 hour ago
		RevokedAt:  nil,
	}
	require.NoError(t, store.Create(ctx, session))

	// Capture original last seen (service mutates the same pointer in the store)
	originalLastSeen := session.LastSeenAt

	// Move clock forward
	clock.now = clock.Now().Add(1 * time.Hour)

	// Validate should update last_seen
	validated, err := service.Validate(ctx, tenantID, session.ID)
	require.NoError(t, err)
	assert.True(t, validated.LastSeenAt.After(originalLastSeen))
}

func TestService_Revoke(t *testing.T) {
	service, store, clock := setupSessionService()
	ctx := context.Background()

	tenantID := "tenant-123"
	session := &core.Session{
		ID:         "session-1",
		TenantID:   tenantID,
		UserID:     "user-456",
		CreatedAt:  clock.Now(),
		LastSeenAt: clock.Now(),
		RevokedAt:  nil,
	}
	require.NoError(t, store.Create(ctx, session))

	// Revoke session
	err := service.Revoke(ctx, tenantID, session.ID)
	require.NoError(t, err)

	// Verify session is revoked
	stored, err := store.GetByID(ctx, tenantID, session.ID)
	require.NoError(t, err)
	assert.NotNil(t, stored.RevokedAt)

	// Validation should now fail
	_, err = service.Validate(ctx, tenantID, session.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}

func TestService_Revoke_Nonexistent(t *testing.T) {
	service, _, _ := setupSessionService()
	ctx := context.Background()

	err := service.Revoke(ctx, "tenant-123", "nonexistent-id")
	assert.Error(t, err)
}

func TestService_Revoke_WrongTenant(t *testing.T) {
	service, store, clock := setupSessionService()
	ctx := context.Background()

	session := &core.Session{
		ID:         "session-1",
		TenantID:   "tenant-123",
		UserID:     "user-456",
		CreatedAt:  clock.Now(),
		LastSeenAt: clock.Now(),
	}
	require.NoError(t, store.Create(ctx, session))

	// Try to revoke with wrong tenant
	err := service.Revoke(ctx, "wrong-tenant", session.ID)
	assert.Error(t, err)

	// Verify session is NOT revoked
	stored, err := store.GetByID(ctx, "tenant-123", session.ID)
	require.NoError(t, err)
	assert.Nil(t, stored.RevokedAt)
}

func TestService_Create_MultipleSessions(t *testing.T) {
	service, store, _ := setupSessionService()
	ctx := context.Background()

	tenantID := "tenant-123"
	userID := "user-456"

	// Create multiple sessions for same user
	session1, err := service.Create(ctx, tenantID, userID, "client-1", "192.168.1.1", "Mozilla/5.0")
	require.NoError(t, err)

	session2, err := service.Create(ctx, tenantID, userID, "client-2", "192.168.1.2", "Chrome/120")
	require.NoError(t, err)

	session3, err := service.Create(ctx, tenantID, userID, "client-3", "192.168.1.3", "Safari/17")
	require.NoError(t, err)

	// Verify all sessions exist and have different IDs
	assert.NotEqual(t, session1.ID, session2.ID)
	assert.NotEqual(t, session2.ID, session3.ID)

	// List all sessions for user
	sessions, _, err := store.List(ctx, tenantID, &userID, nil, false, 100, "")
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Revoke one session
	err = service.Revoke(ctx, tenantID, session2.ID)
	require.NoError(t, err)

	// List active sessions only
	activeSessions, _, err := store.List(ctx, tenantID, &userID, nil, true, 100, "")
	require.NoError(t, err)
	assert.Len(t, activeSessions, 2)
}

func TestService_SessionIsolation(t *testing.T) {
	service, _, _ := setupSessionService()
	ctx := context.Background()

	// Create sessions for different tenants
	tenant1Session, err := service.Create(ctx, "tenant-1", "user-1", "client-1", "", "")
	require.NoError(t, err)

	tenant2Session, err := service.Create(ctx, "tenant-2", "user-2", "client-2", "", "")
	require.NoError(t, err)

	// Validate with correct tenants
	_, err = service.Validate(ctx, "tenant-1", tenant1Session.ID)
	assert.NoError(t, err)

	_, err = service.Validate(ctx, "tenant-2", tenant2Session.ID)
	assert.NoError(t, err)

	// Cross-tenant validation should fail
	_, err = service.Validate(ctx, "tenant-1", tenant2Session.ID)
	assert.Error(t, err)

	_, err = service.Validate(ctx, "tenant-2", tenant1Session.ID)
	assert.Error(t, err)
}

func strPtr(s string) *string {
	return &s
}
