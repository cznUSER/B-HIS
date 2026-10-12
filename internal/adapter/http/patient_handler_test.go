package httpapi

import (
	"encoding/json"
	"net/http"
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

func newPatientHTTP(t *testing.T) (*httptest.Server, *memory.Store, *auth.Issuer) {
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
		Tokens:      issuer,
		Log:         log,
		TTL:         ttl,
		Now:         func() time.Time { return fixed },
	}
	ts := httptest.NewServer(New(svc, log, nil, "memory", time.Minute))
	t.Cleanup(ts.Close)
	return ts, st, issuer
}

func bearerFor(t *testing.T, issuer *auth.Issuer, u domain.User) string {
	t.Helper()
	tok, _, _, err := issuer.Issue(u, time.Date(2026, 10, 12, 4, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func authHdr(tok string) map[string]string {
	if tok == "" {
		return nil
	}
	return map[string]string{"Authorization": "Bearer " + tok}
}

func decodeErr(t *testing.T, body []byte) errDTO {
	t.Helper()
	var e errDTO
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("json: %v body %s", err, body)
	}
	return e
}

func TestPatientHTTP(t *testing.T) {
	ts, st, issuer := newPatientHTTP(t)
	tok := bearerFor(t, issuer, seed.DemoUser())
	base := ts.URL + "/api/v1/patients"

	t.Run("unauthorized", func(t *testing.T) {
		for _, tc := range []struct {
			name, method, url, body string
		}{
			{"create", http.MethodPost, base, `{"name":"张三","sex":"M","birthday":"1990-01-15","fareIdentity":"SELF"}`},
			{"search", http.MethodGet, base + "?name=张", ""},
			{"get", http.MethodGet, base + "/missing", ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var code int
				var body []byte
				if tc.method == http.MethodPost {
					code, body, _ = postJSON(t, tc.url, tc.body, nil)
				} else {
					code, body = getAuth(t, tc.url, "")
				}
				if code != 401 {
					t.Fatalf("status %d %s", code, body)
				}
				e := decodeErr(t, body)
				if e.Success || e.Data != nil || e.Error.Code != "UNAUTHORIZED" || e.RequestID == "" {
					t.Fatalf("%+v", e)
				}
			})
		}
	})

	type createCase struct {
		name       string
		body       string
		hdr        map[string]string
		wantStatus int
		wantCode   string
		fields     string
	}
	cases := []createCase{
		{
			name:       "missing sex",
			body:       `{"name":"张三","birthday":"1990-01-15","fareIdentity":"SELF"}`,
			wantStatus: 400, wantCode: "VALIDATION_ERROR", fields: "sex",
		},
		{
			name:       "missing fareIdentity",
			body:       `{"name":"张三","sex":"F","birthday":"1990-01-15"}`,
			wantStatus: 400, wantCode: "VALIDATION_ERROR", fields: "fareIdentity",
		},
		{
			name:       "birthday floor",
			body:       `{"name":"张三","sex":"M","birthday":"1900-01-01","fareIdentity":"SELF"}`,
			wantStatus: 400, wantCode: "VALIDATION_ERROR", fields: "birthday",
		},
		{
			name:       "birthday future",
			body:       `{"name":"张三","sex":"U","birthday":"2026-10-13","fareIdentity":"SELF"}`,
			wantStatus: 400, wantCode: "VALIDATION_ERROR", fields: "birthday",
		},
		{
			name:       "address null",
			body:       `{"name":"空地址","sex":"M","birthday":"1990-01-15","fareIdentity":"SELF","address":null}`,
			wantStatus: 201,
		},
		{
			name:       "address empty",
			body:       `{"name":"空地址二","sex":"F","birthday":"1990-01-15","fareIdentity":"SELF","address":""}`,
			wantStatus: 201,
		},
		{
			name:       "create",
			body:       `{"name":"张三","sex":"M","birthday":"1990-01-15","idType":"ID_CARD","idNumber":"110101199001151234","cardNo":"C1","phone":"13800000000","address":"浦东","fareIdentity":"SELF"}`,
			wantStatus: 201,
		},
	}
	var created patientJSON
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, body, _ := postJSON(t, base, tc.body, authHdr(tok))
			if code != tc.wantStatus {
				t.Fatalf("status %d body %s", code, body)
			}
			if tc.wantStatus == 201 {
				var raw map[string]any
				if err := json.Unmarshal(body, &raw); err != nil {
					t.Fatal(err)
				}
				if _, ok := raw["success"]; ok {
					t.Fatalf("201 must be PatientDto, got %s", body)
				}
				var dto patientJSON
				if err := json.Unmarshal(body, &dto); err != nil {
					t.Fatal(err)
				}
				if dto.PatientID == "" || dto.Name == "" || dto.FareIdentity != "SELF" || dto.Status != "ACTIVE" {
					t.Fatalf("%+v", dto)
				}
				if tc.name == "create" {
					if dto.CardStatus != "NORMAL" || dto.Address != "浦东" || dto.CardNo != "C1" {
						t.Fatalf("%+v", dto)
					}
					created = dto
				}
				if tc.name == "address null" || tc.name == "address empty" {
					if dto.Address != "" || dto.CardStatus != "NOT_SENT" {
						t.Fatalf("%+v", dto)
					}
				}
				return
			}
			e := decodeErr(t, body)
			if e.Error.Code != tc.wantCode || e.Success || e.Data != nil {
				t.Fatalf("%+v", e)
			}
			if tc.fields != "" && !strings.Contains(flatFields(e), tc.fields) {
				t.Fatalf("fields %v want %s", e.Error.Details["fields"], tc.fields)
			}
		})
	}

	t.Run("duplicate idNumber", func(t *testing.T) {
		body := `{"name":"重复","sex":"M","birthday":"1990-01-15","idNumber":"110101199001151234","cardNo":"C-other","fareIdentity":"SELF"}`
		code, raw, _ := postJSON(t, base, body, authHdr(tok))
		if code != 409 {
			t.Fatalf("status %d %s", code, raw)
		}
		e := decodeErr(t, raw)
		if e.Error.Code != "DUPLICATE_ID_NUMBER" {
			t.Fatalf("%+v", e)
		}
	})

	t.Run("empty idNumber twice", func(t *testing.T) {
		body := `{"name":"无证","sex":"U","birthday":"2001-02-03","fareIdentity":"SELF","idNumber":""}`
		code1, raw1, _ := postJSON(t, base, body, authHdr(tok))
		code2, raw2, _ := postJSON(t, base, body, authHdr(tok))
		if code1 != 201 || code2 != 201 {
			t.Fatalf("%d %s / %d %s", code1, raw1, code2, raw2)
		}
		var a, b patientJSON
		if err := json.Unmarshal(raw1, &a); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw2, &b); err != nil {
			t.Fatal(err)
		}
		if a.PatientID == "" || a.PatientID == b.PatientID || a.IDNumber != "" {
			t.Fatalf("%+v %+v", a, b)
		}
	})

	t.Run("duplicate card", func(t *testing.T) {
		body := `{"name":"卡重复","sex":"F","birthday":"1988-08-08","cardNo":"C1","fareIdentity":"SELF"}`
		code, raw, _ := postJSON(t, base, body, authHdr(tok))
		if code != 409 {
			t.Fatalf("status %d %s", code, raw)
		}
		e := decodeErr(t, raw)
		if e.Error.Code != "DUPLICATE_CARD_NO" {
			t.Fatalf("%+v", e)
		}
	})

	t.Run("get and search", func(t *testing.T) {
		if created.PatientID == "" {
			t.Fatal("create case did not run")
		}
		code, body := getAuth(t, base+"/"+created.PatientID, tok)
		if code != 200 {
			t.Fatalf("get %d %s", code, body)
		}
		var dto patientJSON
		if err := json.Unmarshal(body, &dto); err != nil {
			t.Fatal(err)
		}
		if dto.PatientID != created.PatientID || dto.HospPkNo != created.HospPkNo {
			t.Fatalf("%+v", dto)
		}
		code, body = getAuth(t, base+"/00000000-0000-4000-8000-000000009999", tok)
		if code != 404 {
			t.Fatalf("missing %d %s", code, body)
		}
		e := decodeErr(t, body)
		if e.Error.Code != "NOT_FOUND" || e.Success || e.Data != nil {
			t.Fatalf("%+v", e)
		}
		code, body = getAuth(t, base+"?"+url.Values{"name": {"张"}, "cardNo": {"C1"}}.Encode(), tok)
		if code != 200 {
			t.Fatalf("search %d %s", code, body)
		}
		var list struct {
			Items []patientJSON `json:"items"`
		}
		if err := json.Unmarshal(body, &list); err != nil {
			t.Fatal(err)
		}
		if len(list.Items) != 1 || list.Items[0].PatientID != created.PatientID {
			t.Fatalf("%+v", list)
		}
		code, body = getAuth(t, base+"?"+url.Values{"address": {"浦"}}.Encode(), tok)
		if code != 200 {
			t.Fatalf("address %d %s", code, body)
		}
		if err := json.Unmarshal(body, &list); err != nil {
			t.Fatal(err)
		}
		if len(list.Items) != 1 {
			t.Fatalf("address hits %+v", list.Items)
		}
	})

	t.Run("pharmacist forbidden", func(t *testing.T) {
		pharm := domain.User{
			ID: "00000000-0000-4000-8000-000000000003", Username: "pharm", DisplayName: "药房",
			Roles: []domain.Role{domain.RolePharmacist}, Status: domain.StatusActive,
		}
		st.Add(pharm)
		code, body, _ := postJSON(t, base, `{"name":"张三","sex":"M","birthday":"1990-01-15","fareIdentity":"SELF"}`, authHdr(bearerFor(t, issuer, pharm)))
		if code != 403 {
			t.Fatalf("status %d %s", code, body)
		}
		e := decodeErr(t, body)
		if e.Error.Code != "FORBIDDEN" {
			t.Fatalf("%+v", e)
		}
		code, body = getAuth(t, base+"?"+url.Values{"name": {"张"}}.Encode(), bearerFor(t, issuer, pharm))
		if code != 200 {
			t.Fatalf("search %d %s", code, body)
		}
	})
}

type patientJSON struct {
	PatientID    string `json:"patientId"`
	HospPkNo     string `json:"hospPkNo"`
	PatientNo    string `json:"patientNo"`
	CardNo       string `json:"cardNo"`
	Name         string `json:"name"`
	Sex          string `json:"sex"`
	Birthday     string `json:"birthday"`
	IDNumber     string `json:"idNumber"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	FareIdentity string `json:"fareIdentity"`
	CardStatus   string `json:"cardStatus"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
}

func flatFields(e errDTO) string {
	raw, _ := e.Error.Details["fields"].([]any)
	parts := make([]string, 0, len(raw))
	for _, v := range raw {
		parts = append(parts, strings.TrimSpace(stringify(v)))
	}
	return strings.Join(parts, ",")
}

func stringify(v any) string {
	s, _ := v.(string)
	return s
}
