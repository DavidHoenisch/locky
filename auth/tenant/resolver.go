package tenant

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/locky/auth/core"
)

// HostResolver implements core.TenantResolver using host-based resolution
type HostResolver struct {
	domains    core.DomainStore
	tenants    core.TenantStore
	baseDomain string
}

// NewHostResolver creates a new HostResolver
func NewHostResolver(domains core.DomainStore, tenants core.TenantStore, baseDomain string) *HostResolver {
	return &HostResolver{
		domains:    domains,
		tenants:    tenants,
		baseDomain: baseDomain,
	}
}

// ResolveTenant resolves a tenant from a hostname
func (r *HostResolver) ResolveTenant(ctx context.Context, host string) (*core.Tenant, error) {
	// Normalize host: strip port, lowercase
	host = normalizeHost(host)

	// First, try exact match on custom domains
	domain, err := r.domains.GetByDomain(ctx, host)
	if err == nil && domain != nil {
		if domain.VerifiedAt == nil {
			return nil, fmt.Errorf("domain not verified")
		}
		return r.tenants.GetByID(ctx, domain.TenantID)
	}

	// Try to parse as subdomain of base domain
	slug := extractSlug(host, r.baseDomain)
	if slug != "" {
		return r.tenants.GetBySlug(ctx, slug)
	}

	return nil, fmt.Errorf("tenant not found for host: %s", host)
}

// normalizeHost strips port and lowercases the host
func normalizeHost(host string) string {
	// Handle URLs by parsing them
	if strings.Contains(host, "://") {
		u, err := url.Parse(host)
		if err == nil {
			host = u.Host
		}
	}

	// Strip port if present
	if i := strings.Index(host, ":"); i != -1 {
		host = host[:i]
	}

	return strings.ToLower(host)
}

// extractSlug extracts the tenant slug from a subdomain
// e.g., tenantSlug.auth.example.com -> tenantSlug
func extractSlug(host, baseDomain string) string {
	host = normalizeHost(host)
	baseDomain = normalizeHost(baseDomain)

	// Remove base domain suffix
	if !strings.HasSuffix(host, baseDomain) {
		return ""
	}

	prefix := strings.TrimSuffix(host, baseDomain)
	prefix = strings.TrimSuffix(prefix, ".")

	// Handle wildcard subdomains like *.auth.example.com
	// tenantSlug.auth.example.com -> tenantSlug
	parts := strings.Split(prefix, ".")
	if len(parts) >= 1 && parts[0] != "" {
		return parts[0]
	}

	return ""
}
