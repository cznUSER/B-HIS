package app

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// Dispense API (OpenAPI 0.3.9). Quantities stay in milli (decimal 18,3).
// Charge only checks stock. This package decrements on dispense and credits on return. 2026-10-12
const (
	CodeAlreadyDispensed         = "ALREADY_DISPENSED"
	CodeEmptyDispenseItems       = "EMPTY_DISPENSE_ITEMS"
	CodeDispenseExceedsAvailable = "DISPENSE_EXCEEDS_AVAILABLE"
	CodeReturnReasonRequired     = "RETURN_REASON_REQUIRED"
	CodeReturnExceedsDispense    = "RETURN_EXCEEDS_DISPENSE"
	CodeNotDispensed             = "NOT_DISPENSED"

	msgAlreadyDispensed         = "已全部发完，不能再发"
	msgEmptyDispenseItems       = "没有可发明细"
	msgDispenseExceedsAvailable = "发药数量超过可发数量"
	msgReturnReasonRequired     = "请填写退药原因"
	msgReturnExceedsDispense    = "退药数量超过可退数量"
	msgNotDispensed             = "尚未发药，不能退药"
	msgDispenseNotCharged       = "未收费，不能发药"

	idemScopeDispenseCreate = "dispense.create"
	idemScopeDispenseReturn = "dispense.return"
	dispenseSearchLimit     = 100
)

// DispenseItemInput is one POST /dispenses line. orderItemId and chargeItemId: at least one.
type DispenseItemInput struct {
	OrderItemID  string  `json:"orderItemId"`
	ChargeItemID string  `json:"chargeItemId"`
	DispenseQty  float64 `json:"dispenseQty"`
	Selected     *bool   `json:"selected"`
}

// CreateDispenseInput is DispenseRequest. PathOrderID is set by the deprecated alias.
type CreateDispenseInput struct {
	OrderID        string              `json:"orderId"`
	ChargeID       string              `json:"chargeId"`
	Items          []DispenseItemInput `json:"items"`
	IdempotencyKey string              `json:"-"`
	PathOrderID    string              `json:"-"`
}

// ReturnItemInput is one return line.
type ReturnItemInput struct {
	DispenseItemID string  `json:"dispenseItemId"`
	ReturnQty      float64 `json:"returnQty"`
}

// ReturnDispenseInput is ReturnDispenseRequest. reason is required.
type ReturnDispenseInput struct {
	Reason         string            `json:"reason"`
	Items          []ReturnItemInput `json:"items"`
	IdempotencyKey string            `json:"-"`
}

// DispenseItemDTO is OpenAPI DispenseItemDto.
type DispenseItemDTO struct {
	DispenseItemID     string  `json:"dispenseItemId"`
	OrderItemID        *string `json:"orderItemId"`
	ChargeItemID       *string `json:"chargeItemId"`
	ItemNo             string  `json:"itemNo"`
	ItemName           string  `json:"itemName"`
	Units              string  `json:"units,omitempty"`
	ChargedQty         float64 `json:"chargedQty"`
	DispensedQty       float64 `json:"dispensedQty"`
	ReturnedQty        float64 `json:"returnedQty"`
	AvailableQty       float64 `json:"availableQty"`
	DispenseQty        float64 `json:"dispenseQty"`
	StockHint          string  `json:"stockHint"`
	StockQty           float64 `json:"stockQty"`
	LineDispenseStatus string  `json:"lineDispenseStatus"`
}

// DispenseDTO is OpenAPI DispenseDto.
type DispenseDTO struct {
	DispenseID   string            `json:"dispenseId"`
	DispenseNo   string            `json:"dispenseNo"`
	OrderID      string            `json:"orderId"`
	ChargeID     *string           `json:"chargeId"`
	RegisterID   string            `json:"registerId"`
	PatientID    string            `json:"patientId"`
	Status       string            `json:"status"`
	OrderStatus  string            `json:"orderStatus,omitempty"`
	ReturnReason *string           `json:"returnReason"`
	OperatorID   *string           `json:"operatorId"`
	Items        []DispenseItemDTO `json:"items"`
	DispensedAt  string            `json:"dispensedAt"`
}

