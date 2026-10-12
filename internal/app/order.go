package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

const (
	CodeEncounterRequired       = "ENCOUNTER_REQUIRED"
	CodeOrderNotEditable        = "ORDER_NOT_EDITABLE"
	CodeOrderNotCommittable     = "ORDER_NOT_COMMITTABLE"
	CodeOrderEmptyItems         = "ORDER_EMPTY_ITEMS"
	CodeOrderInvalidAmount      = "ORDER_INVALID_AMOUNT"
	CodeOrderChargedNotAbatable = "ORDER_CHARGED_NOT_ABATABLE"
	CodeCatalogItemNotFound     = "CATALOG_ITEM_NOT_FOUND"
	CodeNotImplemented          = "NOT_IMPLEMENTED"

	msgEncounterRequired       = "需要有效挂号"
	msgOrderNotEditable        = "仅暂存处方可修改"
	msgOrderNotCommittable     = "当前状态不可提交"
	msgOrderEmptyItems         = "提交时至少需要一行明细"
	msgOrderInvalidAmount      = "数量或金额非法"
	msgOrderChargedNotAbatable = "已收费处方不可作废"
	msgCatalogItemNotFound     = "目录项不存在"
	msgNotImplemented          = "尚未实现"

	idemScopeOrderCreate = "order.create"
	orderSearchLimit     = 100
	maxOrderItems        = 200
)

// OrderItemInput is OpenAPI OrderItemInput. 2026-10-12
type OrderItemInput struct {
	LineNo           int      `json:"lineNo"`
	ItemNo           string   `json:"itemNo"`
	ItemType         string   `json:"itemType"`
	ItemName         string   `json:"itemName"`
	Spec             *string  `json:"spec"`
	Units            string   `json:"units"`
	Amount           float64  `json:"amount"`
	Price            float64  `json:"price"`
	TotalFee         *float64 `json:"totalFee"`
	UsageText        *string  `json:"usageText"`
	FrequencyText    *string  `json:"frequencyText"`
	Dosage           *string  `json:"dosage"`
	DosageUnit       *string  `json:"dosageUnit"`
	ProcDepartmentID *string  `json:"procDepartmentId"`
}

// CreateOrderInput is OpenAPI CreateOrderRequest. registerId is required.
// encounterId is an alias and must match registerId when both are sent.
type CreateOrderInput struct {
	RegisterID     string           `json:"registerId"`
	EncounterID    *string          `json:"encounterId"`
	PatientID      *string          `json:"patientId"`
	DepartmentID   *string          `json:"departmentId"`
	DoctorID       *string          `json:"doctorId"`
	DiagnoseName   *string          `json:"diagnoseName"`
	RecipeSort     *string          `json:"recipeSort"`
	Items          []OrderItemInput `json:"items"`
	IdempotencyKey string           `json:"-"`
}

// PatchOrderInput updates a SAVE sheet. A present items array replaces every line.
type PatchOrderInput struct {
	PatientID    *string           `json:"patientId"`
	DepartmentID *string           `json:"departmentId"`
	DoctorID     *string           `json:"doctorId"`
	DiagnoseName *string           `json:"diagnoseName"`
	RecipeSort   *string           `json:"recipeSort"`
	Items        *[]OrderItemInput `json:"items"`
}

// OrderItemDTO is OpenAPI OrderItemDto.
type OrderItemDTO struct {
	OrderItemID      string  `json:"orderItemId"`
	LineNo           int     `json:"lineNo"`
	ItemNo           string  `json:"itemNo"`
	ItemType         string  `json:"itemType"`
	ItemName         string  `json:"itemName"`
	Spec             *string `json:"spec"`
	Units            string  `json:"units"`
	Amount           float64 `json:"amount"`
	Price            float64 `json:"price"`
	TotalFee         float64 `json:"totalFee"`
	UsageText        *string `json:"usageText"`
	FrequencyText    *string `json:"frequencyText"`
	Dosage           *string `json:"dosage"`
	DosageUnit       *string `json:"dosageUnit"`
	ProcDepartmentID *string `json:"procDepartmentId"`
}

