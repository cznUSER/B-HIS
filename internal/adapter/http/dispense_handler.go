package httpapi

// Dispense and inventory HTTP handlers. 2026-10-12

import (
	"net/http"
	"strings"

	"github.com/cznUSER/B-HIS/internal/app"
)

func (h *Handler) createDispense(w http.ResponseWriter, r *http.Request) {
	h.writeDispense(w, r, "", http.StatusCreated)
}

func (h *Handler) dispenseOrderAlias(w http.ResponseWriter, r *http.Request) {
	h.writeDispense(w, r, r.PathValue("orderId"), http.StatusOK)
}

func (h *Handler) writeDispense(w http.ResponseWriter, r *http.Request, pathOrderID string, status int) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	var in app.CreateDispenseInput
	if err := decodeJSON(r, &in); err != nil {
		writeAppError(w, r, err)
		return
	}
	in.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	in.PathOrderID = pathOrderID
	dto, err := h.svc.CreateDispense(r.Context(), claims, in, requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, status, dto)
}

func (h *Handler) searchDispenses(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	q := r.URL.Query()
	list, err := h.svc.SearchDispenses(r.Context(), claims, app.DispenseSearch{
		PatientID:  q.Get("patientId"),
		RegisterID: q.Get("registerId"),
		ChargeID:   q.Get("chargeId"),
		OrderID:    q.Get("orderId"),
		VisitDate:  q.Get("visitDate"),
		From:       q.Get("from"),
		To:         q.Get("to"),
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) getDispense(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.GetDispense(r.Context(), claims, r.PathValue("dispenseId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) returnDispense(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	var in app.ReturnDispenseInput
	if err := decodeJSON(r, &in); err != nil {
		writeAppError(w, r, err)
		return
	}
	in.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	dto, err := h.svc.ReturnDispense(r.Context(), claims, r.PathValue("dispenseId"), in, requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) listInventory(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	q := r.URL.Query()
	list, err := h.svc.ListInventory(r.Context(), claims, app.InventorySearch{
		SkuCode: q.Get("skuCode"),
		ItemNo:  q.Get("itemNo"),
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
