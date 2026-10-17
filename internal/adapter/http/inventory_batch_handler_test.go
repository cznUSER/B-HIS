package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/app"
	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/store/memory"
)

func TestInventoryBatchHTTPReadPermissionAndValidation(t *testing.T) {
	log, err := logging.New(t.TempDir(), "info")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	st := memory.New()
	for _, row := range []domain.InventoryBatch{
		{BatchID: "P2-6A-EXPIRED", ItemNo: "P2-6A-DRUG", ExpireDate: ptrBatchDate(2026, 10, 14), QtyOnHandMilli: 3000},
		{BatchID: "P2-6A-EXPIRING", ItemNo: "P2-6A-DRUG", ExpireDate: ptrBatchDate(2026, 10, 30), QtyOnHandMilli: 5000},
		{BatchID: "P2-6A-VALID", ItemNo: "P2-6A-DRUG", ExpireDate: ptrBatchDate(2027, 1, 14), QtyOnHandMilli: 10000},
		{BatchID: "P2-6A-UNKNOWN", ItemNo: "P2-6A-DRUG", QtyOnHandMilli: 7000},
	} {
		st.AddInventoryBatch(row)
	}
	issuer, err := auth.NewIssuer("test-secret-must-be-long-enough-32b!!", 8*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc := &app.Service{Users: st, Revocations: st, Audit: st, Master: st, Dispenses: st, InventoryBatches: st, Tokens: issuer, Log: log, TTL: 8 * time.Hour, Now: func() time.Time { return time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC) }}
	ts := httptest.NewServer(New(svc, log, nil, "memory", time.Minute))
	defer ts.Close()
	system := domain.User{ID: "batch-system", Username: "batch-system", DisplayName: "系统", Roles: []domain.Role{domain.RoleSystem}, Status: domain.StatusActive}
	pharm := domain.User{ID: "batch-pharm", Username: "batch-pharm", DisplayName: "药房", Roles: []domain.Role{domain.RolePharmacist}, Status: domain.StatusActive}
	doc := domain.User{ID: "batch-doc", Username: "batch-doc", DisplayName: "医生", Roles: []domain.Role{domain.RoleDoctor}, Status: domain.StatusActive}
	st.Add(system)
	st.Add(pharm)
	st.Add(doc)
	systemTok := bearerForBatchHTTP(t, issuer, system)
	pharmTok := bearerForBatchHTTP(t, issuer, pharm)
	docTok := bearerForBatchHTTP(t, issuer, doc)
	path := ts.URL + "/api/v1/inventory/batches?asOfDate=2026-10-15&expiryWindowDays=30"
	if code, _ := getAuth(t, path, ""); code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", code)
	}
	if code, body := getAuth(t, path, docTok); code != http.StatusForbidden {
		t.Fatalf("doctor status=%d body=%s", code, body)
	}
	code, body := getAuth(t, path, pharmTok)
	if code != http.StatusOK {
		t.Fatalf("pharmacist status=%d body=%s", code, body)
	}
	if string(body) == "" || !containsBatchStatus(body, "UNKNOWN") {
		t.Fatalf("response=%s", body)
	}
	if code, body := getAuth(t, ts.URL+"/api/v1/inventory/batches?asOfDate=2026-10-15&expiryWindowDays=-1", systemTok); code != http.StatusBadRequest || decodeErr(t, body).Error.Code != app.CodeValidation {
		t.Fatalf("bad window status=%d body=%s", code, body)
	}
	if code, body := getAuth(t, ts.URL+"/api/v1/inventory/batches?asOfDate=2026-10-15&expiryWindowDays=30&expiryStatus=NOPE", systemTok); code != http.StatusBadRequest || decodeErr(t, body).Error.Code != app.CodeValidation {
		t.Fatalf("bad status=%d body=%s", code, body)
	}
}

func ptrBatchDate(y, m, d int) *time.Time {
	v := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	return &v
}

func bearerForBatchHTTP(t *testing.T, issuer *auth.Issuer, user domain.User) string {
	t.Helper()
	tok, _, _, err := issuer.Issue(user, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func containsBatchStatus(body []byte, status string) bool {
	return bytes.Contains(body, []byte(status))
}
