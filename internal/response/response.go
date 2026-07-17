package response

import (
	"encoding/json"
	"net/http"
)

// ErrorBody is the standard API error envelope for Admin and clients.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{
		Code:    code,
		Message: message,
	})
}

// JSON writes a successful JSON response.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
