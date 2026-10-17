package memory

import (
	"context"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func (s *Store) ListUserRoleScopes(_ context.Context, userID string) ([]domain.UserRoleScope, error) {
	s.accessMu.RLock()
	defer s.accessMu.RUnlock()
	return append([]domain.UserRoleScope(nil), s.roleScopes[userID]...), nil
}

func (s *Store) ReplaceUserRoleScopes(_ context.Context, userID string, scopes []domain.UserRoleScope) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.byID[userID]
	if !ok {
		return port.ErrNotFound
	}
	if err := s.validateUserRoleScopesLocked(userID, user.Roles, scopes); err != nil {
		return err
	}
	s.roleScopes[userID] = normalizeUserRoleScopes(userID, scopes)
	return nil
}
