package app

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// Charge API (OpenAPI 0.3.8). Money stays in cents (decimal 18,2). No float arithmetic. 2026-10-12
const (
	CodeInsufficientStock                = "INSUFFICIENT_STOCK"
	CodeInvalidAmount                    = "INVALID_AMOUNT"
	CodeItemNotCanBack                   = "ITEM_NOT_CAN_BACK"
	CodeRefundRequiresReturn             = "REFUND_REQUIRES_RETURN"
	CodeRefundExceedsRefundable          = "REFUND_EXCEEDS_REFUNDABLE"
	CodePartialRefundDiscountUnsupported = "PARTIAL_REFUND_DISCOUNT_UNSUPPORTED"
	CodeAlreadyAbated                    = "ALREADY_ABATED"
	CodeNotCharged                       = "NOT_CHARGED"
	CodeChargeAmountMismatch             = "CHARGE_AMOUNT_MISMATCH"
	CodeRegisterFeeNotChargeable         = "REGISTER_FEE_NOT_CHARGEABLE"
	CodeRegisterFeeAlreadyCharged        = "REGISTER_FEE_ALREADY_CHARGED"

	msgInsufficientStock                = "库存不足"
	msgInvalidAmount                    = "数量或金额非法"
	msgItemNotCanBack                   = "该明细不可退"
	msgRefundRequiresReturn             = "已发药须先退药"
	msgRefundExceedsRefundable          = "退费数量超过可退数量"
	msgPartialRefundDiscountUnsupported = "含优惠收费单暂不支持部分退费"
	msgAlreadyAbated                    = "结算单已作废"
	msgNotCharged                       = "尚未收费"
	msgChargeAmountMismatch             = "实收必须等于应收"
	msgChargeNotCommitted               = "仅已提交处方可收费"
	msgChargeExists                     = "该处方已有结算单"
	msgChargeAlreadyPaid                = "结算单已支付"
	msgRegisterFeeNotChargeable         = "当前挂号单没有待收挂号费"
	msgRegisterFeeAlreadyCharged        = "挂号费已收费"

	idemScopeChargeCreate = "charge.create"
	idemScopeChargePay    = "charge.pay"
	idemScopeChargeRefund = "charge.refund"
	// Durable stores validate expiry against wall-clock UTC, so this TTL must
	// not depend on the injectable business clock used for document timestamps.
	chargeIdempotencyTTL  = 24 * time.Hour
	chargeSearchLimit     = 100
)

// CreateChargeInput is POST /charges. orderId is required.
type CreateChargeInput struct {
	OrderID        string `json:"orderId"`
	IdempotencyKey string `json:"-"`
}

