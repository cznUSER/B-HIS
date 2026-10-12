package memory

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// Department, doctor, catalog, dict, user, and inventory adjust maintenance. 2026-10-13

func (s *Store) ListDepartments(_ context.Context, q port.DepartmentQuery) ([]domain.Department, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Department, 0)
	for _, d := range s.departments {
		if d.IsDeleted || !matchStatus(d.Status, q.Status) || !matchQ(q.Q, d.DeptCode, d.DeptName) {
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortNo != out[j].SortNo {
			return out[i].SortNo < out[j].SortNo
		}
		return out[i].DeptCode < out[j].DeptCode
	})
	return out, nil
}

func (s *Store) CreateDepartment(_ context.Context, d domain.Department) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.departments[d.DepartmentID]; ok {
		return port.ErrDuplicateDeptCode
	}
	if deptCodeTaken(s.departments, d.DeptCode, "") {
		return port.ErrDuplicateDeptCode
	}
	s.departments[d.DepartmentID] = d
	return nil
}

func (s *Store) UpdateDepartment(_ context.Context, d domain.Department) (*domain.Department, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.departments[d.DepartmentID]
	if !ok || cur.IsDeleted {
		return nil, port.ErrNotFound
	}
	if deptCodeTaken(s.departments, d.DeptCode, d.DepartmentID) {
		return nil, port.ErrDuplicateDeptCode
	}
	d.CreatedAt = cur.CreatedAt
	d.CreatedBy = cur.CreatedBy
	d.IsDeleted = false
	s.departments[d.DepartmentID] = d
	return &d, nil
}

func (s *Store) ListDoctors(_ context.Context, q port.DoctorQuery) ([]domain.Doctor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Doctor, 0)
	for _, d := range s.doctors {
		if d.IsDeleted || !matchStatus(d.Status, q.Status) || !matchQ(q.Q, d.DoctorCode, d.DoctorName) {
			continue
		}
		if q.DepartmentID != "" && d.DepartmentID != q.DepartmentID {
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DoctorCode < out[j].DoctorCode })
	return out, nil
}

func (s *Store) CreateDoctor(_ context.Context, d domain.Doctor) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.doctors[d.DoctorID]; ok {
		return port.ErrDuplicateDoctorCode
	}
	if doctorCodeTaken(s.doctors, d.DoctorCode, "") {
		return port.ErrDuplicateDoctorCode
	}
	s.doctors[d.DoctorID] = d
	return nil
}

func (s *Store) UpdateDoctor(_ context.Context, d domain.Doctor) (*domain.Doctor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.doctors[d.DoctorID]
	if !ok || cur.IsDeleted {
		return nil, port.ErrNotFound
	}
	if doctorCodeTaken(s.doctors, d.DoctorCode, d.DoctorID) {
		return nil, port.ErrDuplicateDoctorCode
	}
	d.CreatedAt = cur.CreatedAt
	d.CreatedBy = cur.CreatedBy
	d.IsDeleted = false
	s.doctors[d.DoctorID] = d
	return &d, nil
}

func (s *Store) ListCatalog(_ context.Context, q port.CatalogQuery) ([]domain.CatalogItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.CatalogItem, 0)
	for _, c := range s.catalog {
		if c.IsDeleted || !matchStatus(c.Status, q.Status) || !matchQ(q.Q, c.ItemNo, c.ItemName) {
			continue
		}
		if q.ItemType != "" && c.ItemType != q.ItemType {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ItemNo < out[j].ItemNo })
	return out, nil
}

func (s *Store) GetCatalog(_ context.Context, catalogItemID string) (*domain.CatalogItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.catalogByID(catalogItemID)
	if !ok {
		return nil, port.ErrNotFound
	}
	return &c, nil
}

func (s *Store) CreateCatalog(_ context.Context, item domain.CatalogItem, skuID string) (*domain.CatalogItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if itemNoTaken(s.catalog, item.ItemNo, "") {
		return nil, port.ErrDuplicateItemNo
	}
	if item.ItemType == domain.ItemTypeDrug {
		if skuID == "" {
			return nil, port.ErrNotDrugSKU
		}
		if _, ok := s.inventory[item.ItemNo]; ok || inventoryNoTaken(s.inventory, item.ItemNo, "") {
			return nil, port.ErrDuplicateItemNo
		}
		s.inventory[item.ItemNo] = domain.InventoryItem{
			SkuID: skuID, ItemNo: item.ItemNo, ItemName: item.ItemName, QtyOnHandMilli: 0,
			Units: item.Units, Status: domain.InventoryStatusActive, RowVersion: "1",
			CreatedAt: item.CreatedAt, CreatedBy: item.CreatedBy, UpdatedAt: item.UpdatedAt, UpdatedBy: item.UpdatedBy,
		}
	}
	s.catalog[item.ItemNo] = item
	cp := item
	return &cp, nil
}

