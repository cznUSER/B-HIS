package app

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
	"github.com/golang-jwt/jwt/v5"
)

func TestListUserOperationsRequiresAuditPermission(t *testing.T) {
	store := memory.New()
	user := domain.User{ID: "user-auditor", Username: "auditor", DisplayName: "审计员", Roles: []domain.Role{domain.RoleDoctor}, Status: domain.StatusActive}
	store.Add(user)
	svc := &Service{Users: store, Master: store, Audit: store, Now: func() time.Time { return time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC) }}
	_, err := svc.ListUserOperations(context.Background(), &auth.Claims{RegisteredClaims: jwtSubject(user.ID)}, UserOperationFilter{})
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Status != 403 {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestListUserOperationsUsesInclusiveDatesAndStableCursor(t *testing.T) {
	store := memory.New()
	base := time.Date(2026, 10, 13, 15, 59, 57, 0, time.UTC) // Shanghai 2026-10-13 23:59:57.
	for i, event := range []domain.AuditEvent{
		{ID: "a", OccurredAt: base, ActorID: seed.DemoUserID, ActorUsername: "system", ModuleID: "module-outpatient", Action: domain.ActionPatientCreate, Outcome: domain.OutcomeSuccess, RequestID: "req-a"},
		{ID: "b", OccurredAt: base.Add(time.Second), ActorID: seed.DemoUserID, ActorUsername: "system", ModuleID: "module-master", Action: domain.ActionUserUpdate, Outcome: domain.OutcomeFail, ErrorCode: "VALIDATION_ERROR", RequestID: "req-b"},
		{ID: "c", OccurredAt: base.Add(2 * time.Second), ActorID: seed.DemoUserID, ActorUsername: "system", ModuleID: "module-outpatient", Action: domain.ActionPatientCreate, Outcome: domain.OutcomeSuccess, RequestID: "req-c"},
	} {
		if event.ID == "" {
			t.Fatalf("test event %d missing id", i)
		}
		if err := store.Append(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	svc := &Service{Users: store, Master: store, Audit: store, Now: func() time.Time { return time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC) }}
	claims := &auth.Claims{RegisteredClaims: jwtSubject(seed.DemoUserID)}
	first, err := svc.ListUserOperations(context.Background(), claims, UserOperationFilter{ModuleID: "module-outpatient", Begin: "2026-10-13", End: "2026-10-13", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].RequestID != "req-c" || first.NextCursor == "" {
		t.Fatalf("unexpected first page: %+v", first)
	}
	second, err := svc.ListUserOperations(context.Background(), claims, UserOperationFilter{ModuleID: "module-outpatient", Begin: "2026-10-13", End: "2026-10-13", Cursor: first.NextCursor, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].RequestID != "req-a" || second.NextCursor != "" {
		t.Fatalf("unexpected second page: %+v", second)
	}
}

func TestUserOperationCursorRejectsTampering(t *testing.T) {
	store := memory.New()
	svc := &Service{Users: store, Master: store, Audit: store}
	_, err := svc.ListUserOperations(context.Background(), &auth.Claims{RegisteredClaims: jwtSubject(seed.DemoUserID)}, UserOperationFilter{Cursor: base64.RawURLEncoding.EncodeToString([]byte(`{"id":"x"}`))})
	var appErr *Error
	if !errors.As(err, &appErr) || appErr.Code != CodeValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestMemoryUserOperationQueryFiltersAndOrders(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	for _, event := range []domain.AuditEvent{
		{ID: "1", OccurredAt: time.Unix(1, 0), ModuleID: "module-master", ActorID: "u1", Action: "user.update", Outcome: domain.OutcomeSuccess},
		{ID: "2", OccurredAt: time.Unix(2, 0), ModuleID: "module-master", ActorID: "u2", Action: "user.update", Outcome: domain.OutcomeSuccess},
		{ID: "3", OccurredAt: time.Unix(3, 0), ModuleID: "module-outpatient", ActorID: "u1", Action: "patient.create", Outcome: domain.OutcomeSuccess},
	} {
		if err := store.Append(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := store.ListUserOperations(ctx, port.UserOperationQuery{ModuleID: "module-master", UserID: "u1", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "1" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func jwtSubject(id string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{Subject: id}
}
