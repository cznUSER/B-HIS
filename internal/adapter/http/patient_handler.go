package httpapi

import (
	"net/http"
	"strings"

	"github.com/cznUSER/B-HIS/internal/app"
)

func (h *Handler) createPatient(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	var in app.CreatePatientInput
	if err := decodeJSON(r, &in); err != nil {
		writeAppError(w, r, err)
		return
	}
	in.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	dto, err := h.svc.CreatePatient(r.Context(), claims, in, requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *Handler) searchPatients(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	q := r.URL.Query()
	list, err := h.svc.SearchPatients(r.Context(), claims, app.PatientSearch{
		CardNo:   q.Get("cardNo"),
		IDNumber: q.Get("idNumber"),
		Name:     q.Get("name"),
		HospPkNo: q.Get("hospPkNo"),
		Address:  q.Get("address"),
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) getPatient(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.GetPatient(r.Context(), claims, r.PathValue("patientId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}
