//go:build unit

package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminAuthJWTValidatesTokenVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil)

	admin := &service.User{
		ID:           1,
		Email:        "admin@example.com",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		TokenVersion: 2,
		Concurrency:  1,
	}

	userRepo := &stubUserRepo{
		getByID: func(ctx context.Context, id int64) (*service.User, error) {
			if id != admin.ID {
				return nil, service.ErrUserNotFound
			}
			clone := *admin
			return &clone, nil
		},
	}
	userService := service.NewUserService(userRepo, nil, nil, nil)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(authService, userService, nil)))
	router.GET("/t", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	t.Run("token_version_mismatch_rejected", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion - 1,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "TOKEN_REVOKED")
	})

	t.Run("token_version_match_allows", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("websocket_token_version_mismatch_rejected", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion - 1,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt."+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "TOKEN_REVOKED")
	})

	t.Run("websocket_token_version_match_allows", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt."+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAdminAuthTenantAPIKeyContractRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil)

	admin := &service.User{
		ID:           1,
		Email:        "admin@example.com",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		TokenVersion: 1,
		Concurrency:  1,
	}
	member := &service.User{
		ID:           2,
		Email:        "member@example.com",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		TokenVersion: 1,
		Concurrency:  1,
	}

	t.Run("missing_auth_uses_application_error_envelope", func(t *testing.T) {
		router := newAdminAuthTestRouter(authService, &stubUserRepo{}, &stubSettingRepo{})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/9/api-keys", nil)
		req.Header.Set(requestIDHeader, "rid-missing-auth")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.JSONEq(t, `{
			"code": 401,
			"message": "Authorization required",
			"reason": "auth_failed",
			"metadata": {"request_id": "rid-missing-auth"}
		}`, w.Body.String())
	})

	t.Run("invalid_admin_api_key_uses_application_error_envelope", func(t *testing.T) {
		router := newAdminAuthTestRouter(authService, &stubUserRepo{}, &stubSettingRepo{
			getValue: func(ctx context.Context, key string) (string, error) {
				return "expected-admin-key", nil
			},
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/7", nil)
		req.Header.Set(requestIDHeader, "rid-invalid-key")
		req.Header.Set("x-api-key", "wrong-key")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.JSONEq(t, `{
			"code": 401,
			"message": "Invalid admin API key",
			"reason": "auth_failed",
			"metadata": {"request_id": "rid-invalid-key"}
		}`, w.Body.String())
	})

	t.Run("non_admin_jwt_uses_permission_denied_envelope", func(t *testing.T) {
		userRepo := &stubUserRepo{
			getByID: func(ctx context.Context, id int64) (*service.User, error) {
				if id != member.ID {
					return nil, service.ErrUserNotFound
				}
				clone := *member
				return &clone, nil
			},
		}
		router := newAdminAuthTestRouter(authService, userRepo, &stubSettingRepo{})

		token, err := authService.GenerateToken(member)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/api-keys/7", nil)
		req.Header.Set(requestIDHeader, "rid-non-admin")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
		require.JSONEq(t, `{
			"code": 403,
			"message": "Admin access required",
			"reason": "permission_denied",
			"metadata": {"request_id": "rid-non-admin"}
		}`, w.Body.String())
	})

	t.Run("internal_setting_error_uses_internal_error_envelope", func(t *testing.T) {
		router := newAdminAuthTestRouter(authService, &stubUserRepo{}, &stubSettingRepo{
			getValue: func(ctx context.Context, key string) (string, error) {
				return "", errors.New("boom")
			},
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/7", nil)
		req.Header.Set(requestIDHeader, "rid-setting-error")
		req.Header.Set("x-api-key", "whatever")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.JSONEq(t, `{
			"code": 500,
			"message": "Internal server error",
			"reason": "internal_error",
			"metadata": {"request_id": "rid-setting-error"}
		}`, w.Body.String())
	})

	t.Run("unrelated_routes_keep_legacy_error_shape", func(t *testing.T) {
		userRepo := &stubUserRepo{
			getByID: func(ctx context.Context, id int64) (*service.User, error) {
				if id != admin.ID {
					return nil, service.ErrUserNotFound
				}
				clone := *admin
				return &clone, nil
			},
		}
		router := newAdminAuthTestRouter(authService, userRepo, &stubSettingRepo{})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/9/api-keys", nil)
		req.Header.Set(requestIDHeader, "rid-unrelated")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.JSONEq(t, `{
			"code": "UNAUTHORIZED",
			"message": "Authorization required"
		}`, w.Body.String())
	})
}

func newAdminAuthTestRouter(authService *service.AuthService, userRepo *stubUserRepo, settingRepo *stubSettingRepo) *gin.Engine {
	userService := service.NewUserService(userRepo, nil, nil, nil)
	settingService := service.NewSettingService(settingRepo, nil)

	router := gin.New()
	router.Use(RequestLogger())
	router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(authService, userService, settingService)))
	router.POST("/api/v1/admin/users/:id/api-keys", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.PUT("/api/v1/admin/api-keys/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.DELETE("/api/v1/admin/api-keys/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/v1/admin/users/:id/api-keys", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	return router
}

type stubUserRepo struct {
	getByID       func(ctx context.Context, id int64) (*service.User, error)
	getFirstAdmin func(ctx context.Context) (*service.User, error)
}

func (s *stubUserRepo) Create(ctx context.Context, user *service.User) error {
	panic("unexpected Create call")
}

func (s *stubUserRepo) GetByID(ctx context.Context, id int64) (*service.User, error) {
	if s.getByID == nil {
		panic("GetByID not stubbed")
	}
	return s.getByID(ctx, id)
}

func (s *stubUserRepo) GetByEmail(ctx context.Context, email string) (*service.User, error) {
	panic("unexpected GetByEmail call")
}

func (s *stubUserRepo) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	if s.getFirstAdmin == nil {
		panic("GetFirstAdmin not stubbed")
	}
	return s.getFirstAdmin(ctx)
}

func (s *stubUserRepo) Update(ctx context.Context, user *service.User) error {
	panic("unexpected Update call")
}

func (s *stubUserRepo) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *stubUserRepo) GetUserAvatar(ctx context.Context, userID int64) (*service.UserAvatar, error) {
	return nil, nil
}

func (s *stubUserRepo) UpsertUserAvatar(ctx context.Context, userID int64, input service.UpsertUserAvatarInput) (*service.UserAvatar, error) {
	panic("unexpected UpsertUserAvatar call")
}

func (s *stubUserRepo) DeleteUserAvatar(ctx context.Context, userID int64) error {
	panic("unexpected DeleteUserAvatar call")
}

func (s *stubUserRepo) List(ctx context.Context, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *stubUserRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *stubUserRepo) GetLatestUsedAtByUserIDs(ctx context.Context, userIDs []int64) (map[int64]*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserIDs call")
}

func (s *stubUserRepo) GetLatestUsedAtByUserID(ctx context.Context, userID int64) (*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserID call")
}

func (s *stubUserRepo) UpdateUserLastActiveAt(ctx context.Context, userID int64, activeAt time.Time) error {
	panic("unexpected UpdateUserLastActiveAt call")
}

func (s *stubUserRepo) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	panic("unexpected UpdateBalance call")
}

func (s *stubUserRepo) DeductBalance(ctx context.Context, id int64, amount float64) error {
	panic("unexpected DeductBalance call")
}

func (s *stubUserRepo) UpdateConcurrency(ctx context.Context, id int64, amount int) error {
	panic("unexpected UpdateConcurrency call")
}

func (s *stubUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}

func (s *stubUserRepo) RemoveGroupFromAllowedGroups(ctx context.Context, groupID int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}

func (s *stubUserRepo) RemoveGroupFromUserAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}

func (s *stubUserRepo) AddGroupToAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}

func (s *stubUserRepo) ListUserAuthIdentities(ctx context.Context, userID int64) ([]service.UserAuthIdentityRecord, error) {
	panic("unexpected ListUserAuthIdentities call")
}

func (s *stubUserRepo) UnbindUserAuthProvider(context.Context, int64, string) error {
	panic("unexpected UnbindUserAuthProvider call")
}

func (s *stubUserRepo) UpdateTotpSecret(ctx context.Context, userID int64, encryptedSecret *string) error {
	panic("unexpected UpdateTotpSecret call")
}

func (s *stubUserRepo) EnableTotp(ctx context.Context, userID int64) error {
	panic("unexpected EnableTotp call")
}

func (s *stubUserRepo) DisableTotp(ctx context.Context, userID int64) error {
	panic("unexpected DisableTotp call")
}

type stubSettingRepo struct {
	getValue func(ctx context.Context, key string) (string, error)
}

func (s *stubSettingRepo) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *stubSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	if s.getValue == nil {
		panic("GetValue not stubbed")
	}
	return s.getValue(ctx, key)
}

func (s *stubSettingRepo) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *stubSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *stubSettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *stubSettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *stubSettingRepo) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}
