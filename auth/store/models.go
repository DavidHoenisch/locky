package store

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// StringSlice is a custom type for handling JSONB arrays
type StringSlice []string

// Scan implements the Scanner interface
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return nil
	}
}

// Value implements the Valuer interface
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Tenant is the GORM model for tenants
type Tenant struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	Slug      string    `gorm:"uniqueIndex;not null"`
	Name      string    `gorm:"not null"`
	Status    string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TenantDomain is the GORM model for tenant domains
type TenantDomain struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	TenantID   string `gorm:"type:uuid;not null;index"`
	Domain     string `gorm:"uniqueIndex;not null"`
	VerifiedAt *time.Time
	CreatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// User is the GORM model for users
type User struct {
	ID            string `gorm:"type:uuid;primaryKey"`
	TenantID      string `gorm:"type:uuid;not null;index;uniqueIndex:idx_tenant_email"`
	Email         string `gorm:"not null;uniqueIndex:idx_tenant_email"`
	EmailVerified bool   `gorm:"not null;default:false"`
	Status        string `gorm:"not null"`
	DisplayName   *string
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     *time.Time
}

// UserPassword is the GORM model for user passwords
type UserPassword struct {
	UserID       string    `gorm:"type:uuid;primaryKey"`
	PasswordHash string    `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// Session is the GORM model for sessions
type Session struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	TenantID   string `gorm:"type:uuid;not null;index"`
	UserID     string `gorm:"type:uuid;not null;index"`
	ClientID   *string
	IP         *string
	UserAgent  *string
	CreatedAt  time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	LastSeenAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	RevokedAt  *time.Time `gorm:"index"`
}

// Client is the GORM model for OAuth clients
type Client struct {
	ID                     string `gorm:"type:uuid;primaryKey"`
	TenantID               string `gorm:"type:uuid;not null;index;uniqueIndex:idx_tenant_client_id"`
	Name                   string `gorm:"not null"`
	ClientID               string `gorm:"not null;uniqueIndex:idx_tenant_client_id"`
	ClientSecretHash       *string
	ClientSecretLast4      *string
	RedirectURIs           StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	PostLogoutRedirectURIs StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	GrantTypes             StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	ResponseTypes          StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	Scopes                 StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	TokenTTLSeconds        int         `gorm:"not null;default:900"`
	RefreshTTLSeconds      int         `gorm:"not null;default:1209600"`
	CreatedAt              time.Time   `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// SigningKey is the GORM model for signing keys
type SigningKey struct {
	ID                  string    `gorm:"type:uuid;primaryKey"`
	TenantID            string    `gorm:"type:uuid;not null;index;uniqueIndex:idx_tenant_kid"`
	KID                 string    `gorm:"not null;uniqueIndex:idx_tenant_kid"`
	PublicJWK           []byte    `gorm:"type:jsonb;not null"`
	PrivateKeyEncrypted []byte    `gorm:"type:bytea;not null"`
	Status              string    `gorm:"not null"`
	CreatedAt           time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	NotBefore           time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	NotAfter            time.Time `gorm:"not null"`
}

// OAuthCode is the GORM model for authorization codes
type OAuthCode struct {
	CodeHash      string    `gorm:"primaryKey"`
	TenantID      string    `gorm:"type:uuid;not null;index"`
	ClientID      string    `gorm:"not null"`
	UserID        string    `gorm:"type:uuid;not null"`
	RedirectURI   string    `gorm:"not null"`
	PKCEChallenge string    `gorm:"not null"`
	PKCEMethod    string    `gorm:"not null"`
	Scope         string    `gorm:"not null"`
	ExpiresAt     time.Time `gorm:"not null;index"`
	UsedAt        *time.Time
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// RefreshToken is the GORM model for refresh tokens
type RefreshToken struct {
	TokenHash       string     `gorm:"primaryKey"`
	TenantID        string     `gorm:"type:uuid;not null;index"`
	ClientID        string     `gorm:"not null"`
	UserID          string     `gorm:"type:uuid;not null"`
	Scope           string     `gorm:"not null"`
	CreatedAt       time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	ExpiresAt       time.Time  `gorm:"not null;index"`
	RevokedAt       *time.Time `gorm:"index"`
	RotatedFromHash *string
}

// AuditEvent is the GORM model for audit events
type AuditEvent struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	TenantID  string `gorm:"type:uuid;not null;index"`
	ActorType string `gorm:"not null"`
	ActorID   *string
	EventType string `gorm:"not null"`
	IP        *string
	UserAgent *string
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index"`
	Data      []byte    `gorm:"type:jsonb;not null;default:'{}'"`
}

// AdminKey is the GORM model for admin API keys
type AdminKey struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	KeyHash   string    `gorm:"uniqueIndex;not null"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy *string
}

// Policy is the GORM model for policies
type Policy struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	TenantID  string    `gorm:"type:uuid;not null;index;uniqueIndex:idx_tenant_name_version"`
	Name      string    `gorm:"not null;uniqueIndex:idx_tenant_name_version"`
	Version   int       `gorm:"not null;uniqueIndex:idx_tenant_name_version"`
	Status    string    `gorm:"not null"`
	Document  []byte    `gorm:"type:jsonb;not null"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// PolicyBinding is the GORM model for policy bindings
type PolicyBinding struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	TenantID  string    `gorm:"type:uuid;not null;index"`
	PolicyID  string    `gorm:"type:uuid;not null;index;uniqueIndex:idx_policy_bind"`
	BindType  string    `gorm:"not null;uniqueIndex:idx_policy_bind"`
	BindID    string    `gorm:"not null;uniqueIndex:idx_policy_bind"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// RbacTuple is the GORM model for RBAC tuples
type RbacTuple struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	TenantID  string `gorm:"type:uuid;not null;index"`
	TupleType string `gorm:"not null"`
	V0        string `gorm:"not null"`
	V1        string `gorm:"not null"`
	V2        string `gorm:"not null"`
	V3        *string
	V4        *string
	V5        *string
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for RbacTuple
func (RbacTuple) TableName() string {
	return "rbac_tuples"
}