// CreateRegisterFeeCharge creates the standalone charge for a register fee.
// It intentionally has no order_id; the normal charge/pay/refund lifecycle is
// reused while the register row is marked by the charge store transaction.
func (s *Service) CreateRegisterFeeCharge(ctx context.Context, claims *auth.Claims, registerID, idempotencyKey, requestID, ip string) (*ChargeDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	registerID = strings.TrimSpace(registerID)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	var fields []string
	if registerID == "" || len(registerID) > 64 {
		fields = append(fields, "registerId")
	}
	if idempotencyKey != "" && !validIdempotencyKey(idempotencyKey) {
		fields = append(fields, "idempotencyKey")
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if s.Registers == nil || s.Charges == nil {
		return nil, errors.New("charge store is not configured")
	}
	reg, err := s.Registers.GetRegister(ctx, registerID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermChargePay, "outpatient", reg.DepartmentID) {
		e := forbidden()
		s.auditCharge(ctx, start, user, domain.ActionChargeCreate, registerID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if _, err := s.requireOpenRegister(ctx, registerID); err != nil {
		return nil, err
	}
	if reg.RegisterFeeCents <= 0 {
		e := &Error{Status: 409, Code: CodeRegisterFeeNotChargeable, Message: msgRegisterFeeNotChargeable}
		s.auditChargeErr(ctx, start, user, domain.ActionChargeCreate, registerID, requestID, ip, e)
		return nil, e
	}
	if reg.RegisterCharged {
		e := &Error{Status: 409, Code: CodeRegisterFeeAlreadyCharged, Message: msgRegisterFeeAlreadyCharged}
		s.auditChargeErr(ctx, start, user, domain.ActionChargeCreate, registerID, requestID, ip, e)
		return nil, e
	}
	hash := hashJSON(map[string]string{"registerId": registerID})
	atomicStore, atomic := s.Charges.(port.AtomicChargeIdempotencyStore)
	// A pending or paid fee charge must be unique for one open registration.
	if !atomic || idempotencyKey == "" {
		rows, err := s.Charges.SearchCharges(ctx, port.ChargeQuery{RegisterID: registerID, Limit: chargeSearchLimit})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row.OrderID == "" && registerFeeCharge(row) && row.Status != domain.ChargeStatusAbate {
				e := &Error{Status: 409, Code: CodeRegisterFeeAlreadyCharged, Message: msgRegisterFeeAlreadyCharged}
				s.auditChargeErr(ctx, start, user, domain.ActionChargeCreate, registerID, requestID, ip, e)
				return nil, e
			}
		}
	}
	if idempotencyKey != "" && !atomic {
		var prev ChargeDTO
		found, err := s.takeIdem(idemScopeChargeCreate, idempotencyKey, hash, &prev)
		if err != nil || found {
			return &prev, err
		}
	}
	now := s.now().UTC()
	var saved domain.Charge
	var replayed bool
	for attempt := 0; attempt < 5; attempt++ {
		id, err := newUUID()
		if err != nil {
			return nil, err
		}
		no, err := newBusinessNo("S", now)
		if err != nil {
			return nil, err
		}
		ch := chargeFromRegisterFee(*reg, id, no, user.ID, now)
		if atomic && idempotencyKey != "" {
			saved, replayed, err = atomicStore.CreateChargeWithIdempotency(ctx, ch, idemScopeChargeCreate, idempotencyKey, hash, time.Now().UTC().Add(chargeIdempotencyTTL))
		} else {
			err = s.Charges.CreateCharge(ctx, ch)
			if err == nil {
				saved = ch
			}
		}
		if errors.Is(err, port.ErrDuplicateBusinessNo) {
			continue
		}
		if errors.Is(err, port.ErrChargeExists) {
			e := &Error{Status: 409, Code: CodeRegisterFeeAlreadyCharged, Message: msgRegisterFeeAlreadyCharged}
			s.auditChargeErr(ctx, start, user, domain.ActionChargeCreate, registerID, requestID, ip, e)
			return nil, e
		}
		if err != nil {
			e := mapChargeErr(err)
			s.auditChargeErr(ctx, start, user, domain.ActionChargeCreate, registerID, requestID, ip, e)
			return nil, e
		}
		break
	}
	if saved.ChargeID == "" {
		return nil, errors.New("charge number allocation failed")
	}
	dto := toChargeDTO(saved)
	if replayed {
		return &dto, nil
	}
	if idempotencyKey != "" && !atomic {
		if err := s.putIdem(idemScopeChargeCreate, idempotencyKey, hash, dto); err != nil {
			return nil, err
		}
	}
	s.auditCharge(ctx, start, user, domain.ActionChargeCreate, dto.ChargeID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func chargeFromRegisterFee(reg domain.Register, chargeID, squareNo, userID string, now time.Time) domain.Charge {
	itemID, _ := newUUID()
	item := domain.ChargeItem{
		ChargeItemID: itemID, ChargeID: chargeID, LineNo: 1, ItemNo: "REGISTER-FEE",
		ItemName: "挂号费", ItemType: domain.ItemTypeTreat, QtyMilli: 1000,
		UnitPriceCents: reg.RegisterFeeCents, TotalFeeCents: reg.RegisterFeeCents, CanBack: true,
	}
	return domain.Charge{
		ChargeID: chargeID, SquareNo: squareNo, RegisterID: reg.RegisterID, PatientID: reg.PatientID,
		TotalFeeCents: reg.RegisterFeeCents, SquareFeeCents: reg.RegisterFeeCents,
		Status: domain.ChargeStatusNormal, Items: []domain.ChargeItem{item},
		CreatedAt: now, CreatedBy: userID, UpdatedAt: now, UpdatedBy: userID,
	}
}

func registerFeeCharge(c domain.Charge) bool {
	return len(c.Items) == 1 && c.Items[0].ItemNo == "REGISTER-FEE"
}

// PayInput is PayRequest. amount must equal squareFee.
type PayInput struct {
	PaymentNo      *string `json:"paymentNo"`
	PayType        string  `json:"payType"`
	Amount         float64 `json:"amount"`
	IdempotencyKey string  `json:"-"`
}

// RefundItemInput is one refund line.
type RefundItemInput struct {
	ChargeItemID string  `json:"chargeItemId"`
	RefundQty    float64 `json:"refundQty"`
}

// RefundInput is RefundRequest. favourableAlloc defaults to PROPORTIONAL (H08 draft, not a hard assert).
// reason is accepted and audited only as part of the request hash; 005_charge has no reason column.
type RefundInput struct {
	Reason          string            `json:"reason"`
	PayType         string            `json:"payType"`
	FavourableAlloc string            `json:"favourableAlloc"`
	Items           []RefundItemInput `json:"items"`
	IdempotencyKey  string            `json:"-"`
}

// ChargeItemDTO is OpenAPI ChargeItemDto.
type ChargeItemDTO struct {
	ChargeItemID string  `json:"chargeItemId"`
	OrderItemID  *string `json:"orderItemId"`
	LineNo       int     `json:"lineNo"`
	ItemNo       string  `json:"itemNo"`
	ItemName     string  `json:"itemName"`
	Qty          float64 `json:"qty"`
	UnitPrice    float64 `json:"unitPrice"`
	TotalFee     float64 `json:"totalFee"`
	CanBack      bool    `json:"canBack"`
	RefundedQty  float64 `json:"refundedQty"`
}

// ChargeDTO is OpenAPI ChargeDto. encounterId is always registerId.
type ChargeDTO struct {
	ChargeID          string          `json:"chargeId"`
	SquareNo          string          `json:"squareNo"`
	OrderID           *string         `json:"orderId"`
	RegisterID        string          `json:"registerId"`
	EncounterID       string          `json:"encounterId"`
	PatientID         string          `json:"patientId"`
	TotalFee          float64         `json:"totalFee"`
	SquareFee         float64         `json:"squareFee"`
	FavourableFee     float64         `json:"favourableFee"`
	Status            string          `json:"status"`
	SuccessorChargeID *string         `json:"successorChargeId"`
	AbatedFromID      *string         `json:"abatedFromId"`
	Items             []ChargeItemDTO `json:"items"`
	PaidAt            *string         `json:"paidAt"`
}

// ChargeListDTO is GET /charges.
type ChargeListDTO struct {
	Items []ChargeDTO `json:"items"`
}

// PaymentDTO is OpenAPI PaymentDto. succeededAt mirrors paidAt.
type PaymentDTO struct {
	PaymentID   string  `json:"paymentId"`
	PaymentNo   string  `json:"paymentNo"`
	ChargeID    string  `json:"chargeId"`
	Direction   string  `json:"direction"`
	PayType     string  `json:"payType"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	PaidAt      *string `json:"paidAt"`
	SucceededAt *string `json:"succeededAt"`
}

// RefundResultDTO is OpenAPI RefundResultDto. newSquare is null on a full refund.
type RefundResultDTO struct {
	AbatedCharge  ChargeDTO  `json:"abatedCharge"`
	NewSquare     *ChargeDTO `json:"newSquare"`
	RefundPayment PaymentDTO `json:"refundPayment"`
}

// ChargeSearch filters are AND-ed. encounterId is an alias of registerId.
// visitDate, or from+to, filters charge.created_at in Asia/Shanghai (not the register visit date).
// An empty query returns no rows. 2026-10-12
type ChargeSearch struct {
	PatientID   string
	RegisterID  string
	EncounterID string
	VisitDate   string
	From        string
	To          string
}

type stockLine struct {
	itemNo   string
	kind     string
	qtyMilli int64
}

// PreviewCharge builds an unsaved ChargeDto from a COMMIT order.
// It does not check stock and does not write a row. F=0 ⇒ totalFee=squareFee. 2026-10-12
func (s *Service) PreviewCharge(ctx context.Context, claims *auth.Claims, orderID string) (*ChargeDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	order, err := s.loadBillableOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermChargePay, "outpatient", order.DepartmentID) {
		return nil, forbidden()
	}
	ch, err := chargeFromOrder(*order, "", "", "", time.Time{})
	if err != nil {
		return nil, err
	}
	// Preview ids are placeholders. POST /charges allocates the real ones.
	ch.ChargeID = ""
	ch.SquareNo = ""
	for i := range ch.Items {
		ch.Items[i].ChargeItemID = ""
		ch.Items[i].ChargeID = ""
	}
	dto := toChargeDTO(ch)
	return &dto, nil
}

// CreateCharge loads one pending square from a COMMIT order (1:1, no split).
// Register status BACK, or a missing register, is ENCOUNTER_REQUIRED.
// Drug qty above the stub on-hand is INSUFFICIENT_STOCK. paidAt stays empty. HTTP 201. 2026-10-12
func (s *Service) CreateCharge(ctx context.Context, claims *auth.Claims, in CreateChargeInput, requestID, ip string) (*ChargeDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	orderID := strings.TrimSpace(in.OrderID)
	var fields []string
	if orderID == "" || len(orderID) > 64 {
		fields = append(fields, "orderId")
	}
	if in.IdempotencyKey != "" && !validIdempotencyKey(in.IdempotencyKey) {
		fields = append(fields, "idempotencyKey")
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	hash := hashJSON(map[string]string{"orderId": orderID})
	if s.Charges == nil || s.Orders == nil || s.Registers == nil {
		return nil, errors.New("charge store is not configured")
	}
	atomicIdemStore, atomicIdem := s.Charges.(port.AtomicChargeIdempotencyStore)
	if in.IdempotencyKey != "" && !atomicIdem {
		var prev ChargeDTO
		found, err := s.takeIdem(idemScopeChargeCreate, in.IdempotencyKey, hash, &prev)
		if err != nil || found {
			return &prev, err
		}
	}
	order, err := s.loadBillableOrder(ctx, orderID)
	if err != nil {
		s.auditChargeErr(ctx, start, user, domain.ActionChargeCreate, orderID, requestID, ip, err)
		return nil, err
	}
	if !hasPermScope(user, domain.PermChargePay, "outpatient", order.DepartmentID) {
		e := forbidden()
		s.auditCharge(ctx, start, user, domain.ActionChargeCreate, orderID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	// 拍板#13/#22 C-09：受控处方权校验默认关闭，收费不因处方权拒绝。
	if err := s.ensureStock(ctx, orderStock(*order)); err != nil {
		s.auditChargeErr(ctx, start, user, domain.ActionChargeCreate, orderID, requestID, ip, err)
		return nil, err
	}
	if !atomicIdem || in.IdempotencyKey == "" {
		existing, err := s.Charges.SearchCharges(ctx, port.ChargeQuery{OrderID: order.OrderID, Limit: 1})
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			e := &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgChargeExists}
			s.auditChargeErr(ctx, start, user, domain.ActionChargeCreate, orderID, requestID, ip, e)
			return nil, e
		}
	}
	now := s.now().UTC()
	var saved domain.Charge
	var replayed bool
	for attempt := 0; attempt < 5; attempt++ {
		id, err := newUUID()
		if err != nil {
			return nil, err
		}
		no, err := newBusinessNo("S", now)
		if err != nil {
			return nil, err
		}
		ch, err := chargeFromOrder(*order, id, no, user.ID, now)
		if err != nil {
			return nil, err
		}
		if atomicIdem && in.IdempotencyKey != "" {
			saved, replayed, err = atomicIdemStore.CreateChargeWithIdempotency(ctx, ch, idemScopeChargeCreate, in.IdempotencyKey, hash, time.Now().UTC().Add(chargeIdempotencyTTL))
		} else {
			err = s.Charges.CreateCharge(ctx, ch)
			if err == nil {
				saved = ch
			}
		}
		if errors.Is(err, port.ErrDuplicateBusinessNo) {
			continue
		}
		if err != nil {
			mapped := mapChargeErr(err)
			s.auditChargeErr(ctx, start, user, domain.ActionChargeCreate, orderID, requestID, ip, mapped)
			return nil, mapped
		}
		break
	}
	if saved.ChargeID == "" {
		return nil, errors.New("charge number allocation failed")
	}
	dto := toChargeDTO(saved)
	if replayed {
		return &dto, nil
	}
	if in.IdempotencyKey != "" && !atomicIdem {
		if err := s.putIdem(idemScopeChargeCreate, in.IdempotencyKey, hash, dto); err != nil {
			return nil, err
		}
	}
	s.auditCharge(ctx, start, user, domain.ActionChargeCreate, dto.ChargeID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

// PayCharge settles a pending square in cash (other pay types are stored, not sent to a provider).
// amount must equal squareFee. Encounter and stock are checked again. 2026-10-12
func (s *Service) PayCharge(ctx context.Context, claims *auth.Claims, chargeID string, in PayInput, requestID, ip string) (*PaymentDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	chargeID = strings.TrimSpace(chargeID)
	if chargeID == "" {
		return nil, notFound()
	}
	payType := strings.TrimSpace(in.PayType)
	var fields []string
	if !validPayType(payType) {
		fields = addField(fields, "payType")
	}
	if in.PaymentNo != nil {
		no := strings.TrimSpace(*in.PaymentNo)
		if no != "" && (len(no) > 32 || utf8.RuneCountInString(no) > 32) {
			fields = addField(fields, "paymentNo")
		}
	}
	if in.IdempotencyKey != "" && !validIdempotencyKey(in.IdempotencyKey) {
		fields = addField(fields, "idempotencyKey")
	}
	cents, amountOK := priceCents(in.Amount)
	if !amountOK || cents <= 0 {
		return nil, invalidAmount()
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	hash := hashJSON(map[string]any{"chargeId": chargeID, "payType": payType, "amountCents": cents})
	if s.Charges == nil || s.Registers == nil {
		return nil, errors.New("charge store is not configured")
	}
	atomicIdemStore, atomicIdem := s.Charges.(port.AtomicChargePaymentIdempotencyStore)
	if in.IdempotencyKey != "" && !atomicIdem {
		var prev PaymentDTO
		found, err := s.takeIdem(idemScopeChargePay, in.IdempotencyKey, hash, &prev)
		if err != nil || found {
			return &prev, err
		}
	}
	cur, err := s.loadCharge(ctx, chargeID)
	if err != nil {
		return nil, err
	}
	reg, err := s.Registers.GetRegister(ctx, cur.RegisterID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, &Error{Status: 409, Code: CodeEncounterRequired, Message: msgEncounterRequired}
	}
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermChargePay, "outpatient", reg.DepartmentID) {
		e := forbidden()
		s.auditCharge(ctx, start, user, domain.ActionChargePay, chargeID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if in.IdempotencyKey != "" && atomicIdem {
		_, savedPayment, found, err := atomicIdemStore.LookupPayIdempotency(ctx, idemScopeChargePay, in.IdempotencyKey, hash)
		if err != nil {
			return nil, mapChargeErr(err)
		}
		if found {
			dto := toPaymentDTO(*savedPayment)
			return &dto, nil
		}
	}
	if cur.Status == domain.ChargeStatusAbate {
		e := &Error{Status: 409, Code: CodeAlreadyAbated, Message: msgAlreadyAbated}
		s.auditChargeErr(ctx, start, user, domain.ActionChargePay, chargeID, requestID, ip, e)
		return nil, e
	}
	if !cur.PaidAt.IsZero() {
		e := &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgChargeAlreadyPaid}
		s.auditChargeErr(ctx, start, user, domain.ActionChargePay, chargeID, requestID, ip, e)
		return nil, e
	}
	if cents != cur.SquareFeeCents {
		e := &Error{Status: 409, Code: CodeChargeAmountMismatch, Message: msgChargeAmountMismatch}
		s.auditChargeErr(ctx, start, user, domain.ActionChargePay, chargeID, requestID, ip, e)
		return nil, e
	}
	if err := assertChargeFormula(*cur); err != nil {
		s.auditChargeErr(ctx, start, user, domain.ActionChargePay, chargeID, requestID, ip, err)
		return nil, err
	}
	if _, err := s.requireOpenRegister(ctx, cur.RegisterID); err != nil {
		s.auditChargeErr(ctx, start, user, domain.ActionChargePay, chargeID, requestID, ip, err)
		return nil, err
	}
	lines, err := s.chargeStockLines(ctx, *cur)
	if err != nil {
		return nil, err
	}
	if err := s.ensureStock(ctx, lines); err != nil {
		s.auditChargeErr(ctx, start, user, domain.ActionChargePay, chargeID, requestID, ip, err)
		return nil, err
	}
	now := s.now().UTC()
	var payment *domain.Payment
	for attempt := 0; attempt < 5; attempt++ {
		pid, err := newUUID()
		if err != nil {
			return nil, err
		}
		pno := ""
		if in.PaymentNo != nil {
			pno = strings.TrimSpace(*in.PaymentNo)
		}
		if pno == "" || attempt > 0 {
			pno, err = newBusinessNo("P", now)
			if err != nil {
				return nil, err
			}
		}
		pay := domain.Payment{
			PaymentID: pid, PaymentNo: pno, ChargeID: cur.ChargeID,
			Direction: domain.PaymentDirectionCharge, PayType: payType, AmountCents: cents,
			Status: domain.PaymentStatusSucceeded, PaidAt: now, OperatorID: user.ID,
			CreatedAt: now, CreatedBy: user.ID, UpdatedAt: now, UpdatedBy: user.ID,
		}
		if atomicIdem && in.IdempotencyKey != "" {
			var replayed bool
			_, payment, replayed, err = atomicIdemStore.PayChargeWithIdempotency(ctx, cur.ChargeID, pay, idemScopeChargePay, in.IdempotencyKey, hash, time.Now().UTC().Add(chargeIdempotencyTTL))
			if replayed {
				if payment == nil {
					return nil, errors.New("idempotent payment replay missing payment")
				}
				dto := toPaymentDTO(*payment)
				return &dto, nil
			}
			if err == nil {
				break
			}
		} else {
			_, payment, err = s.Charges.PayCharge(ctx, cur.ChargeID, pay)
		}
		if errors.Is(err, port.ErrDuplicateBusinessNo) {
			continue
		}
		if err != nil {
			mapped := mapChargeErr(err)
			s.auditChargeErr(ctx, start, user, domain.ActionChargePay, chargeID, requestID, ip, mapped)
			return nil, mapped
		}
		break
	}
	if payment == nil {
		return nil, errors.New("payment number allocation failed")
	}
	if cur.FavourableFeeCents == 0 && (payment.AmountCents != cur.SquareFeeCents || cur.TotalFeeCents != cur.SquareFeeCents) {
		return nil, errors.New("charge fee invariant")
	}
	dto := toPaymentDTO(*payment)
	if in.IdempotencyKey != "" && !atomicIdem {
		if err := s.putIdem(idemScopeChargePay, in.IdempotencyKey, hash, dto); err != nil {
			return nil, err
		}
	}
	s.auditCharge(ctx, start, user, domain.ActionChargePay, chargeID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

// RefundCharge abates a paid NORMAL square in one store transaction.
// A full refund leaves newSquare null. F=0 conserves refund + remainder = squareFee.
// Discounted charges can only be fully refunded until partial allocation is supported.
func (s *Service) RefundCharge(ctx context.Context, claims *auth.Claims, chargeID string, in RefundInput, requestID, ip string) (*RefundResultDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	chargeID = strings.TrimSpace(chargeID)
	if chargeID == "" {
		return nil, notFound()
	}
	if in.IdempotencyKey != "" && !validIdempotencyKey(in.IdempotencyKey) {
		return nil, validationFields([]string{"idempotencyKey"})
	}
	hash := hashJSON(in)
	if s.Charges == nil || s.Registers == nil {
		return nil, errors.New("charge store is not configured")
	}
	atomicIdemStore, atomicIdem := s.Charges.(port.AtomicChargePaymentIdempotencyStore)
	if in.IdempotencyKey != "" && !atomicIdem {
		var prev RefundResultDTO
		found, err := s.takeIdem(idemScopeChargeRefund, in.IdempotencyKey, hash, &prev)
		if err != nil || found {
			return &prev, err
		}
	}
	cur, err := s.loadCharge(ctx, chargeID)
	if err != nil {
		return nil, err
	}
	reg, err := s.Registers.GetRegister(ctx, cur.RegisterID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, &Error{Status: 409, Code: CodeEncounterRequired, Message: msgEncounterRequired}
	}
	if err != nil {
		return nil, err
	}
	if !hasPermScope(user, domain.PermChargeRefund, "outpatient", reg.DepartmentID) {
		e := forbidden()
		s.auditCharge(ctx, start, user, domain.ActionChargeRefund, chargeID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if in.IdempotencyKey != "" && atomicIdem {
		savedOld, savedNew, savedPay, found, err := atomicIdemStore.LookupRefundIdempotency(ctx, idemScopeChargeRefund, in.IdempotencyKey, hash)
		if err != nil {
			return nil, mapChargeErr(err)
		}
		if found {
			result := &RefundResultDTO{AbatedCharge: toChargeDTO(*savedOld), RefundPayment: toPaymentDTO(*savedPay)}
			if savedNew != nil {
				n := toChargeDTO(*savedNew)
				result.NewSquare = &n
			}
			return result, nil
		}
	}
	if cur.Status != domain.ChargeStatusNormal {
		e := &Error{Status: 409, Code: CodeAlreadyAbated, Message: msgAlreadyAbated}
		if cur.Status != domain.ChargeStatusAbate {
			e = &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgIllegalStateTransition}
		}
		s.auditChargeErr(ctx, start, user, domain.ActionChargeRefund, chargeID, requestID, ip, e)
		return nil, e
	}
	if cur.PaidAt.IsZero() {
		e := &Error{Status: 409, Code: CodeNotCharged, Message: msgNotCharged}
		s.auditChargeErr(ctx, start, user, domain.ActionChargeRefund, chargeID, requestID, ip, e)
		return nil, e
	}
	plan, err := s.planRefund(ctx, *cur, in)
	if err != nil {
		s.auditChargeErr(ctx, start, user, domain.ActionChargeRefund, chargeID, requestID, ip, err)
		return nil, err
	}
	now := s.now().UTC()
	var result *RefundResultDTO
	var replayed bool
	for attempt := 0; attempt < 5; attempt++ {
		old, next, pay, err := s.materializeRefund(*cur, plan, user.ID, now, in.PayType)
		if err != nil {
			return nil, err
		}
		var savedOld, savedNew *domain.Charge
		var savedPay *domain.Payment
		if atomicIdem && in.IdempotencyKey != "" {
			savedOld, savedNew, savedPay, replayed, err = atomicIdemStore.RefundChargeWithIdempotency(ctx, old, next, pay, idemScopeChargeRefund, in.IdempotencyKey, hash, time.Now().UTC().Add(chargeIdempotencyTTL))
		} else {
			savedOld, savedNew, savedPay, err = s.Charges.RefundCharge(ctx, old, next, pay)
		}
		if errors.Is(err, port.ErrDuplicateBusinessNo) {
			continue
		}
		if err != nil {
			mapped := mapChargeErr(err)
			s.auditChargeErr(ctx, start, user, domain.ActionChargeRefund, chargeID, requestID, ip, mapped)
			return nil, mapped
		}
		if replayed && savedOld == nil {
			return nil, errors.New("idempotent refund replay missing charge")
		}
		dto := RefundResultDTO{
			AbatedCharge:  toChargeDTO(*savedOld),
			RefundPayment: toPaymentDTO(*savedPay),
		}
		if savedNew != nil {
			n := toChargeDTO(*savedNew)
			dto.NewSquare = &n
		}
		result = &dto
		break
	}
	if result == nil {
		return nil, errors.New("refund number allocation failed")
	}
	if cur.FavourableFeeCents == 0 {
		newR := int64(0)
		if result.NewSquare != nil {
			newR = moneyCents(result.NewSquare.SquareFee)
		}
		if moneyCents(result.RefundPayment.Amount)+newR != cur.SquareFeeCents || (result.NewSquare == nil && moneyCents(result.RefundPayment.Amount) != cur.SquareFeeCents) {
			return nil, errors.New("refund fee invariant")
		}
	}
	if in.IdempotencyKey != "" && !atomicIdem {
		if err := s.putIdem(idemScopeChargeRefund, in.IdempotencyKey, hash, result); err != nil {
			return nil, err
		}
	}
	if !replayed {
		s.auditCharge(ctx, start, user, domain.ActionChargeRefund, chargeID, domain.OutcomeSuccess, "", requestID, ip)
	}
	return result, nil
}

// GetCharge returns one square or 404.
func (s *Service) GetCharge(ctx context.Context, claims *auth.Claims, chargeID string) (*ChargeDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasAnyPermModule(user, "outpatient", domain.PermChargePay, domain.PermChargeRefund) {
		return nil, forbidden()
	}
	cur, err := s.loadCharge(ctx, chargeID)
	if err != nil {
		return nil, err
	}
	if !chargeScopeAllowed(ctx, s.Registers, user, *cur, domain.PermChargePay, domain.PermChargeRefund) {
		return nil, forbidden()
	}
	dto := toChargeDTO(*cur)
	return &dto, nil
}

// SearchCharges lists squares. encounterId is an alias of registerId.
func (s *Service) SearchCharges(ctx context.Context, claims *auth.Claims, q ChargeSearch) (*ChargeListDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasAnyPermModule(user, "outpatient", domain.PermChargePay, domain.PermChargeRefund) {
		return nil, forbidden()
	}
	q.PatientID = strings.TrimSpace(q.PatientID)
	q.RegisterID = strings.TrimSpace(q.RegisterID)
	q.EncounterID = strings.TrimSpace(q.EncounterID)
	q.VisitDate = strings.TrimSpace(q.VisitDate)
	q.From = strings.TrimSpace(q.From)
	q.To = strings.TrimSpace(q.To)
	empty := &ChargeListDTO{Items: []ChargeDTO{}}
	if q.PatientID == "" && q.RegisterID == "" && q.EncounterID == "" && q.VisitDate == "" && q.From == "" && q.To == "" {
		return empty, nil
	}
	if q.RegisterID != "" && q.EncounterID != "" && q.RegisterID != q.EncounterID {
		return nil, validationFields([]string{"encounterId"})
	}
	if q.RegisterID == "" {
		q.RegisterID = q.EncounterID
	}
	var fields []string
	if q.VisitDate != "" && (q.From != "" || q.To != "") {
		fields = addField(fields, "visitDate")
	}
	if q.VisitDate != "" && !validDay(q.VisitDate) {
		fields = addField(fields, "visitDate")
	}
	if (q.From == "") != (q.To == "") && q.VisitDate == "" {
		if q.From == "" {
			fields = addField(fields, "from")
		}
		if q.To == "" {
			fields = addField(fields, "to")
		}
	}
	if q.From != "" && !validDay(q.From) {
		fields = addField(fields, "from")
	}
	if q.To != "" && !validDay(q.To) {
		fields = addField(fields, "to")
	}
	if q.From != "" && q.To != "" && q.From > q.To {
		fields = addField(fields, "from")
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	from, to := q.From, q.To
	if q.VisitDate != "" {
		from, to = q.VisitDate, q.VisitDate
	}
	if s.Charges == nil {
		return nil, errors.New("charge store is not configured")
	}
	rows, err := s.Charges.SearchCharges(ctx, port.ChargeQuery{
		PatientID: q.PatientID, RegisterID: q.RegisterID, FromDate: from, ToDate: to, Limit: chargeSearchLimit,
	})
	if err != nil {
		return nil, err
	}
	items := make([]ChargeDTO, 0, len(rows))
	for _, row := range rows {
		if !chargeScopeAllowed(ctx, s.Registers, user, row, domain.PermChargePay, domain.PermChargeRefund) {
			continue
		}
		items = append(items, toChargeDTO(row))
	}
	return &ChargeListDTO{Items: items}, nil
}

func chargeScopeAllowed(ctx context.Context, registers port.RegisterStore, user *domain.User, charge domain.Charge, permissions ...string) bool {
	if registers == nil {
		return false
	}
	register, err := registers.GetRegister(ctx, charge.RegisterID)
	if err != nil || register == nil {
		return false
	}
	return hasAnyPermScope(user, "outpatient", register.DepartmentID, permissions...)
}

func (s *Service) loadBillableOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" || s.Orders == nil || s.Registers == nil {
		if orderID == "" {
			return nil, notFound()
		}
		return nil, errors.New("order store is not configured")
	}
	order, err := s.Orders.GetOrder(ctx, orderID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if order.Status != domain.OrderStatusCommit {
		return nil, &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgChargeNotCommitted}
	}
	if _, err := s.requireOpenRegister(ctx, order.RegisterID); err != nil {
		return nil, err
	}
	if len(order.Items) == 0 {
		return nil, &Error{Status: 409, Code: CodeOrderEmptyItems, Message: msgOrderEmptyItems}
	}
	return order, nil
}

func (s *Service) loadCharge(ctx context.Context, chargeID string) (*domain.Charge, error) {
	chargeID = strings.TrimSpace(chargeID)
	if chargeID == "" {
		return nil, notFound()
	}
	if s.Charges == nil {
		return nil, errors.New("charge store is not configured")
	}
	cur, err := s.Charges.GetCharge(ctx, chargeID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	return cur, nil
}

func chargeFromOrder(order domain.Order, chargeID, squareNo, userID string, now time.Time) (domain.Charge, error) {
	items := make([]domain.ChargeItem, 0, len(order.Items))
	var total int64
	for _, oi := range order.Items {
		if oi.AmountMilli <= 0 || oi.PriceCents < 0 {
			return domain.Charge{}, invalidAmount()
		}
		fee, ok := lineFeeCents(oi.AmountMilli, oi.PriceCents)
		if !ok {
			return domain.Charge{}, invalidAmount()
		}
		itemID := ""
		if chargeID != "" {
			var err error
			itemID, err = newUUID()
			if err != nil {
				return domain.Charge{}, err
			}
		}
		if total > int64(^uint64(0)>>1)-fee {
			return domain.Charge{}, invalidAmount()
		}
		total += fee
		items = append(items, domain.ChargeItem{
			ChargeItemID: itemID, ChargeID: chargeID, OrderItemID: oi.OrderItemID,
			LineNo: oi.LineNo, ItemNo: oi.ItemNo, ItemName: oi.ItemName, ItemType: oi.ItemType,
			QtyMilli: oi.AmountMilli, UnitPriceCents: oi.PriceCents, TotalFeeCents: fee, CanBack: true,
		})
	}
	ch := domain.Charge{
		ChargeID: chargeID, SquareNo: squareNo, OrderID: order.OrderID, RegisterID: order.RegisterID,
		PatientID: order.PatientID, TotalFeeCents: total, SquareFeeCents: total, FavourableFeeCents: 0,
		Status: domain.ChargeStatusNormal, Items: items,
		CreatedAt: now, CreatedBy: userID, UpdatedAt: now, UpdatedBy: userID,
	}
	if err := assertChargeFormula(ch); err != nil {
		return domain.Charge{}, err
	}
	return ch, nil
}

type refundPlan struct {
	oldItems                []domain.ChargeItem
	newItems                []domain.ChargeItem
	payCents                int64
	newTotal, newFav, newSq int64
	full                    bool
}

func (s *Service) planRefund(ctx context.Context, cur domain.Charge, in RefundInput) (refundPlan, error) {
	var zero refundPlan
	payType := strings.TrimSpace(in.PayType)
	alloc := strings.TrimSpace(in.FavourableAlloc)
	if alloc == "" {
		alloc = domain.FavourableAllocProportional
	}
	var fields []string
	if !validPayType(payType) {
		fields = addField(fields, "payType")
	}
	if alloc != domain.FavourableAllocProportional {
		fields = addField(fields, "favourableAlloc")
	}
	if utf8.RuneCountInString(strings.TrimSpace(in.Reason)) > 200 {
		fields = addField(fields, "reason")
	}
	if len(in.Items) == 0 {
		fields = addField(fields, "items")
	}
	if len(fields) > 0 {
		return zero, validationFields(fields)
	}
	index := map[string]int{}
	for i, it := range cur.Items {
		index[it.ChargeItemID] = i
	}
	refundMilli := map[string]int64{}
	seen := map[string]bool{}
	for _, req := range in.Items {
		id := strings.TrimSpace(req.ChargeItemID)
		if id == "" || seen[id] {
			return zero, validationFields([]string{"chargeItemId"})
		}
		seen[id] = true
		if _, ok := index[id]; !ok {
			return zero, validationFields([]string{"chargeItemId"})
		}
		milli, ok := amountMilli(req.RefundQty)
		if !ok {
			return zero, invalidAmount()
		}
		refundMilli[id] = milli
	}
	var order *domain.Order
	if cur.OrderID != "" && s.Orders != nil {
		o, err := s.Orders.GetOrder(ctx, cur.OrderID)
		if err != nil && !errors.Is(err, port.ErrNotFound) {
			return zero, err
		}
		if err == nil {
			order = o
		}
	}
	for id, milli := range refundMilli {
		it := cur.Items[index[id]]
		if !it.CanBack {
			return zero, &Error{Status: 409, Code: CodeItemNotCanBack, Message: msgItemNotCanBack}
		}
		refundable := it.QtyMilli - it.RefundedQtyMilli
		if refundable < 0 {
			refundable = 0
		}
		if milli > refundable {
			return zero, &Error{Status: 409, Code: CodeRefundExceedsRefundable, Message: msgRefundExceedsRefundable}
		}
		if err := s.refundDispenseOK(ctx, it, order); err != nil {
			return zero, err
		}
	}
	oldItems := append([]domain.ChargeItem(nil), cur.Items...)
	newItems := make([]domain.ChargeItem, 0)
	var gref, grem int64
	full := true
	for i := range oldItems {
		it := &oldItems[i]
		milli := refundMilli[it.ChargeItemID]
		if milli == 0 {
			if it.QtyMilli-it.RefundedQtyMilli > 0 {
				full = false
				part, ok := portionCents(it.TotalFeeCents, it.QtyMilli-it.RefundedQtyMilli, it.QtyMilli)
				if !ok {
					return zero, invalidAmount()
				}
				grem += part
				newItems = append(newItems, remainderItem(*it, it.QtyMilli-it.RefundedQtyMilli, part))
			}
			continue
		}
		refCents, ok := portionCents(it.TotalFeeCents, milli, it.QtyMilli)
		if !ok {
			return zero, invalidAmount()
		}
		gref += refCents
		it.RefundedQtyMilli += milli
		remainQty := it.QtyMilli - it.RefundedQtyMilli
		if remainQty > 0 {
			full = false
			remCents := it.TotalFeeCents - refCents
			if it.RefundedQtyMilli != milli {
				// Earlier refunds already took a portion; remainder is what's left of the line total.
				remCents, ok = portionCents(it.TotalFeeCents, remainQty, it.QtyMilli)
				if !ok {
					return zero, invalidAmount()
				}
			}
			if remCents < 0 {
				return zero, invalidAmount()
			}
			grem += remCents
			newItems = append(newItems, remainderItem(*it, remainQty, remCents))
		}
	}
	if gref+grem != cur.TotalFeeCents && cur.FavourableFeeCents == 0 {
		// Lines omitted from a partial request still contribute their full total via the milli==0 branch.
		// A mismatch here means the portions did not conserve the stored gross.
		return zero, invalidAmount()
	}
	if cur.FavourableFeeCents > 0 && !full {
		return zero, &Error{Status: 409, Code: CodePartialRefundDiscountUnsupported, Message: msgPartialRefundDiscountUnsupported}
	}
	_, frem, payRef, rnew, ok := allocFavourable(alloc, cur.TotalFeeCents, cur.FavourableFeeCents, gref)
	if !ok {
		return zero, invalidAmount()
	}
	if cur.FavourableFeeCents == 0 && payRef+rnew != cur.SquareFeeCents {
		return zero, errors.New("refund fee invariant")
	}
	if full {
		newItems = nil
		if rnew != 0 {
			return zero, invalidAmount()
		}
	}
	return refundPlan{
		oldItems: oldItems, newItems: newItems, payCents: payRef,
		newTotal: grem, newFav: frem, newSq: rnew, full: full,
	}, nil
}

func (s *Service) materializeRefund(cur domain.Charge, plan refundPlan, userID string, now time.Time, payType string) (domain.Charge, *domain.Charge, domain.Payment, error) {
	var next *domain.Charge
	successor := ""
	if !plan.full {
		id, err := newUUID()
		if err != nil {
			return domain.Charge{}, nil, domain.Payment{}, err
		}
		no, err := newBusinessNo("S", now)
		if err != nil {
			return domain.Charge{}, nil, domain.Payment{}, err
		}
		items := make([]domain.ChargeItem, len(plan.newItems))
		copy(items, plan.newItems)
		for i := range items {
			itemID, err := newUUID()
			if err != nil {
				return domain.Charge{}, nil, domain.Payment{}, err
			}
			items[i].ChargeItemID = itemID
			items[i].ChargeID = id
			items[i].RefundedQtyMilli = 0
		}
		ch := domain.Charge{
			ChargeID: id, SquareNo: no, OrderID: cur.OrderID, RegisterID: cur.RegisterID, PatientID: cur.PatientID,
			TotalFeeCents: plan.newTotal, SquareFeeCents: plan.newSq, FavourableFeeCents: plan.newFav,
			Status: domain.ChargeStatusNormal, AbatedFromID: cur.ChargeID, PaidAt: now, Items: items,
			CreatedAt: now, CreatedBy: userID, UpdatedAt: now, UpdatedBy: userID,
		}
		if err := assertSquareHeader(ch); err != nil {
			return domain.Charge{}, nil, domain.Payment{}, err
		}
		next = &ch
		successor = id
	}
	old := cur
	old.Status = domain.ChargeStatusAbate
	old.SuccessorChargeID = successor
	old.Items = plan.oldItems
	old.UpdatedAt = now
	old.UpdatedBy = userID
	pid, err := newUUID()
	if err != nil {
		return domain.Charge{}, nil, domain.Payment{}, err
	}
	pno, err := newBusinessNo("P", now)
	if err != nil {
		return domain.Charge{}, nil, domain.Payment{}, err
	}
	pay := domain.Payment{
		PaymentID: pid, PaymentNo: pno, ChargeID: cur.ChargeID,
		Direction: domain.PaymentDirectionRefund, PayType: strings.TrimSpace(payType), AmountCents: plan.payCents,
		Status: domain.PaymentStatusSucceeded, PaidAt: now, OperatorID: userID,
		CreatedAt: now, CreatedBy: userID, UpdatedAt: now, UpdatedBy: userID,
	}
	return old, next, pay, nil
}

func remainderItem(it domain.ChargeItem, qtyMilli, totalCents int64) domain.ChargeItem {
	it.ChargeItemID = ""
	it.ChargeID = ""
	it.QtyMilli = qtyMilli
	it.TotalFeeCents = totalCents
	it.RefundedQtyMilli = 0
	return it
}

// refundDispenseOK enforces B-08 / 拍板#22h / #23f.
// Net dispensed > 0 on this charge item, or on the same order item (successor squares get new ids),
// is REFUND_REQUIRES_RETURN. A never-dispensed line stays refundable even when the order is PROC.
// With no dispense rows at all, Order status PROC still blocks DRUG lines (fixture path). 2026-10-12
func (s *Service) refundDispenseOK(ctx context.Context, it domain.ChargeItem, order *domain.Order) error {
	disp, ret, explicit, err := s.Charges.DispenseLedger(ctx, it.ChargeItemID)
	if err != nil {
		return err
	}
	if explicit {
		if disp > ret {
			return &Error{Status: 409, Code: CodeRefundRequiresReturn, Message: msgRefundRequiresReturn}
		}
		return nil
	}
	if s.Dispenses != nil && it.OrderItemID != "" {
		od, orr, ok, err := s.Dispenses.NetByOrderItem(ctx, it.OrderItemID)
		if err != nil {
			return err
		}
		if ok && od > orr {
			return &Error{Status: 409, Code: CodeRefundRequiresReturn, Message: msgRefundRequiresReturn}
		}
		if ok {
			return nil
		}
	}
	if s.Dispenses != nil && order != nil && order.OrderID != "" {
		has, err := s.Dispenses.OrderHasDispense(ctx, order.OrderID)
		if err != nil {
			return err
		}
		if has {
			return nil
		}
	}
	kind, err := s.lineKind(ctx, it, order)
	if err != nil {
		return err
	}
	if order != nil && order.Status == domain.OrderStatusProc && kind == domain.ItemTypeDrug {
		return &Error{Status: 409, Code: CodeRefundRequiresReturn, Message: msgRefundRequiresReturn}
	}
	return nil
}

// ensureStock is the charge-side check (拍板#22g). It reads inventory_item.qty_on_hand and does not
// decrement it. TREAT lines are free. Preview does not call this. 2026-10-12
func (s *Service) ensureStock(ctx context.Context, lines []stockLine) error {
	if s.Charges == nil {
		return errors.New("charge store is not configured")
	}
	need := map[string]int64{}
	for _, ln := range lines {
		if ln.kind != domain.ItemTypeDrug || ln.qtyMilli <= 0 {
			continue
		}
		need[ln.itemNo] += ln.qtyMilli
	}
	for itemNo, qty := range need {
		on, err := s.Charges.DrugOnHandMilli(ctx, itemNo)
		if err != nil {
			return err
		}
		if qty > on {
			return &Error{Status: 409, Code: CodeInsufficientStock, Message: msgInsufficientStock}
		}
	}
	return nil
}

func orderStock(order domain.Order) []stockLine {
	out := make([]stockLine, 0, len(order.Items))
	for _, it := range order.Items {
		out = append(out, stockLine{itemNo: it.ItemNo, kind: it.ItemType, qtyMilli: it.AmountMilli})
	}
	return out
}

func (s *Service) chargeStockLines(ctx context.Context, c domain.Charge) ([]stockLine, error) {
	// SQL charge_item does not retain the transient ItemType. The dedicated
	// registration fee has no prescription or drug stock to validate.
	if c.OrderID == "" && registerFeeCharge(c) {
		return nil, nil
	}
	var order *domain.Order
	if c.OrderID != "" && s.Orders != nil {
		o, err := s.Orders.GetOrder(ctx, c.OrderID)
		if err != nil && !errors.Is(err, port.ErrNotFound) {
			return nil, err
		}
		if err == nil {
			order = o
		}
	}
	out := make([]stockLine, 0, len(c.Items))
	for _, it := range c.Items {
		kind, err := s.lineKind(ctx, it, order)
		if err != nil {
			return nil, err
		}
		out = append(out, stockLine{itemNo: it.ItemNo, kind: kind, qtyMilli: it.QtyMilli})
	}
	return out, nil
}

func (s *Service) lineKind(ctx context.Context, it domain.ChargeItem, order *domain.Order) (string, error) {
	if it.ItemType == domain.ItemTypeDrug || it.ItemType == domain.ItemTypeTreat {
		return it.ItemType, nil
	}
	if order != nil {
		for _, oi := range order.Items {
			if oi.OrderItemID != "" && oi.OrderItemID == it.OrderItemID {
				return oi.ItemType, nil
			}
		}
	}
	if s.Orders != nil && it.ItemNo != "" {
		cat, err := s.Orders.CatalogByItemNo(ctx, it.ItemNo)
		if err == nil {
			return cat.ItemType, nil
		}
		if !errors.Is(err, port.ErrNotFound) {
			return "", err
		}
	}
	return domain.ItemTypeDrug, nil
}

func assertChargeFormula(c domain.Charge) error {
	var sum int64
	for _, it := range c.Items {
		if it.QtyMilli <= 0 || it.UnitPriceCents < 0 || it.TotalFeeCents < 0 {
			return invalidAmount()
		}
		fee, ok := lineFeeCents(it.QtyMilli, it.UnitPriceCents)
		if !ok || fee != it.TotalFeeCents {
			return invalidAmount()
		}
		sum += fee
	}
	if sum != c.TotalFeeCents {
		return invalidAmount()
	}
	return assertSquareHeader(c)
}

func assertSquareHeader(c domain.Charge) error {
	var sum int64
	for _, it := range c.Items {
		if it.QtyMilli <= 0 || it.TotalFeeCents < 0 {
			return invalidAmount()
		}
		sum += it.TotalFeeCents
	}
	if sum != c.TotalFeeCents || c.FavourableFeeCents < 0 || c.SquareFeeCents < 0 {
		return invalidAmount()
	}
	if c.SquareFeeCents != c.TotalFeeCents-c.FavourableFeeCents {
		return invalidAmount()
	}
	if c.FavourableFeeCents == 0 && c.TotalFeeCents != c.SquareFeeCents {
		return invalidAmount()
	}
	return nil
}

// allocFavourable splits a favourable fee. Round half away from zero (half-up for positive).
// F=0 ignores strategy and returns the gross split. PROPORTIONAL is the default draft. 2026-10-12
func allocFavourable(strategy string, g, f, gref int64) (fref, frem, payRef, rnew int64, ok bool) {
	if g < 0 || f < 0 || gref < 0 || gref > g || f > g {
		return 0, 0, 0, 0, false
	}
	grem := g - gref
	if f == 0 || g == 0 {
		return 0, 0, gref, grem, true
	}
	switch strategy {
	case domain.FavourableAllocDiscountOnRemain:
		if grem == 0 {
			fref = f
		} else if f <= grem {
			frem = f
		} else {
			return 0, 0, 0, 0, false
		}
	case domain.FavourableAllocDiscountFirst:
		if f <= gref {
			fref = f
		} else {
			fref = gref
		}
		frem = f - fref
		if frem > grem {
			return 0, 0, 0, 0, false
		}
	default:
		if g == 0 {
			fref = 0
		} else {
			fref = roundHalfUp(f*gref, g)
		}
		if fref > f {
			fref = f
		}
		if fref > gref {
			fref = gref
		}
		frem = f - fref
		if frem > grem {
			frem = grem
			fref = f - frem
		}
		if fref > gref || fref < 0 {
			return 0, 0, 0, 0, false
		}
	}
	payRef = gref - fref
	rnew = grem - frem
	if payRef < 0 || rnew < 0 || fref+frem != f || payRef+rnew != g-f {
		return 0, 0, 0, 0, false
	}
	return fref, frem, payRef, rnew, true
}

func roundHalfUp(num, den int64) int64 {
	if den <= 0 {
		return 0
	}
	if num < 0 {
		return -roundHalfUp(-num, den)
	}
	return (num + den/2) / den
}

func portionCents(total, part, whole int64) (int64, bool) {
	if total < 0 || part < 0 || whole <= 0 || part > whole {
		return 0, false
	}
	if part == 0 {
		return 0, true
	}
	if part == whole {
		return total, true
	}
	if total > 0 && part > int64(^uint64(0)>>1)/total {
		return 0, false
	}
	return roundHalfUp(total*part, whole), true
}

func validPayType(s string) bool {
	return s == domain.PayTypeCash
}

func validDay(s string) bool {
	_, err := time.ParseInLocation("2006-01-02", s, shanghai)
	return err == nil
}

func invalidAmount() *Error {
	return &Error{Status: 400, Code: CodeInvalidAmount, Message: msgInvalidAmount}
}

func moneyCents(v float64) int64 {
	c, ok := priceCents(v)
	if !ok {
		return -1
	}
	return c
}

func (s *Service) takeIdem(scope, key, hash string, dst any) (bool, error) {
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	hit, ok := s.idem[scope+"\x00"+key]
	if !ok {
		return false, nil
	}
	if hit.hash != hash {
		return false, &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
	}
	if err := json.Unmarshal(hit.body, dst); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) putIdem(scope, key, hash string, src any) error {
	body, err := json.Marshal(src)
	if err != nil {
		return err
	}
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	if s.idem == nil {
		s.idem = map[string]idemHit{}
	}
	k := scope + "\x00" + key
	if hit, ok := s.idem[k]; ok {
		if hit.hash != hash {
			return &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
		}
		return nil
	}
	s.idem[k] = idemHit{hash: hash, body: body}
	return nil
}

func mapChargeErr(err error) error {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return notFound()
	case errors.Is(err, port.ErrChargeExists):
		return &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgChargeExists}
	case errors.Is(err, port.ErrChargeAlreadyPaid):
		return &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgChargeAlreadyPaid}
	case errors.Is(err, port.ErrChargeAlreadyAbated):
		return &Error{Status: 409, Code: CodeAlreadyAbated, Message: msgAlreadyAbated}
	case errors.Is(err, port.ErrChargeNotPaid):
		return &Error{Status: 409, Code: CodeNotCharged, Message: msgNotCharged}
	case errors.Is(err, port.ErrIdempotencyConflict):
		return &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
	case errors.Is(err, port.ErrIdempotencyInProgress):
		return &Error{Status: 409, Code: CodeIdempotencyInProgress, Message: msgIdempotencyInProgress}
	default:
		return err
	}
}

func toChargeDTO(c domain.Charge) ChargeDTO {
	items := make([]ChargeItemDTO, len(c.Items))
	for i, it := range c.Items {
		items[i] = ChargeItemDTO{
			ChargeItemID: it.ChargeItemID,
			OrderItemID:  strPtrOrNil(it.OrderItemID),
			LineNo:       it.LineNo,
			ItemNo:       it.ItemNo,
			ItemName:     it.ItemName,
			Qty:          float64(it.QtyMilli) / 1000,
			UnitPrice:    float64(it.UnitPriceCents) / 100,
			TotalFee:     float64(it.TotalFeeCents) / 100,
			CanBack:      it.CanBack,
			RefundedQty:  float64(it.RefundedQtyMilli) / 1000,
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].LineNo == items[j].LineNo {
			return items[i].ChargeItemID < items[j].ChargeItemID
		}
		return items[i].LineNo < items[j].LineNo
	})
	return ChargeDTO{
		ChargeID:          c.ChargeID,
		SquareNo:          c.SquareNo,
		OrderID:           strPtrOrNil(c.OrderID),
		RegisterID:        c.RegisterID,
		EncounterID:       c.RegisterID,
		PatientID:         c.PatientID,
		TotalFee:          float64(c.TotalFeeCents) / 100,
		SquareFee:         float64(c.SquareFeeCents) / 100,
		FavourableFee:     float64(c.FavourableFeeCents) / 100,
		Status:            c.Status,
		SuccessorChargeID: strPtrOrNil(c.SuccessorChargeID),
		AbatedFromID:      strPtrOrNil(c.AbatedFromID),
		Items:             items,
		PaidAt:            timePtrOrNil(c.PaidAt),
	}
}

func toPaymentDTO(p domain.Payment) PaymentDTO {
	paid := timePtrOrNil(p.PaidAt)
	return PaymentDTO{
		PaymentID: p.PaymentID, PaymentNo: p.PaymentNo, ChargeID: p.ChargeID,
		Direction: p.Direction, PayType: p.PayType, Amount: float64(p.AmountCents) / 100,
		Status: p.Status, PaidAt: paid, SucceededAt: paid,
	}
}

func (s *Service) auditCharge(ctx context.Context, start time.Time, user *domain.User, action, resourceID, outcome, errorCode, requestID, ip string) {
	actor := ""
	roles := ""
	if user != nil {
		actor = user.ID
		roles = strings.Join(user.RoleCodes(), ",")
	}
	level := "info"
	if outcome != domain.OutcomeSuccess {
		level = "warn"
	}
	event := s.enrichBusinessAudit(ctx, domain.AuditEvent{
		ActorID: actor, ActorRoles: roles, Action: action, ResourceType: "Charge",
		ResourceID: resourceID, Outcome: outcome, ErrorCode: errorCode, RequestID: requestID, ClientIP: ip,
	})
	s.record(ctx, start, event, level)
}

func (s *Service) auditChargeErr(ctx context.Context, start time.Time, user *domain.User, action, resourceID, requestID, ip string, err error) {
	var ae *Error
	if errors.As(err, &ae) && (ae.Status == 400 || ae.Status == 403 || ae.Status == 409) {
		s.auditCharge(ctx, start, user, action, resourceID, domain.OutcomeFail, ae.Code, requestID, ip)
	}
}
