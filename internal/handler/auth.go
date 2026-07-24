package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/magicvr/allinme.core-api/internal/middleware"
	"github.com/magicvr/allinme.core-api/internal/port"
	"github.com/magicvr/allinme.core-api/internal/response"
	"github.com/magicvr/allinme.core-api/internal/service/auth"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginData struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
	User        any    `json:"user"`
}

func login(authSvc *auth.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if req.Username == "" || req.Password == "" {
			response.Error(w, http.StatusBadRequest, "bad_request", "username and password required")
			return
		}
		res, err := authSvc.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			if errors.Is(err, port.ErrInvalidCredentials) {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid username or password")
				return
			}
			response.Error(w, http.StatusInternalServerError, "internal", "login failed")
			return
		}
		response.OK(w, loginData{
			AccessToken: res.AccessToken,
			ExpiresIn:   res.ExpiresIn,
			User:        res.User,
		})
	})
}

func me(authSvc *auth.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.UserFromContext(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
			return
		}
		pub, err := authSvc.Me(r.Context(), user.ID)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid session")
			return
		}
		response.OK(w, pub)
	})
}
