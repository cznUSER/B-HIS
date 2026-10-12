package mssql

import (
	"errors"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"

	"github.com/cznUSER/B-HIS/internal/port"
)

// classifyWriteError maps 002_patient unique-index violations to port errors.
// 2601/2627 are SQL Server duplicate-key numbers. The index name decides the code.
func classifyWriteError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	dup := false
	var msErr mssql.Error
	if errors.As(err, &msErr) {
		if msErr.Number == 2601 || msErr.Number == 2627 {
			dup = true
		}
		if msErr.Message != "" {
			msg = msErr.Message + " " + msg
		}
	}
	if strings.Contains(msg, "ux_patient_") {
		dup = true
	}
	if !dup {
		return err
	}
	switch {
	case strings.Contains(msg, "ux_patient_card_no"):
		return port.ErrDuplicateCardNo
	case strings.Contains(msg, "ux_patient_id_number"):
		return port.ErrDuplicateIDNumber
	case strings.Contains(msg, "ux_patient_hosp_pk"):
		return port.ErrDuplicateBusinessNo
	case strings.Contains(msg, "ux_register_no"), strings.Contains(msg, "pk_register_bill"):
		return port.ErrDuplicateBusinessNo
	case strings.Contains(msg, "ux_order_no"), strings.Contains(msg, "pk_order_sheet"):
		return port.ErrDuplicateBusinessNo
	case strings.Contains(msg, "ux_charge_square_no"), strings.Contains(msg, "pk_charge"),
		strings.Contains(msg, "ux_payment_no"), strings.Contains(msg, "pk_payment"):
		return port.ErrDuplicateBusinessNo
	default:
		return err
	}
}
