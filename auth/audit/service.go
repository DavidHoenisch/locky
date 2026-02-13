package audit

import (
	"context"

	"github.com/locky/auth/core"
)

// Service implements core.AuditSink
type Service struct {
	events core.AuditEventStore
}

// NewService creates a new audit service
func NewService(events core.AuditEventStore) *Service {
	return &Service{events: events}
}

// Log creates an audit log entry
func (s *Service) Log(ctx context.Context, event *core.AuditEvent) error {
	return s.events.Create(ctx, event)
}
