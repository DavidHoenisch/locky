-- 000001_initial_schema.down.sql

DROP TABLE IF EXISTS rbac_tuples;
DROP TABLE IF EXISTS policy_bindings;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS admin_keys;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS oauth_refresh_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS signing_keys;
DROP TABLE IF EXISTS clients;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_passwords;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenant_domains;
DROP TABLE IF EXISTS tenants;