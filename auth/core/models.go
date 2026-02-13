package core

import "time"

// Tenant represents a tenant/organization
type Tenant struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // active, suspended
	CreatedAt time.Time `json:"created_at"`
}

// TenantDomain represents a custom domain mapping
type TenantDomain struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Domain     string     `json:"domain"`
	VerifiedAt *time.Time `json:"verified_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// User represents an identity
type User struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	Status        string     `json:"status"` // active, disabled
	DisplayName   *string    `json:"display_name"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// Session represents a web session
type Session struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	UserID     string     `json:"user_id"`
	ClientID   *string    `json:"client_id"`
	IP         string     `json:"ip"`
	UserAgent  string     `json:"user_agent"`
	CreatedAt  time.Time  `json:"created_at"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

// Client represents an OAuth2/OIDC client
type Client struct {
	ID                     string    `json:"id"`
	TenantID               string    `json:"tenant_id"`
	Name                   string    `json:"name"`
	ClientID               string    `json:"client_id"`
	ClientSecretHash       *string   `json:"-"`
	ClientSecretLast4      *string   `json:"client_secret_last4"`
	RedirectURIs           []string  `json:"redirect_uris"`
	PostLogoutRedirectURIs []string  `json:"post_logout_redirect_uris"`
	GrantTypes             []string  `json:"grant_types"`
	ResponseTypes          []string  `json:"response_types"`
	Scopes                 []string  `json:"scopes"`
	TokenTTLSeconds        int       `json:"token_ttl_seconds"`
	RefreshTTLSeconds      int       `json:"refresh_ttl_seconds"`
	CreatedAt              time.Time `json:"created_at"`
}

// Policy represents a declarative policy document
type Policy struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	Name      string                 `json:"name"`
	Version   int                    `json:"version"`
	Status    string                 `json:"status"` // active, inactive
	Document  map[string]interface{} `json:"document"`
	CreatedAt time.Time              `json:"created_at"`
}

// PolicyBinding binds a policy to a target
type PolicyBinding struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	PolicyID  string    `json:"policy_id"`
	BindType  string    `json:"bind_type"` // tenant, client, user, group
	BindID    string    `json:"bind_id"`
	CreatedAt time.Time `json:"created_at"`
}

// SigningKey represents a JWT signing key
type SigningKey struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	KID                 string    `json:"kid"`
	PublicJWK           []byte    `json:"public_jwk"`
	PrivateKeyEncrypted []byte    `json:"-"`
	Status              string    `json:"status"` // active, inactive, retired
	CreatedAt           time.Time `json:"created_at"`
	NotBefore           time.Time `json:"not_before"`
	NotAfter            time.Time `json:"not_after"`
}

// OAuthCode represents an authorization code
type OAuthCode struct {
	CodeHash      string     `json:"-"`
	TenantID      string     `json:"tenant_id"`
	ClientID      string     `json:"client_id"`
	UserID        string     `json:"user_id"`
	RedirectURI   string     `json:"redirect_uri"`
	PKCEChallenge string     `json:"pkce_challenge"`
	PKCEMethod    string     `json:"pkce_method"`
	Scope         string     `json:"scope"`
	ExpiresAt     time.Time  `json:"expires_at"`
	UsedAt        *time.Time `json:"used_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// RefreshToken represents a refresh token
type RefreshToken struct {
	TokenHash       string     `json:"-"`
	TenantID        string     `json:"tenant_id"`
	ClientID        string     `json:"client_id"`
	UserID          string     `json:"user_id"`
	Scope           string     `json:"scope"`
	CreatedAt       time.Time  `json:"created_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	RevokedAt       *time.Time `json:"revoked_at"`
	RotatedFromHash *string    `json:"-"`
}

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	ActorType string                 `json:"actor_type"` // admin, user, system
	ActorID   *string                `json:"actor_id"`
	Type      string                 `json:"type"`
	IP        *string                `json:"ip"`
	UserAgent *string                `json:"user_agent"`
	CreatedAt time.Time              `json:"created_at"`
	Data      map[string]interface{} `json:"data"`
}

// AdminKey represents an admin API key
type AdminKey struct {
	ID        string    `json:"id"`
	KeyHash   string    `json:"-"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy *string   `json:"created_by"`
}

// RbacTuple represents a Casbin policy or grouping
type RbacTuple struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	TupleType string    `json:"tuple_type"` // p, g
	V0        string    `json:"v0"`
	V1        string    `json:"v1"`
	V2        string    `json:"v2"`
	V3        *string   `json:"v3"`
	V4        *string   `json:"v4"`
	V5        *string   `json:"v5"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	Audience  string   `json:"aud"`
	TenantID  string   `json:"tid"`
	SessionID *string  `json:"sid,omitempty"`
	Roles     []string `json:"roles"`
	Scope     string   `json:"scope"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
	NotBefore int64    `json:"nbf"`
	JWTID     string   `json:"jti"`
}

// AuthorizeRequest represents an OAuth2 authorize request
type AuthorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
	TenantID            string
	UserID              string // Set after authentication
}

// AuthorizeResponse represents an OAuth2 authorize response
type AuthorizeResponse struct {
	Code        string
	State       string
	RedirectURI string
}

// TokenRequest represents an OAuth2 token request
type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	ClientID     string
	ClientSecret string
	Scope        string
	TenantID     string
}

// TokenResponse represents an OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// UserInfo represents OIDC userinfo
type UserInfo struct {
	Subject       string                 `json:"sub"`
	Email         string                 `json:"email,omitempty"`
	EmailVerified bool                   `json:"email_verified,omitempty"`
	DisplayName   string                 `json:"name,omitempty"`
	Extra         map[string]interface{} `json:"-"`
}

// IntrospectResponse represents token introspection response
type IntrospectResponse struct {
	Active    bool     `json:"active"`
	Subject   *string  `json:"sub,omitempty"`
	Audience  *string  `json:"aud,omitempty"`
	Issuer    *string  `json:"iss,omitempty"`
	ExpiresAt *int64   `json:"exp,omitempty"`
	IssuedAt  *int64   `json:"iat,omitempty"`
	Scope     *string  `json:"scope,omitempty"`
	ClientID  *string  `json:"client_id,omitempty"`
	TenantID  *string  `json:"tid,omitempty"`
	Roles     []string `json:"roles,omitempty"`
}
