package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/locky/auth/core"
	"gorm.io/gorm"
)

// tenantStore implements core.TenantStore
type tenantStore struct {
	db *gorm.DB
}

func (s *tenantStore) Create(ctx context.Context, tenant *core.Tenant) error {
	model := &Tenant{
		ID:        tenant.ID,
		Slug:      tenant.Slug,
		Name:      tenant.Name,
		Status:    tenant.Status,
		CreatedAt: tenant.CreatedAt,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *tenantStore) GetByID(ctx context.Context, id string) (*core.Tenant, error) {
	var model Tenant
	if err := s.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return toCoreTenant(&model), nil
}

func (s *tenantStore) GetBySlug(ctx context.Context, slug string) (*core.Tenant, error) {
	var model Tenant
	if err := s.db.WithContext(ctx).First(&model, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return toCoreTenant(&model), nil
}

func (s *tenantStore) Update(ctx context.Context, tenant *core.Tenant) error {
	return s.db.WithContext(ctx).Model(&Tenant{}).Where("id = ?", tenant.ID).Updates(map[string]interface{}{
		"slug":   tenant.Slug,
		"name":   tenant.Name,
		"status": tenant.Status,
	}).Error
}

func (s *tenantStore) List(ctx context.Context, limit int, cursor string) ([]*core.Tenant, string, error) {
	var models []Tenant
	query := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit + 1)
	if cursor != "" {
		query = query.Where("created_at < ?", cursor)
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(models) > limit {
		nextCursor = models[limit].CreatedAt.Format(time.RFC3339)
		models = models[:limit]
	}

	tenants := make([]*core.Tenant, len(models))
	for i, m := range models {
		tenants[i] = toCoreTenant(&m)
	}
	return tenants, nextCursor, nil
}

func toCoreTenant(m *Tenant) *core.Tenant {
	return &core.Tenant{
		ID:        m.ID,
		Slug:      m.Slug,
		Name:      m.Name,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
	}
}

// userStore implements core.UserStore
type userStore struct {
	db *gorm.DB
}

func (s *userStore) Create(ctx context.Context, user *core.User) error {
	model := &User{
		ID:            user.ID,
		TenantID:      user.TenantID,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Status:        user.Status,
		DisplayName:   user.DisplayName,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *userStore) GetByID(ctx context.Context, tenantID, id string) (*core.User, error) {
	var model User
	if err := s.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id).Error; err != nil {
		return nil, err
	}
	return toCoreUser(&model), nil
}

func (s *userStore) GetByEmail(ctx context.Context, tenantID, email string) (*core.User, error) {
	var model User
	if err := s.db.WithContext(ctx).First(&model, "tenant_id = ? AND email = ?", tenantID, email).Error; err != nil {
		return nil, err
	}
	return toCoreUser(&model), nil
}

func (s *userStore) Update(ctx context.Context, user *core.User) error {
	return s.db.WithContext(ctx).Model(&User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"status":         user.Status,
		"display_name":   user.DisplayName,
		"updated_at":     user.UpdatedAt,
	}).Error
}

func (s *userStore) List(ctx context.Context, tenantID string, limit int, cursor string) ([]*core.User, string, error) {
	var models []User
	query := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit + 1)
	if cursor != "" {
		query = query.Where("created_at < ?", cursor)
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(models) > limit {
		nextCursor = models[limit].CreatedAt.Format(time.RFC3339)
		models = models[:limit]
	}

	users := make([]*core.User, len(models))
	for i, m := range models {
		users[i] = toCoreUser(&m)
	}
	return users, nextCursor, nil
}

func (s *userStore) SetPassword(ctx context.Context, userID string, hash string) error {
	return s.db.WithContext(ctx).Exec(
		`INSERT INTO user_passwords (user_id, password_hash, updated_at) 
		 VALUES (?, ?, ?) 
		 ON CONFLICT (user_id) DO UPDATE SET 
		 password_hash = EXCLUDED.password_hash, updated_at = EXCLUDED.updated_at`,
		userID, hash, time.Now(),
	).Error
}

func (s *userStore) GetPassword(ctx context.Context, userID string) (string, error) {
	var model UserPassword
	if err := s.db.WithContext(ctx).First(&model, "user_id = ?", userID).Error; err != nil {
		return "", err
	}
	return model.PasswordHash, nil
}

func toCoreUser(m *User) *core.User {
	return &core.User{
		ID:            m.ID,
		TenantID:      m.TenantID,
		Email:         m.Email,
		EmailVerified: m.EmailVerified,
		Status:        m.Status,
		DisplayName:   m.DisplayName,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

// sessionStore implements core.SessionStore
type sessionStore struct {
	db *gorm.DB
}

func (s *sessionStore) Create(ctx context.Context, session *core.Session) error {
	model := &Session{
		ID:         session.ID,
		TenantID:   session.TenantID,
		UserID:     session.UserID,
		ClientID:   session.ClientID,
		IP:         &session.IP,
		UserAgent:  &session.UserAgent,
		CreatedAt:  session.CreatedAt,
		LastSeenAt: session.LastSeenAt,
		RevokedAt:  session.RevokedAt,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *sessionStore) GetByID(ctx context.Context, tenantID, id string) (*core.Session, error) {
	var model Session
	if err := s.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id).Error; err != nil {
		return nil, err
	}
	return toCoreSession(&model), nil
}

func (s *sessionStore) Update(ctx context.Context, session *core.Session) error {
	return s.db.WithContext(ctx).Model(&Session{}).Where("id = ?", session.ID).Update("last_seen_at", session.LastSeenAt).Error
}

func (s *sessionStore) Revoke(ctx context.Context, tenantID, id string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Session{}).Where("tenant_id = ? AND id = ?", tenantID, id).Update("revoked_at", &now).Error
}

func (s *sessionStore) List(ctx context.Context, tenantID string, userID, clientID *string, activeOnly bool, limit int, cursor string) ([]*core.Session, string, error) {
	var models []Session
	query := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit + 1)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if clientID != nil {
		query = query.Where("client_id = ?", *clientID)
	}
	if activeOnly {
		query = query.Where("revoked_at IS NULL")
	}
	if cursor != "" {
		query = query.Where("created_at < ?", cursor)
	}

	if err := query.Find(&models).Error; err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(models) > limit {
		nextCursor = models[limit].CreatedAt.Format(time.RFC3339)
		models = models[:limit]
	}

	sessions := make([]*core.Session, len(models))
	for i, m := range models {
		sessions[i] = toCoreSession(&m)
	}
	return sessions, nextCursor, nil
}

func (s *sessionStore) DeleteExpired(ctx context.Context, before time.Time) error {
	return s.db.WithContext(ctx).Where("revoked_at IS NOT NULL OR created_at < ?", before).Delete(&Session{}).Error
}

func toCoreSession(m *Session) *core.Session {
	s := &core.Session{
		ID:         m.ID,
		TenantID:   m.TenantID,
		UserID:     m.UserID,
		ClientID:   m.ClientID,
		CreatedAt:  m.CreatedAt,
		LastSeenAt: m.LastSeenAt,
		RevokedAt:  m.RevokedAt,
	}
	if m.IP != nil {
		s.IP = *m.IP
	}
	if m.UserAgent != nil {
		s.UserAgent = *m.UserAgent
	}
	return s
}

// clientStore implements core.ClientStore
type clientStore struct {
	db *gorm.DB
}

func (s *clientStore) Create(ctx context.Context, client *core.Client) error {
	model := &Client{
		ID:                     client.ID,
		TenantID:               client.TenantID,
		Name:                   client.Name,
		ClientID:               client.ClientID,
		ClientSecretHash:       client.ClientSecretHash,
		ClientSecretLast4:      client.ClientSecretLast4,
		RedirectURIs:           StringSlice(client.RedirectURIs),
		PostLogoutRedirectURIs: StringSlice(client.PostLogoutRedirectURIs),
		GrantTypes:             StringSlice(client.GrantTypes),
		ResponseTypes:          StringSlice(client.ResponseTypes),
		Scopes:                 StringSlice(client.Scopes),
		TokenTTLSeconds:        client.TokenTTLSeconds,
		RefreshTTLSeconds:      client.RefreshTTLSeconds,
		CreatedAt:              client.CreatedAt,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *clientStore) GetByID(ctx context.Context, tenantID, id string) (*core.Client, error) {
	var model Client
	if err := s.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id).Error; err != nil {
		return nil, err
	}
	return toCoreClient(&model), nil
}

func (s *clientStore) GetByClientID(ctx context.Context, tenantID, clientID string) (*core.Client, error) {
	var model Client
	if err := s.db.WithContext(ctx).First(&model, "tenant_id = ? AND client_id = ?", tenantID, clientID).Error; err != nil {
		return nil, err
	}
	return toCoreClient(&model), nil
}

func (s *clientStore) Update(ctx context.Context, client *core.Client) error {
	return s.db.WithContext(ctx).Model(&Client{}).Where("id = ?", client.ID).Updates(map[string]interface{}{
		"name":                      client.Name,
		"redirect_uris":             StringSlice(client.RedirectURIs),
		"post_logout_redirect_uris": StringSlice(client.PostLogoutRedirectURIs),
		"grant_types":               StringSlice(client.GrantTypes),
		"response_types":            StringSlice(client.ResponseTypes),
		"scopes":                    StringSlice(client.Scopes),
		"token_ttl_seconds":         client.TokenTTLSeconds,
		"refresh_ttl_seconds":       client.RefreshTTLSeconds,
	}).Error
}

func (s *clientStore) Delete(ctx context.Context, tenantID, id string) error {
	return s.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&Client{}).Error
}

