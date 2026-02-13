-- Seed data for testing - run directly in PostgreSQL
-- This creates a test tenant, user, and OAuth client

-- Insert test tenant
INSERT INTO tenants (id, slug, name, status, created_at)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'test',
    'Test Tenant',
    'active',
    CURRENT_TIMESTAMP
) ON CONFLICT (slug) DO NOTHING;

-- Insert tenant domain (for localhost testing)
INSERT INTO tenant_domains (id, tenant_id, domain, verified_at, created_at)
VALUES (
    uuid_generate_v4(),
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'localhost',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (domain) DO NOTHING;

-- Insert test user
INSERT INTO users (id, tenant_id, email, email_verified, status, display_name, created_at, updated_at)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'test@example.com',
    true,
    'active',
    'Test User',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (tenant_id, email) DO NOTHING;

-- Insert test user password (password: 'password123')
-- This is an Argon2id hash of 'password123' - generated with: argon2id hash of 'password123'
INSERT INTO user_passwords (user_id, password_hash, updated_at)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12',
    '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$NK+ZC+lnNZ2xB2PqMm+jx0zZQe0HkZnQ3dFp9hZGJr0',
    CURRENT_TIMESTAMP
) ON CONFLICT (user_id) DO NOTHING;

-- Insert OAuth client for testing
-- Client secret: 'test-client-secret'
-- We'll leave client_secret_hash NULL for now and rely on client_id only for testing
INSERT INTO clients (
    id, tenant_id, name, client_id, client_secret_hash, client_secret_last4,
    redirect_uris, post_logout_redirect_uris, grant_types, response_types, scopes,
    token_ttl_seconds, refresh_ttl_seconds, created_at
)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Test Application',
    'test-client-id',
    NULL,  -- No client secret for public client (PKCE flow)
    NULL,
    '["http://localhost:3000/callback"]'::jsonb,
    '["http://localhost:3000"]'::jsonb,
    '["authorization_code", "refresh_token"]'::jsonb,
    '["code"]'::jsonb,
    '["openid", "profile", "email"]'::jsonb,
    900,
    1209600,
    CURRENT_TIMESTAMP
) ON CONFLICT (tenant_id, client_id) DO NOTHING;

-- Insert sample audit event
INSERT INTO audit_events (id, tenant_id, actor_type, actor_id, event_type, ip, user_agent, created_at, data)
VALUES (
    uuid_generate_v4(),
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'system',
    NULL,
    'tenant.created',
    '127.0.0.1',
    'docker-compose-seed',
    CURRENT_TIMESTAMP,
    '{"source": "docker-compose-seed"}'::jsonb
);
