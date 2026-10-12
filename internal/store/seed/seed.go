// Package seed holds the documented local demo accounts. Passwords are bcrypt only.
package seed

import "github.com/cznUSER/B-HIS/internal/domain"

const (
	DemoUserID      = "00000000-0000-4000-8000-000000000001"
	DemoUsername    = "admin"
	DemoPassword    = "Admin@123"
	DemoDisplayName = "系统管理员"

	DisabledUserID      = "00000000-0000-4000-8000-000000000002"
	DisabledUsername    = "disabled"
	DisabledDisplayName = "已停用账号"

	// DemoPasswordHash is bcrypt(Admin@123, cost 10). Shared by the memory store and the SQL seed.
	DemoPasswordHash = "$2a$10$uVUbI161ahIWvB2/rbXUjuWubRAMog3zNNfldqp4mx/M.ZtuD8TgS"
)

func DemoUser() domain.User {
	return domain.User{
		ID:           DemoUserID,
		Username:     DemoUsername,
		DisplayName:  DemoDisplayName,
		PasswordHash: DemoPasswordHash,
		Roles:        []domain.Role{domain.RoleSystem},
		Status:       domain.StatusActive,
	}
}

func DisabledUser() domain.User {
	return domain.User{
		ID:           DisabledUserID,
		Username:     DisabledUsername,
		DisplayName:  DisabledDisplayName,
		PasswordHash: DemoPasswordHash,
		Roles:        []domain.Role{},
		Status:       domain.StatusDisabled,
	}
}
