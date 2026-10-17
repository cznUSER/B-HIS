package httpapi

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestInventoryLedgerHTTPFiltersAndPages(t *testing.T) {
	ts, st, issuer := newOrderHTTP(t)
	systemToken := bearerFor(t, issuer, seed.DemoUser())
	doctor := domain.User{
		ID: "00000000-0000-4000-8000-0000000000d2", Username: "doctor-ledger", DisplayName: "门诊医生",
		Roles: []domain.Role{domain.RoleDoctor}, Status: domain.StatusActive,
	}
	st.Add(doctor)
	doctorToken := bearerFor(t, issuer, doctor)

	for _, row := range []struct {
		id    string
		delta int64
		at    time.Time
	}{
		{id: "ledger-before-range", delta: 9000, at: time.Date(2026, 10, 14, 15, 59, 59, 0, time.UTC)},
		{id: "ledger-first", delta: 1000, at: time.Date(2026, 10, 14, 17, 0, 0, 0, time.UTC)},
		{id: "ledger-second", delta: -500, at: time.Date(2026, 10, 14, 18, 0, 0, 0, time.UTC)},
	} {
		if _, err := st.AdjustInventory(context.Background(), "sku-drug-a", row.delta, row.id, seed.DemoUserID, "", row.id, row.at); err != nil {
			t.Fatalf("seed ledger %s: %v", row.id, err)
		}
	}

	path := ts.URL + "/api/v1/inventory/ledger?skuCode=DRUG-A&bizType=ADJUST&from=2026-10-15&to=2026-10-15&limit=1"
	if code, body := getAuth(t, path, ""); code != 401 {
		t.Fatalf("anonymous %d %s", code, body)
	}
	if code, body := getAuth(t, path, doctorToken); code != 403 {
		t.Fatalf("doctor %d %s", code, body)
	}

	code, body := getAuth(t, path, systemToken)
	if code != 200 {
		t.Fatalf("page 1 %d %s", code, body)
	}
	first := decodeInventoryLedgerPage(t, body)
	if len(first.Items) != 1 || first.Items[0].LedgerID != "ledger-second" || first.NextCursor == "" {
		t.Fatalf("page 1 %+v", first)
	}
	if first.Items[0].SkuCode != "DRUG-A" || first.Items[0].BizType != domain.LedgerBizAdjust || first.Items[0].QtyDelta != -0.5 || first.Items[0].QtyAfter != 109.5 {
		t.Fatalf("latest row %+v", first.Items[0])
	}

	page2URL := path + "&cursor=" + url.QueryEscape(first.NextCursor)
	code, body = getAuth(t, page2URL, systemToken)
	if code != 200 {
		t.Fatalf("page 2 %d %s", code, body)
	}
	second := decodeInventoryLedgerPage(t, body)
	if len(second.Items) != 1 || second.Items[0].LedgerID != "ledger-first" || second.NextCursor != "" {
		t.Fatalf("page 2 %+v", second)
	}
	if second.Items[0].QtyAfter != 110 {
		t.Fatalf("earlier balance %+v", second.Items[0])
	}

	code, body = getAuth(t, ts.URL+"/api/v1/inventory/ledger?from=2026-13-01", systemToken)
	if code != 400 || decodeErr(t, body).Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("invalid date %d %s", code, body)
	}
}

type inventoryLedgerPageJSON struct {
	Items      []inventoryLedgerJSON `json:"items"`
	NextCursor string                `json:"nextCursor"`
}

type inventoryLedgerJSON struct {
	LedgerID     string  `json:"ledgerId"`
	SkuCode      string  `json:"skuCode"`
	BizType      string  `json:"bizType"`
	QtyDelta     float64 `json:"qtyDelta"`
	QtyAfter     float64 `json:"qtyAfter"`
	OperatorName string  `json:"operatorName"`
}

func decodeInventoryLedgerPage(t *testing.T, body []byte) inventoryLedgerPageJSON {
	t.Helper()
	var page inventoryLedgerPageJSON
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("decode inventory ledger page: %v %s", err, body)
	}
	return page
}
