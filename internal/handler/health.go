package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/magicvr/allinme.core-api/internal/service/meta"
)

type healthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ok",
			Timestamp: time.Now().UTC(),
		})
	})
}

func readyz(metaSvc *meta.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if metaSvc != nil {
			if err := metaSvc.Ready(r.Context()); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, healthResponse{
					Status:    "not_ready",
					Timestamp: time.Now().UTC(),
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ready",
			Timestamp: time.Now().UTC(),
		})
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