func (s *clientStore) List(ctx context.Context, tenantID string, limit int, cursor string) ([]*core.Client, string, error) {
	var models []Client
	query := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit + 1)
	if cursor != "" {
		query = query.Where("created_at < ?", cursor)
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(models) > limit {
		nextCursor = models[limit].CreatedAt.Format(time.RFC3339)
		models = models[:limit]
	}

	clients := make([]*core.Client, len(models))
	for i, m := range models {
		clients[i] = toCoreClient(&m)
	}
	return clients, nextCursor, nil
}

func toCoreClient(m *Client) *core.Client {
	return &core.Client{
		ID:                     m.ID,
		TenantID:               m.TenantID,
		Name:                   m.Name,
		ClientID:               m.ClientID,
		ClientSecretHash:       m.ClientSecretHash,
		ClientSecretLast4:      m.ClientSecretLast4,
		RedirectURIs:           []string(m.RedirectURIs),
		PostLogoutRedirectURIs: []string(m.PostLogoutRedirectURIs),
		GrantTypes:             []string(m.GrantTypes),
		ResponseTypes:          []string(m.ResponseTypes),
		Scopes:                 []string(m.Scopes),
		TokenTTLSeconds:        m.TokenTTLSeconds,
		RefreshTTLSeconds:      m.RefreshTTLSeconds,
		CreatedAt:              m.CreatedAt,
	}
}

