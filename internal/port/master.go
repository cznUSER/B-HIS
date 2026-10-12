package port

import (
	"context"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
)

// DepartmentQuery filters are AND-ed. Empty strings are ignored.
type DepartmentQuery struct {
	Status string
	Q      string
}

// DoctorQuery filters are AND-ed. Empty strings are ignored.
type DoctorQuery struct {
	DepartmentID string
	Status       string
	Q            string
}

// CatalogQuery filters are AND-ed. Empty strings are ignored.
type CatalogQuery struct {
	ItemType string
	Status   string
	Q        string
}

// UserQuery filters are AND-ed. Empty strings are ignored.
// Q matches username or display name.
type UserQuery struct {
	Status string
	Q      string
}

// MasterStore is department, doctor, catalog, inventory adjust, dict, and lightweight user admin.
// memory and sqlserver both implement it. Register/order reads use RegisterStore.GetDepartment
// and OrderStore.CatalogByItemNo; this port is the maintenance write side. 2026-10-13
type MasterStore interface {
	ListDepartments(ctx context.Context, q DepartmentQuery) ([]domain.Department, error)
	GetDepartment(ctx context.Context, departmentID string) (*domain.Department, error)
	CreateDepartment(ctx context.Context, d domain.Department) error
	UpdateDepartment(ctx context.Context, d domain.Department) (*domain.Department, error)

	ListDoctors(ctx context.Context, q DoctorQuery) ([]domain.Doctor, error)
	GetDoctor(ctx context.Context, doctorID string) (*domain.Doctor, error)
	CreateDoctor(ctx context.Context, d domain.Doctor) error
	UpdateDoctor(ctx context.Context, d domain.Doctor) (*domain.Doctor, error)

	ListCatalog(ctx context.Context, q CatalogQuery) ([]domain.CatalogItem, error)
	GetCatalog(ctx context.Context, catalogItemID string) (*domain.CatalogItem, error)
	// CreateCatalog inserts the catalog row. DRUG also inserts inventory_item at qty 0.
	// TREAT does not. skuID is the new inventory primary key and is ignored for TREAT.
	CreateCatalog(ctx context.Context, item domain.CatalogItem, skuID string) (*domain.CatalogItem, error)
	UpdateCatalog(ctx context.Context, item domain.CatalogItem) (*domain.CatalogItem, error)

	// AdjustInventory adds deltaMilli to qty_on_hand and appends a ledger row with biz_type ADJUST.
	// A negative result is ErrAdjustNegative. A missing SKU is ErrNotDrugSKU.
	// rowVersion empty skips the optimistic check; a mismatch is ErrConflict.
	AdjustInventory(ctx context.Context, skuID string, deltaMilli int64, remark, operatorID, rowVersion, ledgerID string, at time.Time) (*domain.InventoryItem, error)

	ListDictItems(ctx context.Context, dictType, status string) ([]domain.DictItem, error)
	CreateDictItem(ctx context.Context, item domain.DictItem) error
	UpdateDictItem(ctx context.Context, item domain.DictItem) (*domain.DictItem, error)

	ListUsers(ctx context.Context, q UserQuery) ([]domain.User, error)
	CreateUser(ctx context.Context, u domain.User) (*domain.User, error)
	UpdateUser(ctx context.Context, u domain.User, replaceRoles bool) (*domain.User, error)
	ResetPassword(ctx context.Context, userID, passwordHash, operatorID string, at time.Time) error
}
