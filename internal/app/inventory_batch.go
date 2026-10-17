package app

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

const (
	inventoryBatchDefaultPageSize = 50
	inventoryBatchMaxPageSize     = 200
)

type InventoryBatchSearch struct {
	SkuCode             string
	StorageUnitID       string
	BatchNo             string
	ExpiryStatus        string
	AsOfDate            string
	ExpiryWindowDays    int
	ExpiryWindowDaysSet bool
	Cursor              string
	Limit               int
}

type InventoryBatchDTO struct {
	BatchID       string  `json:"batchId"`
	SkuID         string  `json:"skuId"`
	ItemNo        string  `json:"itemNo"`
	ItemName      string  `json:"itemName"`
	StorageUnitID string  `json:"storageUnitId,omitempty"`
	BatchNo       string  `json:"batchNo,omitempty"`
	ProductNo     string  `json:"productNo,omitempty"`
	SupplierID    string  `json:"supplierId,omitempty"`
	ProductDate   *string `json:"productDate"`
	ExpireDate    *string `json:"expireDate"`
	QtyOnHand     string  `json:"qtyOnHand"`
	Units         string  `json:"units"`
	SourceRef     string  `json:"sourceRef,omitempty"`
	ExpiryStatus  string  `json:"expiryStatus"`
	RowVersion    string  `json:"rowVersion,omitempty"`
}

type InventoryBatchSummaryDTO struct {
	BatchCount        int    `json:"batchCount"`
	QtyTotal          string `json:"qtyTotal"`
	QtyUnknown        string `json:"qtyUnknown"`
	DuplicateKeyCount int    `json:"duplicateKeyCount"`
}

type InventoryBatchPageDTO struct {
	Items            []InventoryBatchDTO      `json:"items"`
	Summary          InventoryBatchSummaryDTO `json:"summary"`
	AsOfDate         string                   `json:"asOfDate"`
	ExpiryWindowDays int                      `json:"expiryWindowDays"`
	NextCursor       string                   `json:"nextCursor"`
}

func (s *Service) ListInventoryBatches(ctx context.Context, claims *auth.Claims, q InventoryBatchSearch) (*InventoryBatchPageDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPermAcrossScopes(user, domain.PermInventoryBatchRead, "master") {
		return nil, forbidden()
	}
	if s.InventoryBatches == nil {
		return nil, errors.New("inventory batch store is not configured")
	}

	sku := strings.TrimSpace(q.SkuCode)
	storage := strings.TrimSpace(q.StorageUnitID)
	batch := strings.TrimSpace(q.BatchNo)
	if len(sku) > 64 {
		return nil, validationFields([]string{"skuCode"})
	}
	if len(storage) > 64 {
		return nil, validationFields([]string{"storageUnitId"})
	}
	if len(batch) > 64 {
		return nil, validationFields([]string{"batchNo"})
	}
	status := strings.ToUpper(strings.TrimSpace(q.ExpiryStatus))
	switch status {
	case "", domain.ExpiryStatusUnknown, domain.ExpiryStatusValid, domain.ExpiryStatusExpiring, domain.ExpiryStatusExpired:
	default:
		return nil, validationFields([]string{"expiryStatus"})
	}
	if q.AsOfDate == "" {
		return nil, validationFields([]string{"asOfDate"})
	}
	asOf, ok := parseInventoryBatchDate(q.AsOfDate)
	if !ok {
		return nil, validationFields([]string{"asOfDate"})
	}
	if !q.ExpiryWindowDaysSet || q.ExpiryWindowDays < 0 || q.ExpiryWindowDays > 3660 {
		return nil, validationFields([]string{"expiryWindowDays"})
	}
	if q.Limit < 0 || q.Limit > inventoryBatchMaxPageSize {
		return nil, validationFields([]string{"limit"})
	}
	pageSize := q.Limit
	if pageSize == 0 {
		pageSize = inventoryBatchDefaultPageSize
	}
	cursor := ""
	if q.Cursor != "" {
		cursor, err = decodeInventoryBatchCursor(q.Cursor)
		if err != nil {
			return nil, validationFields([]string{"cursor"})
		}
	}
	rows, err := s.InventoryBatches.ListInventoryBatches(ctx, port.InventoryBatchQuery{
		SkuCode: sku, StorageUnitID: storage, BatchNo: batch, ExpiryStatus: status,
		AsOfDate: asOf, ExpiryWindowDays: q.ExpiryWindowDays, CursorBatchID: cursor, Limit: pageSize + 1,
	})
	if err != nil {
		return nil, err
	}
	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}
	items := make([]InventoryBatchDTO, 0, len(rows))
	var total, unknown int64
	keyCounts := make(map[string]int)
	for _, row := range rows {
		dto := toInventoryBatchDTO(row)
		items = append(items, dto)
		total += row.QtyOnHandMilli
		if row.ExpiryStatus == domain.ExpiryStatusUnknown {
			unknown += row.QtyOnHandMilli
		}
		keyCounts[row.StorageUnitID+"\x00"+row.SkuID+"\x00"+row.BatchNo+"\x00"+row.SourceRef]++
	}
	duplicateKeys := 0
	for _, count := range keyCounts {
		if count > 1 {
			duplicateKeys++
		}
	}
	page := &InventoryBatchPageDTO{
		Items:    items,
		Summary:  InventoryBatchSummaryDTO{BatchCount: len(items), QtyTotal: formatMilli(total), QtyUnknown: formatMilli(unknown), DuplicateKeyCount: duplicateKeys},
		AsOfDate: asOf.Format("2006-01-02"), ExpiryWindowDays: q.ExpiryWindowDays,
	}
	if hasMore && len(rows) > 0 {
		page.NextCursor = encodeInventoryBatchCursor(rows[len(rows)-1].BatchID)
	}
	return page, nil
}

func parseInventoryBatchDate(value string) (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), shanghai)
	return t, err == nil
}

func encodeInventoryBatchCursor(batchID string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(batchID))
}

func decodeInventoryBatchCursor(cursor string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil || len(decoded) == 0 || len(decoded) > 64 || strings.ContainsAny(string(decoded), "\r\n") {
		return "", errors.New("invalid inventory batch cursor")
	}
	return string(decoded), nil
}

func toInventoryBatchDTO(row domain.InventoryBatch) InventoryBatchDTO {
	format := func(value *time.Time) *string {
		if value == nil {
			return nil
		}
		v := value.In(shanghai).Format("2006-01-02")
		return &v
	}
	return InventoryBatchDTO{
		BatchID: row.BatchID, SkuID: row.SkuID, ItemNo: row.ItemNo, ItemName: row.ItemName,
		StorageUnitID: row.StorageUnitID, BatchNo: row.BatchNo, ProductNo: row.ProductNo, SupplierID: row.SupplierID,
		ProductDate: format(row.ProductDate), ExpireDate: format(row.ExpireDate), QtyOnHand: formatMilli(row.QtyOnHandMilli),
		Units: row.Units, SourceRef: row.SourceRef, ExpiryStatus: row.ExpiryStatus, RowVersion: row.RowVersion,
	}
}

func formatMilli(value int64) string {
	if value < 0 {
		return "-" + formatMilli(-value)
	}
	return fmt.Sprintf("%d.%03d", value/1000, value%1000)
}