// domainStore implements core.DomainStore
type domainStore struct {
	db *gorm.DB
}

func (s *domainStore) Create(ctx context.Context, domain *core.TenantDomain) error {
	model := &TenantDomain{
		ID:         domain.ID,
		TenantID:   domain.TenantID,
		Domain:     domain.Domain,
		VerifiedAt: domain.VerifiedAt,
		CreatedAt:  domain.CreatedAt,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *domainStore) GetByID(ctx context.Context, tenantID, id string) (*core.TenantDomain, error) {
	var model TenantDomain
	if err := s.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id).Error; err != nil {
		return nil, err
	}
	return toCoreDomain(&model), nil
}

func (s *domainStore) GetByDomain(ctx context.Context, domain string) (*core.TenantDomain, error) {
	var model TenantDomain
	if err := s.db.WithContext(ctx).First(&model, "domain = ?", domain).Error; err != nil {
		return nil, err
	}
	return toCoreDomain(&model), nil
}

func (s *domainStore) Delete(ctx context.Context, tenantID, id string) error {
	return s.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&TenantDomain{}).Error
}

func (s *domainStore) List(ctx context.Context, tenantID string) ([]*core.TenantDomain, error) {
	var models []TenantDomain
	if err := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&models).Error; err != nil {
		return nil, err
	}
	domains := make([]*core.TenantDomain, len(models))
	for i, m := range models {
		domains[i] = toCoreDomain(&m)
	}
	return domains, nil
}

