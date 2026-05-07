package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubAdminAPIKeyLifecycle struct {
	createFn       func(context.Context, int64, service.CreateAPIKeyRequest) (*service.APIKey, error)
	updateStatusFn func(context.Context, int64, string) (*service.APIKey, error)
	deleteFn       func(context.Context, int64) (*service.AdminDeleteAPIKeyResult, error)
}

func (s *stubAdminAPIKeyLifecycle) AdminCreate(ctx context.Context, userID int64, req service.CreateAPIKeyRequest) (*service.APIKey, error) {
	if s.createFn == nil {
		return nil, errors.New("create not stubbed")
	}
	return s.createFn(ctx, userID, req)
}

func (s *stubAdminAPIKeyLifecycle) AdminUpdateStatus(ctx context.Context, keyID int64, status string) (*service.APIKey, error) {
	if s.updateStatusFn == nil {
		return nil, errors.New("update status not stubbed")
	}
	return s.updateStatusFn(ctx, keyID, status)
}

func (s *stubAdminAPIKeyLifecycle) AdminDelete(ctx context.Context, keyID int64) (*service.AdminDeleteAPIKeyResult, error) {
	if s.deleteFn == nil {
		return nil, errors.New("delete not stubbed")
	}
	return s.deleteFn(ctx, keyID)
}

func setupAPIKeyHandler(lifecycle adminAPIKeyLifecycle) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := &AdminAPIKeyHandler{apiKeyService: lifecycle}
	router.POST("/api/v1/admin/users/:id/api-keys", h.Create)
	router.PUT("/api/v1/admin/api-keys/:id", h.Update)
	router.DELETE("/api/v1/admin/api-keys/:id", h.Delete)
	return router
}

func TestAdminAPIKeyHandler_Create_Success(t *testing.T) {
	router := setupAPIKeyHandler(&stubAdminAPIKeyLifecycle{
		createFn: func(_ context.Context, userID int64, req service.CreateAPIKeyRequest) (*service.APIKey, error) {
			require.Equal(t, int64(12), userID)
			require.Equal(t, "tenant-key", req.Name)
			return &service.APIKey{
				ID:        101,
				UserID:    userID,
				Name:      req.Name,
				Status:    service.StatusAPIKeyActive,
				Key:       "sk-secret",
				CreatedAt: mustParseTime(t, "2026-05-05T10:00:00Z"),
			}, nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/12/api-keys", bytes.NewBufferString(`{"name":"tenant-key"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-create-1")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID        int64  `json:"id"`
			UserID    int64  `json:"user_id"`
			Name      string `json:"name"`
			Status    string `json:"status"`
			Secret    string `json:"secret"`
			CreatedAt string `json:"created_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(101), resp.Data.ID)
	require.Equal(t, "active", resp.Data.Status)
	require.Equal(t, "sk-secret", resp.Data.Secret)
}

func TestAdminAPIKeyHandler_Create_UserNotFound(t *testing.T) {
	router := setupAPIKeyHandler(&stubAdminAPIKeyLifecycle{
		createFn: func(_ context.Context, _ int64, _ service.CreateAPIKeyRequest) (*service.APIKey, error) {
			return nil, service.ErrUserNotFound
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/99/api-keys", bytes.NewBufferString(`{"name":"tenant-key"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-create-404")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), `"reason":"resource_not_found"`)
	require.Contains(t, rec.Body.String(), `"target_id":"99"`)
}

func TestAdminAPIKeyHandler_Update_SuccessMapsInactive(t *testing.T) {
	router := setupAPIKeyHandler(&stubAdminAPIKeyLifecycle{
		updateStatusFn: func(_ context.Context, keyID int64, status string) (*service.APIKey, error) {
			require.Equal(t, int64(55), keyID)
			require.Equal(t, "inactive", status)
			return &service.APIKey{
				ID:        keyID,
				UserID:    7,
				Status:    service.StatusAPIKeyDisabled,
				UpdatedAt: mustParseTime(t, "2026-05-05T10:01:00Z"),
			}, nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/55", bytes.NewBufferString(`{"status":"inactive"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"inactive"`)
}

func TestAdminAPIKeyHandler_Update_InvalidStatus(t *testing.T) {
	router := setupAPIKeyHandler(&stubAdminAPIKeyLifecycle{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/55", bytes.NewBufferString(`{"status":"disabled"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-status-422")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Contains(t, rec.Body.String(), `"reason":"request_invalid"`)
	require.Contains(t, rec.Body.String(), `"field":"status"`)
	require.Contains(t, rec.Body.String(), `"request_id":"req-status-422"`)
}

func TestAdminAPIKeyHandler_Update_DeletedConflict(t *testing.T) {
	router := setupAPIKeyHandler(&stubAdminAPIKeyLifecycle{
		updateStatusFn: func(_ context.Context, _ int64, _ string) (*service.APIKey, error) {
			return nil, infraerrors.Conflict("API_KEY_ALREADY_DELETED", "api key has been deleted")
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/55", bytes.NewBufferString(`{"status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), `"reason":"request_conflict"`)
}

func TestAdminAPIKeyHandler_Create_RateLimitedMapsTo429(t *testing.T) {
	router := setupAPIKeyHandler(&stubAdminAPIKeyLifecycle{
		createFn: func(_ context.Context, _ int64, _ service.CreateAPIKeyRequest) (*service.APIKey, error) {
			return nil, service.ErrAPIKeyRateLimited
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/12/api-keys", bytes.NewBufferString(`{"name":"tenant-key"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-create-429")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Contains(t, rec.Body.String(), `"reason":"upstream_error"`)
	require.Contains(t, rec.Body.String(), `"retryable":"true"`)
	require.Contains(t, rec.Body.String(), `"target_id":"12"`)
}

func TestAdminAPIKeyHandler_Delete_Success(t *testing.T) {
	deletedAt := mustParseTime(t, "2026-05-05T10:02:00Z")
	router := setupAPIKeyHandler(&stubAdminAPIKeyLifecycle{
		deleteFn: func(_ context.Context, keyID int64) (*service.AdminDeleteAPIKeyResult, error) {
			require.Equal(t, int64(88), keyID)
			return &service.AdminDeleteAPIKeyResult{DeletedAt: deletedAt}, nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/api-keys/88", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"deleted":true`)
	require.Contains(t, rec.Body.String(), `"id":88`)
	require.Contains(t, rec.Body.String(), `"deleted_at":"2026-05-05T10:02:00Z"`)
}

func TestAdminAPIKeyHandler_Delete_UnknownKey(t *testing.T) {
	router := setupAPIKeyHandler(&stubAdminAPIKeyLifecycle{
		deleteFn: func(_ context.Context, _ int64) (*service.AdminDeleteAPIKeyResult, error) {
			return nil, service.ErrAPIKeyNotFound
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/api-keys/777", nil)
	req.Header.Set("X-Request-ID", "req-delete-404")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), `"reason":"resource_not_found"`)
	require.Contains(t, rec.Body.String(), `"request_id":"req-delete-404"`)
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}
