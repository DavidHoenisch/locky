package rbac

import (
	"context"
	"fmt"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"github.com/locky/auth/core"
	"gorm.io/gorm"
)

// Service implements core.Authorizer using Casbin
type Service struct {
	enforcer *casbin.Enforcer
	db       *gorm.DB
}

// NewService creates a new RBAC service
func NewService(db *gorm.DB) (*Service, error) {
	// Create Casbin model
	m, err := model.NewModelFromString(`
		[request_definition]
		r = sub, dom, obj, act

		[policy_definition]
		p = sub, dom, obj, act

		[role_definition]
		g = _, _, _

		[policy_effect]
		e = some(where (p.eft == allow))

		[matchers]
		m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
	`)
	if err != nil {
		return nil, fmt.Errorf("create casbin model: %w", err)
	}

	// Create enforcer
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("create enforcer: %w", err)
	}

	return &Service{
		enforcer: enforcer,
		db:       db,
	}, nil
}

// Enforce checks if a subject can perform an action on an object within a domain
func (s *Service) Enforce(ctx context.Context, tenantID, subject, object, action string) (bool, error) {
	// Reload policies from database
	if err := s.loadPolicies(ctx, tenantID); err != nil {
		return false, err
	}

	return s.enforcer.Enforce(subject, tenantID, object, action)
}

// RolesForUser returns all roles for a user within a domain
func (s *Service) RolesForUser(ctx context.Context, tenantID, userID string) ([]string, error) {
	var tuples []core.RbacTuple
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND tuple_type = 'g' AND v0 = ?", tenantID, fmt.Sprintf("user:%s", userID)).
		Find(&tuples).Error; err != nil {
		return nil, err
	}

	roles := make([]string, len(tuples))
	for i, t := range tuples {
		roles[i] = t.V2 // V2 contains the role name
	}
	return roles, nil
}

// AddPolicy adds a policy or grouping
func (s *Service) AddPolicy(ctx context.Context, tenantID string, policy core.RbacTuple) error {
	tuple := &core.RbacTuple{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		TupleType: policy.TupleType,
		V0:        policy.V0,
		V1:        policy.V1,
		V2:        policy.V2,
		V3:        policy.V3,
		V4:        policy.V4,
		V5:        policy.V5,
		CreatedAt: time.Now(),
	}

	return s.db.WithContext(ctx).Create(tuple).Error
}

// RemovePolicy removes a policy by ID
func (s *Service) RemovePolicy(ctx context.Context, tenantID string, policyID string) error {
	return s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", policyID, tenantID).Delete(&core.RbacTuple{}).Error
}

// ListPolicies lists policies with optional filters
func (s *Service) ListPolicies(ctx context.Context, tenantID string, filters core.RbacFilters) ([]core.RbacTuple, string, error) {
	var tuples []core.RbacTuple
	query := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC")

	if filters.TupleType != nil {
		query = query.Where("tuple_type = ?", *filters.TupleType)
	}
	if filters.V0 != nil {
		query = query.Where("v0 = ?", *filters.V0)
	}
	if filters.V1 != nil {
		query = query.Where("v1 = ?", *filters.V1)
	}
	if filters.V2 != nil {
		query = query.Where("v2 = ?", *filters.V2)
	}
	if filters.V3 != nil {
		query = query.Where("v3 = ?", *filters.V3)
	}

	if err := query.Find(&tuples).Error; err != nil {
		return nil, "", err
	}

	return tuples, "", nil
}

// loadPolicies loads policies from the database into Casbin
func (s *Service) loadPolicies(ctx context.Context, tenantID string) error {
	var tuples []core.RbacTuple
	if err := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&tuples).Error; err != nil {
		return err
	}

	s.enforcer.ClearPolicy()

	for _, t := range tuples {
		if t.TupleType == "p" {
			// Policy rule
			v3 := ""
			if t.V3 != nil {
				v3 = *t.V3
			}
			s.enforcer.AddPolicy(t.V0, t.V1, t.V2, v3)
		} else if t.TupleType == "g" {
			// Grouping rule (user-role assignment)
			s.enforcer.AddGroupingPolicy(t.V0, t.V1, t.V2)
		}
	}

	return nil
}