// DispenseListDTO is GET /dispenses.
type DispenseListDTO struct {
	Items []DispenseDTO `json:"items"`
}

// InventoryItemDTO is OpenAPI InventoryItemDto. skuCode ≡ itemNo.
type InventoryItemDTO struct {
	SkuID     string  `json:"skuId"`
	SkuCode   string  `json:"skuCode"`
	ItemNo    string  `json:"itemNo"`
	Name      string  `json:"name"`
	QtyOnHand float64 `json:"qtyOnHand"`
	Units     string  `json:"units"`
	Status    string  `json:"status"`
}

// InventoryListDTO is GET /inventory/items.
type InventoryListDTO struct {
	Items []InventoryItemDTO `json:"items"`
}

// DispenseSearch filters are AND-ed. An empty query returns no rows.
type DispenseSearch struct {
	PatientID  string
	RegisterID string
	ChargeID   string
	OrderID    string
	VisitDate  string
	From       string
	To         string
}

// InventorySearch accepts skuCode or itemNo. They must match when both are sent.
type InventorySearch struct {
	SkuCode string
	ItemNo  string
}

type dispenseItemCanon struct {
	OrderItemID  string `json:"orderItemId"`
	ChargeItemID string `json:"chargeItemId"`
	QtyMilli     int64  `json:"qtyMilli"`
}

type dispenseCanon struct {
	OrderID  string              `json:"orderId"`
	ChargeID string              `json:"chargeId"`
	Items    []dispenseItemCanon `json:"items"`
}

type returnCanon struct {
	Reason string            `json:"reason"`
	Items  []returnItemCanon `json:"items"`
}

type returnItemCanon struct {
	DispenseItemID string `json:"dispenseItemId"`
	QtyMilli       int64  `json:"qtyMilli"`
}

type plannedLine struct {
	item  domain.ChargeItem
	qty   int64
	units string
}

