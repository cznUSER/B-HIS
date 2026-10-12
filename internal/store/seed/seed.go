// Package seed holds the documented local demo accounts. Passwords are bcrypt only.
package seed

import "github.com/cznUSER/B-HIS/internal/domain"

const (
	// StaffDefaultPassword is the password for every seeded staff account.
	// Future staff seeds must use this value. Only the bcrypt hash is stored.
	StaffDefaultPassword = "1"

	DemoUserID      = "00000000-0000-4000-8000-000000000001"
	DemoUsername    = "system"
	DemoPassword    = StaffDefaultPassword
	DemoDisplayName = "系统管理员"

	DisabledUserID      = "00000000-0000-4000-8000-000000000002"
	DisabledUsername    = "disabled"
	DisabledDisplayName = "已停用账号"

	// DemoPasswordHash is bcrypt(StaffDefaultPassword, cost 10).
	// Shared by the memory store, the disabled test account, and the SQL seed.
	DemoPasswordHash = "$2a$10$04.ABDU39oM7NW1GGCkdmuvvMrlm2fxFLxg1YuqDWJR.Zy.lXBPGK"
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
