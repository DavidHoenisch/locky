package core

import (
	"context"
	"time"
)

// Clock provides time for testability
type Clock interface {
	Now() time.Time
}

// RealClock is the production clock implementation
type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

// Config holds the core configuration
type Config struct {
	DatabaseURL           string
	AdminAPIKey           string
	BaseDomain            string
	SessionCookieName     string
	SessionCookieSecure   bool
	SessionCookieSameSite string
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
	SessionTTL            time.Duration
	MaxLoginAttempts      int
	PasswordMinLength     int
	EnableHostedUI        bool
	EnableAdminUI         bool
	AdminUIUsername       string
	AdminUIPassword       string
}

// Core is the main entry point for library usage
type Core struct {
	Config         Config
	Store          Store
	Authorizer     Authorizer
	PolicyEngine   PolicyEngine
	AuditSink      AuditSink
	Clock          Clock
	KeyManager     KeyManager
	TenantResolver TenantResolver

	// Services
	TokenService   TokenService
	SessionService SessionService
	UserService    UserService
	OAuthService   OAuthService
}

// NewCore creates a new Core instance
func NewCore(cfg Config, store Store, authorizer Authorizer, auditSink AuditSink) (*Core, error) {
	core := &Core{
		Config:     cfg,
		Store:      store,
		Authorizer: authorizer,
		AuditSink:  auditSink,
		Clock:      RealClock{},
	}

	return core, nil
}

// Store is the main persistence interface
type Store interface {
	Tenants() TenantStore
	Users() UserStore
	Sessions() SessionStore
	Clients() ClientStore
	Domains() DomainStore
	Policies() PolicyStore
	SigningKeys() SigningKeyStore
	OAuthCodes() OAuthCodeStore
	RefreshTokens() RefreshTokenStore
	AuditEvents() AuditEventStore
	AdminKeys() AdminKeyStore
}

// TenantStore manages tenant persistence
type TenantStore interface {
	Create(ctx context.Context, tenant *Tenant) error
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	Update(ctx context.Context, tenant *Tenant) error
	List(ctx context.Context, limit int, cursor string) ([]*Tenant, string, error)
}

// UserStore manages user persistence
type UserStore interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, tenantID, id string) (*User, error)
	GetByEmail(ctx context.Context, tenantID, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	List(ctx context.Context, tenantID string, limit int, cursor string) ([]*User, string, error)
	SetPassword(ctx context.Context, userID string, hash string) error
	GetPassword(ctx context.Context, userID string) (string, error)
}

// SessionStore manages session persistence
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, tenantID, id string) (*Session, error)
	Update(ctx context.Context, session *Session) error
	Revoke(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string, userID, clientID *string, activeOnly bool, limit int, cursor string) ([]*Session, string, error)
	DeleteExpired(ctx context.Context, before time.Time) error
}

// ClientStore manages OAuth client persistence
type ClientStore interface {
	Create(ctx context.Context, client *Client) error
	GetByID(ctx context.Context, tenantID, id string) (*Client, error)
	GetByClientID(ctx context.Context, tenantID, clientID string) (*Client, error)
	Update(ctx context.Context, client *Client) error
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string, limit int, cursor string) ([]*Client, string, error)
}

// DomainStore manages custom domain persistence
type DomainStore interface {
	Create(ctx context.Context, domain *TenantDomain) error
	GetByID(ctx context.Context, tenantID, id string) (*TenantDomain, error)
	GetByDomain(ctx context.Context, domain string) (*TenantDomain, error)
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string) ([]*TenantDomain, error)
	MarkVerified(ctx context.Context, tenantID, id string) error
}

// PolicyStore manages policy persistence
type PolicyStore interface {
	Create(ctx context.Context, policy *Policy) error
	GetByID(ctx context.Context, tenantID, id string) (*Policy, error)
	Update(ctx context.Context, policy *Policy) error
	List(ctx context.Context, tenantID string, status *string, limit int, cursor string) ([]*Policy, string, error)
}

// SigningKeyStore manages signing key persistence
type SigningKeyStore interface {
	Create(ctx context.Context, key *SigningKey) error
	GetActive(ctx context.Context, tenantID string) (*SigningKey, error)
	GetByKID(ctx context.Context, tenantID, kid string) (*SigningKey, error)
	ListActive(ctx context.Context, tenantID string) ([]*SigningKey, error)
	MarkInactive(ctx context.Context, tenantID, id string) error
	MarkRetired(ctx context.Context, tenantID, id string) error
}