// OrderDTO is OpenAPI OrderDto. encounterId is always the same value as registerId.
type OrderDTO struct {
	OrderID      string         `json:"orderId"`
	OrderNo      string         `json:"orderNo"`
	RegisterID   string         `json:"registerId"`
	EncounterID  string         `json:"encounterId"`
	PatientID    string         `json:"patientId"`
	DepartmentID string         `json:"departmentId"`
	DoctorID     string         `json:"doctorId"`
	DiagnoseName *string        `json:"diagnoseName"`
	RecipeSort   string         `json:"recipeSort"`
	Status       string         `json:"status"`
	TotalFee     float64        `json:"totalFee"`
	Charged      bool           `json:"charged"`
	Items        []OrderItemDTO `json:"items"`
	SubmittedAt  *string        `json:"submittedAt"`
	AbatedAt     *string        `json:"abatedAt"`
	AbatedBy     *string        `json:"abatedBy"`
}

// OrderListDTO is the GET /orders body.
type OrderListDTO struct {
	Items []OrderDTO `json:"items"`
}

// OrderSearch filters are AND-ed and exact. An empty query returns no rows,
// same as patient and register search. encounterId is an alias of registerId.
type OrderSearch struct {
	RegisterID  string
	EncounterID string
	PatientID   string
	OrderNo     string
	Status      string
}

type orderCanon struct {
	RegisterID   string           `json:"registerId"`
	EncounterID  string           `json:"encounterId"`
	PatientID    string           `json:"patientId"`
	DepartmentID string           `json:"departmentId"`
	DoctorID     string           `json:"doctorId"`
	DiagnoseName string           `json:"diagnoseName"`
	RecipeSort   string           `json:"recipeSort"`
	Items        []orderItemCanon `json:"items"`
}

type orderItemCanon struct {
	LineNo           int    `json:"lineNo"`
	ItemNo           string `json:"itemNo"`
	ItemType         string `json:"itemType"`
	ItemName         string `json:"itemName"`
	Spec             string `json:"spec"`
	Units            string `json:"units"`
	AmountMilli      int64  `json:"amountMilli"`
	PriceCents       int64  `json:"priceCents"`
	UsageText        string `json:"usageText"`
	FrequencyText    string `json:"frequencyText"`
	Dosage           string `json:"dosage"`
	DosageUnit       string `json:"dosageUnit"`
	ProcDepartmentID string `json:"procDepartmentId"`
}

// NotImplemented is the Phase1 stub for recipe-check. charge-preview and dispense are implemented.
func NotImplemented() *Error {
	return &Error{Status: 501, Code: CodeNotImplemented, Message: msgNotImplemented}
}

