package httpapi

import (
	"net/http"
)

func (h *Handler) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dto, err := h.svc.DashboardSummary(r.Context(), claims, r.URL.Query().Get("period"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}
