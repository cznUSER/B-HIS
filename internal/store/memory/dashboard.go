package memory

import (
	"context"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.DashboardStore = (*Store)(nil)

func (s *Store) SummarizeDashboard(_ context.Context, query port.DashboardQuery) (port.DashboardMetrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	encounterDepartments := departmentSet(query.EncounterDepartmentIDs)
	chargeDepartments := departmentSet(query.ChargeDepartmentIDs)
	dispenseDepartments := departmentSet(query.DispenseDepartmentIDs)
	var result port.DashboardMetrics

	for _, register := range s.registers {
		if register.IsDeleted || register.Status == domain.RegisterStatusBack || !encounterDepartments[register.DepartmentID] {
			continue
		}
		if inDashboardWindow(register.RegisteredAt, query.StartAt, query.EndAt) {
			result.EncounterCount++
		}
	}

	for _, payment := range s.payments {
		if payment.Status != domain.PaymentStatusSucceeded || !inDashboardWindow(payment.PaidAt, query.StartAt, query.EndAt) {
			continue
		}
		charge, ok := s.charges[payment.ChargeID]
		if !ok || charge.IsDeleted || !chargeDepartments[registerDepartment(s.registers, charge.RegisterID)] {
			continue
		}
		switch payment.Direction {
		case domain.PaymentDirectionCharge:
			result.RevenueCents += payment.AmountCents
			result.ChargeCount++
		case domain.PaymentDirectionRefund:
			result.RefundCents += payment.AmountCents
			result.RefundCount++
		}
	}
	result.NetRevenueCents = result.RevenueCents - result.RefundCents

	for _, charge := range s.charges {
		if charge.IsDeleted {
			continue
		}
		departmentID := registerDepartment(s.registers, charge.RegisterID)
		if charge.Status == domain.ChargeStatusNormal && charge.PaidAt.IsZero() && chargeDepartments[departmentID] {
			result.PendingChargeCount++
		}
		if charge.Status == domain.ChargeStatusNormal && !charge.PaidAt.IsZero() && dispenseDepartments[departmentID] && chargeNeedsDispense(charge, s.catalog, s.dispenses) {
			result.PendingDispenseCount++
		}
	}
	return result, nil
}

func departmentSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func registerDepartment(registers map[string]domain.Register, registerID string) string {
	return registers[registerID].DepartmentID
}

func inDashboardWindow(value, start, end time.Time) bool {
	return !value.IsZero() && !value.Before(start) && value.Before(end)
}

func chargeNeedsDispense(charge domain.Charge, catalog map[string]domain.CatalogItem, dispenses map[string]domain.Dispense) bool {
	for _, item := range charge.Items {
		itemType := item.ItemType
		if itemType == "" {
			itemType = catalog[item.ItemNo].ItemType
		}
		if itemType != domain.ItemTypeDrug || item.QtyMilli <= 0 {
			continue
		}
		var netMilli int64
		for _, dispense := range dispenses {
			if dispense.IsDeleted || dispense.ChargeID != charge.ChargeID {
				continue
			}
			for _, dispensedItem := range dispense.Items {
				if dispensedItem.ChargeItemID == item.ChargeItemID {
					netMilli += dispensedItem.QtyDispensedMilli - dispensedItem.QtyReturnedMilli
				}
			}
		}
		if netMilli < item.QtyMilli {
			return true
		}
	}
	return false
}
