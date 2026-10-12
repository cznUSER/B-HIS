package httpapi

import (
	"encoding/json"
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

func newMasterHTTP(t *testing.T) (*httptest.Server, *memory.Store, *auth.Issuer) {
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
		Users: st, Revocations: st, Audit: st, Patients: st, Registers: st, Orders: st,
		Charges: st, Dispenses: st, Master: st, Tokens: issuer, Log: log, TTL: ttl,
		Now: func() time.Time { return fixed },
	}
	ts := httptest.NewServer(New(svc, log, nil, "memory", time.Minute))
	t.Cleanup(ts.Close)
	return ts, st, issuer
}

func TestMasterHTTP(t *testing.T) {
	ts, st, issuer := newMasterHTTP(t)
	tok := bearerFor(t, issuer, seed.DemoUser())
	doc := domain.User{
		ID: "00000000-0000-4000-8000-0000000000d1", Username: "doctor", DisplayName: "医生",
		Roles: []domain.Role{domain.RoleDoctor}, Status: domain.StatusActive,
	}
	pharm := domain.User{
		ID: "00000000-0000-4000-8000-0000000000d2", Username: "pharmacist", DisplayName: "药房",
		Roles: []domain.Role{domain.RolePharmacist}, Status: domain.StatusActive,
	}
	st.Add(doc)
	st.Add(pharm)
	docTok := bearerFor(t, issuer, doc)
	pharmTok := bearerFor(t, issuer, pharm)
	api := ts.URL + "/api/v1"

	t.Run("health", func(t *testing.T) {
		code, body := getAuth(t, ts.URL+"/health", "")
		if code != 200 ||
			!strings.Contains(string(body), `"version":"0.3.10-draft"`) ||
			!strings.Contains(string(body), `"contractVersion":"0.3.10-draft"`) ||
			!strings.Contains(string(body), `"implementedVersion":"0.3.10-draft"`) {
			t.Fatalf("%d %s", code, body)
		}
	})

	t.Run("authz", func(t *testing.T) {
		code, body := getAuth(t, api+"/departments", "")
		if code != 401 {
			t.Fatalf("anon %d %s", code, body)
		}
		code, body = getAuth(t, api+"/departments", docTok)
		if code != 403 || decodeErr(t, body).Error.Code != "FORBIDDEN" {
			t.Fatalf("doctor departments %d %s", code, body)
		}
		code, body = getAuth(t, api+"/dicts", docTok)
		if code != 200 || !strings.Contains(string(body), "register_sort") || !strings.Contains(string(body), `"dictType":"sex"`) {
			t.Fatalf("dicts %d %s", code, body)
		}
	})

	patientID := mustCreatePatient(t, ts.URL, tok)

	t.Run("departments", func(t *testing.T) {
		code, body := getAuth(t, api+"/departments", tok)
		if code != 200 || !strings.Contains(string(body), "dept-1") || !strings.Contains(string(body), "dept-2") {
			t.Fatalf("list %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/departments", `{"deptCode":"dept1","deptName":"重复"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "DEPT_CODE_DUPLICATE" {
			t.Fatalf("dup %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/departments", `{"deptCode":"DEPT9","deptName":"测试科","sortNo":3}`, authHdr(tok))
		if code != 201 || !strings.Contains(string(body), `"deptCode":"DEPT9"`) {
			t.Fatalf("create %d %s", code, body)
		}
		code, body = patchJSON(t, api+"/departments/dept-1", `{"deptName":"内科改","status":"INACTIVE"}`, authHdr(tok))
		if code != 200 || !strings.Contains(string(body), "内科改") || !strings.Contains(string(body), "INACTIVE") {
			t.Fatalf("patch %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/registers", registerBody(patientID, "dept-1", "doc-1", "2026-10-12"), authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "DEPARTMENT_INACTIVE" {
			t.Fatalf("register %d %s", code, body)
		}
		code, body = patchJSON(t, api+"/departments/dept-1", `{"status":"ACTIVE"}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("reactivate %d %s", code, body)
		}
	})

	t.Run("doctors", func(t *testing.T) {
		code, body := getAuth(t, api+"/doctors?departmentId=dept-1", tok)
		if code != 200 || !strings.Contains(string(body), "doc-1") || strings.Contains(string(body), "doc-2") {
			t.Fatalf("filter %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/doctors", `{"doctorCode":"DOC9","doctorName":"王医生","departmentId":"missing"}`, authHdr(tok))
		if code != 404 || decodeErr(t, body).Error.Code != "DEPARTMENT_NOT_FOUND" {
			t.Fatalf("dept %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/doctors", `{"doctorCode":"doc1","doctorName":"重复"}`, authHdr(tok))
		if code != 400 {
			t.Fatalf("missing dept should validate, got %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/doctors", `{"doctorCode":"doc1","doctorName":"重复","departmentId":"dept-1"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "DOCTOR_CODE_DUPLICATE" {
			t.Fatalf("dup %d %s", code, body)
		}
		code, body = patchJSON(t, api+"/doctors/doc-1", `{"doctorName":"张医生改","status":"INACTIVE"}`, authHdr(tok))
		if code != 200 || !strings.Contains(string(body), "张医生改") {
			t.Fatalf("patch %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/registers", registerBody(patientID, "dept-1", "doc-1", "2026-10-12"), authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "DOCTOR_INACTIVE" {
			t.Fatalf("register %d %s", code, body)
		}
		code, body = patchJSON(t, api+"/doctors/doc-1", `{"status":"ACTIVE"}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("reactivate %d %s", code, body)
		}
	})

	t.Run("catalog and inactive order", func(t *testing.T) {
		code, body, _ := postJSON(t, api+"/catalog-items", `{"itemNo":"DRUG-C","itemName":"丙","itemType":"DRUG","units":"盒","price":1.5}`, authHdr(tok))
		if code != 201 || !strings.Contains(string(body), `"itemType":"DRUG"`) {
			t.Fatalf("drug %d %s", code, body)
		}
		code, body = getAuth(t, api+"/inventory/items?skuCode=DRUG-C", tok)
		if code != 200 || !strings.Contains(string(body), `"qtyOnHand":0`) {
			t.Fatalf("stock %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/catalog-items", `{"itemNo":"TREAT-9","itemName":"治疗九","itemType":"TREAT","units":"次","price":2}`, authHdr(tok))
		if code != 201 {
			t.Fatalf("treat %d %s", code, body)
		}
		code, body = getAuth(t, api+"/inventory/items?itemNo=TREAT-9", tok)
		if code != 200 || strings.Contains(string(body), "TREAT-9") {
			t.Fatalf("treat stock %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/catalog-items", `{"itemNo":"drug-a","itemName":"重复","itemType":"DRUG","units":"盒","price":1}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ITEM_NO_DUPLICATE" {
			t.Fatalf("dup %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/catalog-items", `{"itemNo":"DRUG-Z","itemName":"负价","itemType":"DRUG","units":"盒","price":-1}`, authHdr(tok))
		if code != 400 || decodeErr(t, body).Error.Code != "INVALID_PRICE" {
			t.Fatalf("price %d %s", code, body)
		}
		code, body = patchJSON(t, api+"/catalog-items/cat-drug-a", `{"price":12.5}`, authHdr(tok))
		if code != 200 || !strings.Contains(string(body), "12.5") {
			t.Fatalf("price patch %d %s", code, body)
		}
		code, body = patchJSON(t, api+"/catalog-items/cat-treat-1", `{"status":"INACTIVE"}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("stop treat %d %s", code, body)
		}
		regID := mustRegisterID(t, ts.URL, tok, patientID)
		code, body, _ = postJSON(t, api+"/orders", orderBody(regID, `{"itemNo":"TREAT-1","itemType":"TREAT","itemName":"演示诊查费","amount":1,"price":15}`), authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "CATALOG_INACTIVE" {
			t.Fatalf("order %d %s", code, body)
		}
		code, body = patchJSON(t, api+"/catalog-items/cat-treat-1", `{"status":"ACTIVE"}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("restore %d %s", code, body)
		}
	})

	t.Run("adjust", func(t *testing.T) {
		code, body := getAuth(t, api+"/inventory/items?skuCode=DRUG-A", tok)
		if code != 200 {
			t.Fatalf("list %d %s", code, body)
		}
		var listed struct {
			Items []struct {
				SkuID      string  `json:"skuId"`
				QtyOnHand  float64 `json:"qtyOnHand"`
				RowVersion string  `json:"rowVersion"`
			} `json:"items"`
		}
		if err := json.Unmarshal(body, &listed); err != nil || len(listed.Items) != 1 || listed.Items[0].SkuID != "sku-drug-a" {
			t.Fatalf("parse %v %s", err, body)
		}
		code, body, _ = postJSON(t, api+"/inventory/items/sku-drug-a/adjust", `{"qtyDelta":1,"remark":"盘盈"}`, authHdr(pharmTok))
		if code != 403 || decodeErr(t, body).Error.Code != "FORBIDDEN" {
			t.Fatalf("pharm %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/inventory/items/sku-drug-a/adjust", `{"qtyDelta":1}`, authHdr(tok))
		if code != 400 {
			t.Fatalf("remark %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/inventory/items/sku-drug-a/adjust", `{"qtyDelta":1,"remark":"盘盈","rowVersion":"999"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "CONFLICT" {
			t.Fatalf("conflict %d %s", code, body)
		}
		rv := listed.Items[0].RowVersion
		code, body, _ = postJSON(t, api+"/inventory/items/sku-drug-a/adjust", `{"qtyDelta":1,"remark":"盘盈","rowVersion":"`+rv+`"}`, authHdr(tok))
		if code != 200 || !strings.Contains(string(body), `"qtyOnHand":101`) {
			t.Fatalf("adjust %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/inventory/items/sku-drug-a/adjust", `{"qtyDelta":-1000,"remark":"盘亏"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ADJUST_RESULT_NEGATIVE" {
			t.Fatalf("negative %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/inventory/items/no-sku/adjust", `{"qtyDelta":1,"remark":"期初"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "NOT_DRUG_SKU" {
			t.Fatalf("sku %d %s", code, body)
		}
		n := 0
		for _, ev := range st.Audits() {
			if ev.Action == domain.ActionInventoryAdjust && ev.Outcome == domain.OutcomeSuccess {
				n++
			}
			if strings.Contains(ev.Action, "INIT") {
				t.Fatalf("audit %s", ev.Action)
			}
		}
		if n != 1 {
			t.Fatalf("audit success %d", n)
		}
	})

	t.Run("dicts", func(t *testing.T) {
		code, body := getAuth(t, api+"/dicts/register_sort/items", tok)
		if code != 200 || !strings.Contains(string(body), "sort-normal") || !strings.Contains(string(body), "普通号") {
			t.Fatalf("list %d %s", code, body)
		}
		code, body = patchJSON(t, api+"/dicts/register_sort/items/dict-sort-normal", `{"name":"普通号改"}`, authHdr(tok))
		if code != 200 || !strings.Contains(string(body), "普通号改") {
			t.Fatalf("patch %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/dicts/register_sort/items", `{"code":"sort-normal","name":"重复"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "DICT_CODE_DUPLICATE" {
			t.Fatalf("dup %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/dicts/sex/items", `{"code":"X","name":"未知"}`, authHdr(docTok))
		if code != 403 {
			t.Fatalf("doctor write %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/dicts/sex/items", `{"code":"X","name":"未知"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "DICT_TYPE_READONLY" {
			t.Fatalf("readonly %d %s", code, body)
		}
	})

	t.Run("users", func(t *testing.T) {
		code, body, _ := postJSON(t, api+"/users", `{"username":"System","displayName":"撞名"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "USERNAME_DUPLICATE" {
			t.Fatalf("dup %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/users", `{"username":"clerk-new","displayName":"新挂号员","roles":["registrar"]}`, authHdr(tok))
		if code != 201 || strings.Contains(strings.ToLower(string(body)), "password") || strings.Contains(string(body), "$2a$") {
			t.Fatalf("create %d %s", code, body)
		}
		var created struct {
			UserID string   `json:"userId"`
			Roles  []string `json:"roles"`
		}
		if err := json.Unmarshal(body, &created); err != nil || created.UserID == "" || len(created.Roles) != 1 || created.Roles[0] != "REGISTRAR" {
			t.Fatalf("dto %v %s", err, body)
		}
		code, body, _ = postJSON(t, api+"/auth/login", `{"username":"clerk-new","password":"1"}`, nil)
		if code != 200 || !strings.Contains(string(body), "accessToken") {
			t.Fatalf("login %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/users/"+created.UserID+"/reset-password", `{}`, authHdr(tok))
		if code != 204 || len(strings.TrimSpace(string(body))) != 0 {
			t.Fatalf("reset %d %q", code, body)
		}
		code, body, _ = postJSON(t, api+"/auth/login", `{"username":"clerk-new","password":"1"}`, nil)
		if code != 200 {
			t.Fatalf("login after reset %d %s", code, body)
		}
		code, body = patchJSON(t, api+"/users/"+created.UserID, `{"status":"DISABLED"}`, authHdr(tok))
		if code != 200 || !strings.Contains(string(body), "DISABLED") {
			t.Fatalf("disable %d %s", code, body)
		}
		code, body, _ = postJSON(t, api+"/auth/login", `{"username":"clerk-new","password":"1"}`, nil)
		if code != 403 || decodeErr(t, body).Error.Code != "ACCOUNT_DISABLED" {
			t.Fatalf("disabled login %d %s", code, body)
		}
		for _, ev := range st.Audits() {
			if strings.Contains(ev.ErrorCode, "$2a$") || strings.Contains(ev.ResourceID, "$2a$") || ev.ResourceID == "1" {
				t.Fatalf("audit leaked secret: %+v", ev)
			}
		}
	})
}
