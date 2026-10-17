package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestDiagnoseLifecycleHTTP(t *testing.T) {
	ts, st, issuer, _ := newRegisterHTTP(t)
	tok := bearerFor(t, issuer, seed.DemoUser())
	patientID := mustCreatePatient(t, ts.URL, tok)
	code, raw, _ := postJSON(t, ts.URL+"/api/v1/registers", registerBody(patientID, "dept-1", "doc-1", "2026-10-15"), authHdr(tok))
	if code != 201 {
		t.Fatalf("register status=%d body=%s", code, raw)
	}
	registerID := decodeRegister(t, raw).RegisterID
	base := ts.URL + "/api/v1/registers/" + registerID + "/diagnoses"
	st.Add(domain.User{ID: "limited-diagnose", Username: "limited-diagnose", DisplayName: "无诊断权限", Status: domain.StatusActive})
	limitedTok := bearerFor(t, issuer, domain.User{ID: "limited-diagnose", Username: "limited-diagnose", DisplayName: "无诊断权限", Status: domain.StatusActive})
	code, raw, _ = postJSON(t, base, `{"content":"不应保存"}`, authHdr(limitedTok))
	if code != 403 || decodeErr(t, raw).Error.Code != "FORBIDDEN" {
		t.Fatalf("permission status=%d body=%s", code, raw)
	}
	registrar := domain.User{ID: "registrar-diagnose", Username: "registrar-diagnose", DisplayName: "挂号员", Roles: []domain.Role{domain.RoleRegistrar}, Status: domain.StatusActive}
	st.Add(registrar)
	registrarTok := bearerFor(t, issuer, registrar)
	code, raw, _ = postJSON(t, base, `{"content":"挂号员不应录入"}`, authHdr(registrarTok))
	if code != 403 || decodeErr(t, raw).Error.Code != "FORBIDDEN" {
		t.Fatalf("registrar permission status=%d body=%s", code, raw)
	}
	code, raw, _ = postJSON(t, base, `{"content":"上感"}`, authHdr(tok))
	if code != 201 {
		t.Fatalf("create status=%d body=%s", code, raw)
	}
	var created struct {
		DiagnoseID  string `json:"diagnoseId"`
		EncounterID string `json:"encounterId"`
		Status      string `json:"status"`
		Content     string `json:"content"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatal(err)
	}
	if created.DiagnoseID == "" || created.EncounterID != registerID || created.Status != "SAVE" || created.Content != "上感" {
		t.Fatalf("created=%+v", created)
	}
	code, raw = getAuth(t, base, tok)
	if code != 200 || !json.Valid(raw) || !containsJSON(raw, created.DiagnoseID) {
		t.Fatalf("list status=%d body=%s", code, raw)
	}
	code, raw, _ = postJSON(t, ts.URL+"/api/v1/diagnoses/"+created.DiagnoseID+"/commit", `{}`, authHdr(tok))
	if code != 200 || !containsJSON(raw, `"status":"COMMIT"`) {
		t.Fatalf("commit status=%d body=%s", code, raw)
	}
	code, raw, _ = postJSON(t, ts.URL+"/api/v1/diagnoses/"+created.DiagnoseID+"/commit", `{}`, authHdr(tok))
	if code != 409 || decodeErr(t, raw).Error.Code != "DIAGNOSE_NOT_COMMITTABLE" {
		t.Fatalf("second commit status=%d body=%s", code, raw)
	}
	code, raw, _ = postJSON(t, ts.URL+"/api/v1/diagnoses/"+created.DiagnoseID+"/abate", `{}`, authHdr(tok))
	if code != 200 || !containsJSON(raw, `"status":"ABATE"`) {
		t.Fatalf("abate status=%d body=%s", code, raw)
	}
	n := 0
	for _, event := range st.Audits() {
		if event.Action == "diagnose.create" && event.Outcome == "SUCCESS" && event.ResourceType == "Diagnose" && event.ResourceID == created.DiagnoseID {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("create audit=%d", n)
	}
}

func containsJSON(raw []byte, fragment string) bool {
	return string(raw) != "" && (fragment == "" || string(raw) == fragment || (len(raw) >= len(fragment) && string(raw) != "" && json.Valid(raw) && stringContains(string(raw), fragment)))
}

func stringContains(s, fragment string) bool {
	for i := 0; i+len(fragment) <= len(s); i++ {
		if s[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
