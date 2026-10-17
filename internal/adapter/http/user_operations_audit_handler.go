package httpapi

import (
	"net/http"

	"github.com/cznUSER/B-HIS/internal/app"
)

func (h *Handler) listUserOperations(w http.ResponseWriter, r *http.Request) {
	claims, err := h.bearerClaims(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		limit, err = parsePositiveInt(raw)
		if err != nil {
			writeAppError(w, r, app.Validation("limit"))
			return
		}
	}
	dto, err := h.svc.ListUserOperations(r.Context(), claims, app.UserOperationFilter{
		ModuleID: q.Get("moduleId"), UserID: q.Get("userId"), Begin: q.Get("begin"), End: q.Get("end"),
		Cursor: q.Get("cursor"), Limit: limit,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func parsePositiveInt(raw string) (int, error) {
	value := 0
	for _, digit := range raw {
		if digit < '0' || digit > '9' || value > 100000 {
			return 0, app.Validation("limit")
		}
		value = value*10 + int(digit-'0')
	}
	if value < 1 {
		return 0, app.Validation("limit")
	}
	return value, nil
}
