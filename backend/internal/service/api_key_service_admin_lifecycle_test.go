//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type adminLifecycleAPIKeyRepoStub struct {
	createCalls int
	createdKey  *APIKey
	createErr   error

	getByIDCalls int
	apiKey       *APIKey
	getByIDErr   error

	updateCalls int
	updatedKey  *APIKey
	updateErr   error

	deleteCalls    int
	deleteErr      error
	deleteEvidence bool
	deletedAt      time.Time
	deletedAtErr   error

	existsByKey bool
	existsErr   error
}

func (s *adminLifecycleAPIKeyRepoStub) Create(ctx context.Context, key *APIKey) error {
	s.createCalls++
	if s.createErr != nil {
		return s.createErr
	}
	clone := *key
	if clone.ID == 0 {
		clone.ID = 9001
	}
	now := time.Now()
	if clone.CreatedAt.IsZero() {
		clone.CreatedAt = now
	}
	if clone.UpdatedAt.IsZero() {
		clone.UpdatedAt = now
	}
	s.createdKey = &clone
	*key = clone
	return nil
}

func (s *adminLifecycleAPIKeyRepoStub) GetByID(ctx context.Context, id int64) (*APIKey, error) {
	s.getByIDCalls++
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	if s.apiKey == nil {
		panic("unexpected GetByID call")
	}
	clone := *s.apiKey
	return &clone, nil
}

func (s *adminLifecycleAPIKeyRepoStub) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	if s.getByIDErr != nil {
		return "", 0, s.getByIDErr
	}
	if s.apiKey == nil {
		panic("unexpected GetKeyAndOwnerID call")
	}
	return s.apiKey.Key, s.apiKey.UserID, nil
}

func (s *adminLifecycleAPIKeyRepoStub) GetByKey(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKey call")
}

func (s *adminLifecycleAPIKeyRepoStub) GetByKeyForAuth(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKeyForAuth call")
}

func (s *adminLifecycleAPIKeyRepoStub) Update(ctx context.Context, key *APIKey) error {
	s.updateCalls++
	if s.updateErr != nil {
		return s.updateErr
	}
	clone := *key
	clone.UpdatedAt = time.Now()
	s.updatedKey = &clone
	*key = clone
	return nil
}

func (s *adminLifecycleAPIKeyRepoStub) Delete(context.Context, int64) error {
	s.deleteCalls++
	return s.deleteErr
}

func (s *adminLifecycleAPIKeyRepoStub) HasDeleteEvidence(context.Context, int64) (bool, error) {
	return s.deleteEvidence, nil
}

func (s *adminLifecycleAPIKeyRepoStub) GetDeletedAt(context.Context, int64) (time.Time, error) {
	if s.deletedAtErr != nil {
		return time.Time{}, s.deletedAtErr
	}
	if s.deletedAt.IsZero() {
		panic("unexpected GetDeletedAt call")
	}
	return s.deletedAt, nil
}

func (s *adminLifecycleAPIKeyRepoStub) ListByUserID(context.Context, int64, pagination.PaginationParams, APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserID call")
}

func (s *adminLifecycleAPIKeyRepoStub) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	panic("unexpected VerifyOwnership call")
}

func (s *adminLifecycleAPIKeyRepoStub) CountByUserID(context.Context, int64) (int64, error) {
	panic("unexpected CountByUserID call")
}

func (s *adminLifecycleAPIKeyRepoStub) ExistsByKey(context.Context, string) (bool, error) {
	return s.existsByKey, s.existsErr
}

func (s *adminLifecycleAPIKeyRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID call")
}

func (s *adminLifecycleAPIKeyRepoStub) SearchAPIKeys(context.Context, int64, string, int) ([]APIKey, error) {
	panic("unexpected SearchAPIKeys call")
}

func (s *adminLifecycleAPIKeyRepoStub) ClearGroupIDByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected ClearGroupIDByGroupID call")
}

func (s *adminLifecycleAPIKeyRepoStub) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	panic("unexpected UpdateGroupIDByUserAndGroup call")
}

func (s *adminLifecycleAPIKeyRepoStub) CountByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected CountByGroupID call")
}

func (s *adminLifecycleAPIKeyRepoStub) ListKeysByUserID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByUserID call")
}

func (s *adminLifecycleAPIKeyRepoStub) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByGroupID call")
}