// CreateDispense is POST /dispenses (and the deprecated order alias).
// Only DRUG lines on a paid NORMAL charge. Partial qty is allowed.
// The store transaction checks stock again and does not mutate on INSUFFICIENT_STOCK.
// 毒麻本版不拦（拍板#23g）。2026-10-12
func (s *Service) CreateDispense(ctx context.Context, claims *auth.Claims, in CreateDispenseInput, requestID, ip string) (*DispenseDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermDispenseCreate) {
		e := forbidden()
		s.auditDispense(ctx, start, user, domain.ActionDispenseCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	orderID, chargeID, canon, fields, biz := normalizeDispense(in)
	if in.IdempotencyKey != "" && !validIdempotencyKey(in.IdempotencyKey) {
		fields = addField(fields, "idempotencyKey")
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if biz != nil {
		s.auditDispenseErr(ctx, start, user, domain.ActionDispenseCreate, orderID, requestID, ip, biz)
		return nil, biz
	}
	hash := hashJSON(canon)
	if in.IdempotencyKey != "" {
		var prev DispenseDTO
		found, err := s.takeIdem(idemScopeDispenseCreate, in.IdempotencyKey, hash, &prev)
		if err != nil || found {
			return &prev, err
		}
	}
	if s.Dispenses == nil || s.Orders == nil || s.Charges == nil {
		return nil, errors.New("dispense store is not configured")
	}
	order, err := s.Orders.GetOrder(ctx, orderID)
	if errors.Is(err, port.ErrNotFound) {
		e := notFound()
		s.auditDispenseErr(ctx, start, user, domain.ActionDispenseCreate, orderID, requestID, ip, e)
		return nil, e
	}
	if err != nil {
		return nil, err
	}
	if order.Status != domain.OrderStatusCommit && order.Status != domain.OrderStatusProc {
		e := &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgIllegalStateTransition}
		s.auditDispenseErr(ctx, start, user, domain.ActionDispenseCreate, orderID, requestID, ip, e)
		return nil, e
	}
	ch, err := s.resolvePaidCharge(ctx, orderID, chargeID)
	if err != nil {
		s.auditDispenseErr(ctx, start, user, domain.ActionDispenseCreate, orderID, requestID, ip, err)
		return nil, err
	}
	lines, err := s.planDispenseLines(ctx, *order, *ch, in.Items)
	if err != nil {
		s.auditDispenseErr(ctx, start, user, domain.ActionDispenseCreate, orderID, requestID, ip, err)
		return nil, err
	}
	now := s.now().UTC()
	var saved *domain.Dispense
	for attempt := 0; attempt < 5; attempt++ {
		id, err := newUUID()
		if err != nil {
			return nil, err
		}
		no, err := newBusinessNo("D", now)
		if err != nil {
			return nil, err
		}
		items := make([]domain.DispenseItem, len(lines))
		for i, ln := range lines {
			itemID, err := newUUID()
			if err != nil {
				return nil, err
			}
			items[i] = domain.DispenseItem{
				DispenseItemID: itemID, DispenseID: id, OrderItemID: ln.item.OrderItemID, ChargeItemID: ln.item.ChargeItemID,
				ItemNo: ln.item.ItemNo, ItemName: ln.item.ItemName, Units: ln.units,
				QtyChargedMilli: ln.item.QtyMilli, QtyDispensedMilli: ln.qty,
				LineStatus: domain.LineDispensePartial, CreatedAt: now, UpdatedAt: now,
			}
		}
		slip := domain.Dispense{
			DispenseID: id, DispenseNo: no, OrderID: order.OrderID, ChargeID: ch.ChargeID,
			RegisterID: ch.RegisterID, PatientID: ch.PatientID, Status: domain.DispenseStatusDone,
			DispensedAt: now, OperatorID: user.ID, DeptID: order.DepartmentID, Items: items,
			CreatedAt: now, CreatedBy: user.ID, UpdatedAt: now, UpdatedBy: user.ID,
		}
		saved, err = s.Dispenses.CommitDispense(ctx, slip)
		if errors.Is(err, port.ErrDuplicateBusinessNo) {
			saved = nil
			continue
		}
		if err != nil {
			mapped := mapDispenseErr(err)
			s.auditDispenseErr(ctx, start, user, domain.ActionDispenseCreate, orderID, requestID, ip, mapped)
			return nil, mapped
		}
		break
	}
	if saved == nil {
		return nil, errors.New("dispense number allocation failed")
	}
	dto, err := s.toDispenseDTO(ctx, *saved)
	if err != nil {
		return nil, err
	}
	if in.IdempotencyKey != "" {
		if err := s.putIdem(idemScopeDispenseCreate, in.IdempotencyKey, hash, dto); err != nil {
			return nil, err
		}
	}
	s.auditDispense(ctx, start, user, domain.ActionDispenseCreate, dto.DispenseID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

// ReturnDispense is POST /dispenses/{id}/return. reason is required.
// The store credits stock and writes a positive RETURN ledger in one transaction. 2026-10-12
func (s *Service) ReturnDispense(ctx context.Context, claims *auth.Claims, dispenseID string, in ReturnDispenseInput, requestID, ip string) (*DispenseDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermDispenseReturn) {
		e := forbidden()
		s.auditDispense(ctx, start, user, domain.ActionDispenseReturn, dispenseID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	dispenseID = strings.TrimSpace(dispenseID)
	reason := strings.TrimSpace(in.Reason)
	var fields []string
	if dispenseID == "" || len(dispenseID) > 64 {
		fields = addField(fields, "dispenseId")
	}
	if in.IdempotencyKey != "" && !validIdempotencyKey(in.IdempotencyKey) {
		fields = addField(fields, "idempotencyKey")
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	if reason == "" {
		e := &Error{Status: 400, Code: CodeReturnReasonRequired, Message: msgReturnReasonRequired}
		s.auditDispenseErr(ctx, start, user, domain.ActionDispenseReturn, dispenseID, requestID, ip, e)
		return nil, e
	}
	if utf8.RuneCountInString(reason) > 256 {
		return nil, validationFields([]string{"reason"})
	}
	if len(in.Items) == 0 {
		e := &Error{Status: 409, Code: CodeNotDispensed, Message: msgNotDispensed}
		s.auditDispenseErr(ctx, start, user, domain.ActionDispenseReturn, dispenseID, requestID, ip, e)
		return nil, e
	}
	if len(in.Items) > 200 {
		return nil, validationFields([]string{"items"})
	}
	canon := returnCanon{Reason: reason, Items: make([]returnItemCanon, 0, len(in.Items))}
	lines := make([]port.ReturnLine, 0, len(in.Items))
	seen := map[string]bool{}
	for _, it := range in.Items {
		id := strings.TrimSpace(it.DispenseItemID)
		if id == "" || seen[id] || len(id) > 64 {
			return nil, validationFields([]string{"dispenseItemId"})
		}
		seen[id] = true
		milli, ok := amountMilli(it.ReturnQty)
		if !ok {
			e := &Error{Status: 409, Code: CodeReturnExceedsDispense, Message: msgReturnExceedsDispense}
			s.auditDispenseErr(ctx, start, user, domain.ActionDispenseReturn, dispenseID, requestID, ip, e)
			return nil, e
		}
		canon.Items = append(canon.Items, returnItemCanon{DispenseItemID: id, QtyMilli: milli})
		lines = append(lines, port.ReturnLine{DispenseItemID: id, ReturnMilli: milli})
	}
	hash := hashJSON(canon)
	if in.IdempotencyKey != "" {
		var prev DispenseDTO
		found, err := s.takeIdem(idemScopeDispenseReturn, in.IdempotencyKey, hash, &prev)
		if err != nil || found {
			return &prev, err
		}
	}
	if s.Dispenses == nil {
		return nil, errors.New("dispense store is not configured")
	}
	saved, err := s.Dispenses.CommitReturn(ctx, dispenseID, reason, user.ID, s.now().UTC(), lines)
	if err != nil {
		mapped := mapDispenseErr(err)
		s.auditDispenseErr(ctx, start, user, domain.ActionDispenseReturn, dispenseID, requestID, ip, mapped)
		return nil, mapped
	}
	dto, err := s.toDispenseDTO(ctx, *saved)
	if err != nil {
		return nil, err
	}
	if in.IdempotencyKey != "" {
		if err := s.putIdem(idemScopeDispenseReturn, in.IdempotencyKey, hash, dto); err != nil {
			return nil, err
		}
	}
	s.auditDispense(ctx, start, user, domain.ActionDispenseReturn, dto.DispenseID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

// GetDispense is GET /dispenses/{id}.
func (s *Service) GetDispense(ctx context.Context, claims *auth.Claims, dispenseID string) (*DispenseDTO, error) {
	if _, err := s.requireDispenseRead(ctx, claims); err != nil {
		return nil, err
	}
	dispenseID = strings.TrimSpace(dispenseID)
	if dispenseID == "" || s.Dispenses == nil {
		return nil, notFound()
	}
	d, err := s.Dispenses.GetDispense(ctx, dispenseID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	dto, err := s.toDispenseDTO(ctx, *d)
	if err != nil {
		return nil, err
	}
	return &dto, nil
}

// SearchDispenses is GET /dispenses. No filter returns an empty list.
func (s *Service) SearchDispenses(ctx context.Context, claims *auth.Claims, q DispenseSearch) (*DispenseListDTO, error) {
	if _, err := s.requireDispenseRead(ctx, claims); err != nil {
		return nil, err
	}
	q.PatientID = strings.TrimSpace(q.PatientID)
	q.RegisterID = strings.TrimSpace(q.RegisterID)
	q.ChargeID = strings.TrimSpace(q.ChargeID)
	q.OrderID = strings.TrimSpace(q.OrderID)
	q.VisitDate = strings.TrimSpace(q.VisitDate)
	q.From = strings.TrimSpace(q.From)
	q.To = strings.TrimSpace(q.To)
	empty := &DispenseListDTO{Items: []DispenseDTO{}}
	if q.PatientID == "" && q.RegisterID == "" && q.ChargeID == "" && q.OrderID == "" && q.VisitDate == "" && q.From == "" && q.To == "" {
		return empty, nil
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
	if s.Dispenses == nil {
		return nil, errors.New("dispense store is not configured")
	}
	rows, err := s.Dispenses.SearchDispenses(ctx, port.DispenseQuery{
		PatientID: q.PatientID, RegisterID: q.RegisterID, ChargeID: q.ChargeID, OrderID: q.OrderID,
		FromDate: from, ToDate: to, Limit: dispenseSearchLimit,
	})
	if err != nil {
		return nil, err
	}
	items := make([]DispenseDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := s.toDispenseDTO(ctx, row)
		if err != nil {
			return nil, err
		}
		items = append(items, dto)
	}
	return &DispenseListDTO{Items: items}, nil
}

// ListInventory is GET /inventory/items. TREAT is absent. skuCode ≡ itemNo. 2026-10-12
func (s *Service) ListInventory(ctx context.Context, claims *auth.Claims, q InventorySearch) (*InventoryListDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermInventoryRead) {
		return nil, forbidden()
	}
	sku := strings.TrimSpace(q.SkuCode)
	itemNo := strings.TrimSpace(q.ItemNo)
	if sku != "" && itemNo != "" && sku != itemNo {
		return nil, validationFields([]string{"itemNo"})
	}
	if sku == "" {
		sku = itemNo
	}
	if s.Dispenses == nil {
		return nil, errors.New("dispense store is not configured")
	}
	rows, err := s.Dispenses.ListInventory(ctx, sku)
	if err != nil {
		return nil, err
	}
	items := make([]InventoryItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, InventoryItemDTO{
			SkuID: row.SkuID, SkuCode: row.ItemNo, ItemNo: row.ItemNo, Name: row.ItemName,
			QtyOnHand: float64(row.QtyOnHandMilli) / 1000, Units: row.Units, Status: row.Status,
		})
	}
	return &InventoryListDTO{Items: items}, nil
}

func normalizeDispense(in CreateDispenseInput) (orderID, chargeID string, canon dispenseCanon, fields []string, biz *Error) {
	orderID = strings.TrimSpace(in.OrderID)
	pathID := strings.TrimSpace(in.PathOrderID)
	chargeID = strings.TrimSpace(in.ChargeID)
	if pathID != "" && orderID != "" && pathID != orderID {
		fields = addField(fields, "orderId")
	}
	if orderID == "" {
		orderID = pathID
	}
	if len(in.Items) == 0 {
		return orderID, chargeID, canon, nil, &Error{Status: 409, Code: CodeEmptyDispenseItems, Message: msgEmptyDispenseItems}
	}
	if len(in.Items) > 200 {
		fields = addField(fields, "items")
	}
	if orderID == "" || len(orderID) > 64 {
		fields = addField(fields, "orderId")
	}
	if len(chargeID) > 64 {
		fields = addField(fields, "chargeId")
	}
	canon.OrderID = orderID
	canon.ChargeID = chargeID
	seenC := map[string]bool{}
	seenO := map[string]bool{}
	for _, it := range in.Items {
		oid := strings.TrimSpace(it.OrderItemID)
		cid := strings.TrimSpace(it.ChargeItemID)
		if oid == "" && cid == "" {
			fields = addField(fields, "chargeItemId")
			continue
		}
		if (oid != "" && (len(oid) > 64 || seenO[oid])) || (cid != "" && (len(cid) > 64 || seenC[cid])) {
			fields = addField(fields, "chargeItemId")
		}
		if oid != "" {
			seenO[oid] = true
		}
		if cid != "" {
			seenC[cid] = true
		}
		milli, ok := amountMilli(it.DispenseQty)
		if !ok {
			return orderID, chargeID, canon, nil, &Error{Status: 409, Code: CodeDispenseExceedsAvailable, Message: msgDispenseExceedsAvailable}
		}
		canon.Items = append(canon.Items, dispenseItemCanon{OrderItemID: oid, ChargeItemID: cid, QtyMilli: milli})
	}
	return orderID, chargeID, canon, fields, nil
}

func (s *Service) resolvePaidCharge(ctx context.Context, orderID, chargeID string) (*domain.Charge, error) {
	notCharged := &Error{Status: 409, Code: CodeNotCharged, Message: msgDispenseNotCharged}
	if chargeID != "" {
		ch, err := s.Charges.GetCharge(ctx, chargeID)
		if errors.Is(err, port.ErrNotFound) {
			return nil, notFound()
		}
		if err != nil {
			return nil, err
		}
		if ch.OrderID != orderID || ch.Status != domain.ChargeStatusNormal || ch.PaidAt.IsZero() {
			return nil, notCharged
		}
		return ch, nil
	}
	rows, err := s.Charges.SearchCharges(ctx, port.ChargeQuery{OrderID: orderID, Limit: 50})
	if err != nil {
		return nil, err
	}
	var best domain.Charge
	found := false
	for _, ch := range rows {
		if ch.Status != domain.ChargeStatusNormal || ch.PaidAt.IsZero() {
			continue
		}
		if !found || ch.CreatedAt.After(best.CreatedAt) {
			best = ch
			found = true
		}
	}
	if !found {
		return nil, notCharged
	}
	return &best, nil
}

func (s *Service) planDispenseLines(ctx context.Context, order domain.Order, ch domain.Charge, req []DispenseItemInput) ([]plannedLine, error) {
	exceeds := &Error{Status: 409, Code: CodeDispenseExceedsAvailable, Message: msgDispenseExceedsAvailable}
	already := &Error{Status: 409, Code: CodeAlreadyDispensed, Message: msgAlreadyDispensed}
	out := make([]plannedLine, 0, len(req))
	need := map[string]int64{}
	for _, raw := range req {
		line, err := matchChargeLine(ch, strings.TrimSpace(raw.ChargeItemID), strings.TrimSpace(raw.OrderItemID))
		if err != nil {
			return nil, err
		}
		kind, err := s.lineKind(ctx, line, &order)
		if err != nil {
			return nil, err
		}
		if kind != domain.ItemTypeDrug {
			return nil, validationFields([]string{"items"})
		}
		milli, ok := amountMilli(raw.DispenseQty)
		if !ok {
			return nil, exceeds
		}
		disp, ret, _, err := s.Dispenses.NetByChargeItem(ctx, line.ChargeItemID)
		if err != nil {
			return nil, err
		}
		net := disp - ret
		avail := line.QtyMilli - net
		if avail < 0 {
			avail = 0
		}
		if avail <= 0 {
			return nil, already
		}
		if milli > avail {
			return nil, exceeds
		}
		units := ""
		for _, oi := range order.Items {
			if oi.OrderItemID == line.OrderItemID {
				units = oi.Units
				break
			}
		}
		need[line.ItemNo] += milli
		out = append(out, plannedLine{item: line, qty: milli, units: units})
	}
	for itemNo, qty := range need {
		on, err := s.Dispenses.QtyOnHandMilli(ctx, itemNo)
		if err != nil {
			return nil, err
		}
		if qty > on {
			return nil, &Error{Status: 409, Code: CodeInsufficientStock, Message: msgInsufficientStock}
		}
	}
	return out, nil
}

func matchChargeLine(ch domain.Charge, chargeItemID, orderItemID string) (domain.ChargeItem, error) {
	for _, it := range ch.Items {
		byCharge := chargeItemID != "" && it.ChargeItemID == chargeItemID
		byOrder := orderItemID != "" && it.OrderItemID == orderItemID
		if chargeItemID != "" && orderItemID != "" {
			if byCharge && it.OrderItemID != orderItemID {
				return domain.ChargeItem{}, validationFields([]string{"orderItemId"})
			}
			if byCharge {
				return it, nil
			}
			continue
		}
		if byCharge || byOrder {
			return it, nil
		}
	}
	return domain.ChargeItem{}, validationFields([]string{"chargeItemId"})
}

func (s *Service) toDispenseDTO(ctx context.Context, d domain.Dispense) (DispenseDTO, error) {
	orderStatus := ""
	if s.Orders != nil && d.OrderID != "" {
		o, err := s.Orders.GetOrder(ctx, d.OrderID)
		if err != nil && !errors.Is(err, port.ErrNotFound) {
			return DispenseDTO{}, err
		}
		if err == nil {
			orderStatus = o.Status
		}
	}
	items := make([]DispenseItemDTO, 0, len(d.Items))
	stock := map[string]int64{}
	for _, it := range d.Items {
		on, ok := stock[it.ItemNo]
		if !ok && s.Dispenses != nil {
			q, err := s.Dispenses.QtyOnHandMilli(ctx, it.ItemNo)
			if err != nil {
				return DispenseDTO{}, err
			}
			on = q
			stock[it.ItemNo] = on
		}
		net := int64(0)
		if s.Dispenses != nil && it.ChargeItemID != "" {
			disp, ret, _, err := s.Dispenses.NetByChargeItem(ctx, it.ChargeItemID)
			if err != nil {
				return DispenseDTO{}, err
			}
			net = disp - ret
		}
		avail := it.QtyChargedMilli - net
		if avail < 0 {
			avail = 0
		}
		hint := "库存充足"
		if on <= 0 {
			hint = "库存不足"
		}
		items = append(items, DispenseItemDTO{
			DispenseItemID: it.DispenseItemID, OrderItemID: strPtrOrNil(it.OrderItemID), ChargeItemID: strPtrOrNil(it.ChargeItemID),
			ItemNo: it.ItemNo, ItemName: it.ItemName, Units: it.Units,
			ChargedQty: float64(it.QtyChargedMilli) / 1000, DispensedQty: float64(it.QtyDispensedMilli) / 1000,
			ReturnedQty: float64(it.QtyReturnedMilli) / 1000, AvailableQty: float64(avail) / 1000,
			DispenseQty: float64(it.QtyDispensedMilli) / 1000, StockHint: hint, StockQty: float64(on) / 1000,
			LineDispenseStatus: domain.LineStatus(it.QtyChargedMilli, net),
		})
	}
	at := d.DispensedAt
	if at.IsZero() {
		at = d.CreatedAt
	}
	return DispenseDTO{
		DispenseID: d.DispenseID, DispenseNo: d.DispenseNo, OrderID: d.OrderID, ChargeID: strPtrOrNil(d.ChargeID),
		RegisterID: d.RegisterID, PatientID: d.PatientID, Status: d.Status, OrderStatus: orderStatus,
		ReturnReason: strPtrOrNil(d.ReturnReason), OperatorID: strPtrOrNil(d.OperatorID), Items: items,
		DispensedAt: at.UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) requireDispenseRead(ctx context.Context, claims *auth.Claims) (*domain.User, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermDispenseCreate) && !hasPerm(user, domain.PermDispenseReturn) {
		return nil, forbidden()
	}
	return user, nil
}

func mapDispenseErr(err error) error {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return notFound()
	case errors.Is(err, port.ErrNotCharged), errors.Is(err, port.ErrChargeNotPaid):
		return &Error{Status: 409, Code: CodeNotCharged, Message: msgDispenseNotCharged}
	case errors.Is(err, port.ErrInsufficientStock):
		return &Error{Status: 409, Code: CodeInsufficientStock, Message: msgInsufficientStock}
	case errors.Is(err, port.ErrAlreadyDispensed):
		return &Error{Status: 409, Code: CodeAlreadyDispensed, Message: msgAlreadyDispensed}
	case errors.Is(err, port.ErrDispenseExceeds):
		return &Error{Status: 409, Code: CodeDispenseExceedsAvailable, Message: msgDispenseExceedsAvailable}
	case errors.Is(err, port.ErrReturnExceeds):
		return &Error{Status: 409, Code: CodeReturnExceedsDispense, Message: msgReturnExceedsDispense}
	case errors.Is(err, port.ErrNotDispensed):
		return &Error{Status: 409, Code: CodeNotDispensed, Message: msgNotDispensed}
	case errors.Is(err, port.ErrIllegalStateTransition):
		return &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgIllegalStateTransition}
	default:
		return err
	}
}

func (s *Service) auditDispense(ctx context.Context, start time.Time, user *domain.User, action, resourceID, outcome, errorCode, requestID, ip string) {
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
	s.record(ctx, start, domain.AuditEvent{
		ActorID: actor, ActorRoles: roles, Action: action, ResourceType: "Dispense",
		ResourceID: resourceID, Outcome: outcome, ErrorCode: errorCode, RequestID: requestID, ClientIP: ip,
	}, level)
}

func (s *Service) auditDispenseErr(ctx context.Context, start time.Time, user *domain.User, action, resourceID, requestID, ip string, err error) {
	var ae *Error
	if errors.As(err, &ae) && (ae.Status == 400 || ae.Status == 403 || ae.Status == 409) {
		s.auditDispense(ctx, start, user, action, resourceID, domain.OutcomeFail, ae.Code, requestID, ip)
	}
}
