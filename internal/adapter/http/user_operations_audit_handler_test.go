package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/app"
	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestListUserOperationsHandlerReturnsHERPProjection(t *testing.T) {
	store := memory.New()
	if err := store.Append(context.Background(), domain.AuditEvent{
		ID: "audit-1", OccurredAt: time.Date(2026, 10, 13, 15, 59, 59, 0, time.UTC),
		ActorID: seed.DemoUserID, Action: domain.ActionPatientCreate, Outcome: domain.OutcomeSuccess,
		RequestID: "request-1", ClientIP: "127.0.0.1",
	}); err != nil {
		t.Fatal(err)
	}
	user, err := store.FindByID(context.Background(), seed.DemoUserID)
	if err != nil {
		t.Fatal(err)
	}
	issuer, err := auth.NewIssuer("test-secret-long-enough-for-audit", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, _, _, err := issuer.Issue(*user, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	svc := &app.Service{Users: store, Revocations: store, Audit: store, Master: store, Tokens: issuer}
	h := &Handler{svc: svc}
	req := httptest.NewRequest("GET", "/api/v1/audit/user-operations?moduleId=module-outpatient&begin=2026-10-13&end=2026-10-13", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	h.listUserOperations(response, req)
	if response.Code != 200 {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	var got struct {
		Items      []map[string]any `json:"items"`
		NextCursor string           `json:"nextCursor"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0]["moduleId"] != "module-outpatient" || got.Items[0]["moduleName"] != "门诊业务" || got.Items[0]["actorUserId"] != seed.DemoUserID || got.Items[0]["requestId"] != "request-1" {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
	if _, exists := got.Items[0]["id"]; exists {
		t.Fatalf("internal cursor ID leaked: %s", response.Body.String())
	}
	if got.Items[0]["clientMac"] != "" || got.Items[0]["buildVersion"] != "" {
		t.Fatalf("unavailable values should remain empty: %s", response.Body.String())
	}
}
