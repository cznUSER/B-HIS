package mssql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.DashboardStore = (*Store)(nil)

func (s *Store) SummarizeDashboard(ctx context.Context, query port.DashboardQuery) (port.DashboardMetrics, error) {
	var result port.DashboardMetrics
	if len(query.EncounterDepartmentIDs) == 0 && len(query.ChargeDepartmentIDs) == 0 && len(query.DispenseDepartmentIDs) == 0 {
		return result, nil
	}

	encounterScope, encounterArgs := dashboardScope("r.department_id", "encounter_dept", query.EncounterDepartmentIDs)
	chargeScope, chargeArgs := dashboardScope("r.department_id", "charge_dept", query.ChargeDepartmentIDs)
	dispenseScope, dispenseArgs := dashboardScope("r.department_id", "dispense_dept", query.DispenseDepartmentIDs)
	args := []any{
		sql.Named("start_at", query.StartAt.UTC()),
		sql.Named("end_at", query.EndAt.UTC()),
	}
	args = append(args, chargeArgs...)
	args = append(args, encounterArgs...)
	args = append(args, dispenseArgs...)

	q := `
SELECT
	(
		SELECT COALESCE(SUM(CONVERT(bigint, ROUND(p.amount * 100, 0))), 0)
		FROM dbo.payment p
		JOIN dbo.charge c ON c.charge_id = p.charge_id AND c.is_deleted = 0
		JOIN dbo.register_bill r ON r.register_id = c.register_id AND r.is_deleted = 0
		WHERE p.status = 'SUCCEEDED' AND p.direction = 'CHARGE'
		  AND p.paid_at >= @start_at AND p.paid_at < @end_at
		  AND ` + chargeScope + `
	),
	(
		SELECT COALESCE(SUM(CONVERT(bigint, ROUND(p.amount * 100, 0))), 0)
		FROM dbo.payment p
		JOIN dbo.charge c ON c.charge_id = p.charge_id AND c.is_deleted = 0
		JOIN dbo.register_bill r ON r.register_id = c.register_id AND r.is_deleted = 0
		WHERE p.status = 'SUCCEEDED' AND p.direction = 'REFUND'
		  AND p.paid_at >= @start_at AND p.paid_at < @end_at
		  AND ` + chargeScope + `
	),
	(
		SELECT COUNT_BIG(1)
		FROM dbo.payment p
		JOIN dbo.charge c ON c.charge_id = p.charge_id AND c.is_deleted = 0
		JOIN dbo.register_bill r ON r.register_id = c.register_id AND r.is_deleted = 0
		WHERE p.status = 'SUCCEEDED' AND p.direction = 'CHARGE'
		  AND p.paid_at >= @start_at AND p.paid_at < @end_at
		  AND ` + chargeScope + `
	),
	(
		SELECT COUNT_BIG(1)
		FROM dbo.payment p
		JOIN dbo.charge c ON c.charge_id = p.charge_id AND c.is_deleted = 0
		JOIN dbo.register_bill r ON r.register_id = c.register_id AND r.is_deleted = 0
		WHERE p.status = 'SUCCEEDED' AND p.direction = 'REFUND'
		  AND p.paid_at >= @start_at AND p.paid_at < @end_at
		  AND ` + chargeScope + `
	),
	(
		SELECT COUNT_BIG(1)
		FROM dbo.register_bill r
		WHERE r.is_deleted = 0 AND r.status <> 'BACK'
		  AND r.registered_at >= @start_at AND r.registered_at < @end_at
		  AND ` + encounterScope + `
	),
	(
		SELECT COUNT_BIG(1)
		FROM dbo.charge c
		JOIN dbo.register_bill r ON r.register_id = c.register_id AND r.is_deleted = 0
		WHERE c.is_deleted = 0 AND c.status = 'NORMAL' AND c.paid_at IS NULL
		  AND ` + chargeScope + `
	),
	(
		SELECT COUNT_BIG(1)
		FROM dbo.charge c
		JOIN dbo.register_bill r ON r.register_id = c.register_id AND r.is_deleted = 0
		WHERE c.is_deleted = 0 AND c.status = 'NORMAL' AND c.paid_at IS NOT NULL
		  AND ` + dispenseScope + `
		  AND EXISTS (
			SELECT 1
			FROM dbo.charge_item ci
			LEFT JOIN dbo.order_item oi ON oi.order_item_id = ci.order_item_id
			LEFT JOIN dbo.catalog_item catalog ON catalog.item_no = ci.item_no
			OUTER APPLY (
				SELECT SUM(di.qty_dispensed - di.qty_returned) AS net_dispensed
				FROM dbo.dispense_item di
				JOIN dbo.dispense d ON d.dispense_id = di.dispense_id
					AND d.is_deleted = 0 AND d.charge_id = c.charge_id
				WHERE di.charge_item_id = ci.charge_item_id
			) dispensed
			WHERE ci.charge_id = c.charge_id
			  AND COALESCE(oi.item_type, catalog.item_type) = 'DRUG'
			  AND ci.qty > COALESCE(dispensed.net_dispensed, 0)
		  )
	)`

	err := s.db.QueryRowContext(ctx, q, args...).Scan(
		&result.RevenueCents,
		&result.RefundCents,
		&result.ChargeCount,
		&result.RefundCount,
		&result.EncounterCount,
		&result.PendingChargeCount,
		&result.PendingDispenseCount,
	)
	if err != nil {
		return port.DashboardMetrics{}, err
	}
	result.NetRevenueCents = result.RevenueCents - result.RefundCents
	return result, nil
}

func dashboardScope(column, parameterPrefix string, departmentIDs []string) (string, []any) {
	if len(departmentIDs) == 0 {
		return "1 = 0", nil
	}
	placeholders := make([]string, len(departmentIDs))
	args := make([]any, len(departmentIDs))
	for i, id := range departmentIDs {
		name := fmt.Sprintf("%s_%d", parameterPrefix, i)
		placeholders[i] = "@" + name
		args[i] = sql.Named(name, id)
	}
	return column + " IN (" + joinComma(placeholders) + ")", args
}

func joinComma(values []string) string {
	if len(values) == 0 {
		return ""
	}
	joined := values[0]
	for _, value := range values[1:] {
		joined += ", " + value
	}
	return joined
}
