# Locky - Open Source Identity Platform

Locky is a single-binary, library-first identity platform implementing OAuth2/OIDC, multi-tenancy, and RBAC. It's designed as an embeddable alternative to Auth0.

## Features

- **Host-based Multi-tenancy**: Supports wildcard subdomains (e.g., `tenant.auth.example.com`) and custom domains
- **OAuth2/OIDC**: Full Authorization Code + PKCE flow, Refresh Token, and Client Credentials grants
- **JWT Access Tokens**: Short-lived, tenant-scoped, signed with ECDSA
- **Stateful Sessions**: Server-side session management with secure cookies
- **RBAC via Casbin**: Domain-scoped role-based access control
- **Audit Logging**: All sensitive operations are logged
- **Embeddable UI**: Optional Svelte-based hosted login UI
- **Library-First**: Core logic importable as a Go module

## Quick Start

### Prerequisites

- Go 1.22+
- PostgreSQL 14+
- Node.js 18+ (for building UI)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/locky.git
cd locky

# Install dependencies
cd auth && go mod tidy
cd ../ui && npm install

# Build the UI
cd ../ui && npm run build

# Run migrations and start server
cd ../cmd/locky
go run main.go \
  -database-url="postgres://user:pass@localhost/locky?sslmode=disable" \
  -admin-api-key="your-secret-key" \
  -base-domain="auth.example.com" \
  -auto-migrate=true
```

### Creating Your First Tenant

```bash
# Create a tenant
curl -X POST http://localhost:8080/admin/tenants \
  -H "X-Admin-Key: your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"slug": "acme", "name": "Acme Corp"}'

# Create a user
curl -X POST http://localhost:8080/admin/tenants/{tenant-id}/users \
  -H "X-Admin-Key: your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"email": "user@acme.com", "display_name": "John Doe"}'

# Set user password
curl -X PUT http://localhost:8080/admin/tenants/{tenant-id}/users/{user-id}/password \
  -H "X-Admin-Key: your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"password": "secure-password-123"}'

# Create an OAuth client
curl -X POST http://localhost:8080/admin/tenants/{tenant-id}/clients \
  -H "X-Admin-Key: your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My App",
    "redirect_uris": ["http://localhost:3000/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "scopes": ["openid", "profile", "email"]
  }'
```

### OAuth2 Flow

```bash
# 1. Authorization request (redirect user to this URL)
https://acme.auth.example.com/oauth2/authorize?
  response_type=code&
  client_id=your-client-id&
  redirect_uri=http://localhost:3000/callback&
  scope=openid+profile+email&
  state=random-state&
  code_challenge=pkce-challenge&
  code_challenge_method=S256

# 2. Exchange code for tokens
curl -X POST https://acme.auth.example.com/oauth2/token \
  -d "grant_type=authorization_code" \
  -d "code=authorization-code" \
  -d "redirect_uri=http://localhost:3000/callback" \
  -d "code_verifier=pkce-verifier" \
  -d "client_id=your-client-id" \
  -d "client_secret=your-client-secret"
```

## Architecture

```
                    HTTP Layer
                         │
        ┌────────────────┼────────────────┐
        │                │                │
TenantResolver      Core Services      UI (optional)
        │                │                │
        │                │                │
    Store (PG)    Authorizer (Casbin)
```

### Package Layout

```
/auth
  /core       - Domain models and interfaces
  /store      - Postgres persistence layer
  /tenant     - Host-based tenant resolution
  /tokens     - JWT token service
  /sessions   - Session management
  /oauth      - OAuth2/OIDC implementation
  /policy     - Policy engine (JSON-based)
  /rbac       - Casbin RBAC implementation
  /audit      - Audit logging
  /crypto     - Cryptographic utilities
  /http       - HTTP handlers
  /ui         - Embeddable UI assets

/cmd/locky    - Main server binary
/ui           - Svelte UI source
/migrations   - Database migrations
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://localhost/locky?sslmode=disable` |
| `ADMIN_API_KEY` | Bootstrap admin API key | (none) |
| `BASE_DOMAIN` | Base domain for tenant subdomains | `auth.example.com` |
| `HTTP_ADDR` | HTTP server address | `:8080` |
| `ENABLE_UI` | Enable hosted UI | `true` |
| `ENABLE_ADMIN_UI` | Enable optional admin web UI at `/admin/ui` | `false` |
| `ADMIN_UI_USERNAME` | Username for admin UI login | `admin` |
| `ADMIN_UI_PASSWORD` | Password for admin UI login | `admin123` |
| `AUTO_MIGRATE` | Auto-run migrations | `true` |

