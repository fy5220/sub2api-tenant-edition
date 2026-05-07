package admin

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type adminAPIKeyLifecycle interface {
	AdminCreate(ctx context.Context, userID int64, req service.CreateAPIKeyRequest) (*service.APIKey, error)
	AdminUpdateStatus(ctx context.Context, keyID int64, status string) (*service.APIKey, error)
	AdminDelete(ctx context.Context, keyID int64) (*service.AdminDeleteAPIKeyResult, error)
}

// AdminAPIKeyHandler handles the TokenHub-facing admin API key lifecycle contract.
type AdminAPIKeyHandler struct {
	apiKeyService adminAPIKeyLifecycle
}

// NewAdminAPIKeyHandler creates a new admin API key handler.
func NewAdminAPIKeyHandler(apiKeyService *service.APIKeyService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

type AdminCreateAPIKeyRequest struct {
	Name          string   `json:"name" binding:"required"`
	GroupID       *int64   `json:"group_id"`
	CustomKey     *string  `json:"custom_key"`
	IPWhitelist   []string `json:"ip_whitelist"`
	IPBlacklist   []string `json:"ip_blacklist"`
	Quota         *float64 `json:"quota"`
	ExpiresInDays *int     `json:"expires_in_days"`
	RateLimit5h   *float64 `json:"rate_limit_5h"`
	RateLimit1d   *float64 `json:"rate_limit_1d"`
	RateLimit7d   *float64 `json:"rate_limit_7d"`
}

type AdminUpdateAPIKeyStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type adminAPIKeyCreateResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Secret    string    `json:"secret"`
}

type adminAPIKeyStatusResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type adminAPIKeyDeleteResponse struct {
	ID        int64     `json:"id"`
	Deleted   bool      `json:"deleted"`
	DeletedAt time.Time `json:"deleted_at"`
}

// Create handles POST /api/v1/admin/users/:id/api-keys.
func (h *AdminAPIKeyHandler) Create(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		tenantAPIKeyBadRequest(c, "invalid user id", map[string]string{"field": "id"})
		return
	}

	var req AdminCreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tenantAPIKeyInvalidField(c, "name", "invalid request")
		return
	}

	svcReq := service.CreateAPIKeyRequest{
		Name:          req.Name,
		GroupID:       req.GroupID,
		CustomKey:     req.CustomKey,
		IPWhitelist:   req.IPWhitelist,
		IPBlacklist:   req.IPBlacklist,
		ExpiresInDays: req.ExpiresInDays,
	}
	if req.Quota != nil {
		svcReq.Quota = *req.Quota
	}
	if req.RateLimit5h != nil {
		svcReq.RateLimit5h = *req.RateLimit5h
	}
	if req.RateLimit1d != nil {
		svcReq.RateLimit1d = *req.RateLimit1d
	}
	if req.RateLimit7d != nil {
		svcReq.RateLimit7d = *req.RateLimit7d
	}

	key, err := h.apiKeyService.AdminCreate(c.Request.Context(), userID, svcReq)
	if err != nil {
		tenantAPIKeyWriteServiceError(c, err, tenantTargetTypeUser, userID)
		return
	}

	response.Success(c, adminAPIKeyCreateResponse{
		ID:        key.ID,
		UserID:    key.UserID,
		Name:      key.Name,
		Status:    tenantAPIKeyStatusFromInternal(key.Status),
		CreatedAt: key.CreatedAt,
		Secret:    key.Key,
	})
}

// Update handles PUT /api/v1/admin/api-keys/:id and only accepts status changes.
func (h *AdminAPIKeyHandler) Update(c *gin.Context) {
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		tenantAPIKeyBadRequest(c, "invalid api key id", map[string]string{"field": "id"})
		return
	}

	var req AdminUpdateAPIKeyStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tenantAPIKeyInvalidField(c, "status", "invalid request")
		return
	}

	if _, err := tenantAPIKeyStatusFromRequest(req.Status); err != nil {
		tenantAPIKeyInvalidField(c, "status", "status must be active or inactive")
		return
	}

	key, err := h.apiKeyService.AdminUpdateStatus(c.Request.Context(), keyID, req.Status)
	if err != nil {
		tenantAPIKeyWriteServiceError(c, err, tenantTargetTypeAPIKey, keyID)
		return
	}

	response.Success(c, adminAPIKeyStatusResponse{
		ID:        key.ID,
		UserID:    key.UserID,
		Status:    tenantAPIKeyStatusFromInternal(key.Status),
		UpdatedAt: key.UpdatedAt,
	})
}

// Delete handles DELETE /api/v1/admin/api-keys/:id.
func (h *AdminAPIKeyHandler) Delete(c *gin.Context) {
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		tenantAPIKeyBadRequest(c, "invalid api key id", map[string]string{"field": "id"})
		return
	}

	result, err := h.apiKeyService.AdminDelete(c.Request.Context(), keyID)
	if err != nil {
		tenantAPIKeyWriteServiceError(c, err, tenantTargetTypeAPIKey, keyID)
		return
	}

	response.Success(c, adminAPIKeyDeleteResponse{
		ID:        keyID,
		Deleted:   true,
		DeletedAt: result.DeletedAt,
	})
}
