package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/app"
)

func TestAuthLoginMeMenu(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DB.SQLitePath = t.TempDir() + "/auth.db"
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Auth.JWTTTL = time.Hour

	application, err := app.New(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close() })

	// login
	body := bytes.NewBufferString(`{"username":"admin","password":"Demo@1234"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	rr := httptest.NewRecorder()
	application.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", rr.Code, rr.Body.String())
	}
	var loginEnv struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&loginEnv); err != nil {
		t.Fatal(err)
	}
	if loginEnv.Code != 0 || loginEnv.Data.AccessToken == "" {
		t.Fatalf("login env = %+v", loginEnv)
	}

	// me
	req = httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+loginEnv.Data.AccessToken)
	rr = httptest.NewRecorder()
	application.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", rr.Code, rr.Body.String())
	}

	// menu
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/menu", nil)
	req.Header.Set("Authorization", "Bearer "+loginEnv.Data.AccessToken)
	rr = httptest.NewRecorder()
	application.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("menu status = %d body=%s", rr.Code, rr.Body.String())
	}

	// unauthorized me
	req = httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	rr = httptest.NewRecorder()
	application.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("me without token status = %d", rr.Code)
	}
}
