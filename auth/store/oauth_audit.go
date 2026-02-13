package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/locky/auth/core"
	"gorm.io/gorm"
)

// signingKeyStore implements core.SigningKeyStore
type signingKeyStore struct {
	db *gorm.DB
}

func (s *signingKeyStore) Create(ctx context.Context, key *core.SigningKey) error {
	model := &SigningKey{
		ID:                  key.ID,
		TenantID:            key.TenantID,
		KID:                 key.KID,
		PublicJWK:           key.PublicJWK,
		PrivateKeyEncrypted: key.PrivateKeyEncrypted,
		Status:              key.Status,
		CreatedAt:           key.CreatedAt,
		NotBefore:           key.NotBefore,
		NotAfter:            key.NotAfter,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *signingKeyStore) GetActive(ctx context.Context, tenantID string) (*core.SigningKey, error) {
	var model SigningKey
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ? AND not_before <= ? AND not_after > ?",
			tenantID, "active", time.Now(), time.Now()).
		Order("created_at DESC").
		First(&model).Error; err != nil {
		return nil, err
	}
	return toCoreSigningKey(&model), nil
}

func (s *signingKeyStore) GetByKID(ctx context.Context, tenantID, kid string) (*core.SigningKey, error) {
	var model SigningKey
	if err := s.db.WithContext(ctx).First(&model, "tenant_id = ? AND kid = ?", tenantID, kid).Error; err != nil {
		return nil, err
	}
	return toCoreSigningKey(&model), nil
}

func (s *signingKeyStore) ListActive(ctx context.Context, tenantID string) ([]*core.SigningKey, error) {
	var models []SigningKey
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND status IN (?, ?) AND not_after > ?",
			tenantID, "active", "inactive", time.Now()).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	keys := make([]*core.SigningKey, len(models))
	for i, m := range models {
		keys[i] = toCoreSigningKey(&m)
	}
	return keys, nil
}

func (s *signingKeyStore) MarkInactive(ctx context.Context, tenantID, id string) error {
	return s.db.WithContext(ctx).Model(&SigningKey{}).Where("id = ?", id).Update("status", "inactive").Error
}

func (s *signingKeyStore) MarkRetired(ctx context.Context, tenantID, id string) error {
	return s.db.WithContext(ctx).Model(&SigningKey{}).Where("id = ?", id).Update("status", "retired").Error
}

func toCoreSigningKey(m *SigningKey) *core.SigningKey {
	return &core.SigningKey{
		ID:                  m.ID,
		TenantID:            m.TenantID,
		KID:                 m.KID,
		PublicJWK:           m.PublicJWK,
		PrivateKeyEncrypted: m.PrivateKeyEncrypted,
		Status:              m.Status,
		CreatedAt:           m.CreatedAt,
		NotBefore:           m.NotBefore,
		NotAfter:            m.NotAfter,
	}
}

// oauthCodeStore implements core.OAuthCodeStore
type oauthCodeStore struct {
	db *gorm.DB
}

