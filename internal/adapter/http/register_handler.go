package httpapi

import (
	"net/http"
	"strings"

	"github.com/cznUSER/B-HIS/internal/app"
)

func (h *Handler) createRegister(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	var in app.CreateRegisterInput
	if err := decodeJSON(r, &in); err != nil {
		writeAppError(w, r, err)
		return
	}
	in.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	dto, err := h.svc.CreateRegister(r.Context(), claims, in, requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *Handler) searchRegisters(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	q := r.URL.Query()
	list, err := h.svc.SearchRegisters(r.Context(), claims, app.RegisterSearch{
		PatientID:  q.Get("patientId"),
		RegisterNo: q.Get("registerNo"),
		VisitDate:  q.Get("visitDate"),
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) getRegister(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.GetRegister(r.Context(), claims, r.PathValue("registerId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) checkInRegister(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.CheckInRegister(r.Context(), claims, r.PathValue("registerId"), requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) cancelRegister(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.CancelRegister(r.Context(), claims, r.PathValue("registerId"), requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}
