package domain

import "time"

// Order status values match OpenAPI 0.3.7 and 004_order CHECK. 2026-10-12
const (
	OrderStatusSave   = "SAVE"
	OrderStatusCommit = "COMMIT"
	OrderStatusProc   = "PROC"
	OrderStatusAbate  = "ABATE"

	ItemTypeDrug  = "DRUG"
	ItemTypeTreat = "TREAT"
)

// CatalogItem is dbo.catalog_item. IDs match 004_order_seed (cat-drug-a / cat-drug-b / cat-treat-1).
// PriceCents is yuan × 100.
type CatalogItem struct {
	CatalogItemID string
	ItemNo        string
	ItemName      string
	ItemType      string
	Spec          string
	Units         string
	PriceCents    int64
	Status        string
	IsDeleted     bool
}

// OrderItem is dbo.order_item. AmountMilli is quantity × 1000 (decimal 18,3).
// PriceCents and TotalFeeCents are yuan × 100.
type OrderItem struct {
	OrderItemID      string
	OrderID          string
	LineNo           int
	ItemNo           string
	ItemType         string
	ItemName         string
	Spec             string
	Units            string
	AmountMilli      int64
	PriceCents       int64
	TotalFeeCents    int64
	UsageText        string
	FrequencyText    string
	Dosage           string
	DosageUnit       string
	ProcDepartmentID string
}

// Order is dbo.order_sheet. encounterId ≡ RegisterID. 2026-10-12
type Order struct {
	OrderID       string
	OrderNo       string
	RegisterID    string
	PatientID     string
	DepartmentID  string
	DoctorID      string
	DiagnoseName  string
	RecipeSort    string
	Status        string
	TotalFeeCents int64
	Charged       bool
	Items         []OrderItem
	SubmittedAt   time.Time
	AbatedAt      time.Time
	AbatedBy      string
	CreatedAt     time.Time
	CreatedBy     string
	UpdatedAt     time.Time
	UpdatedBy     string
	IsDeleted     bool
}
