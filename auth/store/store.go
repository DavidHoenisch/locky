package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/locky/auth/core"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// GormStore implements core.Store using GORM
type GormStore struct {
	db *gorm.DB
}

// setUUIDBeforeCreate sets UUID for empty primary key ID fields (so SQLite and Postgres both work)
func setUUIDBeforeCreate(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}
	for _, field := range db.Statement.Schema.Fields {
		if field.Name == "ID" && field.DBName == "id" && field.PrimaryKey {
			val, zero := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue)
			if zero || val == nil {
				_ = field.Set(db.Statement.Context, db.Statement.ReflectValue, uuid.New().String())
				return
			}
			if s, ok := val.(string); ok && s == "" {
				_ = field.Set(db.Statement.Context, db.Statement.ReflectValue, uuid.New().String())
			}
			return
		}
	}
}

// New creates a new GormStore
func New(databaseURL string) (*GormStore, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	db.Callback().Create().Before("gorm:before_create").Register("store:set_uuid", func(d *gorm.DB) {
		setUUIDBeforeCreate(d)
	})
	return &GormStore{db: db}, nil
}

// NewWithDB creates a new GormStore from an existing GORM DB
func NewWithDB(db *gorm.DB) *GormStore {
	db.Callback().Create().Before("gorm:before_create").Register("store:set_uuid", func(d *gorm.DB) {
		setUUIDBeforeCreate(d)
	})
	return &GormStore{db: db}
}

// DB returns the underlying GORM DB
func (s *GormStore) DB() *gorm.DB {
	return s.db
}

// AutoMigrate runs database migrations
func (s *GormStore) AutoMigrate() error {
	return s.db.AutoMigrate(
		&Tenant{},
		&TenantDomain{},
		&User{},
		&UserPassword{},
		&Session{},
		&Client{},
		&SigningKey{},
		&OAuthCode{},
		&RefreshToken{},
		&AuditEvent{},
		&AdminKey{},
		&Policy{},
		&PolicyBinding{},
		&RbacTuple{},
	)
}

// Tenants returns the tenant store
func (s *GormStore) Tenants() core.TenantStore {
	return &tenantStore{db: s.db}
}

// Users returns the user store
func (s *GormStore) Users() core.UserStore {
	return &userStore{db: s.db}
}

// Sessions returns the session store
func (s *GormStore) Sessions() core.SessionStore {
	return &sessionStore{db: s.db}
}

// Clients returns the client store
func (s *GormStore) Clients() core.ClientStore {
	return &clientStore{db: s.db}
}

// Domains returns the domain store
func (s *GormStore) Domains() core.DomainStore {
	return &domainStore{db: s.db}
}

// Policies returns the policy store
func (s *GormStore) Policies() core.PolicyStore {
	return &policyStore{db: s.db}
}

// SigningKeys returns the signing key store
func (s *GormStore) SigningKeys() core.SigningKeyStore {
	return &signingKeyStore{db: s.db}
}

// OAuthCodes returns the OAuth code store
func (s *GormStore) OAuthCodes() core.OAuthCodeStore {
	return &oauthCodeStore{db: s.db}
}

// RefreshTokens returns the refresh token store
func (s *GormStore) RefreshTokens() core.RefreshTokenStore {
	return &refreshTokenStore{db: s.db}
}

// AuditEvents returns the audit event store
func (s *GormStore) AuditEvents() core.AuditEventStore {
	return &auditEventStore{db: s.db}
}

// AdminKeys returns the admin key store
func (s *GormStore) AdminKeys() core.AdminKeyStore {
	return &adminKeyStore{db: s.db}
}

// CleanupExpired deletes all expired records
func (s *GormStore) CleanupExpired(ctx context.Context, before time.Time) error {
	// Delete expired authorization codes
	if err := s.db.WithContext(ctx).
		Where("expires_at < ? OR used_at IS NOT NULL", before).
		Delete(&OAuthCode{}).Error; err != nil {
		return fmt.Errorf("cleanup oauth codes: %w", err)
	}

	// Delete expired refresh tokens
	if err := s.db.WithContext(ctx).
		Where("expires_at < ? OR revoked_at IS NOT NULL", before).
		Delete(&RefreshToken{}).Error; err != nil {
		return fmt.Errorf("cleanup refresh tokens: %w", err)
	}

	// Delete expired/revoked sessions
	if err := s.db.WithContext(ctx).
		Where("revoked_at IS NOT NULL OR created_at < ?", before.Add(-30*24*time.Hour)).
		Delete(&Session{}).Error; err != nil {
		return fmt.Errorf("cleanup sessions: %w", err)
	}

	return nil
}
