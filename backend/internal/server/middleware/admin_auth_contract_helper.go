package middleware

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	tenantAdminAPIKeyUsersRoute = "/api/v1/admin/users/:id/api-keys"
	tenantAdminAPIKeyRoute      = "/api/v1/admin/api-keys/:id"
)

func abortAdminAuthError(c *gin.Context, statusCode int, legacyCode, message, contractReason string) {
	if !isTenantAdminAPIKeyContractRoute(c) {
		AbortWithError(c, statusCode, legacyCode, message)
		return
	}

	var appErr *infraerrors.ApplicationError
	switch statusCode {
	case http.StatusUnauthorized:
		appErr = infraerrors.Unauthorized(contractReason, message)
	case http.StatusForbidden:
		appErr = infraerrors.Forbidden(contractReason, message)
	default:
		appErr = infraerrors.InternalServer(contractReason, message)
	}

	if requestID := adminAuthRequestID(c); requestID != "" {
		appErr = appErr.WithMetadata(map[string]string{"request_id": requestID})
	}

	response.ErrorFrom(c, appErr)
	c.Abort()
}

func isTenantAdminAPIKeyContractRoute(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}

	switch c.Request.Method {
	case http.MethodPost:
		if c.FullPath() == tenantAdminAPIKeyUsersRoute {
			return true
		}
		path := c.Request.URL.Path
		return strings.HasPrefix(path, "/api/v1/admin/users/") && strings.HasSuffix(path, "/api-keys")
	case http.MethodPut, http.MethodDelete:
		if c.FullPath() == tenantAdminAPIKeyRoute {
			return true
		}
		return strings.HasPrefix(c.Request.URL.Path, "/api/v1/admin/api-keys/")
	default:
		return false
	}
}

func adminAuthRequestID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		return strings.TrimSpace(requestID)
	}
	if requestID := strings.TrimSpace(c.GetHeader(requestIDHeader)); requestID != "" {
		return requestID
	}
	return strings.TrimSpace(c.Writer.Header().Get(requestIDHeader))
}
