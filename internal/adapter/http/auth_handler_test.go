package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/app"
	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
	"golang.org/x/crypto/bcrypt"
)

type loginDTO struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int64  `json:"expiresIn"`
	User        struct {
		ID          string   `json:"id"`
		Username    string   `json:"username"`
		DisplayName string   `json:"displayName"`
		Roles       []string `json:"roles"`
	} `json:"user"`
}

type errDTO struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
	Error   struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
	RequestID string `json:"requestId"`
}

func newServer(t *testing.T, maxFail int) (*httptest.Server, *memory.Store, *auth.Issuer, string) {
	t.Helper()
	dir := t.TempDir()
	log, err := logging.New(dir, "info")
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
	svc := &app.Service{
		Users:       st,
		Revocations: st,
		Audit:       st,
		Tokens:      issuer,
		Limiter:     auth.NewLimiter(maxFail, 15*time.Minute),
		Log:         log,
		TTL:         ttl,
	}
	h := New(svc, log, []string{"http://localhost:5173"}, "memory", 15*time.Minute)
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts, st, issuer, dir
}

func postJSON(t *testing.T, url, body string, hdr map[string]string) (int, []byte, http.Header) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
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
	return resp.StatusCode, b, resp.Header
}

func getAuth(t *testing.T, url, token string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func loginBody(username, password string) string {
	return fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
}

func TestLoginSuccess(t *testing.T) {
	ts, _, _, dir := newServer(t, 8)
	code, body, _ := postJSON(t, ts.URL+"/api/v1/auth/login",
		loginBody(seed.DemoUsername, seed.DemoPassword), nil)
	if code != 200 {
		t.Fatalf("status %d body %s", code, body)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["success"]; ok {
		t.Fatalf("login 200 must be LoginResponse, got %#v", raw)
	}
	var got loginDTO
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.TokenType != "Bearer" || got.ExpiresIn != 28800 || got.AccessToken == "" {
		t.Fatalf("%+v", got)
	}
	if got.User.ID != seed.DemoUserID || got.User.Username != seed.DemoUsername || got.User.DisplayName != seed.DemoDisplayName {
		t.Fatalf("%+v", got.User)
	}
	if len(got.User.Roles) != 1 || got.User.Roles[0] != "SYSTEM" {
		t.Fatalf("roles %#v", got.User.Roles)
	}
	logs := readLogs(t, dir)
	if !strings.Contains(logs, "LOGIN_SUCCESS") {
		t.Fatalf("missing LOGIN_SUCCESS: %s", logs)
	}
	if strings.Contains(logs, "Admin@123") || strings.Contains(logs, "admin123") || strings.Contains(logs, `"password"`) {
		t.Fatalf("password leaked into log: %s", logs)
	}
}

func TestWrongPasswordAndUnknownUserSameError(t *testing.T) {
	ts, _, _, dir := newServer(t, 8)
	code1, body1, _ := postJSON(t, ts.URL+"/api/v1/auth/login",
		loginBody(seed.DemoUsername, "WrongPass!9"), map[string]string{"X-Request-Id": "req-1"})
	code2, body2, _ := postJSON(t, ts.URL+"/api/v1/auth/login",
		`{"username":"nobody","password":"WrongPass!9"}`, nil)
	if code1 != 401 || code2 != 401 {
		t.Fatalf("codes %d %d", code1, code2)
	}
	var a, b errDTO
	if err := json.Unmarshal(body1, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body2, &b); err != nil {
		t.Fatal(err)
	}
	if a.Success || a.Data != nil || a.Error.Code != "INVALID_CREDENTIALS" {
		t.Fatalf("%+v", a)
	}
	if a.Error.Message != b.Error.Message || a.Error.Code != b.Error.Code {
		t.Fatalf("enumeration leak %s vs %s", body1, body2)
	}
	if a.RequestID != "req-1" {
		t.Fatalf("request id %q", a.RequestID)
	}
	logs := readLogs(t, dir)
	if !strings.Contains(logs, "LOGIN_FAILURE") || strings.Contains(logs, "WrongPass!9") {
		t.Fatalf("log: %s", logs)
	}
}

func TestMeWithoutToken(t *testing.T) {
	ts, _, _, _ := newServer(t, 8)
	code, body := getAuth(t, ts.URL+"/api/v1/auth/me", "")
	if code != 401 {
		t.Fatalf("status %d %s", code, body)
	}
	var e errDTO
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatal(err)
	}
	if e.Error.Code != "UNAUTHORIZED" || e.Success || e.Data != nil {
		t.Fatalf("%+v", e)
	}
}

func TestMeWithTokenAndLogout(t *testing.T) {
	ts, _, _, dir := newServer(t, 8)
	_, body, _ := postJSON(t, ts.URL+"/api/v1/auth/login",
		loginBody(seed.DemoUsername, seed.DemoPassword), nil)
	var logged loginDTO
	if err := json.Unmarshal(body, &logged); err != nil {
		t.Fatal(err)
	}
	code, me := getAuth(t, ts.URL+"/api/v1/auth/me", logged.AccessToken)
	if code != 200 {
		t.Fatalf("me %d %s", code, me)
	}
	var user struct {
		ID       string   `json:"id"`
		Username string   `json:"username"`
		Roles    []string `json:"roles"`
	}
	if err := json.Unmarshal(me, &user); err != nil {
		t.Fatal(err)
	}
	if user.ID != seed.DemoUserID || user.Username != seed.DemoUsername || len(user.Roles) != 1 {
		t.Fatalf("%+v", user)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+logged.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 204 || len(bytes.TrimSpace(out)) != 0 {
		t.Fatalf("logout %d %q", resp.StatusCode, out)
	}
	code, again := getAuth(t, ts.URL+"/api/v1/auth/me", logged.AccessToken)
	if code != 401 {
		t.Fatalf("after logout %d %s", code, again)
	}
	var e errDTO
	_ = json.Unmarshal(again, &e)
	if e.Error.Code != "TOKEN_INVALID" {
		t.Fatalf("%+v", e)
	}
	if !strings.Contains(readLogs(t, dir), "LOGOUT") {
		t.Fatal("missing LOGOUT audit")
	}
}

func TestTokenExpiredAndInvalid(t *testing.T) {
	ts, st, issuer, _ := newServer(t, 8)
	user, err := st.FindByUsername(nil, seed.DemoUsername)
	if err != nil {
		t.Fatal(err)
	}
	tok, _, _, err := issuer.Issue(*user, time.Now().Add(-9*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	code, body := getAuth(t, ts.URL+"/api/v1/auth/me", tok)
	if code != 401 {
		t.Fatalf("expired %d %s", code, body)
	}
	var e errDTO
	_ = json.Unmarshal(body, &e)
	if e.Error.Code != "TOKEN_EXPIRED" {
		t.Fatalf("%+v", e)
	}

	code, body = getAuth(t, ts.URL+"/api/v1/auth/me", "not-a-jwt")
	_ = json.Unmarshal(body, &e)
	if code != 401 || e.Error.Code != "TOKEN_INVALID" {
		t.Fatalf("%d %+v", code, e)
	}
}

func TestAccountDisabled(t *testing.T) {
	ts, _, _, _ := newServer(t, 8)
	code, body, _ := postJSON(t, ts.URL+"/api/v1/auth/login",
		loginBody(seed.DisabledUsername, seed.DemoPassword), nil)
	var e errDTO
	_ = json.Unmarshal(body, &e)
	if code != 403 || e.Error.Code != "ACCOUNT_DISABLED" {
		t.Fatalf("%d %+v", code, e)
	}
	code, body, _ = postJSON(t, ts.URL+"/api/v1/auth/login",
		`{"username":"disabled","password":"nope"}`, nil)
	_ = json.Unmarshal(body, &e)
	if code != 401 || e.Error.Code != "INVALID_CREDENTIALS" {
		t.Fatalf("disabled wrong password %d %+v", code, e)
	}
}

func TestRateLimitAndValidation(t *testing.T) {
	ts, _, _, _ := newServer(t, 2)
	for i := 0; i < 2; i++ {
		code, body, _ := postJSON(t, ts.URL+"/api/v1/auth/login",
			loginBody(seed.DemoUsername, "nope"), nil)
		var e errDTO
		_ = json.Unmarshal(body, &e)
		if code != 401 || e.Error.Code != "INVALID_CREDENTIALS" {
			t.Fatalf("attempt %d %d %+v", i, code, e)
		}
	}
	code, body, hdr := postJSON(t, ts.URL+"/api/v1/auth/login",
		loginBody(seed.DemoUsername, "nope"), nil)
	var e errDTO
	_ = json.Unmarshal(body, &e)
	if code != 429 || e.Error.Code != "AUTH_RATE_LIMITED" {
		t.Fatalf("%d %+v", code, e)
	}
	if hdr.Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}

	code, body, _ = postJSON(t, ts.URL+"/api/v1/auth/login", `{}`, nil)
	_ = json.Unmarshal(body, &e)
	if code != 400 || e.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("validation %d %+v", code, e)
	}
}

func TestCORSPreflight(t *testing.T) {
	ts, _, _, _ := newServer(t, 8)
	req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("acao %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestDemoHashMatchesPassword(t *testing.T) {
	if seed.DemoUsername != "system" || seed.DemoPassword != "1" || seed.StaffDefaultPassword != "1" {
		t.Fatal("documented demo account must be system / staff default password 1")
	}
	if bcrypt.CompareHashAndPassword([]byte(seed.DemoPasswordHash), []byte(seed.DemoPassword)) != nil {
		t.Fatal("demo hash does not match documented memory password")
	}
	if bcrypt.CompareHashAndPassword([]byte(seed.DisabledUser().PasswordHash), []byte(seed.StaffDefaultPassword)) != nil {
		t.Fatal("disabled account must use the staff default password")
	}
	if auth.CheckPassword(auth.DummyHash, seed.DemoPassword) {
		t.Fatal("dummy hash must not accept the demo password")
	}
	_ = domain.RoleSystem
}

func TestOpenAPIAuthSection(t *testing.T) {
	b := readRepoFile(t, "api/openapi.yaml")
	for _, s := range []string{
		"version: 0.3.10-draft",
		"contractVersion:",
		"/auth/login:",
		"/auth/logout:",
		"/auth/me:",
		"INVALID_CREDENTIALS",
		"TOKEN_EXPIRED",
		"TOKEN_INVALID",
		"UNAUTHORIZED",
		"ACCOUNT_DISABLED",
		"AUTH_RATE_LIMITED",
		"accessToken:",
		"expiresIn:",
	} {
		if !strings.Contains(b, s) {
			t.Errorf("openapi missing %s", s)
		}
	}
}

func readLogs(t *testing.T, dir string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "h-api-*.log"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("logs: %v %v", err, matches)
	}
	var b strings.Builder
	for _, m := range matches {
		raw, err := os.ReadFile(m)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(raw)
	}
	return b.String()
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			b, err := os.ReadFile(filepath.Join(dir, rel))
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