// OAuthCodeStore manages authorization code persistence
type OAuthCodeStore interface {
	Create(ctx context.Context, code *OAuthCode) error
	GetAndConsume(ctx context.Context, tenantID, codeHash string) (*OAuthCode, error)
	DeleteExpired(ctx context.Context, before time.Time) error
}

// RefreshTokenStore manages refresh token persistence
type RefreshTokenStore interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByHash(ctx context.Context, tenantID, hash string) (*RefreshToken, error)
	Revoke(ctx context.Context, tenantID, hash string) error
	DeleteExpired(ctx context.Context, before time.Time) error
}

// AuditEventStore manages audit event persistence
type AuditEventStore interface {
	Create(ctx context.Context, event *AuditEvent) error
	List(ctx context.Context, tenantID string, filters AuditFilters, limit int, cursor string) ([]*AuditEvent, string, error)
}

// AdminKeyStore manages admin API key persistence
type AdminKeyStore interface {
	Create(ctx context.Context, key *AdminKey) error
	GetByHash(ctx context.Context, hash string) (*AdminKey, error)
	List(ctx context.Context) ([]*AdminKey, error)
	Delete(ctx context.Context, id string) error
}

// Authorizer handles RBAC enforcement
type Authorizer interface {
	Enforce(ctx context.Context, tenantID, subject, object, action string) (bool, error)
	RolesForUser(ctx context.Context, tenantID, userID string) ([]string, error)
	AddPolicy(ctx context.Context, tenantID string, policy RbacTuple) error
	RemovePolicy(ctx context.Context, tenantID string, policyID string) error
	ListPolicies(ctx context.Context, tenantID string, filters RbacFilters) ([]RbacTuple, string, error)
}

// PolicyEngine handles policy evaluation
type PolicyEngine interface {
	Evaluate(ctx context.Context, tenantID string, document map[string]interface{}, context map[string]interface{}) (*PolicyResult, error)
}

// AuditSink handles audit logging
type AuditSink interface {
	Log(ctx context.Context, event *AuditEvent) error
}

// KeyManager handles cryptographic keys
type KeyManager interface {
	GenerateKey(ctx context.Context, tenantID string) (*SigningKey, error)
	Sign(ctx context.Context, tenantID string, claims map[string]interface{}) (string, error)
	GetPublicJWKS(ctx context.Context, tenantID string) (map[string]interface{}, error)
}

// TenantResolver resolves tenants from hostnames
type TenantResolver interface {
	ResolveTenant(ctx context.Context, host string) (*Tenant, error)
}

// TokenService handles token operations
type TokenService interface {
	IssueAccessToken(ctx context.Context, tenantID, userID, clientID string, scope string, roles []string, sessionID *string) (string, error)
	IssueRefreshToken(ctx context.Context, tenantID, userID, clientID string, scope string) (string, error)
	ValidateAccessToken(ctx context.Context, token string) (*TokenClaims, error)
	RotateRefreshToken(ctx context.Context, tenantID, oldToken string) (string, error)
}

// SessionService handles session operations
type SessionService interface {
	Create(ctx context.Context, tenantID, userID, clientID string, ip, userAgent string) (*Session, error)
	Validate(ctx context.Context, tenantID, sessionID string) (*Session, error)
	Revoke(ctx context.Context, tenantID, sessionID string) error
}

// UserService handles user operations
type UserService interface {
	Authenticate(ctx context.Context, tenantID, email, password string) (*User, error)
	Create(ctx context.Context, tenantID, email, displayName string) (*User, error)
	SetPassword(ctx context.Context, tenantID, userID, password string) error
}

// OAuthService handles OAuth2/OIDC operations
type OAuthService interface {
	Authorize(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error)
	Token(ctx context.Context, req *TokenRequest) (*TokenResponse, error)
	UserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
	Revoke(ctx context.Context, tenantID, token string, tokenType string) error
	Introspect(ctx context.Context, tenantID, token string) (*IntrospectResponse, error)
}

// AuditFilters for querying audit events
type AuditFilters struct {
	Type      *string
	ActorType *string
	ActorID   *string
	Since     *time.Time
	Until     *time.Time
}

// RbacFilters for querying RBAC policies
type RbacFilters struct {
	TupleType *string
	V0        *string
	V1        *string
	V2        *string
	V3        *string
}

// PolicyResult is the outcome of policy evaluation
type PolicyResult struct {
	Allowed bool
	Reason  string
	Mods    map[string]interface{}
}
