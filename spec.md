Locky — System Specification (v0.1)

1. Design Principles
	1.	Library-first architecture
	•	Core logic is importable as a Go module.
	•	HTTP server is an adapter layer over core services.
	•	No global state.
	•	All dependencies injected via interfaces.
	2.	Single binary deployment
	•	No required external services other than Postgres.
	•	Optional embedded login UI.
	•	Deterministic configuration via environment + config struct.
	3.	Host-based multi-tenancy
	•	Tenant resolved from request host.
	•	Supports wildcard subdomain and custom domains.
	4.	JWT access tokens + stateful sessions
	•	Access tokens: short-lived, stateless JWT.
	•	Refresh tokens: stateful, stored hashed.
	•	Web sessions: stateful, stored server-side.
	5.	Declarative security
	•	Policy documents (JSONB).
	•	RBAC via Casbin (domain-scoped).
	•	Limited, auditable hooks only.
	6.	Auditability by default
	•	Every sensitive operation logged.
	•	Policy version + RBAC state hash captured during token issuance.

⸻

2. High-Level Architecture

                    ┌─────────────────────┐
                    │     HTTP Layer      │
                    │ net/http handlers   │
                    └──────────┬──────────┘
                               │
                ┌──────────────┼──────────────┐
                │              │              │
        ┌───────▼───────┐  ┌───▼──────┐  ┌────▼─────┐
        │ TenantResolver │  │  Core    │  │  UI      │
        └───────┬───────┘  │ Services │  │ (optional)│
                │          └─────┬─────┘  └───────────┘
                │                │
        ┌───────▼────────┐  ┌────▼──────────┐
        │   Store (PG)   │  │ Authorizer    │
        │  (interfaces)  │  │ (Casbin)      │
        └────────────────┘  └───────────────┘


⸻

3. Package Layout

/auth
  /core
  /store
  /tenant
  /tokens
  /sessions
  /oauth
  /policy
  /rbac
  /audit
  /crypto
  /http
  /ui

Core services depend only on interfaces:
	•	Store
	•	Authorizer
	•	PolicyEngine
	•	AuditSink
	•	Clock
	•	KeyManager
	•	TenantResolver

⸻

4. Multi-Tenant Architecture (Host-Based)

4.1 Resolution Strategy

Supported patterns:
	1.	tenantSlug.auth.example.com
	2.	Custom domain (e.g., login.customer.com)

Resolution algorithm:
	1.	Normalize host (strip port, lowercase).
	2.	Lookup exact match in tenant_domains.
	3.	If not found, parse subdomain → lookup tenants.slug.
	4.	If not found → fail closed.

Interface:

type TenantResolver interface {
  ResolveTenant(ctx context.Context, host string) (*Tenant, error)
}

Tenant context is injected into request context at middleware.

⸻

5. Identity Model

5.1 Tenants

tenants (
  id UUID PK,
  slug TEXT UNIQUE,
  name TEXT,
  status TEXT CHECK (status IN ('active','suspended')),
  created_at TIMESTAMP
)

tenant_domains (
  id UUID PK,
  tenant_id UUID FK,
  domain TEXT UNIQUE,
  verified_at TIMESTAMP NULL,
  created_at TIMESTAMP
)


⸻

5.2 Users

users (
  id UUID PK,
  tenant_id UUID FK,
  email TEXT,
  email_verified BOOLEAN,
  status TEXT CHECK (status IN ('active','disabled')),
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  UNIQUE (tenant_id, email)
)

user_passwords (
  user_id UUID PK,
  password_hash TEXT,
  updated_at TIMESTAMP
)

Password hashing:
	•	Argon2id (external lib allowed).
	•	Hashes never returned.

⸻

6. Sessions

Web sessions are stateful.

sessions (
  id UUID PK,
  tenant_id UUID,
  user_id UUID,
  client_id TEXT NULL,
  ip TEXT,
  user_agent TEXT,
  created_at TIMESTAMP,
  last_seen_at TIMESTAMP,
  revoked_at TIMESTAMP NULL
)

Cookie:
	•	session_id
	•	HttpOnly
	•	Secure
	•	SameSite=Lax (Strict optional)

Session revocation invalidates browser login immediately.

⸻

7. OAuth2 / OIDC

7.1 Supported Grants (v1)
	•	Authorization Code + PKCE (S256 required)
	•	Refresh Token (rotation required)
	•	Client Credentials

⸻

7.2 Authorization Codes

oauth_authorization_codes (
  code_hash TEXT PK,
  tenant_id UUID,
  client_id TEXT,
  user_id UUID,
  redirect_uri TEXT,
  pkce_challenge TEXT,
  pkce_method TEXT,
  scope TEXT,
  expires_at TIMESTAMP,
  used_at TIMESTAMP NULL
)

Single use only.

⸻

7.3 Refresh Tokens

oauth_refresh_tokens (
  token_hash TEXT PK,
  tenant_id UUID,
  client_id TEXT,
  user_id UUID,
  scope TEXT,
  created_at TIMESTAMP,
  expires_at TIMESTAMP,
  revoked_at TIMESTAMP NULL,
  rotated_from_hash TEXT NULL
)

	•	Stored hashed.
	•	Rotated on every use.
	•	Revocation supported.

⸻

7.4 Access Tokens (JWT)

Signed per tenant.

Claims:

iss = https://{tenant-host}
sub = user_id
aud = resource identifier
tid = tenant_id
sid = session_id (optional)
roles = [...]
scope = string
iat, exp, nbf
jti = uuid

