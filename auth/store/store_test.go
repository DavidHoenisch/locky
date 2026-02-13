package store

import (
	"context"
	"testing"
	"time"

	"github.com/locky/auth/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type StoreTestSuite struct {
	suite.Suite
	db    *gorm.DB
	store *GormStore
	ctx   context.Context
}

func (s *StoreTestSuite) SetupTest() {
	var err error
	s.db, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(s.T(), err)

	s.store = NewWithDB(s.db)
	err = s.store.AutoMigrate()
	require.NoError(s.T(), err)

	s.ctx = context.Background()
}

func (s *StoreTestSuite) TearDownTest() {
	sqlDB, err := s.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}

func (s *StoreTestSuite) TestTenantStore() {
	tenant := &core.Tenant{
		ID:        "tenant-123",
		Slug:      "acme-corp",
		Name:      "Acme Corporation",
		Status:    "active",
		CreatedAt: time.Now(),
	}

	// Create
	err := s.store.Tenants().Create(s.ctx, tenant)
	s.Require().NoError(err)

	// Get by ID
	retrieved, err := s.store.Tenants().GetByID(s.ctx, tenant.ID)
	s.Require().NoError(err)
	s.Equal(tenant.ID, retrieved.ID)
	s.Equal(tenant.Slug, retrieved.Slug)

	// Get by slug
	retrieved, err = s.store.Tenants().GetBySlug(s.ctx, tenant.Slug)
	s.Require().NoError(err)
	s.Equal(tenant.ID, retrieved.ID)

	// Update
	tenant.Name = "Acme Corp Updated"
	err = s.store.Tenants().Update(s.ctx, tenant)
	s.Require().NoError(err)

	retrieved, err = s.store.Tenants().GetByID(s.ctx, tenant.ID)
	s.Require().NoError(err)
	s.Equal("Acme Corp Updated", retrieved.Name)

	// List
	tenants, cursor, err := s.store.Tenants().List(s.ctx, 10, "")
	s.Require().NoError(err)
	s.Len(tenants, 1)
	s.Empty(cursor)
}

