package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/app"
	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/logging"
)

// Handler serves Auth and Patients routes. Other outpatient paths are not registered.
type Handler struct {
	svc        *app.Service
	log        *logging.Logger
	driver     string
	rateWindow time.Duration
}

func New(svc *app.Service, log *logging.Logger, origins []string, driver string, rateWindow time.Duration) http.Handler {
	h := &Handler{svc: svc, log: log, driver: driver, rateWindow: rateWindow}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
	mux.HandleFunc("POST /api/v1/patients", h.createPatient)
	mux.HandleFunc("GET /api/v1/patients", h.searchPatients)
	mux.HandleFunc("GET /api/v1/patients/{patientId}", h.getPatient)
	mux.HandleFunc("GET /health", h.health)
	return withRequestID(recoverPanic(log, accessLog(log, cors(origins, mux))))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeAppError(w, r, err)
		return
	}
	res, err := h.svc.Login(r.Context(), req.Username, req.Password, requestID(r.Context()), clientIP(r))
	if err != nil {
		var ae *app.Error
		if errors.As(err, &ae) && ae.Code == app.CodeRateLimited && h.rateWindow > 0 {
			w.Header().Set("Retry-After", formatSeconds(h.rateWindow))
		}
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	if err := h.svc.Logout(r.Context(), claims, requestID(r.Context()), clientIP(r)); err != nil {
		writeAppError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	user, err := h.svc.Me(r.Context(), claims)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "0.3.5-draft",
		"driver":  h.driver,
	})
}

func (h *Handler) bearerClaims(r *http.Request) (*auth.Claims, error) {
	raw, err := bearerToken(r)
	if err != nil {
		return nil, err
	}
	return h.svc.Authenticate(r.Context(), raw)
}

func bearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if strings.TrimSpace(h) == "" {
		return "", app.Unauthorized()
	}
	parts := strings.SplitN(strings.TrimSpace(h), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", app.TokenInvalid()
	}
	tok := strings.TrimSpace(parts[1])
	if tok == "" {
		return "", app.Unauthorized()
	}
	return tok, nil
}

func decodeJSON(r *http.Request, dst any) error {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(strings.ToLower(ct), "application/json") {
		return app.Validation("content-type")
	}
	body := http.MaxBytesReader(nil, r.Body, 1<<20)
	defer body.Close()
	dec := json.NewDecoder(body)
	if err := dec.Decode(dst); err != nil {
		return app.Validation("json")
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return app.Validation("json")
	}
	return nil
}

func formatSeconds(d time.Duration) string {
	sec := int(d.Seconds())
	if sec < 1 {
		sec = 1
	}
	return itoa(sec)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
