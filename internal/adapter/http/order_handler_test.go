package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/app"
	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func newOrderHTTP(t *testing.T) (*httptest.Server, *memory.Store, *auth.Issuer) {
	t.Helper()
	log, err := logging.New(t.TempDir(), "info")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	st := memory.New()
	ttl := 8 * time.Hour
	issuer, err := auth.NewIssuer("test-secret-must-be-long-enough-32b!!", ttl)
	if err != nil {
		t.Fatal(err)
	}
	fixed := time.Date(2026, 10, 12, 4, 0, 0, 0, time.UTC)
	svc := &app.Service{
		Users:       st,
		Revocations: st,
		Audit:       st,
		Patients:    st,
		Registers:   st,
		Orders:      st,
		Charges:     st,
		Tokens:      issuer,
		Log:         log,
		TTL:         ttl,
		Now:         func() time.Time { return fixed },
	}
	ts := httptest.NewServer(New(svc, log, nil, "memory", time.Minute))
	t.Cleanup(ts.Close)
	return ts, st, issuer
}

func TestOrderHTTP(t *testing.T) {
	ts, st, issuer := newOrderHTTP(t)
	tok := bearerFor(t, issuer, seed.DemoUser())
	base := ts.URL + "/api/v1/orders"
	patientID := mustCreatePatient(t, ts.URL, tok)
	registerID := mustRegisterID(t, ts.URL, tok, patientID)

	doc := domain.User{
		ID: "00000000-0000-4000-8000-0000000000a2", Username: "doctor", DisplayName: "门诊医生",
		Roles: []domain.Role{domain.RoleDoctor}, Status: domain.StatusActive,
	}
	regUser := domain.User{
		ID: "00000000-0000-4000-8000-0000000000a1", Username: "registrar", DisplayName: "挂号员",
		Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive,
	}
	st.Add(doc)
	st.Add(regUser)
	docTok := bearerFor(t, issuer, doc)
	regTok := bearerFor(t, issuer, regUser)

	t.Run("health version", func(t *testing.T) {
		code, body := getAuth(t, ts.URL+"/health", "")
		if code != 200 || !strings.Contains(string(body), `"version":"0.3.8-draft"`) {
			t.Fatalf("%d %s", code, body)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		code, body, _ := postJSON(t, base, orderBody(registerID, drugA(2)), nil)
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("create %d %s", code, body)
		}
		code, body = getAuth(t, base+"?registerId="+registerID, "")
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("list %d %s", code, body)
		}
		code, body = getAuth(t, base+"/missing", "")
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("get %d %s", code, body)
		}
		code, body = patchJSON(t, base+"/missing", `{}`, nil)
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("patch %d %s", code, body)
		}
		code, body, _ = postJSON(t, base+"/missing/commit", `{}`, nil)
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("commit %d %s", code, body)
		}
	})

	t.Run("registrar forbidden write but can read", func(t *testing.T) {
		code, body, _ := postJSON(t, base, orderBody(registerID, ""), authHdr(regTok))
		if code != 403 || decodeErr(t, body).Error.Code != "FORBIDDEN" {
			t.Fatalf("create %d %s", code, body)
		}
		code, body = getAuth(t, base+"?registerId="+registerID, regTok)
		if code != 200 {
			t.Fatalf("list %d %s", code, body)
		}
	})

	var created orderJSON
	t.Run("create save", func(t *testing.T) {
		code, body, _ := postJSON(t, base, orderBody(registerID, drugA(2)), authHdr(tok))
		if code != 201 {
			t.Fatalf("%d %s", code, body)
		}
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatal(err)
		}
		if _, ok := raw["success"]; ok {
			t.Fatalf("201 must be OrderDto, got %s", body)
		}
		created = decodeOrder(t, body)
		if created.OrderID == "" || created.OrderNo == "" || created.Status != "SAVE" || created.Charged {
			t.Fatalf("%+v", created)
		}
		if created.RegisterID != registerID || created.EncounterID != registerID || created.PatientID != patientID {
			t.Fatalf("%+v", created)
		}
		if created.DepartmentID != "dept-1" || created.DoctorID != "doc-1" || created.RecipeSort != "NORMAL" {
			t.Fatalf("%+v", created)
		}
		if created.DiagnoseName != nil || created.SubmittedAt != nil || created.AbatedAt != nil {
			t.Fatalf("%+v", created)
		}
		if !near(created.TotalFee, 20) || len(created.Items) != 1 {
			t.Fatalf("%+v", created)
		}
		line := created.Items[0]
		if line.ItemNo != "DRUG-A" || line.LineNo != 1 || line.Units != "盒" || !near(line.Amount, 2) || !near(line.Price, 10) || !near(line.TotalFee, 20) {
			t.Fatalf("%+v", line)
		}
		if line.OrderItemID == "" || line.Spec == nil || *line.Spec != "0.25g×24" {
			t.Fatalf("%+v", line)
		}
		if n := countOrderAudit(st, "order.create", "SUCCESS", created.OrderID); n != 1 {
			t.Fatalf("audit %d", n)
		}
	})

	t.Run("doctor can create and defaults doctor", func(t *testing.T) {
		code, body, _ := postJSON(t, base, `{"registerId":"`+registerID+`"}`, authHdr(docTok))
		if code != 201 {
			t.Fatalf("%d %s", code, body)
		}
		dto := decodeOrder(t, body)
		if dto.Status != "SAVE" || dto.DoctorID != "doc-1" || len(dto.Items) != 0 || !near(dto.TotalFee, 0) {
			t.Fatalf("%+v", dto)
		}
	})

	t.Run("query and missing detail", func(t *testing.T) {
		if created.OrderID == "" {
			t.Fatal("create did not run")
		}
		code, body := getAuth(t, base+"/"+created.OrderID, tok)
		if code != 200 {
			t.Fatalf("get %d %s", code, body)
		}
		dto := decodeOrder(t, body)
		if dto.OrderID != created.OrderID || dto.Status != "SAVE" || !near(dto.TotalFee, 20) {
			t.Fatalf("%+v", dto)
		}
		code, body = getAuth(t, base+"/no-such-order", tok)
		if code != 404 || decodeErr(t, body).Error.Code != "NOT_FOUND" {
			t.Fatalf("missing %d %s", code, body)
		}
		code, body = getAuth(t, base, tok)
		if code != 200 {
			t.Fatalf("empty %d %s", code, body)
		}
		if items := decodeOrderList(t, body); len(items) != 0 {
			t.Fatalf("unfiltered %+v", items)
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(body, &envelope); err != nil {
			t.Fatal(err)
		}
		if string(envelope["items"]) != "[]" {
			t.Fatalf("items %s", envelope["items"])
		}
		code, body = getAuth(t, base+"?registerId="+registerID, tok)
		if code != 200 || !orderListHas(decodeOrderList(t, body), created.OrderID) {
			t.Fatalf("by register %d %s", code, body)
		}
		code, body = getAuth(t, base+"?encounterId="+registerID+"&status=SAVE", tok)
		if code != 200 || !orderListHas(decodeOrderList(t, body), created.OrderID) {
			t.Fatalf("alias %d %s", code, body)
		}
		code, body = getAuth(t, base+"?orderNo="+created.OrderNo, tok)
		list := decodeOrderList(t, body)
		if code != 200 || len(list) != 1 || list[0].OrderID != created.OrderID {
			t.Fatalf("orderNo %d %s", code, body)
		}
		code, body = getAuth(t, base+"?patientId="+patientID+"&status=COMMIT", tok)
		if code != 200 || orderListHas(decodeOrderList(t, body), created.OrderID) {
			t.Fatalf("commit filter %d %s", code, body)
		}
	})

	t.Run("patch save replaces items", func(t *testing.T) {
		code, body := patchJSON(t, base+"/"+created.OrderID, `{"diagnoseName":"上感","doctorId":"doc-2","items":[`+drugB(1)+`]}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("%d %s", code, body)
		}
		dto := decodeOrder(t, body)
		if dto.Status != "SAVE" || dto.DoctorID != "doc-2" || dto.DiagnoseName == nil || *dto.DiagnoseName != "上感" {
			t.Fatalf("%+v", dto)
		}
		if len(dto.Items) != 1 || dto.Items[0].ItemNo != "DRUG-B" || !near(dto.TotalFee, 8) || !near(dto.Items[0].TotalFee, 8) {
			t.Fatalf("%+v", dto)
		}
		code, body = patchJSON(t, base+"/"+created.OrderID, `{"items":[`+drugA(2)+`]}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("restore %d %s", code, body)
		}
		dto = decodeOrder(t, body)
		if len(dto.Items) != 1 || !near(dto.TotalFee, 20) || dto.Items[0].ItemNo != "DRUG-A" {
			t.Fatalf("%+v", dto)
		}
	})

	t.Run("patch invalid amount", func(t *testing.T) {
		code, body := patchJSON(t, base+"/"+created.OrderID, `{"items":[`+drugA(0)+`]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ORDER_INVALID_AMOUNT" {
			t.Fatalf("%d %s", code, body)
		}
		code, body = getAuth(t, base+"/"+created.OrderID, tok)
		dto := decodeOrder(t, body)
		if code != 200 || !near(dto.TotalFee, 20) || dto.Items[0].ItemNo != "DRUG-A" {
			t.Fatalf("unchanged %d %+v", code, dto)
		}
	})

	t.Run("commit", func(t *testing.T) {
		code, body, _ := postJSON(t, base+"/"+created.OrderID+"/commit", `{}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("%d %s", code, body)
		}
		dto := decodeOrder(t, body)
		if dto.Status != "COMMIT" || !near(dto.TotalFee, 20) || dto.Charged || dto.SubmittedAt == nil || *dto.SubmittedAt == "" {
			t.Fatalf("%+v", dto)
		}
		if n := countOrderAudit(st, "order.commit", "SUCCESS", created.OrderID); n != 1 {
			t.Fatalf("audit %d", n)
		}
		code, body = patchJSON(t, base+"/"+created.OrderID, `{"diagnoseName":"不再改"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ORDER_NOT_EDITABLE" {
			t.Fatalf("patch %d %s", code, body)
		}
		code, body, _ = postJSON(t, base+"/"+created.OrderID+"/commit", `{}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ORDER_NOT_COMMITTABLE" {
			t.Fatalf("recommit %d %s", code, body)
		}
	})

	t.Run("submit alias", func(t *testing.T) {
		code, body, _ := postJSON(t, base, orderBody(registerID, drugA(1)), authHdr(tok))
		if code != 201 {
			t.Fatalf("create %d %s", code, body)
		}
		dto := decodeOrder(t, body)
		code, body, _ = postJSON(t, base+"/"+dto.OrderID+"/submit", `{}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("submit %d %s", code, body)
		}
		got := decodeOrder(t, body)
		if got.Status != "COMMIT" || !near(got.TotalFee, 10) || got.SubmittedAt == nil {
			t.Fatalf("%+v", got)
		}
		if n := countOrderAudit(st, "order.commit", "SUCCESS", dto.OrderID); n != 1 {
			t.Fatalf("audit %d", n)
		}
	})

	t.Run("abate uncharged", func(t *testing.T) {
		code, body, _ := postJSON(t, base, orderBody(registerID, drugA(1)), authHdr(tok))
		dto := decodeOrder(t, body)
		if code != 201 {
			t.Fatalf("create %d %s", code, body)
		}
		code, body, _ = postJSON(t, base+"/"+dto.OrderID+"/abate", `{}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("abate %d %s", code, body)
		}
		got := decodeOrder(t, body)
		if got.Status != "ABATE" || got.AbatedAt == nil || got.AbatedBy == nil || *got.AbatedBy != seed.DemoUserID {
			t.Fatalf("%+v", got)
		}
		if n := countOrderAudit(st, "order.abate", "SUCCESS", dto.OrderID); n != 1 {
			t.Fatalf("audit %d", n)
		}
		code, body, _ = postJSON(t, base+"/"+dto.OrderID+"/abate", `{}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ILLEGAL_STATE_TRANSITION" {
			t.Fatalf("again %d %s", code, body)
		}
	})

	t.Run("abate charged and proc", func(t *testing.T) {
		code, body, _ := postJSON(t, base, orderBody(registerID, drugA(1)), authHdr(tok))
		dto := decodeOrder(t, body)
		if code != 201 {
			t.Fatalf("%d %s", code, body)
		}
		if err := st.SetOrderCharged(dto.OrderID, true); err != nil {
			t.Fatal(err)
		}
		code, body, _ = postJSON(t, base+"/"+dto.OrderID+"/abate", `{}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ORDER_CHARGED_NOT_ABATABLE" {
			t.Fatalf("charged %d %s", code, body)
		}
		code, body = getAuth(t, base+"/"+dto.OrderID, tok)
		if code != 200 || decodeOrder(t, body).Status != "SAVE" {
			t.Fatalf("status %d %s", code, body)
		}
		code, body, _ = postJSON(t, base, `{"registerId":"`+registerID+`"}`, authHdr(tok))
		proc := decodeOrder(t, body)
		if err := st.SetOrderStatus(proc.OrderID, "PROC"); err != nil {
			t.Fatal(err)
		}
		code, body, _ = postJSON(t, base+"/"+proc.OrderID+"/abate", `{}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ILLEGAL_STATE_TRANSITION" {
			t.Fatalf("proc %d %s", code, body)
		}
	})

	t.Run("encounter required", func(t *testing.T) {
		code, body, _ := postJSON(t, base, `{}`, authHdr(tok))
		if code != 400 || decodeErr(t, body).Error.Code != "VALIDATION_ERROR" || !strings.Contains(flatFields(decodeErr(t, body)), "registerId") {
			t.Fatalf("missing %d %s", code, body)
		}
		code, body, _ = postJSON(t, base, orderBody("no-such-register", ""), authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ENCOUNTER_REQUIRED" {
			t.Fatalf("unknown %d %s", code, body)
		}
		backPatient := mustCreatePatient(t, ts.URL, tok)
		backID := mustRegisterID(t, ts.URL, tok, backPatient)
		code, body, _ = postJSON(t, ts.URL+"/api/v1/registers/"+backID+"/cancel", `{}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("cancel %d %s", code, body)
		}
		code, body, _ = postJSON(t, base, orderBody(backID, drugA(1)), authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ENCOUNTER_REQUIRED" {
			t.Fatalf("back %d %s", code, body)
		}
		fresh := mustRegisterID(t, ts.URL, tok, patientID)
		code, body, _ = postJSON(t, base, orderBody(fresh, drugA(1)), authHdr(tok))
		draft := decodeOrder(t, body)
		if code != 201 || draft.Status != "SAVE" {
			t.Fatalf("draft %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/registers/"+fresh+"/cancel", `{}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("cancel draft register %d %s", code, body)
		}
		code, body, _ = postJSON(t, base+"/"+draft.OrderID+"/commit", `{}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ENCOUNTER_REQUIRED" {
			t.Fatalf("commit back %d %s", code, body)
		}
		code, body = getAuth(t, base+"/"+draft.OrderID, tok)
		if code != 200 || decodeOrder(t, body).Status != "SAVE" {
			t.Fatalf("still save %d %s", code, body)
		}
		code, body = patchJSON(t, base+"/"+draft.OrderID, `{"diagnoseName":"退号后"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ENCOUNTER_REQUIRED" {
			t.Fatalf("patch back %d %s", code, body)
		}
	})

	t.Run("empty items and illegal amount on commit", func(t *testing.T) {
		code, body, _ := postJSON(t, base, `{"registerId":"`+registerID+`"}`, authHdr(tok))
		draft := decodeOrder(t, body)
		if code != 201 {
			t.Fatalf("%d %s", code, body)
		}
		code, body, _ = postJSON(t, base+"/"+draft.OrderID+"/commit", `{}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ORDER_EMPTY_ITEMS" {
			t.Fatalf("empty %d %s", code, body)
		}
		code, body, _ = postJSON(t, base, orderBody(registerID, drugA(0)), authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ORDER_INVALID_AMOUNT" {
			t.Fatalf("zero %d %s", code, body)
		}
		code, body, _ = postJSON(t, base, orderBody(registerID, drugA(-1)), authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ORDER_INVALID_AMOUNT" {
			t.Fatalf("neg %d %s", code, body)
		}
		code, body, _ = postJSON(t, base, orderBody(registerID, drugA(1)), authHdr(tok))
		bad := decodeOrder(t, body)
		if code != 201 || len(bad.Items) != 1 {
			t.Fatalf("%d %s", code, body)
		}
		if err := st.SetOrderItemAmountMilli(bad.OrderID, 1, 0); err != nil {
			t.Fatal(err)
		}
		code, body, _ = postJSON(t, base+"/"+bad.OrderID+"/commit", `{}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ORDER_INVALID_AMOUNT" {
			t.Fatalf("commit amount %d %s", code, body)
		}
		code, body = getAuth(t, base+"/"+bad.OrderID, tok)
		if decodeOrder(t, body).Status != "SAVE" {
			t.Fatalf("still %s", body)
		}
	})

	t.Run("catalog and patient", func(t *testing.T) {
		code, body, _ := postJSON(t, base, `{"registerId":"`+registerID+`","items":[{"itemNo":"NO-SUCH","itemType":"DRUG","itemName":"x","amount":1,"price":1}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "CATALOG_ITEM_NOT_FOUND" {
			t.Fatalf("catalog %d %s", code, body)
		}
		code, body, _ = postJSON(t, base, `{"registerId":"`+registerID+`","patientId":"someone-else"}`, authHdr(tok))
		if code != 400 || decodeErr(t, body).Error.Code != "VALIDATION_ERROR" || !strings.Contains(flatFields(decodeErr(t, body)), "patientId") {
			t.Fatalf("patient %d %s", code, body)
		}
		code, body, _ = postJSON(t, base, `{"registerId":"`+registerID+`","encounterId":"other"}`, authHdr(tok))
		if code != 400 || !strings.Contains(flatFields(decodeErr(t, body)), "encounterId") {
			t.Fatalf("alias %d %s", code, body)
		}
	})

	t.Run("multi line total and second sheet", func(t *testing.T) {
		body := `{"registerId":"` + registerID + `","items":[` + drugA(1) + `,` + drugB(1) + `]}`
		code, raw, _ := postJSON(t, base, body, authHdr(tok))
		if code != 201 {
			t.Fatalf("%d %s", code, raw)
		}
		dto := decodeOrder(t, raw)
		if len(dto.Items) != 2 || !near(dto.TotalFee, 18) {
			t.Fatalf("%+v", dto)
		}
		code, raw, _ = postJSON(t, base+"/"+dto.OrderID+"/commit", `{}`, authHdr(tok))
		got := decodeOrder(t, raw)
		if code != 200 || got.Status != "COMMIT" || !near(got.TotalFee, 18) {
			t.Fatalf("%d %+v %s", code, got, raw)
		}
		code, raw, _ = postJSON(t, base, orderBody(registerID, drugA(1)), authHdr(tok))
		second := decodeOrder(t, raw)
		if code != 201 || second.OrderID == dto.OrderID {
			t.Fatalf("second %d %s", code, raw)
		}
		code, raw = getAuth(t, base+"?registerId="+registerID, tok)
		if code != 200 || !orderListHas(decodeOrderList(t, raw), dto.OrderID) || !orderListHas(decodeOrderList(t, raw), second.OrderID) {
			t.Fatalf("list %d %s", code, raw)
		}
	})

	t.Run("idempotency", func(t *testing.T) {
		hdr := authHdr(tok)
		hdr["Idempotency-Key"] = "order-key-1"
		payload := orderBody(registerID, drugA(1))
		code, body, _ := postJSON(t, base, payload, hdr)
		first := decodeOrder(t, body)
		if code != 201 {
			t.Fatalf("%d %s", code, body)
		}
		code, body, _ = postJSON(t, base, payload, hdr)
		second := decodeOrder(t, body)
		if code != 201 || second.OrderID != first.OrderID {
			t.Fatalf("replay %d %s", code, body)
		}
		code, body, _ = postJSON(t, base, orderBody(registerID, drugB(1)), hdr)
		if code != 409 || decodeErr(t, body).Error.Code != "IDEMPOTENCY_KEY_CONFLICT" {
			t.Fatalf("conflict %d %s", code, body)
		}
	})

	t.Run("not implemented", func(t *testing.T) {
		code, body, _ := postJSON(t, base+"/"+created.OrderID+"/recipe-check", `{}`, nil)
		if code != 401 {
			t.Fatalf("recipe auth %d %s", code, body)
		}
		code, body, _ = postJSON(t, base+"/"+created.OrderID+"/recipe-check", `{}`, authHdr(tok))
		if code != 501 || decodeErr(t, body).Error.Code != "NOT_IMPLEMENTED" {
			t.Fatalf("recipe %d %s", code, body)
		}
		code, body = getAuth(t, base+"/"+created.OrderID+"/charge-preview", tok)
		if code != 200 || !strings.Contains(string(body), `"squareFee":20`) || !strings.Contains(string(body), `"favourableFee":0`) {
			t.Fatalf("preview %d %s", code, body)
		}
		code, body, _ = postJSON(t, base+"/"+created.OrderID+"/dispense", `{}`, authHdr(tok))
		if code != 501 || decodeErr(t, body).Error.Code != "NOT_IMPLEMENTED" {
			t.Fatalf("dispense %d %s", code, body)
		}
		code, body = getAuth(t, base+"/"+created.OrderID, tok)
		if decodeOrder(t, body).Status != "COMMIT" {
			t.Fatalf("unchanged %s", body)
		}
	})
}

func mustRegisterID(t *testing.T, origin, tok, patientID string) string {
	t.Helper()
	code, body, _ := postJSON(t, origin+"/api/v1/registers", registerBody(patientID, "dept-1", "doc-1", "2026-10-12"), authHdr(tok))
	if code != 201 {
		t.Fatalf("register %d %s", code, body)
	}
	dto := decodeRegister(t, body)
	if dto.RegisterID == "" {
		t.Fatalf("register id %s", body)
	}
	return dto.RegisterID
}

func orderBody(registerID, items string) string {
	if items == "" {
		return `{"registerId":"` + registerID + `","items":[]}`
	}
	return `{"registerId":"` + registerID + `","items":[` + items + `]}`
}

func drugA(amount float64) string {
	return `{"itemNo":"DRUG-A","itemType":"DRUG","itemName":"演示阿莫西林胶囊","amount":` + jsonNumber(amount) + `,"price":10}`
}

func drugB(amount float64) string {
	return `{"itemNo":"DRUG-B","itemType":"DRUG","itemName":"演示布洛芬片","amount":` + jsonNumber(amount) + `,"price":8}`
}

func jsonNumber(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func near(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

type orderItemJSON struct {
	OrderItemID string  `json:"orderItemId"`
	LineNo      int     `json:"lineNo"`
	ItemNo      string  `json:"itemNo"`
	ItemType    string  `json:"itemType"`
	ItemName    string  `json:"itemName"`
	Spec        *string `json:"spec"`
	Units       string  `json:"units"`
	Amount      float64 `json:"amount"`
	Price       float64 `json:"price"`
	TotalFee    float64 `json:"totalFee"`
}

type orderJSON struct {
	OrderID      string          `json:"orderId"`
	OrderNo      string          `json:"orderNo"`
	RegisterID   string          `json:"registerId"`
	EncounterID  string          `json:"encounterId"`
	PatientID    string          `json:"patientId"`
	DepartmentID string          `json:"departmentId"`
	DoctorID     string          `json:"doctorId"`
	DiagnoseName *string         `json:"diagnoseName"`
	RecipeSort   string          `json:"recipeSort"`
	Status       string          `json:"status"`
	TotalFee     float64         `json:"totalFee"`
	Charged      bool            `json:"charged"`
	Items        []orderItemJSON `json:"items"`
	SubmittedAt  *string         `json:"submittedAt"`
	AbatedAt     *string         `json:"abatedAt"`
	AbatedBy     *string         `json:"abatedBy"`
}

func decodeOrder(t *testing.T, body []byte) orderJSON {
	t.Helper()
	var dto orderJSON
	if err := json.Unmarshal(body, &dto); err != nil {
		t.Fatalf("json %v %s", err, body)
	}
	return dto
}

func decodeOrderList(t *testing.T, body []byte) []orderJSON {
	t.Helper()
	var list struct {
		Items []orderJSON `json:"items"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("json %v %s", err, body)
	}
	return list.Items
}

func orderListHas(list []orderJSON, id string) bool {
	for _, item := range list {
		if item.OrderID == id {
			return true
		}
	}
	return false
}

func countOrderAudit(st *memory.Store, action, outcome, resourceID string) int {
	n := 0
	for _, e := range st.Audits() {
		if e.Action == action && e.Outcome == outcome && e.ResourceType == "Order" && e.ResourceID == resourceID {
			n++
		}
	}
	return n
}

func patchJSON(t *testing.T, url, body string, hdr map[string]string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, b
}