func (s *Store) UpdateCatalog(_ context.Context, item domain.CatalogItem) (*domain.CatalogItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.catalogByID(item.CatalogItemID)
	if !ok {
		return nil, port.ErrNotFound
	}
	if itemNoTaken(s.catalog, item.ItemNo, item.CatalogItemID) {
		return nil, port.ErrDuplicateItemNo
	}
	item.ItemType = cur.ItemType
	item.CreatedAt = cur.CreatedAt
	item.CreatedBy = cur.CreatedBy
	item.IsDeleted = false
	if cur.ItemType == domain.ItemTypeDrug {
		if err := s.syncDrugLocked(cur.ItemNo, item); err != nil {
			return nil, err
		}
	}
	if cur.ItemNo != item.ItemNo {
		delete(s.catalog, cur.ItemNo)
	}
	s.catalog[item.ItemNo] = item
	cp := item
	return &cp, nil
}

func (s *Store) syncDrugLocked(oldNo string, item domain.CatalogItem) error {
	row, ok := s.inventory[oldNo]
	if !ok {
		for k, it := range s.inventory {
			if !it.IsDeleted && strings.EqualFold(it.ItemNo, oldNo) {
				row = it
				oldNo = k
				ok = true
				break
			}
		}
	}
	if inventoryNoTaken(s.inventory, item.ItemNo, row.SkuID) {
		return port.ErrDuplicateItemNo
	}
	if !ok {
		s.inventory[item.ItemNo] = domain.InventoryItem{
			SkuID: "sku-" + item.CatalogItemID, ItemNo: item.ItemNo, ItemName: item.ItemName,
			Units: item.Units, Status: domain.InventoryStatusActive, RowVersion: "1",
			CreatedAt: item.UpdatedAt, CreatedBy: item.UpdatedBy, UpdatedAt: item.UpdatedAt, UpdatedBy: item.UpdatedBy,
		}
		return nil
	}
	if oldNo != item.ItemNo {
		delete(s.inventory, oldNo)
	}
	row.ItemNo = item.ItemNo
	row.ItemName = item.ItemName
	row.Units = item.Units
	row.UpdatedAt = item.UpdatedAt
	row.UpdatedBy = item.UpdatedBy
	row.RowVersion = bumpRowVersion(row.RowVersion)
	s.inventory[item.ItemNo] = row
	return nil
}

func (s *Store) AdjustInventory(_ context.Context, skuID string, deltaMilli int64, remark, operatorID, rowVersion, ledgerID string, at time.Time) (*domain.InventoryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ""
	var row domain.InventoryItem
	found := false
	for k, it := range s.inventory {
		if it.SkuID == skuID && !it.IsDeleted {
			key = k
			row = it
			found = true
			break
		}
	}
	if !found {
		return nil, port.ErrNotDrugSKU
	}
	if rowVersion != "" && rowVersion != row.RowVersion {
		return nil, port.ErrConflict
	}
	if deltaMilli < 0 {
		need := -deltaMilli
		if deltaMilli == math.MinInt64 || row.QtyOnHandMilli < need {
			return nil, port.ErrAdjustNegative
		}
	}
	if deltaMilli > 0 && row.QtyOnHandMilli > math.MaxInt64-deltaMilli {
		return nil, port.ErrAdjustNegative
	}
	row.QtyOnHandMilli += deltaMilli
	row.RowVersion = bumpRowVersion(row.RowVersion)
	row.UpdatedAt = at.UTC()
	row.UpdatedBy = operatorID
	s.inventory[key] = row
	s.ledgers = append(s.ledgers, domain.InventoryLedger{
		LedgerID: ledgerID, SkuID: row.SkuID, QtyDeltaMilli: deltaMilli,
		BizType: domain.LedgerBizAdjust, BizID: ledgerID, Remark: remark,
		CreatedAt: row.UpdatedAt, CreatedBy: operatorID,
	})
	cp := row
	return &cp, nil
}

func (s *Store) ListDictItems(_ context.Context, dictType, status string) ([]domain.DictItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.DictItem, 0)
	for _, it := range s.dicts {
		if it.IsDeleted || it.DictType != dictType || !matchStatus(it.Status, status) {
			continue
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortNo != out[j].SortNo {
			return out[i].SortNo < out[j].SortNo
		}
		return out[i].Code < out[j].Code
	})
	return out, nil
}

