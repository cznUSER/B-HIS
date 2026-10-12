package domain

import "time"

// Role is a Phase1 role code from 权限与审计.
type Role string

const (
	RoleRegistrar  Role = "REGISTRAR"
	RoleDoctor     Role = "DOCTOR"
	RoleCashier    Role = "CASHIER"
	RolePharmacist Role = "PHARMACIST"
	RoleSystem     Role = "SYSTEM"
)

const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
	StatusDisabled = "DISABLED"
)

// User is the identity aggregate used by login. PasswordHash is bcrypt; never plaintext.
type User struct {
	ID           string
	Username     string
	DisplayName  string
	PasswordHash string
	Roles        []Role
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    string
	UpdatedBy    string
}

func (u User) Active() bool {
	return u.Status == StatusActive
}

func (u User) RoleCodes() []string {
	out := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		out = append(out, string(r))
	}
	return out
}
