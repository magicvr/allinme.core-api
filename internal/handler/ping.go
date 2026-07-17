package handler

import (
	"log/slog"
	"net/http"
	"time"
)

type pingResponse struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func ping(logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("ping", "remote", r.RemoteAddr)
		writeJSON(w, http.StatusOK, pingResponse{
			Message:   "pong",
			Timestamp: time.Now().UTC(),
		})
	})
}
