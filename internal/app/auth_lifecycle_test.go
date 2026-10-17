package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func lifecycleService(t *testing.T, store *memory.Store, now time.Time) *Service {
	t.Helper()
	issuer, err := auth.NewIssuer("lifecycle-test-secret-at-least-32-bytes", 8*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return &Service{
		Users: store, Revocations: store, Audit: store, Master: store, Tokens: issuer,
		Now: func() time.Time { return now },
	}
}

func TestLoginLocksAccountAfterThreeFailures(t *testing.T) {
	store := memory.New()
	user := domain.User{ID: "lock-test", Username: "lock-test", DisplayName: "锁定测试", PasswordHash: seed.DemoPasswordHash, Status: domain.StatusActive}
	store.Add(user)
	svc := lifecycleService(t, store, time.Date(2026, 10, 14, 2, 0, 0, 0, time.UTC))
	for i := 0; i < 3; i++ {
		if _, err := svc.Login(context.Background(), user.Username, "wrong", "req", "127.0.0.1"); err == nil {
			t.Fatal("wrong password must fail")
		}
	}
	if _, err := svc.Login(context.Background(), user.Username, seed.DemoPassword, "req", "127.0.0.1"); err == nil {
		t.Fatal("valid password must remain blocked during account lock")
	} else {
		var ae *Error
		if !errors.As(err, &ae) || ae.Code != CodeAccountLocked || ae.Status != 423 {
			t.Fatalf("want locked response, got %v", err)
		}
	}
	got, err := store.FindByID(context.Background(), user.ID)
	if err != nil || !got.AccountLocked || got.FailedLoginCount != 3 || !got.LockUntil.After(svc.now()) {
		t.Fatalf("lock state: err=%v user=%+v", err, got)
	}
}

func TestLoginRejectsStoppedAndManuallyLockedUsers(t *testing.T) {
	store := memory.New()
	now := time.Date(2026, 10, 14, 2, 0, 0, 0, time.UTC)
	stopped := domain.User{ID: "stopped-test", Username: "stopped-test", DisplayName: "停用", PasswordHash: seed.DemoPasswordHash, Status: domain.StatusActive, AccountStopped: true}
	locked := domain.User{ID: "manual-lock-test", Username: "manual-lock-test", DisplayName: "锁定", PasswordHash: seed.DemoPasswordHash, Status: domain.StatusActive, AccountLocked: true, LockUntil: now.Add(time.Hour)}
	store.Add(stopped)
	store.Add(locked)
	svc := lifecycleService(t, store, now)
	if _, err := svc.Login(context.Background(), stopped.Username, seed.DemoPassword, "req", "127.0.0.1"); err == nil {
		t.Fatal("stopped account must not log in")
	} else {
		var ae *Error
		if !errors.As(err, &ae) || ae.Code != CodeAccountDisabled {
			t.Fatalf("stopped response: %v", err)
		}
	}
	if _, err := svc.Login(context.Background(), locked.Username, seed.DemoPassword, "req", "127.0.0.1"); err == nil {
		t.Fatal("locked account must not log in")
	} else {
		var ae *Error
		if !errors.As(err, &ae) || ae.Code != CodeAccountLocked {
			t.Fatalf("locked response: %v", err)
		}
	}
}

func TestAdminResetRequiresPasswordChangeAndInvalidatesOldToken(t *testing.T) {
	store := memory.New()
	now := time.Date(2026, 10, 14, 2, 0, 0, 0, time.UTC)
	user := domain.User{ID: "reset-target", Username: "reset-target", DisplayName: "重置目标", PasswordHash: seed.DemoPasswordHash, Status: domain.StatusActive}
	store.Add(user)
	svc := lifecycleService(t, store, now)
	oldToken, _, _, err := svc.Tokens.Issue(user, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetUserPassword(context.Background(), systemClaims(), user.ID, "req-reset", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(context.Background(), oldToken); err == nil {
		t.Fatal("password reset must invalidate previously issued tokens")
	} else {
		var ae *Error
		if !errors.As(err, &ae) || ae.Code != CodeTokenInvalid {
			t.Fatalf("old token error: %v", err)
		}
	}
	result, err := svc.Login(context.Background(), user.Username, seed.DemoPassword, "req-login", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.User.MustChangePassword {
		t.Fatalf("reset account should be forced to change password: %+v", result.User)
	}
	if _, err := svc.Me(context.Background(), mustClaims(t, svc, result.AccessToken)); err == nil {
		t.Fatal("forced-change token must be blocked from ordinary authenticated operations")
	} else {
		var ae *Error
		if !errors.As(err, &ae) || ae.Code != CodePasswordChangeRequired {
			t.Fatalf("force-change response: %v", err)
		}
	}
}

func TestLoginIncludesProtectedAccountFlagsInSessionUser(t *testing.T) {
	store := memory.New()
	user := domain.User{
		ID: "super-test", Username: "super-test", DisplayName: "超级账号",
		PasswordHash: seed.DemoPasswordHash, Status: domain.StatusActive,
		System: true, Super: true, Developer: true,
	}
	store.Add(user)
	svc := lifecycleService(t, store, time.Date(2026, 10, 14, 2, 0, 0, 0, time.UTC))

	result, err := svc.Login(context.Background(), user.Username, seed.DemoPassword, "req-login", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.User.System || !result.User.Super || !result.User.Developer {
		t.Fatalf("session user must include protected account flags: %+v", result.User)
	}
}

func mustClaims(t *testing.T, svc *Service, raw string) *auth.Claims {
	t.Helper()
	claims, err := svc.Authenticate(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	return claims
}