TTL:
	•	5–15 minutes recommended.

No access token storage (stateless).

⸻

7.5 Signing Keys

signing_keys (
  id UUID PK,
  tenant_id UUID,
  kid TEXT,
  public_jwk JSONB,
  private_key_encrypted BYTEA,
  status TEXT CHECK (status IN ('active','inactive','retired')),
  created_at TIMESTAMP,
  not_before TIMESTAMP,
  not_after TIMESTAMP
)

Rotation:
	•	New key created.
	•	Mark previous as inactive but valid until grace window ends.
	•	JWKS endpoint publishes active + grace keys.

⸻

8. RBAC (Casbin-Based)

8.1 Model

Use RBAC with domains.

Model:

r = sub, dom, obj, act
p = sub, dom, obj, act
g = _, _, _

Domain = tenant_id.

⸻

8.2 Policy Storage

Use Casbin Postgres adapter.

Examples:

User-role binding:

g, user:123, role:admin, tenant:abc

Role-permission binding:

p, role:admin, tenant:abc, project:*, write


⸻

8.3 Authorization Interface

type Authorizer interface {
  Enforce(ctx context.Context, tenantID, subject, object, action string) (bool, error)
  RolesForUser(ctx context.Context, tenantID, userID string) ([]string, error)
}

Core depends only on Authorizer.

⸻

8.4 JWT Claim Strategy

JWT includes:
	•	roles
	•	scope

Access token TTL short.
Fine-grained enforcement can occur in services via Casbin.

⸻

9. Policy Engine

Policies stored as JSONB.

policies (
  id UUID,
  tenant_id UUID,
  name TEXT,
  version INT,
  document JSONB,
  status TEXT,
  created_at TIMESTAMP
)

Used for:
	•	Login gating
	•	Scope ceilings
	•	Conditional logic (future MFA enforcement)

Policy evaluation occurs before token issuance.

⸻

10. Hooks (Limited + Audited)

Hook interface:

type Hook interface {
  Name() string
  BeforeTokenIssue(ctx HookContext) (TokenMod, error)
}

Constraints:
	•	Cannot override deny decision.
	•	Cannot elevate beyond policy ceiling.
	•	Every invocation audited.

⸻

11. Audit Logging

audit_events (
  id UUID,
  tenant_id UUID,
  actor_type TEXT,
  actor_id TEXT,
  event_type TEXT,
  ip TEXT,
  user_agent TEXT,
  created_at TIMESTAMP,
  data JSONB
)

Events logged for:
	•	Login success/failure
	•	Token issuance
	•	Policy changes
	•	RBAC changes
	•	Key rotation
	•	Session revocation

⸻

12. HTTP Surfaces

12.1 Public (Per Tenant Host)
	•	/.well-known/openid-configuration
	•	/oauth2/authorize
	•	/oauth2/token
	•	/oauth2/userinfo
	•	/oauth2/revoke
	•	/oauth2/introspect
	•	/oauth2/logout
	•	/oauth2/jwks.json
	•	/ui/login
	•	/ui/logout

⸻

12.2 Admin API (Dedicated Host)
	•	Tenant CRUD
	•	Domain mapping
	•	User management
	•	Client management
	•	Key rotation
	•	Policy CRUD
	•	RBAC role + binding management
	•	Session listing/revocation
	•	Audit query

Admin authentication:
	•	Admin API key (initial)
	•	Later: OIDC-protected admin console

⸻

13. Security Controls
	•	Argon2id password hashing.
	•	CSRF protection for login forms.
	•	PKCE required.
	•	Rate limiting middleware (IP + account-based).
	•	Refresh token hashing.
	•	Secure cookie flags enforced.
	•	All cryptographic randomness via crypto/rand.

⸻

14. Library-First Usage Model

Example embedding:

cfg := auth.Config{...}

core, err := auth.NewCore(cfg)

srv := authhttp.NewServer(core)
http.ListenAndServe(":8080", srv)

Or direct usage:

token, err := core.TokenService.IssueAccessToken(ctx, input)

No HTTP required to use core logic.

⸻

15. Cleanup & Background Tasks

Single binary includes:
	•	Expired auth code cleanup
	•	Expired refresh token cleanup
	•	Expired session cleanup
	•	Key rotation expiration enforcement

These run via internal goroutines with configurable intervals.

⸻

16. Future Extensions (Reserved)
	•	MFA (TOTP → WebAuthn)
	•	SCIM provisioning
	•	SAML (SP mode)
	•	Organization-based IdP routing
	•	Device Code flow
	•	Token introspection caching layer

⸻

17. Deployment Model

Single container:
	•	Go binary
	•	Postgres connection
	•	Optional TLS termination externally
	•	No Redis required
	•	No background workers separate from binary

⸻

18. Operational Guarantees
	•	Deterministic tenant isolation via domain scoping.
	•	RBAC enforced per-tenant via Casbin domain model.
	•	JWT iss tenant-specific.
	•	Revocation achieved via:
	•	Session invalidation
	•	Refresh token revocation
	•	Short access token TTL

⸻

Final Architecture Characteristics

Property	Achieved
Multi-tenant	Host-based
Library-first	Yes
Single binary	Yes
Standards compliant	OAuth2 + OIDC
RBAC	Casbin with domains
Policy model	Declarative JSON
Audit	First-class
Extensible	Yes
