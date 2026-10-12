package httpapi

// Order HTTP handlers. 2026-10-12

import (
	"net/http"
	"strings"

	"github.com/cznUSER/B-HIS/internal/app"
)

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	var in app.CreateOrderInput
	if err := decodeJSON(r, &in); err != nil {
		writeAppError(w, r, err)
		return
	}
	in.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	dto, err := h.svc.CreateOrder(r.Context(), claims, in, requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *Handler) searchOrders(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	q := r.URL.Query()
	list, err := h.svc.SearchOrders(r.Context(), claims, app.OrderSearch{
		RegisterID:  q.Get("registerId"),
		EncounterID: q.Get("encounterId"),
		PatientID:   q.Get("patientId"),
		OrderNo:     q.Get("orderNo"),
		Status:      q.Get("status"),
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.GetOrder(r.Context(), claims, r.PathValue("orderId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) patchOrder(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	var in app.PatchOrderInput
	if err := decodeJSON(r, &in); err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.PatchOrder(r.Context(), claims, r.PathValue("orderId"), in, requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) commitOrder(w http.ResponseWriter, r *http.Request) {
	h.commitOrderAction(w, r)
}

func (h *Handler) submitOrder(w http.ResponseWriter, r *http.Request) {
	h.commitOrderAction(w, r)
}

func (h *Handler) commitOrderAction(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.CommitOrder(r.Context(), claims, r.PathValue("orderId"), requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) abateOrder(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.AbateOrder(r.Context(), claims, r.PathValue("orderId"), requestID(r.Context()), clientIP(r))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) orderNotImplemented(w http.ResponseWriter, r *http.Request) {
	if _, err := h.bearerClaims(r); err != nil {
		writeAppError(w, r, err)
		return
	}
	writeAppError(w, r, app.NotImplemented())
}
