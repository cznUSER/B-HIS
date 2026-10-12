package domain

import "time"

// Charge status, payment direction, and pay types match OpenAPI 0.3.8 and 005_charge. 2026-10-12
const (
	ChargeStatusNormal = "NORMAL"
	ChargeStatusAbate  = "ABATE"

	PaymentDirectionCharge = "CHARGE"
	PaymentDirectionRefund = "REFUND"

	PaymentStatusPending   = "PENDING"
	PaymentStatusSucceeded = "SUCCEEDED"
	PaymentStatusFailed    = "FAILED"

	PayTypeCash   = "CASH"
	PayTypeOtherE = "OTHER_E"
	PayTypeWechat = "WECHAT"
	PayTypeAlipay = "ALIPAY"
	PayTypeOther  = "OTHER"

	FavourableAllocProportional     = "PROPORTIONAL"
	FavourableAllocDiscountOnRemain = "DISCOUNT_ON_REMAIN"
	FavourableAllocDiscountFirst    = "DISCOUNT_FIRST"

	// DefaultDrugOnHandMilli is the charge-side stock stub: 100.000 per drug item_no
	// when no explicit on-hand has been set. TREAT lines do not consume it.
	// There is no inventory table yet. Tests lower this to reach INSUFFICIENT_STOCK.
	// The demo catalog (DRUG-A / DRUG-B) stays payable at the default. 2026-10-12
	DefaultDrugOnHandMilli int64 = 100000
)

// ChargeItem is dbo.charge_item. QtyMilli is quantity × 1000. Money is yuan × 100.
// ItemType is not a 005 column: memory keeps it, SQL reloads leave it empty and the
// app derives DRUG/TREAT from the order line or catalog.
type ChargeItem struct {
	ChargeItemID     string
	ChargeID         string
	OrderItemID      string
	LineNo           int
	ItemNo           string
	ItemName         string
	ItemType         string
	QtyMilli         int64
	UnitPriceCents   int64
	TotalFeeCents    int64
	CanBack          bool
	RefundedQtyMilli int64
}

// Charge is dbo.charge (Square). encounterId ≡ RegisterID. Money is yuan × 100.
// PaidAt zero means 待收. Status stays NORMAL until pay or refund; there is no third state.
type Charge struct {
	ChargeID           string
	SquareNo           string
	OrderID            string
	RegisterID         string
	PatientID          string
	TotalFeeCents      int64
	SquareFeeCents     int64
	FavourableFeeCents int64
	Status             string
	SuccessorChargeID  string
	AbatedFromID       string
	PaidAt             time.Time
	Items              []ChargeItem
	CreatedAt          time.Time
	CreatedBy          string
	UpdatedAt          time.Time
	UpdatedBy          string
	IsDeleted          bool
}

// Payment is dbo.payment. AmountCents is yuan × 100.
type Payment struct {
	PaymentID   string
	PaymentNo   string
	ChargeID    string
	Direction   string
	PayType     string
	AmountCents int64
	Status      string
	PaidAt      time.Time
	OperatorID  string
	CreatedAt   time.Time
	CreatedBy   string
	UpdatedAt   time.Time
	UpdatedBy   string
}