// CreateOrder stores a SAVE sheet. registerId must name a register whose status is not BACK.
// Items may be empty. A non-empty line must reference an active catalog item_no and amount>0.
// patientId, when sent, must match the register. departmentId and doctorId default to the register.
// POST requires order:submit or admin:* (DOCTOR, SYSTEM). Idempotency-Key is process-local. 2026-10-12
func (s *Service) CreateOrder(ctx context.Context, claims *auth.Claims, in CreateOrderInput, requestID, ip string) (*OrderDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermOrderSubmit) {
		e := forbidden()
		s.auditOrder(ctx, start, user, domain.ActionOrderCreate, "", domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	canon, fields := normalizeOrderCreate(in)
	if in.IdempotencyKey != "" && !validIdempotencyKey(in.IdempotencyKey) {
		fields = append(fields, "idempotencyKey")
	}
	if len(fields) > 0 {
		return nil, validationFields(fields)
	}
	hash := hashJSON(canon)
	if in.IdempotencyKey != "" {
		if dto, err := s.lookupOrderIdem(in.IdempotencyKey, hash); err != nil || dto != nil {
			return dto, err
		}
	}
	if s.Orders == nil || s.Registers == nil {
		return nil, errors.New("order store is not configured")
	}
	reg, err := s.requireOpenRegister(ctx, canon.RegisterID)
	if err != nil {
		s.auditOrderErr(ctx, start, user, domain.ActionOrderCreate, "", requestID, ip, err)
		return nil, err
	}
	if canon.PatientID != "" && canon.PatientID != reg.PatientID {
		return nil, validationFields([]string{"patientId"})
	}
	dept := reg.DepartmentID
	if canon.DepartmentID != "" {
		dept = canon.DepartmentID
	}
	doctor := reg.DoctorID
	if canon.DoctorID != "" {
		doctor = canon.DoctorID
	}
	if err := s.requireDeptDoctor(ctx, dept, doctor); err != nil {
		return nil, err
	}
	items, total, err := s.prepareItems(ctx, in.Items)
	if err != nil {
		return nil, err
	}
	recipe := canon.RecipeSort
	if recipe == "" {
		recipe = "NORMAL"
	}
	now := s.now().UTC()
	var created domain.Order
	var createErr error
	for attempt := 0; attempt < 5; attempt++ {
		id, err := newUUID()
		if err != nil {
			return nil, err
		}
		no, err := newBusinessNo("O", now)
		if err != nil {
			return nil, err
		}
		lines := append([]domain.OrderItem(nil), items...)
		for i := range lines {
			itemID, err := newUUID()
			if err != nil {
				return nil, err
			}
			lines[i].OrderItemID = itemID
			lines[i].OrderID = id
		}
		order := domain.Order{
			OrderID:       id,
			OrderNo:       no,
			RegisterID:    reg.RegisterID,
			PatientID:     reg.PatientID,
			DepartmentID:  dept,
			DoctorID:      doctor,
			DiagnoseName:  canon.DiagnoseName,
			RecipeSort:    recipe,
			Status:        domain.OrderStatusSave,
			TotalFeeCents: total,
			Charged:       false,
			Items:         lines,
			CreatedAt:     now,
			CreatedBy:     user.ID,
			UpdatedAt:     now,
			UpdatedBy:     user.ID,
		}
		createErr = s.Orders.CreateOrder(ctx, order)
		if createErr == nil {
			created = order
			break
		}
		if errors.Is(createErr, port.ErrDuplicateBusinessNo) {
			continue
		}
		break
	}
	if createErr != nil {
		return nil, createErr
	}
	dto := toOrderDTO(created)
	if in.IdempotencyKey != "" {
		stored, err := s.saveOrderIdem(in.IdempotencyKey, hash, dto)
		if err != nil {
			return nil, err
		}
		if stored != nil {
			dto = *stored
		}
	}
	s.auditOrder(ctx, start, user, domain.ActionOrderCreate, dto.OrderID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

// SearchOrders returns an AND of the given filters. With no filters the list is empty.
func (s *Service) SearchOrders(ctx context.Context, claims *auth.Claims, q OrderSearch) (*OrderListDTO, error) {
	if _, err := s.actor(ctx, claims); err != nil {
		return nil, err
	}
	q.RegisterID = strings.TrimSpace(q.RegisterID)
	q.EncounterID = strings.TrimSpace(q.EncounterID)
	q.PatientID = strings.TrimSpace(q.PatientID)
	q.OrderNo = strings.TrimSpace(q.OrderNo)
	q.Status = strings.TrimSpace(q.Status)
	empty := &OrderListDTO{Items: []OrderDTO{}}
	if q.RegisterID == "" && q.EncounterID == "" && q.PatientID == "" && q.OrderNo == "" && q.Status == "" {
		return empty, nil
	}
	if q.RegisterID != "" && q.EncounterID != "" && q.RegisterID != q.EncounterID {
		return nil, validationFields([]string{"encounterId"})
	}
	if q.RegisterID == "" {
		q.RegisterID = q.EncounterID
	}
	if q.Status != "" && !validOrderStatus(q.Status) {
		return nil, validationFields([]string{"status"})
	}
	if s.Orders == nil {
		return nil, errors.New("order store is not configured")
	}
	rows, err := s.Orders.SearchOrders(ctx, port.OrderQuery{
		RegisterID: q.RegisterID, PatientID: q.PatientID, OrderNo: q.OrderNo, Status: q.Status, Limit: orderSearchLimit,
	})
	if err != nil {
		return nil, err
	}
	items := make([]OrderDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toOrderDTO(row))
	}
	return &OrderListDTO{Items: items}, nil
}

// GetOrder returns one sheet or 404 NOT_FOUND.
func (s *Service) GetOrder(ctx context.Context, claims *auth.Claims, orderID string) (*OrderDTO, error) {
	if _, err := s.actor(ctx, claims); err != nil {
		return nil, err
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, notFound()
	}
	if s.Orders == nil {
		return nil, errors.New("order store is not configured")
	}
	o, err := s.Orders.GetOrder(ctx, orderID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	dto := toOrderDTO(*o)
	return &dto, nil
}

// PatchOrder changes a SAVE sheet. Items, when present, replace every line.
func (s *Service) PatchOrder(ctx context.Context, claims *auth.Claims, orderID string, in PatchOrderInput, requestID, ip string) (*OrderDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(user, domain.PermOrderSubmit) {
		return nil, forbidden()
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, notFound()
	}
	if s.Orders == nil || s.Registers == nil {
		return nil, errors.New("order store is not configured")
	}
	cur, err := s.Orders.GetOrder(ctx, orderID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if cur.Status != domain.OrderStatusSave {
		return nil, &Error{Status: 409, Code: CodeOrderNotEditable, Message: msgOrderNotEditable}
	}
	reg, err := s.requireOpenRegister(ctx, cur.RegisterID)
	if err != nil {
		return nil, err
	}
	next := *cur
	if in.PatientID != nil {
		p := strings.TrimSpace(*in.PatientID)
		if p == "" || p != reg.PatientID || len(p) > 64 {
			return nil, validationFields([]string{"patientId"})
		}
		next.PatientID = p
	}
	if in.DepartmentID != nil {
		d := strings.TrimSpace(*in.DepartmentID)
		if d == "" || len(d) > 64 {
			return nil, validationFields([]string{"departmentId"})
		}
		next.DepartmentID = d
	}
	if in.DoctorID != nil {
		d := strings.TrimSpace(*in.DoctorID)
		if d == "" || len(d) > 64 {
			return nil, validationFields([]string{"doctorId"})
		}
		next.DoctorID = d
	}
	if in.DiagnoseName != nil {
		d := strings.TrimSpace(*in.DiagnoseName)
		if utf8.RuneCountInString(d) > 128 {
			return nil, validationFields([]string{"diagnoseName"})
		}
		next.DiagnoseName = d
	}
	if in.RecipeSort != nil {
		r := strings.TrimSpace(*in.RecipeSort)
		if len(r) > 32 {
			return nil, validationFields([]string{"recipeSort"})
		}
		if r == "" {
			r = "NORMAL"
		}
		next.RecipeSort = r
	}
	replace := false
	if in.Items != nil {
		lines, total, err := s.prepareItems(ctx, *in.Items)
		if err != nil {
			return nil, err
		}
		for i := range lines {
			itemID, err := newUUID()
			if err != nil {
				return nil, err
			}
			lines[i].OrderItemID = itemID
			lines[i].OrderID = next.OrderID
		}
		next.Items = lines
		next.TotalFeeCents = total
		replace = true
	}
	if err := s.requireDeptDoctor(ctx, next.DepartmentID, next.DoctorID); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	next.UpdatedAt = now
	next.UpdatedBy = user.ID
	updated, err := s.Orders.UpdateOrderDraft(ctx, next, replace)
	if err != nil {
		return nil, mapOrderErr(err)
	}
	dto := toOrderDTO(*updated)
	return &dto, nil
}

// CommitOrder moves SAVE to COMMIT, recomputes totalFee, and writes submittedAt.
// SubmitOrder is the deprecated alias and uses the same path.
func (s *Service) CommitOrder(ctx context.Context, claims *auth.Claims, orderID, requestID, ip string) (*OrderDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	orderID = strings.TrimSpace(orderID)
	if !hasPerm(user, domain.PermOrderSubmit) {
		e := forbidden()
		s.auditOrder(ctx, start, user, domain.ActionOrderCommit, orderID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if orderID == "" {
		return nil, notFound()
	}
	if s.Orders == nil || s.Registers == nil {
		return nil, errors.New("order store is not configured")
	}
	cur, err := s.Orders.GetOrder(ctx, orderID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if cur.Status != domain.OrderStatusSave {
		e := &Error{Status: 409, Code: CodeOrderNotCommittable, Message: msgOrderNotCommittable}
		s.auditOrder(ctx, start, user, domain.ActionOrderCommit, orderID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if len(cur.Items) == 0 {
		e := &Error{Status: 409, Code: CodeOrderEmptyItems, Message: msgOrderEmptyItems}
		s.auditOrder(ctx, start, user, domain.ActionOrderCommit, orderID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	var total int64
	for i := range cur.Items {
		if cur.Items[i].AmountMilli <= 0 || cur.Items[i].PriceCents < 0 {
			e := &Error{Status: 409, Code: CodeOrderInvalidAmount, Message: msgOrderInvalidAmount}
			s.auditOrder(ctx, start, user, domain.ActionOrderCommit, orderID, domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		}
		fee, ok := lineFeeCents(cur.Items[i].AmountMilli, cur.Items[i].PriceCents)
		if !ok || total > math.MaxInt64-fee {
			e := &Error{Status: 409, Code: CodeOrderInvalidAmount, Message: msgOrderInvalidAmount}
			s.auditOrder(ctx, start, user, domain.ActionOrderCommit, orderID, domain.OutcomeFail, e.Code, requestID, ip)
			return nil, e
		}
		cur.Items[i].TotalFeeCents = fee
		total += fee
	}
	if _, err := s.requireOpenRegister(ctx, cur.RegisterID); err != nil {
		s.auditOrderErr(ctx, start, user, domain.ActionOrderCommit, orderID, requestID, ip, err)
		return nil, err
	}
	now := s.now().UTC()
	cur.TotalFeeCents = total
	cur.SubmittedAt = now
	cur.UpdatedAt = now
	cur.UpdatedBy = user.ID
	updated, err := s.Orders.CommitOrder(ctx, *cur)
	if err != nil {
		mapped := mapOrderErr(err)
		s.auditOrderErr(ctx, start, user, domain.ActionOrderCommit, orderID, requestID, ip, mapped)
		return nil, mapped
	}
	dto := toOrderDTO(*updated)
	s.auditOrder(ctx, start, user, domain.ActionOrderCommit, dto.OrderID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

// AbateOrder moves an uncharged SAVE or COMMIT sheet to ABATE.
func (s *Service) AbateOrder(ctx context.Context, claims *auth.Claims, orderID, requestID, ip string) (*OrderDTO, error) {
	start := time.Now()
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	orderID = strings.TrimSpace(orderID)
	if !hasPerm(user, domain.PermOrderSubmit) {
		e := forbidden()
		s.auditOrder(ctx, start, user, domain.ActionOrderAbate, orderID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if orderID == "" {
		return nil, notFound()
	}
	if s.Orders == nil {
		return nil, errors.New("order store is not configured")
	}
	cur, err := s.Orders.GetOrder(ctx, orderID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	if cur.Charged {
		e := &Error{Status: 409, Code: CodeOrderChargedNotAbatable, Message: msgOrderChargedNotAbatable}
		s.auditOrder(ctx, start, user, domain.ActionOrderAbate, orderID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	if cur.Status != domain.OrderStatusSave && cur.Status != domain.OrderStatusCommit {
		e := &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgIllegalStateTransition}
		s.auditOrder(ctx, start, user, domain.ActionOrderAbate, orderID, domain.OutcomeFail, e.Code, requestID, ip)
		return nil, e
	}
	updated, err := s.Orders.AbateOrder(ctx, orderID, user.ID, s.now().UTC())
	if err != nil {
		mapped := mapOrderErr(err)
		s.auditOrderErr(ctx, start, user, domain.ActionOrderAbate, orderID, requestID, ip, mapped)
		return nil, mapped
	}
	dto := toOrderDTO(*updated)
	s.auditOrder(ctx, start, user, domain.ActionOrderAbate, dto.OrderID, domain.OutcomeSuccess, "", requestID, ip)
	return &dto, nil
}

func (s *Service) requireOpenRegister(ctx context.Context, registerID string) (*domain.Register, error) {
	r, err := s.Registers.GetRegister(ctx, registerID)
	if errors.Is(err, port.ErrNotFound) {
		return nil, &Error{Status: 409, Code: CodeEncounterRequired, Message: msgEncounterRequired}
	}
	if err != nil {
		return nil, err
	}
	if r.Status == domain.RegisterStatusBack {
		return nil, &Error{Status: 409, Code: CodeEncounterRequired, Message: msgEncounterRequired}
	}
	return r, nil
}

func (s *Service) requireDeptDoctor(ctx context.Context, departmentID, doctorID string) error {
	ok, err := s.Registers.DepartmentExists(ctx, departmentID)
	if err != nil {
		return err
	}
	if !ok {
		return &Error{Status: 404, Code: CodeDepartmentNotFound, Message: msgDepartmentNotFound}
	}
	ok, err = s.Registers.DoctorExists(ctx, doctorID)
	if err != nil {
		return err
	}
	if !ok {
		return &Error{Status: 404, Code: CodeDoctorNotFound, Message: msgDoctorNotFound}
	}
	return nil
}

func (s *Service) prepareItems(ctx context.Context, inputs []OrderItemInput) ([]domain.OrderItem, int64, error) {
	if len(inputs) > maxOrderItems {
		return nil, 0, validationFields([]string{"items"})
	}
	fields := make([]string, 0)
	used := map[int]bool{}
	next := 1
	type draft struct {
		line             int
		itemNo           string
		itemType         string
		itemName         string
		spec             string
		units            string
		amount           float64
		price            float64
		usage            string
		frequency        string
		dosage           string
		dosageUnit       string
		procDepartmentID string
	}
	drafts := make([]draft, 0, len(inputs))
	for _, in := range inputs {
		itemNo := strings.TrimSpace(in.ItemNo)
		itemType := strings.TrimSpace(in.ItemType)
		itemName := strings.TrimSpace(in.ItemName)
		if itemNo == "" || len(itemNo) > 32 {
			fields = addField(fields, "itemNo")
		}
		if itemType != domain.ItemTypeDrug && itemType != domain.ItemTypeTreat {
			fields = addField(fields, "itemType")
		}
		if itemName == "" || utf8.RuneCountInString(itemName) > 128 {
			fields = addField(fields, "itemName")
		}
		spec := derefTrim(in.Spec)
		if utf8.RuneCountInString(spec) > 64 {
			fields = addField(fields, "spec")
		}
		units := strings.TrimSpace(in.Units)
		if utf8.RuneCountInString(units) > 16 {
			fields = addField(fields, "units")
		}
		usage := derefTrim(in.UsageText)
		if utf8.RuneCountInString(usage) > 64 {
			fields = addField(fields, "usageText")
		}
		freq := derefTrim(in.FrequencyText)
		if utf8.RuneCountInString(freq) > 64 {
			fields = addField(fields, "frequencyText")
		}
		dosage := derefTrim(in.Dosage)
		if utf8.RuneCountInString(dosage) > 64 {
			fields = addField(fields, "dosage")
		}
		dosageUnit := derefTrim(in.DosageUnit)
		if utf8.RuneCountInString(dosageUnit) > 16 {
			fields = addField(fields, "dosageUnit")
		}
		proc := derefTrim(in.ProcDepartmentID)
		if len(proc) > 64 {
			fields = addField(fields, "procDepartmentId")
		}
		ln := in.LineNo
		if ln == 0 {
			for used[next] {
				next++
			}
			ln = next
			next++
		}
		if ln < 1 || used[ln] {
			fields = addField(fields, "lineNo")
		}
		used[ln] = true
		drafts = append(drafts, draft{
			line: ln, itemNo: itemNo, itemType: itemType, itemName: itemName, spec: spec, units: units,
			amount: in.Amount, price: in.Price, usage: usage, frequency: freq, dosage: dosage,
			dosageUnit: dosageUnit, procDepartmentID: proc,
		})
	}
	if len(fields) > 0 {
		return nil, 0, validationFields(fields)
	}
	var total int64
	out := make([]domain.OrderItem, 0, len(drafts))
	for _, d := range drafts {
		milli, okAmt := amountMilli(d.amount)
		cents, okPrice := priceCents(d.price)
		fee, okFee := lineFeeCents(milli, cents)
		if !okAmt || !okPrice || !okFee || total > math.MaxInt64-fee {
			return nil, 0, &Error{Status: 409, Code: CodeOrderInvalidAmount, Message: msgOrderInvalidAmount}
		}
		total += fee
		cat, err := s.Orders.CatalogByItemNo(ctx, d.itemNo)
		if errors.Is(err, port.ErrNotFound) {
			return nil, 0, &Error{Status: 409, Code: CodeCatalogItemNotFound, Message: msgCatalogItemNotFound, Details: map[string]any{"itemNo": d.itemNo}}
		}
		if err != nil {
			return nil, 0, err
		}
		if cat.ItemType != d.itemType {
			return nil, 0, validationFields([]string{"itemType"})
		}
		units := d.units
		if units == "" {
			units = cat.Units
		}
		if units == "" || utf8.RuneCountInString(units) > 16 {
			return nil, 0, validationFields([]string{"units"})
		}
		spec := d.spec
		if spec == "" {
			spec = cat.Spec
		}
		out = append(out, domain.OrderItem{
			LineNo:           d.line,
			ItemNo:           d.itemNo,
			ItemType:         d.itemType,
			ItemName:         d.itemName,
			Spec:             spec,
			Units:            units,
			AmountMilli:      milli,
			PriceCents:       cents,
			TotalFeeCents:    fee,
			UsageText:        d.usage,
			FrequencyText:    d.frequency,
			Dosage:           d.dosage,
			DosageUnit:       d.dosageUnit,
			ProcDepartmentID: d.procDepartmentID,
		})
	}
	return out, total, nil
}

func normalizeOrderCreate(in CreateOrderInput) (orderCanon, []string) {
	c := orderCanon{
		RegisterID:   strings.TrimSpace(in.RegisterID),
		EncounterID:  derefTrim(in.EncounterID),
		PatientID:    derefTrim(in.PatientID),
		DepartmentID: derefTrim(in.DepartmentID),
		DoctorID:     derefTrim(in.DoctorID),
		DiagnoseName: derefTrim(in.DiagnoseName),
		RecipeSort:   derefTrim(in.RecipeSort),
	}
	var fields []string
	if c.RegisterID == "" || len(c.RegisterID) > 64 {
		fields = append(fields, "registerId")
	}
	if c.EncounterID != "" && c.EncounterID != c.RegisterID {
		fields = append(fields, "encounterId")
	}
	if len(c.PatientID) > 64 {
		fields = append(fields, "patientId")
	}
	if len(c.DepartmentID) > 64 {
		fields = append(fields, "departmentId")
	}
	if len(c.DoctorID) > 64 {
		fields = append(fields, "doctorId")
	}
	if utf8.RuneCountInString(c.DiagnoseName) > 128 {
		fields = append(fields, "diagnoseName")
	}
	if len(c.RecipeSort) > 32 {
		fields = append(fields, "recipeSort")
	}
	if len(in.Items) > maxOrderItems {
		fields = append(fields, "items")
	}
	c.Items = make([]orderItemCanon, 0, len(in.Items))
	used := map[int]bool{}
	next := 1
	for _, it := range in.Items {
		ln := it.LineNo
		if ln == 0 {
			for used[next] {
				next++
			}
			ln = next
			next++
		}
		if ln < 1 || used[ln] {
			fields = addField(fields, "lineNo")
		}
		used[ln] = true
		milli, _ := amountMilli(it.Amount)
		cents, _ := priceCents(it.Price)
		c.Items = append(c.Items, orderItemCanon{
			LineNo: ln, ItemNo: strings.TrimSpace(it.ItemNo), ItemType: strings.TrimSpace(it.ItemType),
			ItemName: strings.TrimSpace(it.ItemName), Spec: derefTrim(it.Spec), Units: strings.TrimSpace(it.Units),
			AmountMilli: milli, PriceCents: cents, UsageText: derefTrim(it.UsageText),
			FrequencyText: derefTrim(it.FrequencyText), Dosage: derefTrim(it.Dosage),
			DosageUnit: derefTrim(it.DosageUnit), ProcDepartmentID: derefTrim(it.ProcDepartmentID),
		})
	}
	return c, fields
}

func amountMilli(v float64) (int64, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > 999999999999.999 {
		return 0, false
	}
	m := math.Round(v * 1000)
	if m <= 0 || m > 999999999999999 {
		return 0, false
	}
	return int64(m), true
}

func priceCents(v float64) (int64, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > maxRegisterFee {
		return 0, false
	}
	c := math.Round(v * 100)
	if c < 0 || c > 999999999999999999 {
		return 0, false
	}
	return int64(c), true
}

func lineFeeCents(amountMilli, priceCents int64) (int64, bool) {
	if amountMilli <= 0 || priceCents < 0 {
		return 0, false
	}
	if priceCents > 0 && amountMilli > math.MaxInt64/priceCents {
		return 0, false
	}
	return (amountMilli*priceCents + 500) / 1000, true
}

func validOrderStatus(s string) bool {
	switch s {
	case domain.OrderStatusSave, domain.OrderStatusCommit, domain.OrderStatusProc, domain.OrderStatusAbate:
		return true
	default:
		return false
	}
}

func validationFields(fields []string) *Error {
	return &Error{Status: 400, Code: CodeValidation, Message: msgValidation, Details: map[string]any{"fields": fields}}
}

func addField(fields []string, name string) []string {
	for _, f := range fields {
		if f == name {
			return fields
		}
	}
	return append(fields, name)
}

func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Service) lookupOrderIdem(key, hash string) (*OrderDTO, error) {
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	hit, ok := s.idem[idemScopeOrderCreate+"\x00"+key]
	if !ok {
		return nil, nil
	}
	if hit.hash != hash {
		return nil, &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
	}
	var dto OrderDTO
	if err := json.Unmarshal(hit.body, &dto); err != nil {
		return nil, err
	}
	return &dto, nil
}

func (s *Service) saveOrderIdem(key, hash string, dto OrderDTO) (*OrderDTO, error) {
	body, err := json.Marshal(dto)
	if err != nil {
		return nil, err
	}
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	if s.idem == nil {
		s.idem = map[string]idemHit{}
	}
	k := idemScopeOrderCreate + "\x00" + key
	if hit, ok := s.idem[k]; ok {
		if hit.hash != hash {
			return nil, &Error{Status: 409, Code: CodeIdempotencyConflict, Message: msgIdempotencyConflict}
		}
		var prev OrderDTO
		if err := json.Unmarshal(hit.body, &prev); err != nil {
			return nil, err
		}
		return &prev, nil
	}
	s.idem[k] = idemHit{hash: hash, body: body}
	return &dto, nil
}

func (s *Service) auditOrder(ctx context.Context, start time.Time, user *domain.User, action, resourceID, outcome, errorCode, requestID, ip string) {
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
		ActorID:      actor,
		ActorRoles:   roles,
		Action:       action,
		ResourceType: "Order",
		ResourceID:   resourceID,
		Outcome:      outcome,
		ErrorCode:    errorCode,
		RequestID:    requestID,
		ClientIP:     ip,
	}, level)
}

func (s *Service) auditOrderErr(ctx context.Context, start time.Time, user *domain.User, action, resourceID, requestID, ip string, err error) {
	var ae *Error
	if errors.As(err, &ae) && (ae.Status == 403 || ae.Status == 409) {
		s.auditOrder(ctx, start, user, action, resourceID, domain.OutcomeFail, ae.Code, requestID, ip)
	}
}

func mapOrderErr(err error) error {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return notFound()
	case errors.Is(err, port.ErrOrderNotEditable):
		return &Error{Status: 409, Code: CodeOrderNotEditable, Message: msgOrderNotEditable}
	case errors.Is(err, port.ErrOrderNotCommittable):
		return &Error{Status: 409, Code: CodeOrderNotCommittable, Message: msgOrderNotCommittable}
	case errors.Is(err, port.ErrOrderChargedNotAbatable):
		return &Error{Status: 409, Code: CodeOrderChargedNotAbatable, Message: msgOrderChargedNotAbatable}
	case errors.Is(err, port.ErrIllegalStateTransition):
		return &Error{Status: 409, Code: CodeIllegalStateTransition, Message: msgIllegalStateTransition}
	default:
		return err
	}
}

func toOrderDTO(o domain.Order) OrderDTO {
	recipe := o.RecipeSort
	if recipe == "" {
		recipe = "NORMAL"
	}
	items := make([]OrderItemDTO, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderItemDTO{
			OrderItemID:      it.OrderItemID,
			LineNo:           it.LineNo,
			ItemNo:           it.ItemNo,
			ItemType:         it.ItemType,
			ItemName:         it.ItemName,
			Spec:             strPtrOrNil(it.Spec),
			Units:            it.Units,
			Amount:           float64(it.AmountMilli) / 1000,
			Price:            float64(it.PriceCents) / 100,
			TotalFee:         float64(it.TotalFeeCents) / 100,
			UsageText:        strPtrOrNil(it.UsageText),
			FrequencyText:    strPtrOrNil(it.FrequencyText),
			Dosage:           strPtrOrNil(it.Dosage),
			DosageUnit:       strPtrOrNil(it.DosageUnit),
			ProcDepartmentID: strPtrOrNil(it.ProcDepartmentID),
		})
	}
	return OrderDTO{
		OrderID:      o.OrderID,
		OrderNo:      o.OrderNo,
		RegisterID:   o.RegisterID,
		EncounterID:  o.RegisterID,
		PatientID:    o.PatientID,
		DepartmentID: o.DepartmentID,
		DoctorID:     o.DoctorID,
		DiagnoseName: strPtrOrNil(o.DiagnoseName),
		RecipeSort:   recipe,
		Status:       o.Status,
		TotalFee:     float64(o.TotalFeeCents) / 100,
		Charged:      o.Charged,
		Items:        items,
		SubmittedAt:  timePtrOrNil(o.SubmittedAt),
		AbatedAt:     timePtrOrNil(o.AbatedAt),
		AbatedBy:     strPtrOrNil(o.AbatedBy),
	}
}
