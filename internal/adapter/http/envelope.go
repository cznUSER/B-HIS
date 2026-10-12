package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cznUSER/B-HIS/internal/app"
)

type errorBody struct {
	Success   bool        `json:"success"`
	Data      any         `json:"data"`
	Error     errorDetail `json:"error"`
	RequestID string      `json:"requestId"`
}

type errorDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeAppError(w http.ResponseWriter, r *http.Request, err error) {
	var ae *app.Error
	if errors.As(err, &ae) {
		writeError(w, r, ae)
		return
	}
	writeError(w, r, &app.Error{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "内部错误"})
}

func writeError(w http.ResponseWriter, r *http.Request, ae *app.Error) {
	details := ae.Details
	if details == nil {
		details = map[string]any{}
	}
	status := ae.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, errorBody{
		Success: false,
		Data:    nil,
		Error: errorDetail{
			Code:    ae.Code,
			Message: ae.Message,
			Details: details,
		},
		RequestID: requestID(r.Context()),
	})
}
