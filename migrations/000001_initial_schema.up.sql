-- 000001_initial_schema.up.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'suspended')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Tenant domains table
CREATE TABLE tenant_domains (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    domain TEXT UNIQUE NOT NULL,
    verified_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, domain)
);

CREATE INDEX idx_tenant_domains_domain ON tenant_domains(domain);
CREATE INDEX idx_tenant_domains_tenant ON tenant_domains(tenant_id);

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    display_name TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);

-- User passwords table
CREATE TABLE user_passwords (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id TEXT NULL,
    ip TEXT NULL,
    user_agent TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP NULL
);

CREATE INDEX idx_sessions_tenant ON sessions(tenant_id);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_tenant_user ON sessions(tenant_id, user_id);
CREATE INDEX idx_sessions_revoked ON sessions(revoked_at);

-- OAuth clients table
CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    client_id TEXT NOT NULL,
    client_secret_hash TEXT NULL,
    client_secret_last4 TEXT NULL,
    redirect_uris JSONB NOT NULL DEFAULT '[]',
    post_logout_redirect_uris JSONB NOT NULL DEFAULT '[]',
    grant_types JSONB NOT NULL DEFAULT '[]',
    response_types JSONB NOT NULL DEFAULT '[]',
    scopes JSONB NOT NULL DEFAULT '[]',
    token_ttl_seconds INTEGER NOT NULL DEFAULT 900,
    refresh_ttl_seconds INTEGER NOT NULL DEFAULT 1209600,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, client_id)
);

CREATE INDEX idx_clients_tenant ON clients(tenant_id);
CREATE INDEX idx_clients_tenant_client_id ON clients(tenant_id, client_id);

-- Signing keys table
CREATE TABLE signing_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kid TEXT NOT NULL,
    public_jwk JSONB NOT NULL,
    private_key_encrypted BYTEA NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'inactive', 'retired')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    not_before TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    not_after TIMESTAMP NOT NULL,
    UNIQUE (tenant_id, kid)
);

CREATE INDEX idx_signing_keys_tenant ON signing_keys(tenant_id);
CREATE INDEX idx_signing_keys_tenant_status ON signing_keys(tenant_id, status);

-- OAuth authorization codes table
CREATE TABLE oauth_authorization_codes (
    code_hash TEXT PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    pkce_challenge TEXT NOT NULL,
    pkce_method TEXT NOT NULL,
    scope TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_oauth_codes_tenant ON oauth_authorization_codes(tenant_id);
CREATE INDEX idx_oauth_codes_expires ON oauth_authorization_codes(expires_at);

-- OAuth refresh tokens table
CREATE TABLE oauth_refresh_tokens (
    token_hash TEXT PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP NULL,
    rotated_from_hash TEXT NULL
);

CREATE INDEX idx_refresh_tokens_tenant ON oauth_refresh_tokens(tenant_id);
CREATE INDEX idx_refresh_tokens_expires ON oauth_refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_revoked ON oauth_refresh_tokens(revoked_at);

-- Audit events table
CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_type TEXT NOT NULL CHECK (actor_type IN ('admin', 'user', 'system')),
    actor_id TEXT NULL,
    event_type TEXT NOT NULL,
    ip TEXT NULL,
    user_agent TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_audit_events_tenant ON audit_events(tenant_id);
CREATE INDEX idx_audit_events_tenant_type ON audit_events(tenant_id, event_type);
CREATE INDEX idx_audit_events_created ON audit_events(created_at);

-- Admin API keys table
CREATE TABLE admin_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_hash TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NULL
);

-- Policies table
CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL CHECK (status IN ('active', 'inactive')),
    document JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name, version)
);

CREATE INDEX idx_policies_tenant ON policies(tenant_id);
CREATE INDEX idx_policies_tenant_status ON policies(tenant_id, status);

-- Policy bindings table
CREATE TABLE policy_bindings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    bind_type TEXT NOT NULL CHECK (bind_type IN ('tenant', 'client', 'user', 'group')),
    bind_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (policy_id, bind_type, bind_id)
);

CREATE INDEX idx_policy_bindings_tenant ON policy_bindings(tenant_id);
CREATE INDEX idx_policy_bindings_policy ON policy_bindings(policy_id);

-- RBAC tuples table (for Casbin)
CREATE TABLE rbac_tuples (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tuple_type TEXT NOT NULL CHECK (tuple_type IN ('p', 'g')),
    v0 TEXT NOT NULL,
    v1 TEXT NOT NULL,
    v2 TEXT NOT NULL,
    v3 TEXT NULL,
    v4 TEXT NULL,
    v5 TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_rbac_tuples_tenant ON rbac_tuples(tenant_id);
CREATE INDEX idx_rbac_tuples_type ON rbac_tuples(tenant_id, tuple_type);
CREATE INDEX idx_rbac_tuples_lookup ON rbac_tuples(tenant_id, tuple_type, v0, v1, v2);