func (s *adminLifecycleAPIKeyRepoStub) IncrementQuotaUsed(context.Context, int64, float64) (float64, error) {
	panic("unexpected IncrementQuotaUsed call")
}

func (s *adminLifecycleAPIKeyRepoStub) UpdateLastUsed(context.Context, int64, time.Time) error {
	panic("unexpected UpdateLastUsed call")
}

func (s *adminLifecycleAPIKeyRepoStub) IncrementRateLimitUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementRateLimitUsage call")
}

func (s *adminLifecycleAPIKeyRepoStub) ResetRateLimitWindows(context.Context, int64) error {
	panic("unexpected ResetRateLimitWindows call")
}

func (s *adminLifecycleAPIKeyRepoStub) GetRateLimitData(context.Context, int64) (*APIKeyRateLimitData, error) {
	panic("unexpected GetRateLimitData call")
}

type adminLifecycleUserRepoStub struct {
	user *User
	err  error
}

func (s *adminLifecycleUserRepoStub) Create(context.Context, *User) error {
	panic("unexpected Create call")
}
func (s *adminLifecycleUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user == nil {
		panic("unexpected GetByID call")
	}
	clone := *s.user
	return &clone, nil
}
func (s *adminLifecycleUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	panic("unexpected GetByEmail call")
}
func (s *adminLifecycleUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin call")
}
func (s *adminLifecycleUserRepoStub) Update(context.Context, *User) error {
	panic("unexpected Update call")
}
func (s *adminLifecycleUserRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}
func (s *adminLifecycleUserRepoStub) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	panic("unexpected GetUserAvatar call")
}
func (s *adminLifecycleUserRepoStub) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	panic("unexpected UpsertUserAvatar call")
}
func (s *adminLifecycleUserRepoStub) DeleteUserAvatar(context.Context, int64) error {
	panic("unexpected DeleteUserAvatar call")
}
func (s *adminLifecycleUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *adminLifecycleUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (s *adminLifecycleUserRepoStub) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserIDs call")
}
func (s *adminLifecycleUserRepoStub) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserID call")
}
func (s *adminLifecycleUserRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	panic("unexpected UpdateUserLastActiveAt call")
}
func (s *adminLifecycleUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}
func (s *adminLifecycleUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}
func (s *adminLifecycleUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}
func (s *adminLifecycleUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}
func (s *adminLifecycleUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}
func (s *adminLifecycleUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}
func (s *adminLifecycleUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}
func (s *adminLifecycleUserRepoStub) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	panic("unexpected ListUserAuthIdentities call")
}
func (s *adminLifecycleUserRepoStub) UnbindUserAuthProvider(context.Context, int64, string) error {
	panic("unexpected UnbindUserAuthProvider call")
}
func (s *adminLifecycleUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}
func (s *adminLifecycleUserRepoStub) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}
func (s *adminLifecycleUserRepoStub) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}

func TestAPIKeyService_AdminCreate_UsesGenericCreateFlow(t *testing.T) {
	customKey := "tenant-admin-key-1234"
	repo := &adminLifecycleAPIKeyRepoStub{}
	userRepo := &adminLifecycleUserRepoStub{
		user: &User{ID: 17, Status: StatusActive},
	}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo:   userRepo,
	}

	got, err := svc.AdminCreate(context.Background(), 17, CreateAPIKeyRequest{
		Name:      "tenant-managed",
		CustomKey: &customKey,
	})
	require.NoError(t, err)
	require.Equal(t, 1, repo.createCalls)
	require.NotNil(t, repo.createdKey)
	require.Equal(t, int64(17), repo.createdKey.UserID)
	require.Equal(t, "tenant-managed", repo.createdKey.Name)
	require.Equal(t, customKey, repo.createdKey.Key)
	require.Equal(t, StatusActive, repo.createdKey.Status)
	require.Equal(t, repo.createdKey.Key, got.Key)
	require.Equal(t, StatusActive, got.Status)
}

