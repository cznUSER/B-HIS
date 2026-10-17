package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

const CodeDashboardUnsupported = "DASHBOARD_UNSUPPORTED"

type DashboardPeriod string

const (
	DashboardToday     DashboardPeriod = "TODAY"
	DashboardYesterday DashboardPeriod = "YESTERDAY"
	DashboardWeek      DashboardPeriod = "WEEK"
	DashboardMonth     DashboardPeriod = "MONTH"
)

type DashboardSummaryDTO struct {
	Period               DashboardPeriod          `json:"period"`
	RangeStart           time.Time                `json:"rangeStart"`
	RangeEnd             time.Time                `json:"rangeEnd"`
	RevenueCents         int64                    `json:"revenueCents"`
	RefundCents          int64                    `json:"refundCents"`
	NetRevenueCents      int64                    `json:"netRevenueCents"`
	ChargeCount          int                      `json:"chargeCount"`
	RefundCount          int                      `json:"refundCount"`
	EncounterCount       int                      `json:"encounterCount"`
	AverageTicketCents   int64                    `json:"averageTicketCents"`
	PendingChargeCount   int                      `json:"pendingChargeCount"`
	PendingDispenseCount int                      `json:"pendingDispenseCount"`
	Available            DashboardAvailabilityDTO `json:"available"`
}

type DashboardAvailabilityDTO struct {
	Charges    bool `json:"charges"`
	Encounters bool `json:"encounters"`
	Dispenses  bool `json:"dispenses"`
}

// DashboardSummary returns period totals and live pending-work snapshots, filtered by current RBAC scopes.
func (s *Service) DashboardSummary(ctx context.Context, claims *auth.Claims, rawPeriod string) (*DashboardSummaryDTO, error) {
	user, err := s.actor(ctx, claims)
	if err != nil {
		return nil, err
	}
	period := DashboardPeriod(strings.ToUpper(strings.TrimSpace(rawPeriod)))
	start, end, ok := dashboardWindow(s.now(), period)
	if !ok {
		return nil, &Error{Status: 400, Code: CodeValidation, Message: msgValidation, Details: map[string]any{"fields": []string{"period"}}}
	}
	if s.Master == nil {
		return nil, errors.New("dashboard department store is not configured")
	}
	departments, err := s.Master.ListDepartments(ctx, port.DepartmentQuery{})
	if err != nil {
		return nil, err
	}
	query := port.DashboardQuery{StartAt: start, EndAt: end}
	for _, department := range departments {
		if hasAnyPermScope(user, "outpatient", department.DepartmentID, domain.PermEncounterCreate, domain.PermEncounterCheckIn, domain.PermEncounterCancel) {
			query.EncounterDepartmentIDs = append(query.EncounterDepartmentIDs, department.DepartmentID)
		}
		if hasAnyPermScope(user, "outpatient", department.DepartmentID, domain.PermChargePay, domain.PermChargeRefund) {
			query.ChargeDepartmentIDs = append(query.ChargeDepartmentIDs, department.DepartmentID)
		}
		if hasAnyPermScope(user, "outpatient", department.DepartmentID, domain.PermDispenseCreate, domain.PermDispenseReturn) {
			query.DispenseDepartmentIDs = append(query.DispenseDepartmentIDs, department.DepartmentID)
		}
	}
	if len(query.EncounterDepartmentIDs)+len(query.ChargeDepartmentIDs)+len(query.DispenseDepartmentIDs) == 0 {
		return nil, forbidden()
	}
	if s.Dashboard == nil {
		return nil, &Error{Status: 501, Code: CodeDashboardUnsupported, Message: "当前数据库尚未实现工作台汇总查询"}
	}
	metrics, err := s.Dashboard.SummarizeDashboard(ctx, query)
	if err != nil {
		return nil, err
	}
	dto := &DashboardSummaryDTO{
		Period: period, RangeStart: start, RangeEnd: end,
		RevenueCents: metrics.RevenueCents, RefundCents: metrics.RefundCents,
		NetRevenueCents: metrics.NetRevenueCents, ChargeCount: metrics.ChargeCount,
		RefundCount: metrics.RefundCount, EncounterCount: metrics.EncounterCount,
		PendingChargeCount: metrics.PendingChargeCount, PendingDispenseCount: metrics.PendingDispenseCount,
		Available: DashboardAvailabilityDTO{
			Charges:    len(query.ChargeDepartmentIDs) > 0,
			Encounters: len(query.EncounterDepartmentIDs) > 0,
			Dispenses:  len(query.DispenseDepartmentIDs) > 0,
		},
	}
	if dto.EncounterCount > 0 {
		dto.AverageTicketCents = dto.RevenueCents / int64(dto.EncounterCount)
	}
	return dto, nil
}

func dashboardWindow(now time.Time, period DashboardPeriod) (time.Time, time.Time, bool) {
	local := now.In(shanghai)
	year, month, day := local.Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, shanghai)
	switch period {
	case DashboardToday:
		return today, today.AddDate(0, 0, 1), true
	case DashboardYesterday:
		return today.AddDate(0, 0, -1), today, true
	case DashboardWeek:
		mondayOffset := (int(today.Weekday()) + 6) % 7
		start := today.AddDate(0, 0, -mondayOffset)
		return start, start.AddDate(0, 0, 7), true
	case DashboardMonth:
		start := time.Date(year, month, 1, 0, 0, 0, 0, shanghai)
		return start, start.AddDate(0, 1, 0), true
	default:
		return time.Time{}, time.Time{}, false
	}
}