### Optional Admin UI

Set `ENABLE_ADMIN_UI=true` to expose the built-in admin UI at `http://localhost:8080/admin/ui`.
Sign in with `ADMIN_UI_USERNAME` / `ADMIN_UI_PASSWORD`, then enter `X-Admin-Key` for admin API actions.

### Library Usage

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/locky/auth"
    "github.com/locky/auth/http"
    "github.com/locky/auth/store"
)

func main() {
    // Create store
    gormStore, err := store.New("postgres://localhost/locky")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create core
    cfg := auth.Config{
        DatabaseURL: "postgres://localhost/locky",
        BaseDomain:  "auth.example.com",
    }
    
    core, err := auth.NewCore(cfg, gormStore, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create HTTP server
    srv := authhttp.NewServer(core, cfg)
    http.ListenAndServe(":8080", srv)
}
```

## API Endpoints

### Public Endpoints (per-tenant)

| Endpoint | Description |
|----------|-------------|
| `/.well-known/openid-configuration` | OIDC discovery |
| `/oauth2/authorize` | Authorization endpoint |
| `/oauth2/token` | Token endpoint |
| `/oauth2/userinfo` | UserInfo endpoint |
| `/oauth2/revoke` | Token revocation |
| `/oauth2/introspect` | Token introspection |
| `/oauth2/jwks.json` | JWKS endpoint |
| `/ui/login` | Hosted login page |
| `/ui/logout` | Logout page |

### Admin Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /healthz` | Health check |
| `GET /admin/tenants` | List tenants |
| `POST /admin/tenants` | Create tenant |
| `GET /admin/tenants/{id}` | Get tenant |
| `PATCH /admin/tenants/{id}` | Update tenant |
| `GET /admin/tenants/{id}/users` | List users |
| `POST /admin/tenants/{id}/users` | Create user |
| `PUT /admin/tenants/{id}/users/{id}/password` | Set password |

See `openapi.yaml` for complete API specification.

## Security

- **Argon2id** password hashing
- **PKCE** required for all authorization flows
- **Short-lived** JWT access tokens (default: 15 min)
- **Token rotation** for refresh tokens
- **Secure cookie** attributes enforced
- **AES-GCM** encrypted private keys
- **Audit logging** for all sensitive operations

## Python API Integration Tests

An end-to-end Python suite is available under `integration_tests/` and targets a live Locky instance.

### Install test dependencies

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements-integration.txt
```

### Run against local docker-compose stack

```bash
docker compose up -d --build
pytest
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOCKY_BASE_URL` | `http://localhost:8080` | Base URL of Locky API |
| `LOCKY_ADMIN_API_KEY` | `test-admin-key-123` | Admin API key sent as `X-Admin-Key` |
| `LOCKY_TENANT_HOST` | `localhost` | Host header used for tenant resolution |
| `LOCKY_SEEDED_EMAIL` | `test@example.com` | Seeded user email for OAuth flow tests |
| `LOCKY_SEEDED_PASSWORD` | `password123` | Seeded user password for OAuth flow tests |
| `LOCKY_SEEDED_CLIENT_ID` | `test-client-id` | Seeded OAuth client id |
| `LOCKY_SEEDED_REDIRECT_URI` | `http://localhost:3000/callback` | Redirect URI for auth code flow |
| `LOCKY_SEEDED_SCOPE` | `openid profile email` | Scope used in OAuth tests |

The suite always validates health/admin endpoints and runs OAuth authorization-code tests when seeded OAuth prerequisites are present.

## Roadmap

- [ ] MFA (TOTP, WebAuthn)
- [ ] SCIM provisioning
- [ ] SAML Service Provider
- [ ] Device Code flow
- [ ] Token introspection caching
- [ ] Rate limiting
- [ ] Custom email templates
- [ ] Admin console UI

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- GitHub Issues: https://github.com/DavidHoenisch/locky/issues
