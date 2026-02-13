package tenant

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/locky/auth/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock DomainStore
type mockDomainStore struct {
	domains map[string]*core.TenantDomain
}

func newMockDomainStore() *mockDomainStore {
	return &mockDomainStore{
		domains: make(map[string]*core.TenantDomain),
	}
}

func (m *mockDomainStore) Create(ctx context.Context, domain *core.TenantDomain) error {
	m.domains[domain.Domain] = domain
	return nil
}

func (m *mockDomainStore) GetByID(ctx context.Context, tenantID, id string) (*core.TenantDomain, error) {
	for _, d := range m.domains {
		if d.ID == id && d.TenantID == tenantID {
			return d, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockDomainStore) GetByDomain(ctx context.Context, domain string) (*core.TenantDomain, error) {
	if d, ok := m.domains[domain]; ok {
		return d, nil
	}
	return nil, errors.New("not found")
}

func (m *mockDomainStore) Delete(ctx context.Context, tenantID, id string) error {
	for k, d := range m.domains {
		if d.ID == id && d.TenantID == tenantID {
			delete(m.domains, k)
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockDomainStore) List(ctx context.Context, tenantID string) ([]*core.TenantDomain, error) {
	var result []*core.TenantDomain
	for _, d := range m.domains {
		if d.TenantID == tenantID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockDomainStore) MarkVerified(ctx context.Context, tenantID, id string) error {
	for _, d := range m.domains {
		if d.ID == id && d.TenantID == tenantID {
			now := ctx.Value("now").(interface{})
			_ = now // Would set verified_at
			return nil
		}
	}
	return errors.New("not found")
}

// Mock TenantStore
type mockTenantStore struct {
	tenants map[string]*core.Tenant
}

func newMockTenantStore() *mockTenantStore {
	return &mockTenantStore{
		tenants: make(map[string]*core.Tenant),
	}
}

func (m *mockTenantStore) Create(ctx context.Context, tenant *core.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantStore) GetByID(ctx context.Context, id string) (*core.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("not found")
}

func (m *mockTenantStore) GetBySlug(ctx context.Context, slug string) (*core.Tenant, error) {
	for _, t := range m.tenants {
		if t.Slug == slug {
			return t, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockTenantStore) Update(ctx context.Context, tenant *core.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantStore) List(ctx context.Context, limit int, cursor string) ([]*core.Tenant, string, error) {
	var result []*core.Tenant
	for _, t := range m.tenants {
		result = append(result, t)
		if len(result) >= limit {
			break
		}
	}
	return result, "", nil
}

func setupTestData(domains *mockDomainStore, tenants *mockTenantStore) {
	// Create test tenants
	tenant1 := &core.Tenant{
		ID:     "tenant-1",
		Slug:   "acme-corp",
		Name:   "Acme Corporation",
		Status: "active",
	}
	tenants.tenants[tenant1.ID] = tenant1

	tenant2 := &core.Tenant{
		ID:     "tenant-2",
		Slug:   "startup-io",
		Name:   "Startup.io",
		Status: "active",
	}
	tenants.tenants[tenant2.ID] = tenant2

	tenant3 := &core.Tenant{
		ID:     "tenant-3",
		Slug:   "suspended-tenant",
		Name:   "Suspended Corp",
		Status: "suspended",
	}
	tenants.tenants[tenant3.ID] = tenant3

	// Create custom domain mappings (verified domain must have VerifiedAt set)
	verifiedAt := time.Now()
	verifiedDomain := &core.TenantDomain{
		ID:         "domain-1",
		TenantID:   "tenant-1",
		Domain:     "auth.acme.com",
		VerifiedAt: &verifiedAt,
	}
	domains.domains["auth.acme.com"] = verifiedDomain

	unverifiedDomain := &core.TenantDomain{
		ID:         "domain-2",
		TenantID:   "tenant-2",
		Domain:     "auth.startup.io",
		VerifiedAt: nil,
	}
	domains.domains["auth.startup.io"] = unverifiedDomain
}

func TestHostResolver_ResolveTenant_ByCustomDomain(t *testing.T) {
	domains := newMockDomainStore()
	tenants := newMockTenantStore()
	setupTestData(domains, tenants)

	resolver := NewHostResolver(domains, tenants, "auth.example.com")

	ctx := context.Background()

	// Test verified custom domain
	t.Run("verified_custom_domain", func(t *testing.T) {
		tenant, err := resolver.ResolveTenant(ctx, "auth.acme.com")
		require.NoError(t, err)
		assert.Equal(t, "tenant-1", tenant.ID)
		assert.Equal(t, "acme-corp", tenant.Slug)
	})

	// Test custom domain with port
	t.Run("custom_domain_with_port", func(t *testing.T) {
		tenant, err := resolver.ResolveTenant(ctx, "auth.acme.com:8080")
		require.NoError(t, err)
		assert.Equal(t, "tenant-1", tenant.ID)
	})

	// Test custom domain with scheme
	t.Run("custom_domain_with_scheme", func(t *testing.T) {
		tenant, err := resolver.ResolveTenant(ctx, "https://auth.acme.com")
		require.NoError(t, err)
		assert.Equal(t, "tenant-1", tenant.ID)
	})
}

func TestHostResolver_ResolveTenant_BySubdomain(t *testing.T) {
	domains := newMockDomainStore()
	tenants := newMockTenantStore()
	setupTestData(domains, tenants)

	resolver := NewHostResolver(domains, tenants, "auth.example.com")

	ctx := context.Background()

	// Test subdomain
	t.Run("subdomain", func(t *testing.T) {
		tenant, err := resolver.ResolveTenant(ctx, "acme-corp.auth.example.com")
		require.NoError(t, err)
		assert.Equal(t, "tenant-1", tenant.ID)
	})

	// Test subdomain with port
	t.Run("subdomain_with_port", func(t *testing.T) {
		tenant, err := resolver.ResolveTenant(ctx, "startup-io.auth.example.com:8443")
		require.NoError(t, err)
		assert.Equal(t, "tenant-2", tenant.ID)
	})

	// Test subdomain with scheme
	t.Run("subdomain_with_scheme", func(t *testing.T) {
		tenant, err := resolver.ResolveTenant(ctx, "https://acme-corp.auth.example.com")
		require.NoError(t, err)
		assert.Equal(t, "tenant-1", tenant.ID)
	})
}

func TestHostResolver_ResolveTenant_NotFound(t *testing.T) {
	domains := newMockDomainStore()
	tenants := newMockTenantStore()
	setupTestData(domains, tenants)

	resolver := NewHostResolver(domains, tenants, "auth.example.com")

	ctx := context.Background()

	tests := []struct {
		name string
		host string
	}{
		{
			name: "unknown_subdomain",
			host: "unknown.auth.example.com",
		},
		{
			name: "unknown_custom_domain",
			host: "auth.unknown.com",
		},
		{
			name: "random_host",
			host: "example.com",
		},
		{
			name: "empty_host",
			host: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolver.ResolveTenant(ctx, tt.host)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple_host",
			input:    "auth.example.com",
			expected: "auth.example.com",
		},
		{
			name:     "host_with_port",
			input:    "auth.example.com:8080",
			expected: "auth.example.com",
		},
		{
			name:     "host_with_https",
			input:    "https://auth.example.com",
			expected: "auth.example.com",
		},
		{
			name:     "host_with_http",
			input:    "http://auth.example.com",
			expected: "auth.example.com",
		},
		{
			name:     "mixed_case",
			input:    "Auth.Example.COM",
			expected: "auth.example.com",
		},
		{
			name:     "host_with_path",
			input:    "https://auth.example.com/path",
			expected: "auth.example.com",
		},
		{
			name:     "empty_string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeHost(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSlug(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		baseDomain string
		expected   string
	}{
		{
			name:       "simple_subdomain",
			host:       "acme.auth.example.com",
			baseDomain: "auth.example.com",
			expected:   "acme",
		},
		{
			name:       "subdomain_with_port",
			host:       "acme.auth.example.com:8080",
			baseDomain: "auth.example.com",
			expected:   "acme",
		},
		{
			name:       "multi_level_subdomain",
			host:       "my-app.staging.auth.example.com",
			baseDomain: "auth.example.com",
			expected:   "my-app",
		},
		{
			name:       "no_match_wrong_base",
			host:       "acme.other.com",
			baseDomain: "auth.example.com",
			expected:   "",
		},
		{
			name:       "exact_base_domain",
			host:       "auth.example.com",
			baseDomain: "auth.example.com",
			expected:   "",
		},
		{
			name:       "empty_host",
			host:       "",
			baseDomain: "auth.example.com",
			expected:   "",
		},
		{
			name:       "case_insensitive",
			host:       "ACME.Auth.Example.COM",
			baseDomain: "auth.example.com",
			expected:   "acme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSlug(tt.host, tt.baseDomain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHostResolver_ResolveTenant_CaseInsensitivity(t *testing.T) {
	domains := newMockDomainStore()
	tenants := newMockTenantStore()
	setupTestData(domains, tenants)

	resolver := NewHostResolver(domains, tenants, "auth.example.com")

	ctx := context.Background()

	// Test various cases
	tests := []struct {
		name string
		host string
	}{
		{
			name: "uppercase_subdomain",
			host: "ACME-CORP.AUTH.EXAMPLE.COM",
		},
		{
			name: "mixed_case_subdomain",
			host: "Acme-Corp.Auth.Example.Com",
		},
		{
			name: "uppercase_custom_domain",
			host: "AUTH.ACME.COM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant, err := resolver.ResolveTenant(ctx, tt.host)
			require.NoError(t, err)
			assert.Equal(t, "tenant-1", tenant.ID)
		})
	}
}

func TestHostResolver_VerifiedDomainOnly(t *testing.T) {
	domains := newMockDomainStore()
	tenants := newMockTenantStore()
	setupTestData(domains, tenants)

	// Make one domain unverified
	domains.domains["auth.startup.io"] = &core.TenantDomain{
		ID:         "domain-2",
		TenantID:   "tenant-2",
		Domain:     "auth.startup.io",
		VerifiedAt: nil,
	}

	resolver := NewHostResolver(domains, tenants, "auth.example.com")

	ctx := context.Background()

	// Unverified domain should fail
	t.Run("unverified_domain_fails", func(t *testing.T) {
		_, err := resolver.ResolveTenant(ctx, "auth.startup.io")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not verified")
	})

	// But accessing via subdomain should still work
	t.Run("subdomain_still_works", func(t *testing.T) {
		tenant, err := resolver.ResolveTenant(ctx, "startup-io.auth.example.com")
		require.NoError(t, err)
		assert.Equal(t, "tenant-2", tenant.ID)
	})
}