func (s *domainStore) MarkVerified(ctx context.Context, tenantID, id string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&TenantDomain{}).Where("tenant_id = ? AND id = ?", tenantID, id).Update("verified_at", &now).Error
}

func toCoreDomain(m *TenantDomain) *core.TenantDomain {
	return &core.TenantDomain{
		ID:         m.ID,
		TenantID:   m.TenantID,
		Domain:     m.Domain,
		VerifiedAt: m.VerifiedAt,
		CreatedAt:  m.CreatedAt,
	}
}

// policyStore implements core.PolicyStore
type policyStore struct {
	db *gorm.DB
}

func (s *policyStore) Create(ctx context.Context, policy *core.Policy) error {
	docJSON, err := json.Marshal(policy.Document)
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}
	model := &Policy{
		ID:        policy.ID,
		TenantID:  policy.TenantID,
		Name:      policy.Name,
		Version:   policy.Version,
		Status:    policy.Status,
		Document:  docJSON,
		CreatedAt: policy.CreatedAt,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *policyStore) GetByID(ctx context.Context, tenantID, id string) (*core.Policy, error) {
	var model Policy
	if err := s.db.WithContext(ctx).First(&model, "tenant_id = ? AND id = ?", tenantID, id).Error; err != nil {
		return nil, err
	}
	return toCorePolicy(&model)
}

func (s *policyStore) Update(ctx context.Context, policy *core.Policy) error {
	updates := map[string]interface{}{
		"status": policy.Status,
	}
	if policy.Document != nil {
		docJSON, err := json.Marshal(policy.Document)
		if err != nil {
			return fmt.Errorf("marshal document: %w", err)
		}
		updates["document"] = docJSON
	}
	return s.db.WithContext(ctx).Model(&Policy{}).Where("id = ?", policy.ID).Updates(updates).Error
}

func (s *policyStore) List(ctx context.Context, tenantID string, status *string, limit int, cursor string) ([]*core.Policy, string, error) {
	var models []Policy
	query := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit + 1)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if cursor != "" {
		query = query.Where("created_at < ?", cursor)
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(models) > limit {
		nextCursor = models[limit].CreatedAt.Format(time.RFC3339)
		models = models[:limit]
	}

	policies := make([]*core.Policy, len(models))
	for i, m := range models {
		p, err := toCorePolicy(&m)
		if err != nil {
			return nil, "", err
		}
		policies[i] = p
	}
	return policies, nextCursor, nil
}

func toCorePolicy(m *Policy) (*core.Policy, error) {
	var doc map[string]interface{}
	if err := json.Unmarshal(m.Document, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal document: %w", err)
	}
	return &core.Policy{
		ID:        m.ID,
		TenantID:  m.TenantID,
		Name:      m.Name,
		Version:   m.Version,
		Status:    m.Status,
		Document:  doc,
		CreatedAt: m.CreatedAt,
	}, nil
}