func (s *oauthCodeStore) Create(ctx context.Context, code *core.OAuthCode) error {
	model := &OAuthCode{
		CodeHash:      code.CodeHash,
		TenantID:      code.TenantID,
		ClientID:      code.ClientID,
		UserID:        code.UserID,
		RedirectURI:   code.RedirectURI,
		PKCEChallenge: code.PKCEChallenge,
		PKCEMethod:    code.PKCEMethod,
		Scope:         code.Scope,
		ExpiresAt:     code.ExpiresAt,
		UsedAt:        code.UsedAt,
		CreatedAt:     code.CreatedAt,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *oauthCodeStore) GetAndConsume(ctx context.Context, tenantID, codeHash string) (*core.OAuthCode, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	var model OAuthCode
	if err := tx.Clauses().First(&model, "code_hash = ? AND tenant_id = ?", codeHash, tenantID).Error; err != nil {
		return nil, err
	}

	if model.UsedAt != nil {
		return nil, fmt.Errorf("code already used")
	}

	if time.Now().After(model.ExpiresAt) {
		return nil, fmt.Errorf("code expired")
	}

	now := time.Now()
	if err := tx.Model(&OAuthCode{}).Where("code_hash = ?", codeHash).Update("used_at", &now).Error; err != nil {
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return toCoreOAuthCode(&model), nil
}

func (s *oauthCodeStore) DeleteExpired(ctx context.Context, before time.Time) error {
	return s.db.WithContext(ctx).
		Where("expires_at < ? OR used_at IS NOT NULL", before).
		Delete(&OAuthCode{}).Error
}

func toCoreOAuthCode(m *OAuthCode) *core.OAuthCode {
	return &core.OAuthCode{
		CodeHash:      m.CodeHash,
		TenantID:      m.TenantID,
		ClientID:      m.ClientID,
		UserID:        m.UserID,
		RedirectURI:   m.RedirectURI,
		PKCEChallenge: m.PKCEChallenge,
		PKCEMethod:    m.PKCEMethod,
		Scope:         m.Scope,
		ExpiresAt:     m.ExpiresAt,
		UsedAt:        m.UsedAt,
		CreatedAt:     m.CreatedAt,
	}
}

// refreshTokenStore implements core.RefreshTokenStore
type refreshTokenStore struct {
	db *gorm.DB
}

func (s *refreshTokenStore) Create(ctx context.Context, token *core.RefreshToken) error {
	model := &RefreshToken{
		TokenHash:       token.TokenHash,
		TenantID:        token.TenantID,
		ClientID:        token.ClientID,
		UserID:          token.UserID,
		Scope:           token.Scope,
		CreatedAt:       token.CreatedAt,
		ExpiresAt:       token.ExpiresAt,
		RevokedAt:       token.RevokedAt,
		RotatedFromHash: token.RotatedFromHash,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *refreshTokenStore) GetByHash(ctx context.Context, tenantID, hash string) (*core.RefreshToken, error) {
	var model RefreshToken
	if err := s.db.WithContext(ctx).First(&model, "token_hash = ? AND tenant_id = ?", hash, tenantID).Error; err != nil {
		return nil, err
	}
	return toCoreRefreshToken(&model), nil
}

func (s *refreshTokenStore) Revoke(ctx context.Context, tenantID, hash string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&RefreshToken{}).Where("token_hash = ?", hash).Update("revoked_at", &now).Error
}

func (s *refreshTokenStore) DeleteExpired(ctx context.Context, before time.Time) error {
	return s.db.WithContext(ctx).
		Where("expires_at < ? OR revoked_at IS NOT NULL", before).
		Delete(&RefreshToken{}).Error
}

func toCoreRefreshToken(m *RefreshToken) *core.RefreshToken {
	return &core.RefreshToken{
		TokenHash:       m.TokenHash,
		TenantID:        m.TenantID,
		ClientID:        m.ClientID,
		UserID:          m.UserID,
		Scope:           m.Scope,
		CreatedAt:       m.CreatedAt,
		ExpiresAt:       m.ExpiresAt,
		RevokedAt:       m.RevokedAt,
		RotatedFromHash: m.RotatedFromHash,
	}
}

// auditEventStore implements core.AuditEventStore
type auditEventStore struct {
	db *gorm.DB
}

func (s *auditEventStore) Create(ctx context.Context, event *core.AuditEvent) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}
	model := &AuditEvent{
		ID:        event.ID,
		TenantID:  event.TenantID,
		ActorType: event.ActorType,
		ActorID:   event.ActorID,
		EventType: event.Type,
		IP:        event.IP,
		UserAgent: event.UserAgent,
		CreatedAt: event.CreatedAt,
		Data:      dataJSON,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *auditEventStore) List(ctx context.Context, tenantID string, filters core.AuditFilters, limit int, cursor string) ([]*core.AuditEvent, string, error) {
	var models []AuditEvent
	query := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit + 1)

	if filters.Type != nil {
		query = query.Where("event_type = ?", *filters.Type)
	}
	if filters.ActorType != nil {
		query = query.Where("actor_type = ?", *filters.ActorType)
	}
	if filters.ActorID != nil {
		query = query.Where("actor_id = ?", *filters.ActorID)
	}
	if filters.Since != nil {
		query = query.Where("created_at >= ?", *filters.Since)
	}
	if filters.Until != nil {
		query = query.Where("created_at <= ?", *filters.Until)
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

	events := make([]*core.AuditEvent, len(models))
	for i, m := range models {
		e, err := toCoreAuditEvent(&m)
		if err != nil {
			return nil, "", err
		}
		events[i] = e
	}
	return events, nextCursor, nil
}

func toCoreAuditEvent(m *AuditEvent) (*core.AuditEvent, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(m.Data, &data); err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}
	return &core.AuditEvent{
		ID:        m.ID,
		TenantID:  m.TenantID,
		ActorType: m.ActorType,
		ActorID:   m.ActorID,
		Type:      m.EventType,
		IP:        m.IP,
		UserAgent: m.UserAgent,
		CreatedAt: m.CreatedAt,
		Data:      data,
	}, nil
}

// adminKeyStore implements core.AdminKeyStore
type adminKeyStore struct {
	db *gorm.DB
}

func (s *adminKeyStore) Create(ctx context.Context, key *core.AdminKey) error {
	model := &AdminKey{
		ID:        key.ID,
		KeyHash:   key.KeyHash,
		Name:      key.Name,
		CreatedAt: key.CreatedAt,
		CreatedBy: key.CreatedBy,
	}
	return s.db.WithContext(ctx).Create(model).Error
}

func (s *adminKeyStore) GetByHash(ctx context.Context, hash string) (*core.AdminKey, error) {
	var model AdminKey
	if err := s.db.WithContext(ctx).First(&model, "key_hash = ?", hash).Error; err != nil {
		return nil, err
	}
	return toCoreAdminKey(&model), nil
}

func (s *adminKeyStore) List(ctx context.Context) ([]*core.AdminKey, error) {
	var models []AdminKey
	if err := s.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	keys := make([]*core.AdminKey, len(models))
	for i, m := range models {
		keys[i] = toCoreAdminKey(&m)
	}
	return keys, nil
}

func (s *adminKeyStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&AdminKey{}).Error
}

func toCoreAdminKey(m *AdminKey) *core.AdminKey {
	return &core.AdminKey{
		ID:        m.ID,
		KeyHash:   m.KeyHash,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		CreatedBy: m.CreatedBy,
	}
}