func (s *StoreTestSuite) TestUserStore() {
	// First create a tenant
	tenant := &core.Tenant{
		ID:        "tenant-123",
		Slug:      "acme-corp",
		Name:      "Acme Corporation",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	err := s.store.Tenants().Create(s.ctx, tenant)
	s.Require().NoError(err)

	displayName := "John Doe"
	user := &core.User{
		ID:            "user-456",
		TenantID:      tenant.ID,
		Email:         "john@example.com",
		EmailVerified: true,
		Status:        "active",
		DisplayName:   &displayName,
		CreatedAt:     time.Now(),
		UpdatedAt:     nil,
	}

	// Create
	err = s.store.Users().Create(s.ctx, user)
	s.Require().NoError(err)

	// Get by ID
	retrieved, err := s.store.Users().GetByID(s.ctx, tenant.ID, user.ID)
	s.Require().NoError(err)
	s.Equal(user.ID, retrieved.ID)
	s.Equal(user.Email, retrieved.Email)

	// Get by email
	retrieved, err = s.store.Users().GetByEmail(s.ctx, tenant.ID, user.Email)
	s.Require().NoError(err)
	s.Equal(user.ID, retrieved.ID)

	// Update
	newDisplayName := "Johnny Doe"
	user.DisplayName = &newDisplayName
	now := time.Now()
	user.UpdatedAt = &now
	err = s.store.Users().Update(s.ctx, user)
	s.Require().NoError(err)

	retrieved, err = s.store.Users().GetByID(s.ctx, tenant.ID, user.ID)
	s.Require().NoError(err)
	s.Equal("Johnny Doe", *retrieved.DisplayName)

	// Set password
	passwordHash := "hashedpassword123"
	err = s.store.Users().SetPassword(s.ctx, user.ID, passwordHash)
	s.Require().NoError(err)

	retrievedHash, err := s.store.Users().GetPassword(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Equal(passwordHash, retrievedHash)

	// Update password
	newHash := "newhashedpassword456"
	err = s.store.Users().SetPassword(s.ctx, user.ID, newHash)
	s.Require().NoError(err)

	retrievedHash, err = s.store.Users().GetPassword(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Equal(newHash, retrievedHash)
}

func (s *StoreTestSuite) TestSessionStore() {
	// Create tenant and user
	tenant := &core.Tenant{
		ID:        "tenant-123",
		Slug:      "acme-corp",
		Name:      "Acme Corporation",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	err := s.store.Tenants().Create(s.ctx, tenant)
	s.Require().NoError(err)

	user := &core.User{
		ID:        "user-456",
		TenantID:  tenant.ID,
		Email:     "john@example.com",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	err = s.store.Users().Create(s.ctx, user)
	s.Require().NoError(err)

	clientID := "client-789"
	session := &core.Session{
		ID:         "session-abc",
		TenantID:   tenant.ID,
		UserID:     user.ID,
		ClientID:   &clientID,
		IP:         "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
		RevokedAt:  nil,
	}

	// Create
	err = s.store.Sessions().Create(s.ctx, session)
	s.Require().NoError(err)

	// Get by ID
	retrieved, err := s.store.Sessions().GetByID(s.ctx, tenant.ID, session.ID)
	s.Require().NoError(err)
	s.Equal(session.ID, retrieved.ID)
	s.Equal(session.IP, retrieved.IP)

	// Update
	newLastSeen := time.Now().Add(1 * time.Hour)
	session.LastSeenAt = newLastSeen
	err = s.store.Sessions().Update(s.ctx, session)
	s.Require().NoError(err)

	retrieved, err = s.store.Sessions().GetByID(s.ctx, tenant.ID, session.ID)
	s.Require().NoError(err)

	// Revoke
	err = s.store.Sessions().Revoke(s.ctx, tenant.ID, session.ID)
	s.Require().NoError(err)

	retrieved, err = s.store.Sessions().GetByID(s.ctx, tenant.ID, session.ID)
	s.Require().NoError(err)
	s.NotNil(retrieved.RevokedAt)

	// List
	sessions, cursor, err := s.store.Sessions().List(s.ctx, tenant.ID, &user.ID, &clientID, false, 10, "")
	s.Require().NoError(err)
	s.Len(sessions, 1)
	s.Empty(cursor)

	// List active only
	sessions, _, err = s.store.Sessions().List(s.ctx, tenant.ID, nil, nil, true, 10, "")
	s.Require().NoError(err)
	s.Len(sessions, 0) // Should be empty since session is revoked
}

func (s *StoreTestSuite) TestClientStore() {
	// Create tenant
	tenant := &core.Tenant{
		ID:        "tenant-123",
		Slug:      "acme-corp",
		Name:      "Acme Corporation",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	err := s.store.Tenants().Create(s.ctx, tenant)
	s.Require().NoError(err)

	secretHash := "secrethash"
	secretLast4 := "1234"
	client := &core.Client{
		ID:                     "client-789",
		TenantID:               tenant.ID,
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
		CreatedAt:              time.Now(),
	}

	// Create
	err = s.store.Clients().Create(s.ctx, client)
	s.Require().NoError(err)

	// Get by ID
	retrieved, err := s.store.Clients().GetByID(s.ctx, tenant.ID, client.ID)
	s.Require().NoError(err)
	s.Equal(client.ID, retrieved.ID)
	s.Equal(client.Name, retrieved.Name)

	// Get by ClientID
	retrieved, err = s.store.Clients().GetByClientID(s.ctx, tenant.ID, client.ClientID)
	s.Require().NoError(err)
	s.Equal(client.ID, retrieved.ID)

	// Update
	client.Name = "Updated Application"
	client.RedirectURIs = []string{"http://localhost:3000/callback", "http://localhost:3001/callback"}
	err = s.store.Clients().Update(s.ctx, client)
	s.Require().NoError(err)

	retrieved, err = s.store.Clients().GetByID(s.ctx, tenant.ID, client.ID)
	s.Require().NoError(err)
	s.Equal("Updated Application", retrieved.Name)
	s.Len(retrieved.RedirectURIs, 2)

	// List
	clients, cursor, err := s.store.Clients().List(s.ctx, tenant.ID, 10, "")
	s.Require().NoError(err)
	s.Len(clients, 1)
	s.Empty(cursor)

	// Delete
	err = s.store.Clients().Delete(s.ctx, tenant.ID, client.ID)
	s.Require().NoError(err)

	_, err = s.store.Clients().GetByID(s.ctx, tenant.ID, client.ID)
	s.Require().Error(err)
}

func (s *StoreTestSuite) TestDomainStore() {
	// Create tenant
	tenant := &core.Tenant{
		ID:        "tenant-123",
		Slug:      "acme-corp",
		Name:      "Acme Corporation",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	err := s.store.Tenants().Create(s.ctx, tenant)
	s.Require().NoError(err)

	domain := &core.TenantDomain{
		ID:        "domain-001",
		TenantID:  tenant.ID,
		Domain:    "auth.acme.com",
		CreatedAt: time.Now(),
	}

	// Create
	err = s.store.Domains().Create(s.ctx, domain)
	s.Require().NoError(err)

	// Get by ID
	retrieved, err := s.store.Domains().GetByID(s.ctx, tenant.ID, domain.ID)
	s.Require().NoError(err)
	s.Equal(domain.ID, retrieved.ID)
	s.Equal(domain.Domain, retrieved.Domain)

	// Get by domain
	retrieved, err = s.store.Domains().GetByDomain(s.ctx, domain.Domain)
	s.Require().NoError(err)
	s.Equal(domain.ID, retrieved.ID)

	// Mark verified
	err = s.store.Domains().MarkVerified(s.ctx, tenant.ID, domain.ID)
	s.Require().NoError(err)

	retrieved, err = s.store.Domains().GetByID(s.ctx, tenant.ID, domain.ID)
	s.Require().NoError(err)
	s.NotNil(retrieved.VerifiedAt)

	// List
	domains, err := s.store.Domains().List(s.ctx, tenant.ID)
	s.Require().NoError(err)
	s.Len(domains, 1)

	// Delete
	err = s.store.Domains().Delete(s.ctx, tenant.ID, domain.ID)
	s.Require().NoError(err)

	_, err = s.store.Domains().GetByID(s.ctx, tenant.ID, domain.ID)
	s.Require().Error(err)
}

func (s *StoreTestSuite) TestRefreshTokenStore() {
	// Create tenant and user
	tenant := &core.Tenant{
		ID:        "tenant-123",
		Slug:      "acme-corp",
		Name:      "Acme Corporation",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	err := s.store.Tenants().Create(s.ctx, tenant)
	s.Require().NoError(err)

	user := &core.User{
		ID:        "user-456",
		TenantID:  tenant.ID,
		Email:     "john@example.com",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	err = s.store.Users().Create(s.ctx, user)
	s.Require().NoError(err)

	rotatedFrom := "old-token-hash"
	token := &core.RefreshToken{
		TokenHash:       "token-hash-123",
		TenantID:        tenant.ID,
		ClientID:        "client-789",
		UserID:          user.ID,
		Scope:           "openid profile",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(14 * 24 * time.Hour),
		RevokedAt:       nil,
		RotatedFromHash: &rotatedFrom,
	}

	// Create
	err = s.store.RefreshTokens().Create(s.ctx, token)
	s.Require().NoError(err)

	// Get by hash
	retrieved, err := s.store.RefreshTokens().GetByHash(s.ctx, tenant.ID, token.TokenHash)
	s.Require().NoError(err)
	s.Equal(token.TokenHash, retrieved.TokenHash)
	s.Equal(token.UserID, retrieved.UserID)

	// Revoke
	err = s.store.RefreshTokens().Revoke(s.ctx, tenant.ID, token.TokenHash)
	s.Require().NoError(err)

	retrieved, err = s.store.RefreshTokens().GetByHash(s.ctx, tenant.ID, token.TokenHash)
	s.Require().NoError(err)
	s.NotNil(retrieved.RevokedAt)
}

func (s *StoreTestSuite) TestAuditEventStore() {
	// Create tenant
	tenant := &core.Tenant{
		ID:        "tenant-123",
		Slug:      "acme-corp",
		Name:      "Acme Corporation",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	err := s.store.Tenants().Create(s.ctx, tenant)
	s.Require().NoError(err)

	actorID := "admin-001"
	ip := "192.168.1.1"
	ua := "Mozilla/5.0"
	event := &core.AuditEvent{
		ID:        "event-001",
		TenantID:  tenant.ID,
		ActorType: "admin",
		ActorID:   &actorID,
		Type:      "user_created",
		IP:        &ip,
		UserAgent: &ua,
		CreatedAt: time.Now(),
		Data: map[string]interface{}{
			"user_id": "user-456",
			"email":   "test@example.com",
		},
	}

	// Create
	err = s.store.AuditEvents().Create(s.ctx, event)
	s.Require().NoError(err)

	// List
	filters := core.AuditFilters{}
	events, cursor, err := s.store.AuditEvents().List(s.ctx, tenant.ID, filters, 10, "")
	s.Require().NoError(err)
	s.Len(events, 1)
	s.Empty(cursor)
	s.Equal(event.ID, events[0].ID)

	// List with type filter
	filters.Type = strPtr("user_created")
	events, _, err = s.store.AuditEvents().List(s.ctx, tenant.ID, filters, 10, "")
	s.Require().NoError(err)
	s.Len(events, 1)

	// List with wrong type
	filters.Type = strPtr("user_deleted")
	events, _, err = s.store.AuditEvents().List(s.ctx, tenant.ID, filters, 10, "")
	s.Require().NoError(err)
	s.Len(events, 0)
}

func strPtr(s string) *string {
	return &s
}

// Test with real SQLite to ensure SQL compatibility
func TestGormStore_CleanupExpired(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	store := NewWithDB(db)
	err = store.AutoMigrate()
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Create tenant
	tenant := &core.Tenant{
		ID:        "tenant-123",
		Slug:      "test",
		Name:      "Test",
		Status:    "active",
		CreatedAt: now,
	}
	err = store.Tenants().Create(ctx, tenant)
	require.NoError(t, err)

	// Create expired refresh token
	token := &core.RefreshToken{
		TokenHash: "expired-token",
		TenantID:  tenant.ID,
		ClientID:  "client-1",
		UserID:    "user-1",
		Scope:     "openid",
		CreatedAt: now.Add(-30 * 24 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	err = store.RefreshTokens().Create(ctx, token)
	require.NoError(t, err)

	// Run cleanup
	err = store.CleanupExpired(ctx, now)
	require.NoError(t, err)

	// Verify token was deleted
	_, err = store.RefreshTokens().GetByHash(ctx, tenant.ID, token.TokenHash)
	assert.Error(t, err)
}
