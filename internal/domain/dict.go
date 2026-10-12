package domain

import "time"

// DictItem is dbo.dict_item. Phase1 maintains register_sort (seed sort-normal). 2026-10-13
type DictItem struct {
	DictItemID string
	DictType   string
	Code       string
	Name       string
	Status     string
	SortNo     int
	CreatedAt  time.Time
	CreatedBy  string
	UpdatedAt  time.Time
	UpdatedBy  string
	IsDeleted  bool
}
