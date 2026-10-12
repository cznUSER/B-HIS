package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/logging"
)

type ctxKey int

const requestIDKey ctxKey = 1

func requestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if !validRequestID(id) {
			id = newRequestID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validRequestID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '.' || c == '_' || c == '-':
		default:
			return false
		}
	}
	return true
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func cors(origins []string, next http.Handler) http.Handler {
	allow := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allow[o] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allow[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id, Idempotency-Key")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func accessLog(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		if log == nil {
			return
		}
		level := "info"
		outcome := "SUCCESS"
		if sw.status >= 500 {
			level = "error"
			outcome = "FAIL"
		} else if sw.status >= 400 {
			level = "warn"
			outcome = "FAIL"
		}
		log.Log(logging.Event{
			Level:      level,
			RequestID:  requestID(r.Context()),
			Action:     "http.request",
			Outcome:    outcome,
			DurationMS: time.Since(start).Milliseconds(),
			Message:    r.Method + " " + r.URL.Path,
		})
	})
}

func recoverPanic(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if log != nil {
					log.Log(logging.Event{
						Level:     "error",
						RequestID: requestID(r.Context()),
						Action:    "http.panic",
						Outcome:   "FAIL",
						Message:   string(debug.Stack()),
					})
				}
				writeError(w, r, structAppError())
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	if w.wrote {
		return
	}
	w.status = code
	w.wrote = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.status = http.StatusOK
		w.wrote = true
	}
	return w.ResponseWriter.Write(b)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
