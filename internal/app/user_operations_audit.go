package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

const (
	maxUserOperationPageSize     = 200
	defaultUserOperationPageSize = 50
)

type UserOperationFilter struct {
	ModuleID string
	UserID   string
	Begin    string
	End      string
	Cursor   string
	Limit    int
}

type UserOperationPageDTO struct {
	Items      []domain.UserOperationLog `json:"items"`
	NextCursor string                    `json:"nextCursor"`
}

type userOperationCursor struct {
	OccurredAt time.Time `json:"at"`
	ID         string    `json:"id"`
	ModuleID   string    `json:"moduleId"`
	UserID     string    `json:"userId"`
	Begin      string    `json:"begin"`
	End        string    `json:"end"`
}

func (s *Service) ListUserOperations(ctx context.Context, claims *auth.Claims, filter UserOperationFilter) (*UserOperationPageDTO, error) {
	actor, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !hasPerm(actor, domain.PermAuditRead) && !hasPerm(actor, domain.PermAdminAll) {
		return nil, forbidden()
	}
	store, ok := s.Audit.(port.UserOperationQueryStore)
	if !ok || store == nil {
		return nil, errors.New("user operation query store is not configured")
	}
	if len(filter.ModuleID) > 64 || len(filter.UserID) > 64 {
		return nil, Validation("filters")
	}
	limit := filter.Limit
	if limit == 0 {
		limit = defaultUserOperationPageSize
	}
	if limit < 1 || limit > maxUserOperationPageSize {
		return nil, Validation("limit")
	}

	moduleID, userID := strings.TrimSpace(filter.ModuleID), strings.TrimSpace(filter.UserID)
	query := port.UserOperationQuery{ModuleID: moduleID, UserID: userID, Limit: limit + 1}
	loc := time.FixedZone("Asia/Shanghai", 8*60*60)
	if filter.Begin != "" {
		query.Begin, err = time.ParseInLocation("2006-01-02", filter.Begin, loc)
		if err != nil {
			return nil, Validation("begin")
		}
	}
	if filter.End != "" {
		end, parseErr := time.ParseInLocation("2006-01-02", filter.End, loc)
		if parseErr != nil {
			return nil, Validation("end")
		}
		query.EndExclusive = end.AddDate(0, 0, 1)
	}
	if !query.Begin.IsZero() && !query.EndExclusive.IsZero() && !query.Begin.Before(query.EndExclusive) {
		return nil, Validation("dateRange")
	}
	if filter.Cursor != "" {
		cursor, decodeErr := decodeUserOperationCursor(filter.Cursor)
		if decodeErr != nil || cursor.ModuleID != moduleID || cursor.UserID != userID || cursor.Begin != filter.Begin || cursor.End != filter.End {
			return nil, Validation("cursor")
		}
		query.CursorOccurredAt = cursor.OccurredAt
		query.CursorID = cursor.ID
	}
	items, err := store.ListUserOperations(ctx, query)
	if err != nil {
		return nil, err
	}
	page := &UserOperationPageDTO{Items: items}
	if len(items) > limit {
		last := items[limit-1]
		page.Items = items[:limit]
		page.NextCursor = encodeUserOperationCursor(userOperationCursor{
			OccurredAt: last.OccurredAt, ID: last.ID, ModuleID: moduleID, UserID: userID,
			Begin: filter.Begin, End: filter.End,
		})
	}
	return page, nil
}

func encodeUserOperationCursor(cursor userOperationCursor) string {
	data, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeUserOperationCursor(raw string) (userOperationCursor, error) {
	var cursor userOperationCursor
	if len(raw) > 512 {
		return cursor, errors.New("cursor too long")
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(data) == 0 {
		return cursor, errors.New("invalid cursor encoding")
	}
	if err := json.Unmarshal(data, &cursor); err != nil || cursor.ID == "" || len(cursor.ID) > 64 || cursor.OccurredAt.IsZero() {
		return userOperationCursor{}, errors.New("invalid cursor data")
	}
	return cursor, nil
}
