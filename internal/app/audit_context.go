package app

import (
	"context"

	"github.com/cznUSER/B-HIS/internal/buildinfo"
	"github.com/cznUSER/B-HIS/internal/domain"
)

func (s *Service) auditDepartmentID(ctx context.Context, resourceType, resourceID string) string {
	if resourceID == "" {
		return ""
	}
	switch resourceType {
	case "Register":
		if s.Registers != nil {
			if row, err := s.Registers.GetRegister(ctx, resourceID); err == nil {
				return row.DepartmentID
			}
		}
	case "Diagnose":
		if s.Diagnoses != nil && s.Registers != nil {
			if row, err := s.Diagnoses.GetDiagnose(ctx, resourceID); err == nil {
				if reg, err := s.Registers.GetRegister(ctx, row.RegisterID); err == nil {
					return reg.DepartmentID
				}
			}
		}
	case "Order":
		if s.Orders != nil {
			if row, err := s.Orders.GetOrder(ctx, resourceID); err == nil {
				return row.DepartmentID
			}
		}
	case "Charge":
		if s.Charges != nil {
			if row, err := s.Charges.GetCharge(ctx, resourceID); err == nil {
				if departmentID := s.auditOrderOrRegisterDepartment(ctx, row.OrderID, row.RegisterID); departmentID != "" {
					return departmentID
				}
			}
		}
	case "Dispense":
		if s.Dispenses != nil {
			if row, err := s.Dispenses.GetDispense(ctx, resourceID); err == nil {
				if row.DeptID != "" {
					return row.DeptID
				}
				if departmentID := s.auditOrderOrRegisterDepartment(ctx, row.OrderID, row.RegisterID); departmentID != "" {
					return departmentID
				}
			}
		}
	}
	return ""
}

func (s *Service) auditOrderOrRegisterDepartment(ctx context.Context, orderID, registerID string) string {
	if orderID != "" && s.Orders != nil {
		if row, err := s.Orders.GetOrder(ctx, orderID); err == nil && row.DepartmentID != "" {
			return row.DepartmentID
		}
	}
	if registerID != "" && s.Registers != nil {
		if row, err := s.Registers.GetRegister(ctx, registerID); err == nil {
			return row.DepartmentID
		}
	}
	return ""
}

func (s *Service) enrichBusinessAudit(ctx context.Context, event domain.AuditEvent) domain.AuditEvent {
	if event.DepartmentID == "" {
		event.DepartmentID = s.auditDepartmentID(ctx, event.ResourceType, event.ResourceID)
	}
	if event.BuildVersion == "" {
		event.BuildVersion = buildinfo.Version
	}
	return event
}
