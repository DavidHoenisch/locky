package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/locky/auth/core"
	"github.com/locky/auth/crypto"
)

// AdminHandlers handles admin API endpoints
type AdminHandlers struct {
	store      core.Store
	keyManager core.KeyManager
	auditSink  core.AuditSink
	clock      core.Clock
}

// NewAdminHandlers creates new admin handlers
func NewAdminHandlers(store core.Store, keyManager core.KeyManager, auditSink core.AuditSink, clock core.Clock) *AdminHandlers {
	return &AdminHandlers{
		store:      store,
		keyManager: keyManager,
		auditSink:  auditSink,
		clock:      clock,
	}
}

// HealthHandler handles health checks
func (h *AdminHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":  "ok",
		"version": "0.1.0",
		"time":    h.clock.Now(),
	}
	data, _ := json.Marshal(health)
	writeJSON(w, http.StatusOK, data)
}

// ListTenants lists all tenants
func (h *AdminHandlers) ListTenants(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 || limit > 200 {
		limit = 50
	}
	cursor := r.URL.Query().Get("cursor")

	tenants, nextCursor, err := h.store.Tenants().List(r.Context(), limit, cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	resp := map[string]interface{}{
		"tenants":     tenants,
		"next_cursor": nextCursor,
	}
	data, _ := json.Marshal(resp)
	writeJSON(w, http.StatusOK, data)
}

// CreateTenant creates a new tenant
func (h *AdminHandlers) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	tenant := &core.Tenant{
		ID:        uuid.New().String(),
		Slug:      req.Slug,
		Name:      req.Name,
		Status:    "active",
		CreatedAt: h.clock.Now(),
	}

	if err := h.store.Tenants().Create(r.Context(), tenant); err != nil {
		writeError(w, http.StatusConflict, "conflict", "Tenant already exists")
		return
	}

	// Generate initial signing key for the tenant
	if _, err := h.keyManager.GenerateKey(r.Context(), tenant.ID); err != nil {
		// Log error but don't fail
		_ = err
	}

	// Audit log
	if h.auditSink != nil {
		h.auditSink.Log(r.Context(), &core.AuditEvent{
			ID:        uuid.New().String(),
			TenantID:  tenant.ID,
			ActorType: "admin",
			Type:      "tenant_created",
			CreatedAt: h.clock.Now(),
			Data: map[string]interface{}{
				"tenant_id": tenant.ID,
				"slug":      tenant.Slug,
			},
		})
	}

	data, _ := json.Marshal(tenant)
	writeJSON(w, http.StatusCreated, data)
}

// GetTenant gets a tenant by ID
func (h *AdminHandlers) GetTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		// Try getting from URL query for older Go versions
		// This is for backward compatibility
		tenantID = r.URL.Query().Get("tenant_id")
	}

	tenant, err := h.store.Tenants().GetByID(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Tenant not found")
		return
	}

	data, _ := json.Marshal(tenant)
	writeJSON(w, http.StatusOK, data)
}

// UpdateTenant updates a tenant
func (h *AdminHandlers) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")

	var req struct {
		Name   *string `json:"name"`
		Status *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	tenant, err := h.store.Tenants().GetByID(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Tenant not found")
		return
	}

	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.Status != nil {
		tenant.Status = *req.Status
	}

	if err := h.store.Tenants().Update(r.Context(), tenant); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	data, _ := json.Marshal(tenant)
	writeJSON(w, http.StatusOK, data)
}

// ListUsers lists users for a tenant
func (h *AdminHandlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 || limit > 200 {
		limit = 50
	}
	cursor := r.URL.Query().Get("cursor")

	users, nextCursor, err := h.store.Users().List(r.Context(), tenantID, limit, cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	resp := map[string]interface{}{
		"users":       users,
		"next_cursor": nextCursor,
	}
	data, _ := json.Marshal(resp)
	writeJSON(w, http.StatusOK, data)
}

// CreateUser creates a new user
func (h *AdminHandlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")

	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	if req.Status == "" {
		req.Status = "active"
	}

	now := h.clock.Now()
	user := &core.User{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Email:     req.Email,
		Status:    req.Status,
		CreatedAt: now,
		UpdatedAt: &now,
	}

	if req.DisplayName != "" {
		user.DisplayName = &req.DisplayName
	}

	if err := h.store.Users().Create(r.Context(), user); err != nil {
		writeError(w, http.StatusConflict, "conflict", "User already exists")
		return
	}

	data, _ := json.Marshal(user)
	writeJSON(w, http.StatusCreated, data)
}

// GetUser gets a user by ID
func (h *AdminHandlers) GetUser(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	userID := r.PathValue("user_id")

	user, err := h.store.Users().GetByID(r.Context(), tenantID, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	data, _ := json.Marshal(user)
	writeJSON(w, http.StatusOK, data)
}

// UpdateUser updates a user
func (h *AdminHandlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	userID := r.PathValue("user_id")

	var req struct {
		DisplayName   *string `json:"display_name"`
		Status        *string `json:"status"`
		EmailVerified *bool   `json:"email_verified"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	user, err := h.store.Users().GetByID(r.Context(), tenantID, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	if req.DisplayName != nil {
		user.DisplayName = req.DisplayName
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.EmailVerified != nil {
		user.EmailVerified = *req.EmailVerified
	}

	now := h.clock.Now()
	user.UpdatedAt = &now

	if err := h.store.Users().Update(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	data, _ := json.Marshal(user)
	writeJSON(w, http.StatusOK, data)
}

// SetUserPassword sets a user's password
func (h *AdminHandlers) SetUserPassword(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	// Hash password
	hasher := crypto.NewPasswordHasher()
	hash, err := hasher.Hash(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to hash password")
		return
	}

	if err := h.store.Users().SetPassword(r.Context(), userID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