func (s *Store) CreateDictItem(_ context.Context, item domain.DictItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dicts[item.DictItemID]; ok {
		return port.ErrDuplicateDictCode
	}
	if dictCodeTaken(s.dicts, item.DictType, item.Code, "") {
		return port.ErrDuplicateDictCode
	}
	s.dicts[item.DictItemID] = item
	return nil
}

func (s *Store) UpdateDictItem(_ context.Context, item domain.DictItem) (*domain.DictItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.dicts[item.DictItemID]
	if !ok || cur.IsDeleted || cur.DictType != item.DictType {
		return nil, port.ErrNotFound
	}
	item.Code = cur.Code
	item.CreatedAt = cur.CreatedAt
	item.CreatedBy = cur.CreatedBy
	item.IsDeleted = false
	s.dicts[item.DictItemID] = item
	return &item, nil
}

func (s *Store) ListUsers(_ context.Context, q port.UserQuery) ([]domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.User, 0)
	for _, u := range s.byID {
		if !matchStatus(u.Status, q.Status) || !matchQ(q.Q, u.Username, u.DisplayName) {
			continue
		}
		out = append(out, clone(u))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Username < out[j].Username })
	return out, nil
}

func (s *Store) CreateUser(_ context.Context, u domain.User) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[u.ID]; ok {
		return nil, port.ErrDuplicateUsername
	}
	for name := range s.byName {
		if strings.EqualFold(name, u.Username) {
			return nil, port.ErrDuplicateUsername
		}
	}
	if u.Roles == nil {
		u.Roles = []domain.Role{}
	}
	s.byID[u.ID] = clone(u)
	s.byName[u.Username] = u.ID
	c := clone(u)
	return &c, nil
}

func (s *Store) UpdateUser(_ context.Context, u domain.User, replaceRoles bool) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.byID[u.ID]
	if !ok {
		return nil, port.ErrNotFound
	}
	cur.DisplayName = u.DisplayName
	cur.Status = u.Status
	cur.UpdatedAt = u.UpdatedAt
	cur.UpdatedBy = u.UpdatedBy
	if replaceRoles {
		cur.Roles = append([]domain.Role(nil), u.Roles...)
	}
	s.byID[u.ID] = cur
	c := clone(cur)
	return &c, nil
}

func (s *Store) ResetPassword(_ context.Context, userID, passwordHash, operatorID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byID[userID]
	if !ok {
		return port.ErrNotFound
	}
	u.PasswordHash = passwordHash
	u.UpdatedAt = at.UTC()
	u.UpdatedBy = operatorID
	s.byID[userID] = u
	return nil
}

func (s *Store) catalogByID(id string) (domain.CatalogItem, bool) {
	for _, c := range s.catalog {
		if c.CatalogItemID == id && !c.IsDeleted {
			return c, true
		}
	}
	return domain.CatalogItem{}, false
}

func deptCodeTaken(rows map[string]domain.Department, code, exceptID string) bool {
	for _, d := range rows {
		if d.IsDeleted || d.DepartmentID == exceptID {
			continue
		}
		if strings.EqualFold(d.DeptCode, code) {
			return true
		}
	}
	return false
}

func doctorCodeTaken(rows map[string]domain.Doctor, code, exceptID string) bool {
	for _, d := range rows {
		if d.IsDeleted || d.DoctorID == exceptID {
			continue
		}
		if strings.EqualFold(d.DoctorCode, code) {
			return true
		}
	}
	return false
}

func itemNoTaken(rows map[string]domain.CatalogItem, itemNo, exceptID string) bool {
	for _, c := range rows {
		if c.IsDeleted || c.CatalogItemID == exceptID {
			continue
		}
		if strings.EqualFold(c.ItemNo, itemNo) {
			return true
		}
	}
	return false
}

func inventoryNoTaken(rows map[string]domain.InventoryItem, itemNo, exceptSku string) bool {
	for _, it := range rows {
		if it.IsDeleted || (exceptSku != "" && it.SkuID == exceptSku) {
			continue
		}
		if strings.EqualFold(it.ItemNo, itemNo) {
			return true
		}
	}
	return false
}

func dictCodeTaken(rows map[string]domain.DictItem, dictType, code, exceptID string) bool {
	for _, it := range rows {
		if it.IsDeleted || it.DictItemID == exceptID || it.DictType != dictType {
			continue
		}
		if strings.EqualFold(it.Code, code) {
			return true
		}
	}
	return false
}

func matchStatus(have, want string) bool {
	return want == "" || have == want
}

func matchQ(q string, fields ...string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return true
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), strings.ToLower(q)) {
			return true
		}
	}
	return false
}

func bumpRowVersion(cur string) string {
	n, err := strconv.Atoi(strings.TrimSpace(cur))
	if err != nil || n < 0 {
		n = 0
	}
	return strconv.Itoa(n + 1)
}
