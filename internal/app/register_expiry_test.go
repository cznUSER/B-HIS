package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func TestCheckInRegisterHonorsOptionalExpiry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	svc := &Service{
		Users: store, Registers: store, Audit: store,
		RegisterExpireHours: 2,
		Now:                 func() time.Time { return now },
	}
	reg := domain.Register{
		RegisterID: "reg-expired", RegisterNo: "R-EXPIRED", DepartmentID: "dept-1",
		RegisteredAt: now.Add(-2 * time.Hour), Status: domain.RegisterStatusNormal,
	}
	if err := store.CreateRegister(ctx, reg); err != nil {
		t.Fatal(err)
	}
	_, err := svc.CheckInRegister(ctx, systemClaims(), reg.RegisterID, "req-expired", "127.0.0.1")
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != CodeRegisterExpired || appErr.Status != 409 {
		t.Fatalf("expected REGISTER_EXPIRED, got %v", err)
	}
	got, err := store.GetRegister(ctx, reg.RegisterID)
	if err != nil || got.Status != domain.RegisterStatusNormal {
		t.Fatalf("expired check-in changed register: %#v %v", got, err)
	}
}

func TestRegisterExpiryDisabledAndOpenRegisterGuard(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	reg := domain.Register{
		RegisterID: "reg-open", RegisterNo: "R-OPEN", DepartmentID: "dept-1",
		RegisteredAt: now.Add(-24 * time.Hour), Status: domain.RegisterStatusNormal,
	}
	if err := store.CreateRegister(ctx, reg); err != nil {
		t.Fatal(err)
	}
	noExpiry := &Service{Users: store, Registers: store, Audit: store, Now: func() time.Time { return now }}
	if _, err := noExpiry.CheckInRegister(ctx, systemClaims(), reg.RegisterID, "req-disabled", "127.0.0.1"); err != nil {
		t.Fatalf("zero expiry should preserve behavior: %v", err)
	}

	reg.Status = domain.RegisterStatusNormal
	reg.RegisteredAt = now.Add(-3 * time.Hour)
	if err := store.CreateRegister(ctx, domain.Register{RegisterID: "reg-open-expired", RegisterNo: "R-OPEN-EXPIRED", DepartmentID: "dept-1", RegisteredAt: reg.RegisteredAt, Status: reg.Status}); err != nil {
		t.Fatal(err)
	}
	withExpiry := &Service{Users: store, Registers: store, RegisterExpireHours: 2, Now: func() time.Time { return now }}
	if _, err := withExpiry.requireOpenRegister(ctx, "reg-open-expired"); err != nil {
		var appErr *Error
		if !errors.As(err, &appErr) || appErr.Code != CodeRegisterExpired {
			t.Fatalf("expected open register expiry guard, got %v", err)
		}
	} else {
		t.Fatal("expected open register expiry guard")
	}
}
