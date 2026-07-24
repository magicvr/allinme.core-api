package response

import "net/http"

// Envelope is the standard success body used by Admin APIs (GOAL-002 D-015).
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK writes a code=0 success envelope.
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, Envelope{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}