func TestAPIKeyService_AdminUpdateStatus_MapsExternalStatuses(t *testing.T) {
	tests := []struct {
		name         string
		current      string
		requested    string
		wantInternal string
	}{
		{name: "repeat active", current: StatusActive, requested: "active", wantInternal: StatusActive},
		{name: "repeat inactive", current: StatusDisabled, requested: "inactive", wantInternal: StatusDisabled},
		{name: "transition inactive", current: StatusActive, requested: "inactive", wantInternal: StatusDisabled},
		{name: "transition active", current: StatusDisabled, requested: "active", wantInternal: StatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &adminLifecycleAPIKeyRepoStub{
				apiKey: &APIKey{ID: 51, UserID: 8, Key: "sk-status", Status: tt.current},
			}
			cache := &apiKeyCacheStub{}
			svc := &APIKeyService{apiKeyRepo: repo, cache: cache}

			got, err := svc.AdminUpdateStatus(context.Background(), 51, tt.requested)
			require.NoError(t, err)
			require.Equal(t, 1, repo.getByIDCalls)
			wantUpdates := 1
			if tt.current == tt.wantInternal {
				wantUpdates = 0
			}
			require.Equal(t, wantUpdates, repo.updateCalls)
			if wantUpdates == 1 {
				require.NotNil(t, repo.updatedKey)
				require.Equal(t, tt.wantInternal, repo.updatedKey.Status)
			}
			require.Equal(t, tt.wantInternal, got.Status)
			if wantUpdates == 1 {
				require.Equal(t, []string{svc.authCacheKey("sk-status")}, cache.deleteAuthKeys)
			} else {
				require.Empty(t, cache.deleteAuthKeys)
			}
		})
	}
}

func TestAPIKeyService_AdminUpdateStatus_InvalidExternalStatus(t *testing.T) {
	repo := &adminLifecycleAPIKeyRepoStub{}
	svc := &APIKeyService{apiKeyRepo: repo}

	_, err := svc.AdminUpdateStatus(context.Background(), 99, "paused")
	require.ErrorIs(t, err, ErrAdminAPIKeyStatusInvalid)
	require.Equal(t, 0, repo.getByIDCalls)
	require.Equal(t, 0, repo.updateCalls)
}

func TestAPIKeyService_AdminUpdateStatus_DeletedKeyCannotBeReupdated(t *testing.T) {
	repo := &adminLifecycleAPIKeyRepoStub{getByIDErr: ErrAPIKeyNotFound, deleteEvidence: true}
	svc := &APIKeyService{apiKeyRepo: repo}

	_, err := svc.AdminUpdateStatus(context.Background(), 99, "active")
	require.ErrorIs(t, err, ErrAdminAPIKeyDeleted)
	require.Equal(t, 1, repo.getByIDCalls)
	require.Equal(t, 0, repo.updateCalls)
}

func TestAPIKeyService_AdminDelete_RepeatedDeleteSucceedsWithEvidence(t *testing.T) {
	repo := &adminLifecycleAPIKeyRepoStub{
		getByIDErr: ErrAPIKeyNotFound,
		deletedAt:  mustParseAdminLifecycleTime(t, "2026-05-05T10:04:00Z"),
	}
	cache := &apiKeyCacheStub{}
	svc := &APIKeyService{apiKeyRepo: repo, cache: cache}

	result, err := svc.AdminDelete(context.Background(), 88)
	require.NoError(t, err)
	require.Equal(t, mustParseAdminLifecycleTime(t, "2026-05-05T10:04:00Z"), result.DeletedAt)
	require.Equal(t, 1, repo.deleteCalls)
	require.Empty(t, cache.deleteAuthKeys)
}

func TestAPIKeyService_AdminDelete_FirstDeleteInvalidatesAuthCache(t *testing.T) {
	repo := &adminLifecycleAPIKeyRepoStub{
		apiKey:    &APIKey{ID: 88, UserID: 9, Key: "sk-delete"},
		deletedAt: mustParseAdminLifecycleTime(t, "2026-05-05T10:05:00Z"),
	}
	cache := &apiKeyCacheStub{}
	svc := &APIKeyService{apiKeyRepo: repo, cache: cache}

	result, err := svc.AdminDelete(context.Background(), 88)
	require.NoError(t, err)
	require.Equal(t, mustParseAdminLifecycleTime(t, "2026-05-05T10:05:00Z"), result.DeletedAt)
	require.Equal(t, 1, repo.deleteCalls)
	require.Equal(t, []string{svc.authCacheKey("sk-delete")}, cache.deleteAuthKeys)
}

func mustParseAdminLifecycleTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}
