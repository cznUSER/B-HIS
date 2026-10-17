package mssql

import (
	"errors"
	"testing"

	"github.com/cznUSER/B-HIS/internal/port"
)

func TestClassifyWriteErrorMapsRoleScopePrimaryKeyConflict(t *testing.T) {
	err := classifyWriteError(errors.New("Violation of PRIMARY KEY constraint 'pk_user_role_scope'. Cannot insert duplicate key."))
	if !errors.Is(err, port.ErrDuplicateUserRoleScope) {
		t.Fatalf("classified error = %v, want ErrDuplicateUserRoleScope", err)
	}
}
