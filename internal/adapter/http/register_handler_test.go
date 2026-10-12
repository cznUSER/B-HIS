package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
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

type wall struct{ t time.Time }

func (w *wall) Now() time.Time { return w.t }

func newRegisterHTTP(t *testing.T) (*httptest.Server, *memory.Store, *auth.Issuer, *wall) {
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
	clock := &wall{t: time.Date(2026, 10, 12, 4, 0, 0, 0, time.UTC)}
	svc := &app.Service{
		Users:       st,
		Revocations: st,
		Audit:       st,
		Patients:    st,
		Registers:   st,
		Tokens:      issuer,
		Log:         log,
		TTL:         ttl,
		Now:         clock.Now,
	}
	ts := httptest.NewServer(New(svc, log, nil, "memory", time.Minute))
	t.Cleanup(ts.Close)
	return ts, st, issuer, clock
}

func TestRegisterHTTP(t *testing.T) {
	ts, st, issuer, clock := newRegisterHTTP(t)
	tok := bearerFor(t, issuer, seed.DemoUser())
	base := ts.URL + "/api/v1/registers"
	patientID := mustCreatePatient(t, ts.URL, tok)

	t.Run("unauthorized", func(t *testing.T) {
		code, body, _ := postJSON(t, base, registerBody(patientID, "dept-1", "doc-1", "2026-10-12"), nil)
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("create %d %s", code, body)
		}
		code, body = getAuth(t, base+"?patientId="+patientID, "")
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("list %d %s", code, body)
		}
		code, body = getAuth(t, base+"/missing", "")
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("get %d %s", code, body)
		}
		code, body, _ = postJSON(t, base+"/missing/cancel", `{}`, nil)
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("cancel %d %s", code, body)
		}
	})

	var created registerJSON
	t.Run("create", func(t *testing.T) {
		code, body, _ := postJSON(t, base, registerBody(patientID, "dept-1", "doc-1", "2026-10-12"), authHdr(tok))
		if code != 201 {
			t.Fatalf("status %d %s", code, body)
		}
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatal(err)
		}
		if _, ok := raw["success"]; ok {
			t.Fatalf("201 must be RegisterDto, got %s", body)
		}
		created = decodeRegister(t, body)
		if created.RegisterID == "" || created.RegisterNo == "" || created.Status != "NORMAL" {
			t.Fatalf("%+v", created)
		}
		if created.PatientID != patientID || created.DepartmentID != "dept-1" || created.DoctorID != "doc-1" || created.VisitDate != "2026-10-12" {
			t.Fatalf("%+v", created)
		}
		if created.RegisterFee != 0 || created.RegisterCharged || created.RegisterSort != nil {
			t.Fatalf("%+v", created)
		}
		if created.RegisteredAt == "" || created.BackOperatorID != nil || created.BackOperatedAt != nil {
			t.Fatalf("%+v", created)
		}
		if n := countAudit(st, "register.create", "SUCCESS", created.RegisterID); n != 1 {
			t.Fatalf("audit %d", n)
		}
	})

	t.Run("optional sort and fee", func(t *testing.T) {
		body := `{"patientId":"` + patientID + `","departmentId":"dept-2","doctorId":"doc-2","visitDate":"2026-10-13","registerSort":"sort-normal","registerFee":12.5}`
		code, raw, _ := postJSON(t, base, body, authHdr(tok))
		if code != 201 {
			t.Fatalf("status %d %s", code, raw)
		}
		dto := decodeRegister(t, raw)
		if dto.RegisterSort == nil || *dto.RegisterSort != "sort-normal" || dto.RegisterFee != 12.5 || dto.RegisterCharged {
			t.Fatalf("%+v", dto)
		}
		if dto.DepartmentID != "dept-2" || dto.DoctorID != "doc-2" {
			t.Fatalf("%+v", dto)
		}
	})

	t.Run("validation", func(t *testing.T) {
		full := registerBody(patientID, "dept-1", "doc-1", "2026-10-12")
		cases := []struct{ name, body, field string }{
			{"patientId", strings.Replace(full, `"patientId":"`+patientID+`"`, `"patientId":""`, 1), "patientId"},
			{"departmentId", strings.Replace(full, `"departmentId":"dept-1"`, `"departmentId":""`, 1), "departmentId"},
			{"doctorId", strings.Replace(full, `"doctorId":"doc-1"`, `"doctorId":""`, 1), "doctorId"},
			{"visitDate", strings.Replace(full, `"visitDate":"2026-10-12"`, `"visitDate":""`, 1), "visitDate"},
			{"fee", strings.TrimRight(full, "}") + `,"registerFee":-1}`, "registerFee"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				code, raw, _ := postJSON(t, base, tc.body, authHdr(tok))
				if code != 400 {
					t.Fatalf("status %d %s", code, raw)
				}
				e := decodeErr(t, raw)
				if e.Error.Code != "VALIDATION_ERROR" || e.Success || e.Data != nil {
					t.Fatalf("%+v", e)
				}
				if !strings.Contains(flatFields(e), tc.field) {
					t.Fatalf("fields %v", e.Error.Details["fields"])
				}
			})
		}
	})

	t.Run("masters missing", func(t *testing.T) {
		cases := []struct{ name, body, code string }{
			{"patient", registerBody("no-such-patient", "dept-1", "doc-1", "2026-10-12"), "PATIENT_NOT_FOUND"},
			{"department", registerBody(patientID, "dept-x", "doc-1", "2026-10-12"), "DEPARTMENT_NOT_FOUND"},
			{"doctor", registerBody(patientID, "dept-1", "doc-x", "2026-10-12"), "DOCTOR_NOT_FOUND"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				code, raw, _ := postJSON(t, base, tc.body, authHdr(tok))
				if code != 404 {
					t.Fatalf("status %d %s", code, raw)
				}
				e := decodeErr(t, raw)
				if e.Error.Code != tc.code || e.Success || e.Data != nil {
					t.Fatalf("%+v", e)
				}
			})
		}
	})

	t.Run("query and missing detail", func(t *testing.T) {
		if created.RegisterID == "" {
			t.Fatal("create did not run")
		}
		code, body := getAuth(t, base+"/"+created.RegisterID, tok)
		if code != 200 {
			t.Fatalf("get %d %s", code, body)
		}
		dto := decodeRegister(t, body)
		if dto.RegisterID != created.RegisterID || dto.RegisterNo != created.RegisterNo || dto.Status != "NORMAL" {
			t.Fatalf("%+v", dto)
		}
		code, body = getAuth(t, base+"/00000000-0000-4000-8000-000000009999", tok)
		if code != 404 || decodeErr(t, body).Error.Code != "NOT_FOUND" {
			t.Fatalf("missing %d %s", code, body)
		}
		code, body = getAuth(t, base, tok)
		if code != 200 {
			t.Fatalf("empty query %d %s", code, body)
		}
		if n := len(decodeRegisterList(t, body)); n != 0 {
			t.Fatalf("unfiltered list %d", n)
		}
		for _, q := range []url.Values{
			{"patientId": {patientID}},
			{"registerNo": {created.RegisterNo}},
			{"visitDate": {created.VisitDate}},
		} {
			code, body = getAuth(t, base+"?"+q.Encode(), tok)
			if code != 200 {
				t.Fatalf("search %s %d %s", q.Encode(), code, body)
			}
			list := decodeRegisterList(t, body)
			if !registerListHas(list, created.RegisterID) {
				t.Fatalf("query %s -> %+v", q.Encode(), list)
			}
		}
	})

	t.Run("cancel normal then back", func(t *testing.T) {
		if created.RegisterID == "" {
			t.Fatal("create did not run")
		}
		code, body, _ := postJSON(t, base+"/"+created.RegisterID+"/cancel", `{}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("cancel %d %s", code, body)
		}
		dto := decodeRegister(t, body)
		if dto.Status != "BACK" || dto.BackOperatorID == nil || *dto.BackOperatorID != seed.DemoUserID || dto.BackOperatedAt == nil || *dto.BackOperatedAt == "" {
			t.Fatalf("%+v", dto)
		}
		if n := countAudit(st, "register.cancel", "SUCCESS", created.RegisterID); n != 1 {
			t.Fatalf("cancel audit %d", n)
		}
		firstOp, firstAt := *dto.BackOperatorID, *dto.BackOperatedAt
		clock.t = clock.t.Add(2 * time.Hour)
		regUser := domain.User{
			ID: "00000000-0000-4000-8000-000000000010", Username: "registrar", DisplayName: "挂号员",
			Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive,
		}
		st.Add(regUser)
		code, body, _ = postJSON(t, base+"/"+created.RegisterID+"/cancel", `{}`, authHdr(bearerFor(t, issuer, regUser)))
		if code != 409 || decodeErr(t, body).Error.Code != "REGISTER_BACK_NOT_ALLOWED" {
			t.Fatalf("second %d %s", code, body)
		}
		code, body = getAuth(t, base+"/"+created.RegisterID, tok)
		if code != 200 {
			t.Fatalf("get %d %s", code, body)
		}
		again := decodeRegister(t, body)
		if again.Status != "BACK" || again.BackOperatorID == nil || *again.BackOperatorID != firstOp || again.BackOperatedAt == nil || *again.BackOperatedAt != firstAt {
			t.Fatalf("back columns overwritten: %+v", again)
		}
	})

	t.Run("not cancellable", func(t *testing.T) {
		for _, status := range []string{"USED", "DIAGNOSING", "DIAGNOSED", "PAUSED"} {
			t.Run(status, func(t *testing.T) {
				code, raw, _ := postJSON(t, base, registerBody(patientID, "dept-1", "doc-1", "2026-10-12"), authHdr(tok))
				if code != 201 {
					t.Fatalf("create %d %s", code, raw)
				}
				dto := decodeRegister(t, raw)
				if err := st.SetRegisterStatus(dto.RegisterID, status); err != nil {
					t.Fatal(err)
				}
				code, raw, _ = postJSON(t, base+"/"+dto.RegisterID+"/cancel", `{}`, authHdr(tok))
				if code != 409 || decodeErr(t, raw).Error.Code != "REGISTER_NOT_CANCELLABLE" {
					t.Fatalf("cancel %d %s", code, raw)
				}
				code, raw = getAuth(t, base+"/"+dto.RegisterID, tok)
				got := decodeRegister(t, raw)
				if code != 200 || got.Status != status || got.BackOperatorID != nil || got.BackOperatedAt != nil {
					t.Fatalf("status changed %+v", got)
				}
			})
		}
	})

	t.Run("roles", func(t *testing.T) {
		doc := domain.User{
			ID: "00000000-0000-4000-8000-000000000011", Username: "doctor", DisplayName: "医生",
			Roles: []domain.Role{domain.RoleDoctor}, Status: domain.StatusActive,
		}
		st.Add(doc)
		docTok := bearerFor(t, issuer, doc)
		code, body, _ := postJSON(t, base, registerBody(patientID, "dept-1", "doc-1", "2026-10-12"), authHdr(docTok))
		if code != 403 || decodeErr(t, body).Error.Code != "FORBIDDEN" {
			t.Fatalf("doctor create %d %s", code, body)
		}
		code, body = getAuth(t, base+"?patientId="+url.QueryEscape(patientID), docTok)
		if code != 200 || !registerListHas(decodeRegisterList(t, body), created.RegisterID) {
			t.Fatalf("doctor search %d %s", code, body)
		}
		code, raw, _ := postJSON(t, base, registerBody(patientID, "dept-1", "doc-1", "2026-10-16"), authHdr(tok))
		if code != 201 {
			t.Fatalf("fixture %d %s", code, raw)
		}
		held := decodeRegister(t, raw)
		code, body, _ = postJSON(t, base+"/"+held.RegisterID+"/cancel", `{}`, authHdr(docTok))
		if code != 403 || decodeErr(t, body).Error.Code != "FORBIDDEN" {
			t.Fatalf("doctor cancel %d %s", code, body)
		}
		code, body = getAuth(t, base+"/"+held.RegisterID, tok)
		if code != 200 || decodeRegister(t, body).Status != "NORMAL" {
			t.Fatalf("doctor cancel changed status %d %s", code, body)
		}
		regUser := domain.User{
			ID: "00000000-0000-4000-8000-000000000012", Username: "registrar2", DisplayName: "挂号员二",
			Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive,
		}
		st.Add(regUser)
		code, body, _ = postJSON(t, base, registerBody(patientID, "dept-1", "doc-1", "2026-10-14"), authHdr(bearerFor(t, issuer, regUser)))
		if code != 201 {
			t.Fatalf("registrar %d %s", code, body)
		}
		if decodeRegister(t, body).Status != "NORMAL" {
			t.Fatalf("%s", body)
		}
	})

	t.Run("idempotency", func(t *testing.T) {
		hdr := authHdr(tok)
		hdr["Idempotency-Key"] = "reg-create-1"
		body := registerBody(patientID, "dept-1", "doc-1", "2026-10-15")
		code1, raw1, _ := postJSON(t, base, body, hdr)
		code2, raw2, _ := postJSON(t, base, body, hdr)
		if code1 != 201 || code2 != 201 {
			t.Fatalf("%d %s / %d %s", code1, raw1, code2, raw2)
		}
		a, b := decodeRegister(t, raw1), decodeRegister(t, raw2)
		if a.RegisterID == "" || a.RegisterID != b.RegisterID {
			t.Fatalf("%+v %+v", a, b)
		}
		if n := countAudit(st, "register.create", "SUCCESS", a.RegisterID); n != 1 {
			t.Fatalf("replay audited %d", n)
		}
		other := registerBody(patientID, "dept-2", "doc-2", "2026-10-15")
		code, raw, _ := postJSON(t, base, other, hdr)
		if code != 409 || decodeErr(t, raw).Error.Code != "IDEMPOTENCY_KEY_CONFLICT" {
			t.Fatalf("conflict %d %s", code, raw)
		}
	})
}

func mustCreatePatient(t *testing.T, origin, tok string) string {
	t.Helper()
	code, body, _ := postJSON(t, origin+"/api/v1/patients",
		`{"name":"挂号患者","sex":"M","birthday":"1990-01-15","fareIdentity":"SELF"}`, authHdr(tok))
	if code != 201 {
		t.Fatalf("patient %d %s", code, body)
	}
	var dto struct {
		PatientID string `json:"patientId"`
	}
	if err := json.Unmarshal(body, &dto); err != nil || dto.PatientID == "" {
		t.Fatalf("patient json %v %s", err, body)
	}
	return dto.PatientID
}

func registerBody(patientID, departmentID, doctorID, visitDate string) string {
	return `{"patientId":"` + patientID + `","departmentId":"` + departmentID + `","doctorId":"` + doctorID + `","visitDate":"` + visitDate + `"}`
}

type registerJSON struct {
	RegisterID      string  `json:"registerId"`
	RegisterNo      string  `json:"registerNo"`
	PatientID       string  `json:"patientId"`
	DepartmentID    string  `json:"departmentId"`
	DoctorID        string  `json:"doctorId"`
	RegisterSort    *string `json:"registerSort"`
	VisitDate       string  `json:"visitDate"`
	Status          string  `json:"status"`
	RegisterFee     float64 `json:"registerFee"`
	RegisterCharged bool    `json:"registerCharged"`
	RegisteredAt    string  `json:"registeredAt"`
	BackOperatorID  *string `json:"backOperatorId"`
	BackOperatedAt  *string `json:"backOperatedAt"`
}

func decodeRegister(t *testing.T, body []byte) registerJSON {
	t.Helper()
	var dto registerJSON
	if err := json.Unmarshal(body, &dto); err != nil {
		t.Fatalf("json %v %s", err, body)
	}
	return dto
}

func decodeRegisterList(t *testing.T, body []byte) []registerJSON {
	t.Helper()
	var list struct {
		Items []registerJSON `json:"items"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("json %v %s", err, body)
	}
	return list.Items
}

func registerListHas(list []registerJSON, id string) bool {
	for _, item := range list {
		if item.RegisterID == id {
			return true
		}
	}
	return false
}

func countAudit(st *memory.Store, action, outcome, resourceID string) int {
	n := 0
	for _, e := range st.Audits() {
		if e.Action == action && e.Outcome == outcome && e.ResourceType == "Register" && e.ResourceID == resourceID {
			n++
		}
	}
	return n
}
