package httpapi

import "github.com/cznUSER/B-HIS/internal/app"

func structAppError() *app.Error {
	return &app.Error{Status: 500, Code: "INTERNAL_ERROR", Message: "内部错误"}
}
