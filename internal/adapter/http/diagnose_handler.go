package httpapi

import (
	"net/http"

	"github.com/cznUSER/B-HIS/internal/app"
)

func (h *Handler) createDiagnose(w http.ResponseWriter, r *http.Request) {
	h.createDiagnoseForRegister(w, r, r.PathValue("registerId"))
}

func (h *Handler) createDiagnoseEncounterAlias(w http.ResponseWriter, r *http.Request) {
	h.createDiagnoseForRegister(w, r, r.PathValue("encounterId"))
}

func (h *Handler) createDiagnoseForRegister(w http.ResponseWriter, r *http.Request, registerID string) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	var in app.CreateDiagnoseInput
	if err := decodeJSON(r, &in); err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.CreateDiagnose(r.Context(), claims, registerID, in, requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *Handler) listDiagnoses(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.ListDiagnoses(r.Context(), claims, r.PathValue("registerId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) commitDiagnose(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.CommitDiagnose(r.Context(), claims, r.PathValue("diagnoseId"), requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) abateDiagnose(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.AbateDiagnose(r.Context(), claims, r.PathValue("diagnoseId"), requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}
