package port

import (
	"context"
	"time"
)

// DashboardQuery carries already-authorized department scopes into one aggregate read.
// Empty department lists intentionally produce zero values for that metric group.
type DashboardQuery struct {
	StartAt                time.Time
	EndAt                  time.Time
	EncounterDepartmentIDs []string
	ChargeDepartmentIDs    []string
	DispenseDepartmentIDs  []string
}

// DashboardMetrics contains aggregate values only; it never returns patient or encounter rows.
type DashboardMetrics struct {
	RevenueCents         int64
	RefundCents          int64
	NetRevenueCents      int64
	ChargeCount          int
	RefundCount          int
	EncounterCount       int
	PendingChargeCount   int
	PendingDispenseCount int
}

// DashboardStore must compute these values in the data store without applying list limits.
type DashboardStore interface {
	SummarizeDashboard(ctx context.Context, query DashboardQuery) (DashboardMetrics, error)
}
