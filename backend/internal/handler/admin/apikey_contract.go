package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	tenantAPIKeyStatusActive   = "active"
	tenantAPIKeyStatusInactive = "inactive"
	tenantTargetTypeAPIKey     = "api_key"
	tenantTargetTypeUser       = "user"
)

func tenantAPIKeyRequestID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		return strings.TrimSpace(requestID)
	}
	if requestID := strings.TrimSpace(c.GetHeader("X-Request-ID")); requestID != "" {
		return requestID
	}
	return strings.TrimSpace(c.Writer.Header().Get("X-Request-ID"))
}

func tenantAPIKeyMetadata(c *gin.Context, metadata map[string]string) map[string]string {
	out := map[string]string{}
	if requestID := tenantAPIKeyRequestID(c); requestID != "" {
		out["request_id"] = requestID
	}
	for key, value := range metadata {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func tenantAPIKeyError(c *gin.Context, statusCode int, reason, message string, metadata map[string]string) {
	c.JSON(statusCode, response.Response{
		Code:     statusCode,
		Message:  message,
		Reason:   reason,
		Metadata: tenantAPIKeyMetadata(c, metadata),
	})
}

func tenantAPIKeyInvalidField(c *gin.Context, field, message string) {
	tenantAPIKeyError(c, http.StatusUnprocessableEntity, "request_invalid", message, map[string]string{
		"field": field,
	})
}

func tenantAPIKeyBadRequest(c *gin.Context, message string, metadata map[string]string) {
	tenantAPIKeyError(c, http.StatusBadRequest, "request_invalid", message, metadata)
}

func tenantAPIKeyNotFound(c *gin.Context, targetType string, targetID int64) {
	tenantAPIKeyError(c, http.StatusNotFound, "resource_not_found", "resource not found", map[string]string{
		"target_type": targetType,
		"target_id":   strconv.FormatInt(targetID, 10),
	})
}

func tenantAPIKeyConflict(c *gin.Context, targetType string, targetID int64, message string) {
	tenantAPIKeyError(c, http.StatusConflict, "request_conflict", message, map[string]string{
		"target_type": targetType,
		"target_id":   strconv.FormatInt(targetID, 10),
	})
}

func tenantAPIKeyInternal(c *gin.Context, err error) {
	message := "internal error"
	if appErr := infraerrors.FromError(err); appErr != nil && appErr.Message != "" {
		message = appErr.Message
	}
	tenantAPIKeyError(c, http.StatusInternalServerError, "internal_error", message, nil)
}

func tenantAPIKeyWriteServiceError(c *gin.Context, err error, targetType string, targetID int64) {
	switch {
	case err == nil:
		return
	case errors.Is(err, service.ErrUserNotFound), errors.Is(err, service.ErrAPIKeyNotFound):
		tenantAPIKeyNotFound(c, targetType, targetID)
		return
	}

	appErr := infraerrors.FromError(err)
	if appErr == nil {
		tenantAPIKeyInternal(c, err)
		return
	}

	metadata := map[string]string{}
	for key, value := range appErr.Metadata {
		metadata[key] = value
	}
	if targetID > 0 {
		metadata["target_type"] = targetType
		metadata["target_id"] = strconv.FormatInt(targetID, 10)
	}

	switch int(appErr.Code) {
	case http.StatusConflict:
		tenantAPIKeyError(c, http.StatusConflict, "request_conflict", appErr.Message, metadata)
	case http.StatusTooManyRequests:
		if _, ok := metadata["retryable"]; !ok {
			metadata["retryable"] = "true"
		}
		tenantAPIKeyError(c, http.StatusTooManyRequests, "upstream_error", appErr.Message, metadata)
	case http.StatusUnauthorized:
		tenantAPIKeyError(c, http.StatusUnauthorized, "auth_failed", appErr.Message, metadata)
	case http.StatusForbidden:
		tenantAPIKeyError(c, http.StatusForbidden, "permission_denied", appErr.Message, metadata)
	case http.StatusNotFound:
		tenantAPIKeyError(c, http.StatusNotFound, "resource_not_found", "resource not found", metadata)
	case http.StatusUnprocessableEntity:
		tenantAPIKeyError(c, http.StatusUnprocessableEntity, "request_invalid", appErr.Message, metadata)
	case http.StatusBadGateway:
		tenantAPIKeyError(c, http.StatusBadGateway, "upstream_error", appErr.Message, metadata)
	case http.StatusBadRequest:
		tenantAPIKeyError(c, http.StatusBadRequest, "request_invalid", appErr.Message, metadata)
	default:
		tenantAPIKeyError(c, http.StatusInternalServerError, "internal_error", appErr.Message, metadata)
	}
}

func tenantAPIKeyStatusFromRequest(status string) (string, error) {
	switch strings.TrimSpace(status) {
	case tenantAPIKeyStatusActive:
		return tenantAPIKeyStatusActive, nil
	case tenantAPIKeyStatusInactive:
		return tenantAPIKeyStatusInactive, nil
	default:
		return "", fmt.Errorf("unsupported status %q", status)
	}
}

func tenantAPIKeyStatusFromInternal(status string) string {
	switch strings.TrimSpace(status) {
	case service.StatusAPIKeyDisabled:
		return tenantAPIKeyStatusInactive
	case service.StatusAPIKeyActive:
		return tenantAPIKeyStatusActive
	default:
		return strings.TrimSpace(status)
	}
}
