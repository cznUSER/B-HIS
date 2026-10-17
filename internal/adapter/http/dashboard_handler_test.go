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
	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func newDashboardHTTP(t *testing.T) (*httptest.Server, *memory.Store, *auth.Issuer) {
	t.Helper()
	log, err := logging.New(t.TempDir(), "info")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	st := memory.New()
	ttl := 30 * 24 * time.Hour
	issuer, err := auth.NewIssuer("dashboard-test-secret-must-be-long-32b", ttl)
	if err != nil {
		t.Fatal(err)
	}
	svc := &app.Service{
		Users: st, Revocations: st, Audit: st, Registers: st,
		Tokens: issuer, Master: st, Dashboard: st, Log: log, TTL: ttl,
		Now: func() time.Time { return time.Date(2026, 10, 14, 4, 0, 0, 0, time.UTC) },
	}
	ts := httptest.NewServer(New(svc, log, nil, "memory", time.Minute))
	t.Cleanup(ts.Close)
	return ts, st, issuer
}

func TestDashboardSummaryHTTPUsesCurrentAuthorizationAndShanghaiPeriods(t *testing.T) {
	ts, st, issuer := newDashboardHTTP(t)
	endpoint := ts.URL + "/api/v1/workbench/summary"
	if code, body := getAuth(t, endpoint+"?period=TODAY", ""); code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
		t.Fatalf("unauthenticated summary: %d %s", code, body)
	}

	at := time.Date(2026, 10, 14, 4, 30, 0, 0, time.UTC)
	ctx := context.Background()
	if err := st.CreateRegister(ctx, domain.Register{
		RegisterID: "dashboard-encounter", RegisterNo: "dashboard-reg-1", DepartmentID: "dept-1",
		RegisteredAt: at, Status: domain.RegisterStatusNormal,
	}); err != nil {
		t.Fatal(err)
	}
	charge := domain.Charge{ChargeID: "dashboard-charge", SquareNo: "dashboard-square-1", RegisterID: "dashboard-encounter", TotalFeeCents: 2500, SquareFeeCents: 2500, Status: domain.ChargeStatusNormal, CreatedAt: at}
	if err := st.CreateCharge(ctx, charge); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.PayCharge(ctx, charge.ChargeID, domain.Payment{
		PaymentID: "dashboard-payment", PaymentNo: "dashboard-payment-1", ChargeID: charge.ChargeID,
		Direction: domain.PaymentDirectionCharge, PayType: domain.PayTypeCash, AmountCents: 2500,
		Status: domain.PaymentStatusSucceeded, PaidAt: at,
	}); err != nil {
		t.Fatal(err)
	}
	tok := bearerFor(t, issuer, seed.DemoUser())
	if code, body := getAuth(t, endpoint+"?period=TODAY", tok); code != 200 {
		t.Fatalf("summary: %d %s", code, body)
	} else {
		var got struct {
			Period         string `json:"period"`
			RangeStart     string `json:"rangeStart"`
			RangeEnd       string `json:"rangeEnd"`
			RevenueCents   int64  `json:"revenueCents"`
			EncounterCount int    `json:"encounterCount"`
			AverageTicket  int64  `json:"averageTicketCents"`
			PendingCharges int    `json:"pendingChargeCount"`
			Available      struct {
				Charges    bool `json:"charges"`
				Encounters bool `json:"encounters"`
				Dispenses  bool `json:"dispenses"`
			} `json:"available"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		if got.Period != "TODAY" || got.RangeStart != "2026-10-14T00:00:00+08:00" || got.RangeEnd != "2026-10-15T00:00:00+08:00" || got.RevenueCents != 2500 || got.EncounterCount != 1 || got.AverageTicket != 2500 || got.PendingCharges != 0 || !got.Available.Charges || !got.Available.Encounters || !got.Available.Dispenses {
			t.Fatalf("unexpected summary: %+v", got)
		}
	}
	if code, body := getAuth(t, endpoint+"?period=INVALID", tok); code != 400 || decodeErr(t, body).Error.Code != app.CodeValidation {
		t.Fatalf("invalid period: %d %s", code, body)
	}

	noAccess := domain.User{ID: "no-dashboard", Username: "no-dashboard", DisplayName: "No Access", Status: domain.StatusActive}
	st.Add(noAccess)
	noAccessToken := bearerFor(t, issuer, noAccess)
	if code, body := getAuth(t, endpoint+"?period=TODAY", noAccessToken); code != 403 || decodeErr(t, body).Error.Code != "FORBIDDEN" {
		t.Fatalf("user without outpatient metrics permission: %d %s", code, body)
	}
	registrar := domain.User{ID: "registrar-dashboard", Username: "registrar-dashboard", DisplayName: "Registrar", Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive}
	st.Add(registrar)
	registrarToken := bearerFor(t, issuer, registrar)
	code, body := getAuth(t, endpoint+"?period=TODAY", registrarToken)
	if code != 200 {
		t.Fatalf("registrar summary: %d %s", code, body)
	}
	var scoped struct {
		Available struct {
			Charges    bool `json:"charges"`
			Encounters bool `json:"encounters"`
			Dispenses  bool `json:"dispenses"`
		} `json:"available"`
	}
	if err := json.Unmarshal(body, &scoped); err != nil {
		t.Fatal(err)
	}
	if scoped.Available.Charges || !scoped.Available.Encounters || scoped.Available.Dispenses {
		t.Fatalf("permission groups leaked: %+v", scoped.Available)
	}
}
