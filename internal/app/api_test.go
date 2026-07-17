package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/admin"
	"github.com/magicvr/allinme.core-api/internal/app"
	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/config"
	"github.com/magicvr/allinme.core-api/internal/order"
	"github.com/magicvr/allinme.core-api/internal/store"
)

func TestAPIReadinessTransitionsWithoutRestart(t *testing.T) {
	dataDir := t.TempDir()
	configuration, err := config.Load(mapLookup(map[string]string{"DATA_DIR": dataDir}))
	if err != nil {
		t.Fatal(err)
	}
	application, err := app.NewAPI(configuration, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(application.Close)

	if status := requestStatus(application.Handler(), "/healthz"); status != http.StatusOK {
		t.Fatalf("health status = %d", status)
	}
	if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusServiceUnavailable {
		t.Fatalf("initial ready status = %d", status)
	}

	database, err := store.Open(context.Background(), filepath.Join(dataDir, "allinme.db"), store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(context.Background()); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.Seed(context.Background()); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusOK {
		t.Fatalf("ready status after migrate = %d", status)
	}
	application.Close()
	if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusServiceUnavailable {
		t.Fatalf("ready status after close = %d", status)
	}
	reopened, err := app.NewAPI(configuration, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if status := requestStatus(reopened.Handler(), "/readyz"); status != http.StatusOK {
		t.Fatalf("ready status after reopen = %d", status)
	}
}

func TestAuthenticatedAPIFlowWithSQLite(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	dataDir := t.TempDir()
	base, err := config.LoadBase(mapLookup(map[string]string{"DATA_DIR": dataDir}))
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, base.DatabasePath, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.Seed(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	passwords, err := auth.NewPasswords()
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.SeedAuthDemo(ctx, passwords, "123456789012", now, auth.RandomID); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(ctx, now); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	configuration := config.APIConfig{Config: base, JWTSigningKey: []byte("12345678901234567890123456789012"), CORSAllowedOrigin: "https://ui.example.com"}
	sequence := 0
	dependencies := app.AuthDependencies{
		Clock: func() time.Time { return now }, LimiterClock: func() time.Time { return now },
		NewID: func() (string, error) { sequence++; return "id-" + strconv.Itoa(sequence), nil },
	}
	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(application.Close)
	preflight := httptest.NewRequest(http.MethodOptions, "/api/v1/orders", nil)
	preflight.Header.Set("Origin", configuration.CORSAllowedOrigin)
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflight.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, Idempotency-Key")
	preflightResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent || preflightResponse.Header().Get("Access-Control-Allow-Origin") != configuration.CORSAllowedOrigin {
		t.Fatalf("CORS preflight = %d headers=%v body=%s", preflightResponse.Code, preflightResponse.Header(), preflightResponse.Body.String())
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"viewer","password":"123456789012"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login = %d %s", loginResponse.Code, loginResponse.Body.String())
	}
	var login struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(loginResponse.Body).Decode(&login); err != nil || login.AccessToken == "" {
		t.Fatalf("decode login: %v", err)
	}

	requestWithToken := func(method, path string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, nil)
		request.Header.Set("Authorization", "Bearer "+login.AccessToken)
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response, request)
		return response
	}
	if response := requestWithToken(http.MethodGet, "/api/v1/auth/me"); response.Code != http.StatusOK {
		t.Fatalf("me = %d %s", response.Code, response.Body.String())
	}
	if response := requestWithToken(http.MethodGet, "/api/v1/orders?pageSize=2&status=DRAFT"); response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"total":1`)) || bytes.Contains(response.Body.Bytes(), []byte(`"items":[{"id":"itm_`)) {
		t.Fatalf("orders = %d %s", response.Code, response.Body.String())
	}
	crossOriginRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orders?pageSize=1", nil)
	crossOriginRequest.Header.Set("Authorization", "Bearer "+login.AccessToken)
	crossOriginRequest.Header.Set("Origin", configuration.CORSAllowedOrigin)
	crossOriginResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(crossOriginResponse, crossOriginRequest)
	if crossOriginResponse.Code != http.StatusOK || crossOriginResponse.Header().Get("Access-Control-Allow-Origin") != configuration.CORSAllowedOrigin || crossOriginResponse.Header().Get("Access-Control-Expose-Headers") != "X-Request-ID, Content-Disposition" {
		t.Fatalf("cross origin orders = %d headers=%v body=%s", crossOriginResponse.Code, crossOriginResponse.Header(), crossOriginResponse.Body.String())
	}
	if response := requestWithToken(http.MethodGet, "/api/v1/orders/ord_00000000000000000000000000000001"); response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"items":[{"id":"itm_`)) {
		t.Fatalf("order detail = %d %s", response.Code, response.Body.String())
	}
	if response := requestWithToken(http.MethodHead, "/api/v1/orders"); response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("orders HEAD = %d Allow=%q %s", response.Code, response.Header().Get("Allow"), response.Body.String())
	}
	if response := requestWithToken(http.MethodPost, "/api/v1/auth/logout"); response.Code != http.StatusNoContent {
		t.Fatalf("logout = %d %s", response.Code, response.Body.String())
	}
	if response := requestWithToken(http.MethodGet, "/api/v1/auth/me"); response.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout = %d %s", response.Code, response.Body.String())
	}
	if status := requestStatus(application.Handler(), "/healthz"); status != http.StatusOK {
		t.Fatalf("health = %d", status)
	}
	if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusOK {
		t.Fatalf("readiness = %d", status)
	}
	roleTokens := map[string]string{}
	for _, role := range []string{"viewer", "operator", "approver", "admin"} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"`+role+`","password":"123456789012"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"role":"`+role+`"`)) {
			t.Fatalf("%s login = %d %s", role, response.Code, response.Body.String())
		}
		var roleLogin struct {
			AccessToken string `json:"accessToken"`
		}
		if err := json.NewDecoder(response.Body).Decode(&roleLogin); err != nil || roleLogin.AccessToken == "" {
			t.Fatalf("decode %s login: %v", role, err)
		}
		roleTokens[role] = roleLogin.AccessToken
		roleRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orders?pageSize=1", nil)
		roleRequest.Header.Set("Authorization", "Bearer "+roleLogin.AccessToken)
		roleResponse := httptest.NewRecorder()
		application.Handler().ServeHTTP(roleResponse, roleRequest)
		if roleResponse.Code != http.StatusOK {
			t.Fatalf("%s order list = %d %s", role, roleResponse.Code, roleResponse.Body.String())
		}
		if role == "admin" {
			login.AccessToken = roleLogin.AccessToken
		}
	}
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(`{"customerName":"Created","currency":"CNY","items":[{"sku":"NEW","name":"New Item","quantity":2,"unitPrice":300}]}`))
	createRequest.Header.Set("Authorization", "Bearer "+roleTokens["operator"])
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Idempotency-Key", "app-create-1")
	createResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated || !bytes.Contains(createResponse.Body.Bytes(), []byte(`"totalAmount":600`)) {
		t.Fatalf("create order = %d %s", createResponse.Code, createResponse.Body.String())
	}
	var created struct {
		ID      string `json:"id"`
		Version int64  `json:"version"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil || created.ID == "" || created.Version != 1 {
		t.Fatalf("decode created: %+v %v", created, err)
	}
	replayRequest := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(`{"items":[{"unitPrice":300,"quantity":2,"name":"New Item","sku":"NEW"}],"currency":"CNY","customerName":" Created "}`))
	replayRequest.Header = createRequest.Header.Clone()
	replayResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(replayResponse, replayRequest)
	if replayResponse.Code != http.StatusCreated || !bytes.Contains(replayResponse.Body.Bytes(), []byte(`"id":"`+created.ID+`"`)) {
		t.Fatalf("replay = %d %s", replayResponse.Code, replayResponse.Body.String())
	}
	conflictRequest := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(`{"customerName":"Different","currency":"CNY","items":[{"sku":"NEW","name":"New Item","quantity":2,"unitPrice":300}]}`))
	conflictRequest.Header = createRequest.Header.Clone()
	conflictResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(conflictResponse, conflictRequest)
	if conflictResponse.Code != http.StatusConflict || !bytes.Contains(conflictResponse.Body.Bytes(), []byte(`"code":"IDEMPOTENCY_CONFLICT"`)) {
		t.Fatalf("idempotency conflict = %d %s", conflictResponse.Code, conflictResponse.Body.String())
	}
	editRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/orders/"+created.ID, bytes.NewBufferString(`{"customerName":"Edited","currency":"CNY","items":[{"sku":"EDIT","name":"Edited Item","quantity":1,"unitPrice":700}],"version":1}`))
	editRequest.Header.Set("Authorization", "Bearer "+roleTokens["admin"])
	editRequest.Header.Set("Content-Type", "application/json")
	editResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(editResponse, editRequest)
	if editResponse.Code != http.StatusOK || !bytes.Contains(editResponse.Body.Bytes(), []byte(`"version":2`)) {
		t.Fatalf("edit = %d %s", editResponse.Code, editResponse.Body.String())
	}
	replayAfterEdit := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(`{"customerName":"Created","currency":"CNY","items":[{"sku":"NEW","name":"New Item","quantity":2,"unitPrice":300}]}`))
	replayAfterEdit.Header = createRequest.Header.Clone()
	replayAfterEditResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(replayAfterEditResponse, replayAfterEdit)
	if replayAfterEditResponse.Code != http.StatusCreated || !bytes.Contains(replayAfterEditResponse.Body.Bytes(), []byte(`"version":1`)) || bytes.Contains(replayAfterEditResponse.Body.Bytes(), []byte(`"customerName":"Edited"`)) {
		t.Fatalf("replay after edit = %d %s", replayAfterEditResponse.Code, replayAfterEditResponse.Body.String())
	}
	for _, role := range []string{"viewer", "approver"} {
		denied := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(`{"customerName":"Denied","currency":"CNY","items":[{"sku":"D","name":"Denied","quantity":1,"unitPrice":1}]}`))
		denied.Header.Set("Authorization", "Bearer "+roleTokens[role])
		denied.Header.Set("Content-Type", "application/json")
		denied.Header.Set("Idempotency-Key", "denied-"+role)
		deniedResponse := httptest.NewRecorder()
		application.Handler().ServeHTTP(deniedResponse, denied)
		if deniedResponse.Code != http.StatusForbidden {
			t.Fatalf("%s create = %d %s", role, deniedResponse.Code, deniedResponse.Body.String())
		}
	}
	version := int64(2)
	for _, step := range []struct{ action, status string }{{"confirm", "CONFIRMED"}, {"fulfill", "FULFILLING"}, {"ship", "SHIPPED"}, {"complete", "COMPLETED"}} {
		action := step.action
		actionRequest := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+created.ID+"/"+action, bytes.NewBufferString(`{"version":`+strconv.FormatInt(version, 10)+`}`))
		actionRequest.Header.Set("Authorization", "Bearer "+roleTokens["operator"])
		actionRequest.Header.Set("Content-Type", "application/json")
		actionResponse := httptest.NewRecorder()
		application.Handler().ServeHTTP(actionResponse, actionRequest)
		if actionResponse.Code != http.StatusOK || !bytes.Contains(actionResponse.Body.Bytes(), []byte(`"version":`+strconv.FormatInt(version+1, 10))) || !bytes.Contains(actionResponse.Body.Bytes(), []byte(`"status":"`+step.status+`"`)) {
			t.Fatalf("%s action = %d %s", action, actionResponse.Code, actionResponse.Body.String())
		}
		version++
	}
	cancelRequest := httptest.NewRequest(http.MethodPost, "/api/v1/orders/ord_00000000000000000000000000000001/cancel", bytes.NewBufferString(`{"version":1}`))
	cancelRequest.Header.Set("Authorization", "Bearer "+roleTokens["operator"])
	cancelRequest.Header.Set("Content-Type", "application/json")
	cancelResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(cancelResponse, cancelRequest)
	if cancelResponse.Code != http.StatusOK || !bytes.Contains(cancelResponse.Body.Bytes(), []byte(`"status":"CANCELLED"`)) || !bytes.Contains(cancelResponse.Body.Bytes(), []byte(`"version":2`)) {
		t.Fatalf("cancel action = %d %s", cancelResponse.Code, cancelResponse.Body.String())
	}
	staleAction := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+created.ID+"/cancel", bytes.NewBufferString(`{"version":2}`))
	staleAction.Header.Set("Authorization", "Bearer "+roleTokens["admin"])
	staleAction.Header.Set("Content-Type", "application/json")
	staleResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(staleResponse, staleAction)
	if staleResponse.Code != http.StatusConflict || !bytes.Contains(staleResponse.Body.Bytes(), []byte(`"code":"VERSION_CONFLICT"`)) {
		t.Fatalf("stale action = %d %s", staleResponse.Code, staleResponse.Body.String())
	}

	application.Close()
	reopened, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+login.AccessToken)
	response := httptest.NewRecorder()
	reopened.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("same-key reopen = %d %s", response.Code, response.Body.String())
	}
	reopened.Close()

	disabledDependencies := dependencies
	disabledDependencies.DisableOrderActions = true
	reopened, err = app.NewAuthenticatedAPIWithDependencies(configuration, disabledDependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	disabledCreate := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(`{"customerName":"Rollback","currency":"CNY","items":[{"sku":"R","name":"Rollback","quantity":1,"unitPrice":1}]}`))
	disabledCreate.Header.Set("Authorization", "Bearer "+roleTokens["operator"])
	disabledCreate.Header.Set("Content-Type", "application/json")
	disabledCreate.Header.Set("Idempotency-Key", "rollback-create")
	disabledCreateResponse := httptest.NewRecorder()
	reopened.Handler().ServeHTTP(disabledCreateResponse, disabledCreate)
	if disabledCreateResponse.Code != http.StatusCreated {
		t.Fatalf("rollback create = %d %s", disabledCreateResponse.Code, disabledCreateResponse.Body.String())
	}
	var rollbackCreated struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(disabledCreateResponse.Body).Decode(&rollbackCreated); err != nil || rollbackCreated.ID == "" {
		t.Fatalf("decode rollback create: %+v %v", rollbackCreated, err)
	}
	disabledEdit := httptest.NewRequest(http.MethodPatch, "/api/v1/orders/"+rollbackCreated.ID, bytes.NewBufferString(`{"customerName":"Rollback Edited","currency":"CNY","items":[{"sku":"R","name":"Rollback","quantity":1,"unitPrice":2}],"version":1}`))
	disabledEdit.Header.Set("Authorization", "Bearer "+roleTokens["admin"])
	disabledEdit.Header.Set("Content-Type", "application/json")
	disabledEditResponse := httptest.NewRecorder()
	reopened.Handler().ServeHTTP(disabledEditResponse, disabledEdit)
	if disabledEditResponse.Code != http.StatusOK {
		t.Fatalf("rollback edit = %d %s", disabledEditResponse.Code, disabledEditResponse.Body.String())
	}
	for _, action := range []string{"confirm", "fulfill", "ship", "complete", "cancel"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+rollbackCreated.ID+"/"+action, bytes.NewBufferString(`{"version":2}`))
		request.Header.Set("Authorization", "Bearer "+roleTokens["admin"])
		request.Header.Set("Content-Type", "application/json")
		reopened.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("disabled %s = %d %s", action, response.Code, response.Body.String())
		}
	}
	reopened.Close()

	otherKey := configuration
	otherKey.JWTSigningKey = []byte("abcdefghijklmnopqrstuvwxyz123456")
	reopened, err = app.NewAuthenticatedAPIWithDependencies(otherKey, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	response = httptest.NewRecorder()
	reopened.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("different-key reopen = %d %s", response.Code, response.Body.String())
	}
}

func TestProductionBootstrapToAuthenticatedAPILogin(t *testing.T) {
	dataDir := t.TempDir()
	values := map[string]string{
		"APP_ENV": "production", "PORT": "8080", "DATA_DIR": dataDir,
		"BOOTSTRAP_ADMIN_USERNAME": " Root ", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012",
		"JWT_SIGNING_KEY": "12345678901234567890123456789012",
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"migrate"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	if err := admin.Execute(context.Background(), mapLookup(values), []string{"bootstrap-admin"}, io.Discard, nil); err == nil {
		t.Fatal("repeat bootstrap error = nil")
	}
	configuration, err := config.LoadAPI(mapLookup(values))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	sequence := 0
	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, app.AuthDependencies{
		Clock: func() time.Time { return now }, LimiterClock: func() time.Time { return now },
		NewID: func() (string, error) { sequence++; return "production-id-" + strconv.Itoa(sequence), nil },
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	response := loginResponse(application.Handler(), "root", "123456789012", "production-login")
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"username":"root"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"role":"admin"`)) {
		t.Fatalf("production login = %d %s", response.Code, response.Body.String())
	}
}

func TestAuthenticatedAPIRejectsNegativeIdentityAndSessionStates(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	clock := now
	dataDir := t.TempDir()
	base, err := config.LoadBase(mapLookup(map[string]string{"DATA_DIR": dataDir}))
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, base.DatabasePath, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.Seed(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	passwords, _ := auth.NewPasswords()
	seedSequence := 0
	if _, err := database.SeedAuthDemo(ctx, passwords, "123456789012", now, func() (string, error) {
		seedSequence++
		return "user-" + strconv.Itoa(seedSequence), nil
	}); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	configuration := config.APIConfig{Config: base, JWTSigningKey: []byte("12345678901234567890123456789012")}
	idSequence := 0
	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, app.AuthDependencies{
		Clock: func() time.Time { return clock }, LimiterClock: func() time.Time { return clock },
		NewID: func() (string, error) { idSequence++; return "session-id-" + strconv.Itoa(idSequence), nil },
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	wrong := loginResponse(application.Handler(), "viewer", "wrong-password", "same-failure")
	unknown := loginResponse(application.Handler(), "unknown", "123456789012", "same-failure")
	if wrong.Code != http.StatusUnauthorized || unknown.Code != http.StatusUnauthorized || wrong.Body.String() != unknown.Body.String() {
		t.Fatalf("login failures differ: %d %s / %d %s", wrong.Code, wrong.Body.String(), unknown.Code, unknown.Body.String())
	}

	login := loginResponse(application.Handler(), "viewer", "123456789012", "valid-login")
	if login.Code != http.StatusOK {
		t.Fatalf("login = %d %s", login.Code, login.Body.String())
	}
	var body struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(login.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	requestMe := func(token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response, request)
		return response
	}
	if response := requestMe(body.AccessToken + "tampered"); response.Code != http.StatusUnauthorized {
		t.Fatalf("tampered token status = %d", response.Code)
	}

	mutator, err := store.Open(ctx, base.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	defer mutator.Close()
	if _, err := mutator.SQL().Exec(`UPDATE sessions SET user_id = (SELECT id FROM users WHERE username = 'operator') WHERE user_id = (SELECT id FROM users WHERE username = 'viewer')`); err != nil {
		t.Fatal(err)
	}
	if response := requestMe(body.AccessToken); response.Code != http.StatusUnauthorized {
		t.Fatalf("subject mismatch status = %d", response.Code)
	}
	if _, err := mutator.SQL().Exec(`UPDATE sessions SET user_id = (SELECT id FROM users WHERE username = 'viewer')`); err != nil {
		t.Fatal(err)
	}
	if _, err := mutator.SQL().Exec(`UPDATE users SET disabled_at = ? WHERE username = 'viewer'`, now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if response := requestMe(body.AccessToken); response.Code != http.StatusUnauthorized {
		t.Fatalf("disabled user status = %d", response.Code)
	}
	if _, err := mutator.SQL().Exec(`UPDATE users SET disabled_at = NULL WHERE username = 'viewer'`); err != nil {
		t.Fatal(err)
	}
	clock = now.Add(auth.TokenTTL + auth.ClockLeeway)
	if response := requestMe(body.AccessToken); response.Code != http.StatusUnauthorized {
		t.Fatalf("expired token status = %d", response.Code)
	}
}

func loginResponse(handler http.Handler, username, password, requestID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"`+username+`","password":"`+password+`"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", requestID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestAuthenticatedAPIRefundFlowWithSQLite(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)
	base, err := config.LoadBase(mapLookup(map[string]string{"DATA_DIR": t.TempDir()}))
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, base.DatabasePath, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.Seed(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	passwords, err := auth.NewPasswords()
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.SeedAuthDemo(ctx, passwords, "123456789012", now, auth.RandomID); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.SeedOrderDemo(ctx, now); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.SeedRefundDemo(ctx, now); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	configuration := config.APIConfig{Config: base, JWTSigningKey: []byte("12345678901234567890123456789012"), CORSAllowedOrigin: "https://ui.example.com"}
	authSequence := 0
	refundIDs := []string{
		"rfd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"rfd_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"rfd_cccccccccccccccccccccccccccccccc",
	}
	refundIndex := 0
	dependencies := app.AuthDependencies{
		Clock: func() time.Time { return now }, LimiterClock: func() time.Time { return now }, RefundClock: func() time.Time { return now },
		DashboardClock: func() time.Time { return time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC) },
		NewID:          func() (string, error) { authSequence++; return "refund-flow-id-" + strconv.Itoa(authSequence), nil },
		NewRefundID: func() (string, error) {
			if refundIndex >= len(refundIDs) {
				return "", errors.New("refund ID sequence exhausted")
			}
			value := refundIDs[refundIndex]
			refundIndex++
			return value, nil
		},
	}
	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	login := func(username string) string {
		response := loginResponse(application.Handler(), username, "123456789012", "login-"+username)
		if response.Code != http.StatusOK {
			t.Fatalf("%s login = %d %s", username, response.Code, response.Body.String())
		}
		var body struct {
			AccessToken string `json:"accessToken"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil || body.AccessToken == "" {
			t.Fatalf("decode %s login: %v", username, err)
		}
		return body.AccessToken
	}
	tokens := map[string]string{}
	for _, role := range []string{"viewer", "operator", "approver", "admin"} {
		tokens[role] = login(role)
	}
	request := func(token, method, path, body, key string) *httptest.ResponseRecorder {
		value := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		value.Header.Set("Authorization", "Bearer "+token)
		if body != "" {
			value.Header.Set("Content-Type", "application/json")
		}
		if key != "" {
			value.Header.Set("Idempotency-Key", key)
		}
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response, value)
		return response
	}
	for _, role := range []string{"viewer", "operator", "approver", "admin"} {
		summary := request(tokens[role], http.MethodGet, "/api/v1/dashboard/summary", "", "")
		if summary.Code != http.StatusOK || summary.Body.String() != "{\"orderCount\":10,\"grossAmount\":460000,\"completedRefundAmount\":120000,\"netAmount\":340000,\"currency\":\"CNY\"}\n" {
			t.Fatalf("%s dashboard summary = %d %s", role, summary.Code, summary.Body.String())
		}
	}
	statusSnapshot := request(tokens["viewer"], http.MethodGet, "/api/v1/dashboard/order-status", "", "")
	if statusSnapshot.Code != http.StatusOK || statusSnapshot.Body.String() != "{\"items\":[{\"status\":\"DRAFT\",\"count\":1},{\"status\":\"CONFIRMED\",\"count\":1},{\"status\":\"FULFILLING\",\"count\":2},{\"status\":\"SHIPPED\",\"count\":2},{\"status\":\"COMPLETED\",\"count\":2},{\"status\":\"CANCELLED\",\"count\":2}]}\n" {
		t.Fatalf("dashboard status snapshot = %d %s", statusSnapshot.Code, statusSnapshot.Body.String())
	}
	trendSnapshot := request(tokens["viewer"], http.MethodGet, "/api/v1/dashboard/trend?days=7", "", "")
	if trendSnapshot.Code != http.StatusOK || !bytes.Contains(trendSnapshot.Body.Bytes(), []byte(`"startDate":"2026-01-01"`)) || !bytes.Contains(trendSnapshot.Body.Bytes(), []byte(`{"date":"2026-01-01","orderCount":10,"grossAmount":460000,"completedRefundAmount":0,"netAmount":460000}`)) || !bytes.Contains(trendSnapshot.Body.Bytes(), []byte(`{"date":"2026-01-02","orderCount":0,"grossAmount":0,"completedRefundAmount":120000,"netAmount":-120000}`)) {
		t.Fatalf("dashboard trend snapshot = %d %s", trendSnapshot.Code, trendSnapshot.Body.String())
	}
	orderID := "ord_00000000000000000000000000000007"
	createBody := `{"amount":10000,"reason":" customer request ","orderVersion":1}`
	invalidUTF8 := string(append([]byte(`{"amount":10000,"reason":"`), append([]byte{0xff}, []byte(`","orderVersion":1}`)...)...))
	invalidCreate := request(tokens["operator"], http.MethodPost, "/api/v1/orders/"+orderID+"/refunds", invalidUTF8, "invalid-utf8-create")
	if invalidCreate.Code != http.StatusBadRequest {
		t.Fatalf("invalid UTF-8 create = %d %s", invalidCreate.Code, invalidCreate.Body.String())
	}
	created := request(tokens["operator"], http.MethodPost, "/api/v1/orders/"+orderID+"/refunds", createBody, "refund-flow-1")
	if created.Code != http.StatusCreated || !bytes.Contains(created.Body.Bytes(), []byte(`"id":"`+refundIDs[0]+`"`)) || !bytes.Contains(created.Body.Bytes(), []byte(`"status":"PENDING"`)) || !bytes.Contains(created.Body.Bytes(), []byte(`"canApprove":false`)) {
		t.Fatalf("create refund = %d %s", created.Code, created.Body.String())
	}
	replay := request(tokens["operator"], http.MethodPost, "/api/v1/orders/"+orderID+"/refunds", `{"orderVersion":1,"reason":"customer request","amount":10000}`, "refund-flow-1")
	if replay.Code != http.StatusCreated || replay.Body.String() != created.Body.String() {
		t.Fatalf("refund replay = %d %s", replay.Code, replay.Body.String())
	}
	conflict := request(tokens["operator"], http.MethodPost, "/api/v1/orders/"+orderID+"/refunds", `{"amount":9999,"reason":"customer request","orderVersion":1}`, "refund-flow-1")
	if conflict.Code != http.StatusConflict || !bytes.Contains(conflict.Body.Bytes(), []byte(`"code":"IDEMPOTENCY_CONFLICT"`)) {
		t.Fatalf("refund conflict = %d %s", conflict.Code, conflict.Body.String())
	}
	pendingDetail := request(tokens["operator"], http.MethodGet, "/api/v1/orders/"+orderID, "", "")
	if pendingDetail.Code != http.StatusOK || !bytes.Contains(pendingDetail.Body.Bytes(), []byte(`"availableRefundAmount":60000`)) || !bytes.Contains(pendingDetail.Body.Bytes(), []byte(`"canRequestRefund":true`)) || !bytes.Contains(pendingDetail.Body.Bytes(), []byte(`"version":1`)) {
		t.Fatalf("pending refund order detail = %d %s", pendingDetail.Code, pendingDetail.Body.String())
	}
	for _, role := range []string{"viewer", "approver"} {
		denied := request(tokens[role], http.MethodPost, "/api/v1/orders/"+orderID+"/refunds", createBody, "denied-"+role)
		if denied.Code != http.StatusForbidden {
			t.Fatalf("%s create refund = %d %s", role, denied.Code, denied.Body.String())
		}
	}
	for _, role := range []string{"viewer", "operator"} {
		denied := request(tokens[role], http.MethodPost, "/api/v1/refunds/"+refundIDs[0]+"/approve", `{"version":1}`, "")
		if denied.Code != http.StatusForbidden {
			t.Fatalf("%s approve refund = %d %s", role, denied.Code, denied.Body.String())
		}
	}
	selfApprove := request(tokens["admin"], http.MethodPost, "/api/v1/refunds/rfd_00000000000000000000000000000002/approve", `{"version":1}`, "")
	if selfApprove.Code != http.StatusForbidden {
		t.Fatalf("admin self approve = %d %s", selfApprove.Code, selfApprove.Body.String())
	}
	versionBeforeState := request(tokens["approver"], http.MethodPost, "/api/v1/refunds/rfd_00000000000000000000000000000003/approve", `{"version":1}`, "")
	if versionBeforeState.Code != http.StatusConflict || !bytes.Contains(versionBeforeState.Body.Bytes(), []byte(`"code":"VERSION_CONFLICT"`)) {
		t.Fatalf("version before state = %d %s", versionBeforeState.Code, versionBeforeState.Body.String())
	}
	stateBeforeSelf := request(tokens["admin"], http.MethodPost, "/api/v1/refunds/rfd_00000000000000000000000000000005/approve", `{"version":2}`, "")
	if stateBeforeSelf.Code != http.StatusConflict || !bytes.Contains(stateBeforeSelf.Body.Bytes(), []byte(`"code":"STATE_CONFLICT"`)) {
		t.Fatalf("state before self approval = %d %s", stateBeforeSelf.Code, stateBeforeSelf.Body.String())
	}
	mutator, err := store.Open(ctx, base.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mutator.SQL().Exec(`UPDATE orders SET payment_status = 'REFUNDED' WHERE id = 'ord_00000000000000000000000000000008'`); err != nil {
		mutator.Close()
		t.Fatal(err)
	}
	mutator.Close()
	rejectWithCorruptOrder := request(tokens["approver"], http.MethodPost, "/api/v1/refunds/rfd_00000000000000000000000000000001/reject", `{"version":1}`, "")
	if rejectWithCorruptOrder.Code != http.StatusOK || !bytes.Contains(rejectWithCorruptOrder.Body.Bytes(), []byte(`"status":"REJECTED"`)) {
		t.Fatalf("reject with corrupt order = %d %s", rejectWithCorruptOrder.Code, rejectWithCorruptOrder.Body.String())
	}
	mutator, err = store.Open(ctx, base.DatabasePath, store.OpenExisting)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mutator.SQL().Exec(`UPDATE orders SET payment_status = 'PAID' WHERE id = 'ord_00000000000000000000000000000008'`); err != nil {
		mutator.Close()
		t.Fatal(err)
	}
	mutator.Close()
	invalidApprove := request(tokens["approver"], http.MethodPost, "/api/v1/refunds/"+refundIDs[0]+"/approve", string([]byte{'{', '"', 'v', 'e', 'r', 's', 'i', 'o', 'n', '"', ':', '"', 0xff, '"', '}'}), "")
	if invalidApprove.Code != http.StatusBadRequest {
		t.Fatalf("invalid UTF-8 approve = %d %s", invalidApprove.Code, invalidApprove.Body.String())
	}
	approved := request(tokens["approver"], http.MethodPost, "/api/v1/refunds/"+refundIDs[0]+"/approve", `{"version":1}`, "")
	if approved.Code != http.StatusOK || !bytes.Contains(approved.Body.Bytes(), []byte(`"status":"COMPLETED"`)) || !bytes.Contains(approved.Body.Bytes(), []byte(`"version":2`)) {
		t.Fatalf("approve refund = %d %s", approved.Code, approved.Body.String())
	}
	replayAfterApprove := request(tokens["operator"], http.MethodPost, "/api/v1/orders/"+orderID+"/refunds", createBody, "refund-flow-1")
	if replayAfterApprove.Code != http.StatusCreated || replayAfterApprove.Body.String() != created.Body.String() {
		t.Fatalf("replay after approve = %d %s", replayAfterApprove.Code, replayAfterApprove.Body.String())
	}
	summaryAfterFirstApproval := request(tokens["viewer"], http.MethodGet, "/api/v1/dashboard/summary", "", "")
	if summaryAfterFirstApproval.Code != http.StatusOK || summaryAfterFirstApproval.Body.String() != "{\"orderCount\":10,\"grossAmount\":460000,\"completedRefundAmount\":130000,\"netAmount\":330000,\"currency\":\"CNY\"}\n" {
		t.Fatalf("summary after first approval = %d %s", summaryAfterFirstApproval.Code, summaryAfterFirstApproval.Body.String())
	}
	second := request(tokens["admin"], http.MethodPost, "/api/v1/orders/"+orderID+"/refunds", `{"amount":5000,"reason":"second refund","orderVersion":2}`, "refund-flow-2")
	if second.Code != http.StatusCreated || !bytes.Contains(second.Body.Bytes(), []byte(`"id":"`+refundIDs[1]+`"`)) {
		t.Fatalf("second refund = %d %s", second.Code, second.Body.String())
	}
	secondApprove := request(tokens["approver"], http.MethodPost, "/api/v1/refunds/"+refundIDs[1]+"/approve", `{"version":1}`, "")
	if secondApprove.Code != http.StatusOK {
		t.Fatalf("second approve = %d %s", secondApprove.Code, secondApprove.Body.String())
	}
	third := request(tokens["operator"], http.MethodPost, "/api/v1/orders/"+orderID+"/refunds", `{"amount":1000,"reason":"reject this","orderVersion":3}`, "refund-flow-3")
	if third.Code != http.StatusCreated || !bytes.Contains(third.Body.Bytes(), []byte(`"id":"`+refundIDs[2]+`"`)) {
		t.Fatalf("third refund = %d %s", third.Code, third.Body.String())
	}
	invalidReject := request(tokens["approver"], http.MethodPost, "/api/v1/refunds/"+refundIDs[2]+"/reject", string([]byte{'{', '"', 'v', 'e', 'r', 's', 'i', 'o', 'n', '"', ':', '"', 0xff, '"', '}'}), "")
	if invalidReject.Code != http.StatusBadRequest {
		t.Fatalf("invalid UTF-8 reject = %d %s", invalidReject.Code, invalidReject.Body.String())
	}
	rejected := request(tokens["approver"], http.MethodPost, "/api/v1/refunds/"+refundIDs[2]+"/reject", `{"version":1}`, "")
	if rejected.Code != http.StatusOK || !bytes.Contains(rejected.Body.Bytes(), []byte(`"status":"REJECTED"`)) {
		t.Fatalf("reject refund = %d %s", rejected.Code, rejected.Body.String())
	}
	detail := request(tokens["operator"], http.MethodGet, "/api/v1/orders/"+orderID, "", "")
	if detail.Code != http.StatusOK || !bytes.Contains(detail.Body.Bytes(), []byte(`"paymentStatus":"PARTIALLY_REFUNDED"`)) || !bytes.Contains(detail.Body.Bytes(), []byte(`"availableRefundAmount":55000`)) || !bytes.Contains(detail.Body.Bytes(), []byte(`"version":3`)) || !bytes.Contains(detail.Body.Bytes(), []byte(`"canRequestRefund":true`)) {
		t.Fatalf("refund order detail = %d %s", detail.Code, detail.Body.String())
	}
	list := request(tokens["approver"], http.MethodGet, "/api/v1/refunds?pageSize=20", "", "")
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"total":8`)) || !bytes.Contains(list.Body.Bytes(), []byte(`"canApprove":true`)) {
		t.Fatalf("refund list = %d %s", list.Code, list.Body.String())
	}
	updatedSummary := request(tokens["viewer"], http.MethodGet, "/api/v1/dashboard/summary", "", "")
	if updatedSummary.Code != http.StatusOK || updatedSummary.Body.String() != "{\"orderCount\":10,\"grossAmount\":460000,\"completedRefundAmount\":135000,\"netAmount\":325000,\"currency\":\"CNY\"}\n" {
		t.Fatalf("updated dashboard summary = %d %s", updatedSummary.Code, updatedSummary.Body.String())
	}

	application.Close()
	reopened, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	application = reopened
	detailAfterRestart := request(tokens["operator"], http.MethodGet, "/api/v1/orders/"+orderID, "", "")
	if detailAfterRestart.Code != http.StatusOK || !bytes.Contains(detailAfterRestart.Body.Bytes(), []byte(`"availableRefundAmount":55000`)) || !bytes.Contains(detailAfterRestart.Body.Bytes(), []byte(`"version":3`)) {
		t.Fatalf("refund detail after restart = %d %s", detailAfterRestart.Code, detailAfterRestart.Body.String())
	}
	application.Close()

	disabledDependencies := dependencies
	disabledDependencies.DisableRefundRoutes = true
	disabled, err := app.NewAuthenticatedAPIWithDependencies(configuration, disabledDependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer disabled.Close()
	application = disabled
	if response := request(tokens["operator"], http.MethodGet, "/api/v1/orders/"+orderID, "", ""); response.Code != http.StatusOK {
		t.Fatalf("orders with refund routes disabled = %d %s", response.Code, response.Body.String())
	}
	if response := request(tokens["approver"], http.MethodGet, "/api/v1/refunds", "", ""); response.Code != http.StatusNotFound {
		t.Fatalf("disabled refund list = %d %s", response.Code, response.Body.String())
	}
	if response := request(tokens["viewer"], http.MethodGet, "/api/v1/dashboard/summary", "", ""); response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"netAmount":325000`)) {
		t.Fatalf("dashboard with refund routes disabled = %d %s", response.Code, response.Body.String())
	}
	for _, path := range []string{
		"/api/v1/orders/" + orderID + "/refunds",
		"/api/v1/refunds/" + refundIDs[0] + "/approve",
		"/api/v1/refunds/" + refundIDs[2] + "/reject",
	} {
		if response := request(tokens["admin"], http.MethodPost, path, `{"version":1}`, "disabled-key"); response.Code != http.StatusNotFound {
			t.Fatalf("disabled refund route %s = %d %s", path, response.Code, response.Body.String())
		}
	}
	disabled.Close()
	dashboardDisabledDependencies := dependencies
	dashboardDisabledDependencies.DisableDashboardRoutes = true
	dashboardDisabled, err := app.NewAuthenticatedAPIWithDependencies(configuration, dashboardDisabledDependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboardDisabled.Close()
	application = dashboardDisabled
	if response := request(tokens["approver"], http.MethodGet, "/api/v1/refunds", "", ""); response.Code != http.StatusOK {
		t.Fatalf("refund list with dashboard disabled = %d %s", response.Code, response.Body.String())
	}
	for _, path := range []string{"/api/v1/dashboard/summary", "/api/v1/dashboard/order-status", "/api/v1/dashboard/trend?days=7"} {
		if response := request(tokens["viewer"], http.MethodGet, path, "", ""); response.Code != http.StatusNotFound {
			t.Fatalf("disabled dashboard route %s = %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestAuthenticatedAPIAttachmentCompositionWithDependencies(t *testing.T) {
	ctx := context.Background()
	authNow := time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)
	attachmentNow := time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC)
	base, err := config.LoadBase(mapLookup(map[string]string{"DATA_DIR": t.TempDir()}))
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, base.DatabasePath, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	passwords, err := auth.NewPasswords()
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.SeedAuthDemo(ctx, passwords, "123456789012", authNow, auth.RandomID); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	configuration := config.APIConfig{Config: base, JWTSigningKey: []byte("12345678901234567890123456789012")}
	attachmentID := "att_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	attachmentFiles := &appAttachmentFiles{}
	dependencies := app.AuthDependencies{
		Clock:               func() time.Time { return authNow },
		LimiterClock:        func() time.Time { return authNow },
		NewID:               func() (string, error) { return "attachment-session-id", nil },
		AttachmentClock:     func() time.Time { return attachmentNow },
		NewAttachmentID:     func() (string, error) { return attachmentID, nil },
		AttachmentFileStore: attachmentFiles,
	}
	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	login := loginResponse(application.Handler(), "operator", "123456789012", "attachment-login")
	if login.Code != http.StatusOK {
		application.Close()
		t.Fatalf("login = %d %s", login.Code, login.Body.String())
	}
	var session struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(login.Body).Decode(&session); err != nil || session.AccessToken == "" {
		application.Close()
		t.Fatalf("decode login: %v", err)
	}

	content := []byte("%PDF-1.4\napp attachment composition")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "invoice.pdf")
	if err != nil {
		application.Close()
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		application.Close()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		application.Close()
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/attachments", &body)
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !bytes.Contains(response.Body.Bytes(), []byte(`"id":"`+attachmentID+`"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"createdAt":"2026-07-17T09:30:00Z"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"expiresAt":"2026-07-18T09:30:00Z"`)) {
		application.Close()
		t.Fatalf("upload = %d %s", response.Code, response.Body.String())
	}
	if attachmentFiles.storageKey != attachmentID || attachmentFiles.fileName != "invoice.pdf" || !bytes.Equal(attachmentFiles.content, content) {
		application.Close()
		t.Fatalf("file write = key %q name %q content %q", attachmentFiles.storageKey, attachmentFiles.fileName, attachmentFiles.content)
	}
	if _, err := os.Stat(filepath.Join(base.DataDir, "attachments")); !os.IsNotExist(err) {
		application.Close()
		t.Fatalf("injected file store created local storage: %v", err)
	}
	application.Close()

	dependencies.DisableAttachmentRoutes = true
	disabled, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer disabled.Close()
	disabledRequest := httptest.NewRequest(http.MethodPost, "/api/v1/attachments", nil)
	disabledRequest.Header.Set("Authorization", "Bearer "+session.AccessToken)
	disabledResponse := httptest.NewRecorder()
	disabled.Handler().ServeHTTP(disabledResponse, disabledRequest)
	if disabledResponse.Code != http.StatusNotFound {
		t.Fatalf("disabled attachment route = %d %s", disabledResponse.Code, disabledResponse.Body.String())
	}
}

func TestAuthenticatedAPIAttachmentRequiredFlowWithLocalFilesAndSQLite(t *testing.T) {
	ctx := context.Background()
	authNow := time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC)
	attachmentNow := time.Now().UTC().Truncate(time.Second)
	base, err := config.LoadBase(mapLookup(map[string]string{"DATA_DIR": t.TempDir()}))
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, base.DatabasePath, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	passwords, err := auth.NewPasswords()
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	seedID := 0
	if _, err := database.SeedAuthDemo(ctx, passwords, "123456789012", authNow, func() (string, error) {
		seedID++
		return "attachment-flow-user-" + strconv.Itoa(seedID), nil
	}); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	attachmentIDs := []string{
		"att_00000000000000000000000000000071",
		"att_00000000000000000000000000000072",
	}
	authID := 0
	attachmentID := 0
	dependencies := app.AuthDependencies{
		Clock:        func() time.Time { return authNow },
		LimiterClock: func() time.Time { return authNow },
		NewID: func() (string, error) {
			authID++
			return "attachment-flow-session-" + strconv.Itoa(authID), nil
		},
		AttachmentClock: func() time.Time { return attachmentNow },
		NewAttachmentID: func() (string, error) {
			if attachmentID >= len(attachmentIDs) {
				return "", errors.New("attachment ID sequence exhausted")
			}
			value := attachmentIDs[attachmentID]
			attachmentID++
			return value, nil
		},
	}
	configuration := config.APIConfig{Config: base, JWTSigningKey: []byte("12345678901234567890123456789012")}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if application != nil {
			application.Close()
		}
	}()

	operatorToken := appAttachmentFlowLogin(t, application.Handler(), "operator")
	viewerToken := appAttachmentFlowLogin(t, application.Handler(), "viewer")
	pdfs := [][]byte{appAttachmentFlowPDF("PLN-0007 bound"), appAttachmentFlowPDF("PLN-0007 unbound")}
	fileNames := []string{"pln-0007-bound.pdf", "pln-0007-unbound.pdf"}
	uploaded := make([]appAttachmentFlowDTO, 0, len(pdfs))
	for index, content := range pdfs {
		result := appAttachmentFlowUpload(t, application.Handler(), operatorToken, fileNames[index], content)
		if result.ID != attachmentIDs[index] || result.FileName != fileNames[index] || result.ContentType != "application/pdf" || result.SizeBytes != int64(len(content)) || result.CreatedAt != order.FormatTime(attachmentNow) || result.ExpiresAt != order.FormatTime(attachmentNow.Add(order.AttachmentUploadLifetime)) {
			t.Fatalf("upload %d result = %+v", index+1, result)
		}
		stored, err := os.ReadFile(filepath.Join(base.DataDir, "attachments", "content", result.ID))
		if err != nil || !bytes.Equal(stored, content) {
			t.Fatalf("stored upload %d: error=%v content=%q", index+1, err, stored)
		}
		uploaded = append(uploaded, result)
	}

	createBody := []byte(`{"customerName":"PLN-0007 Required Flow","currency":"CNY","items":[{"sku":"PLN-0007-A","name":"Required item","quantity":2,"unitPrice":1250}],"attachmentIds":["` + attachmentIDs[0] + `"]}`)
	createdResponse := appAttachmentFlowRequest(application.Handler(), operatorToken, http.MethodPost, "/api/v1/orders", "application/json", "pln-0007-create", createBody)
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create order = %d %s", createdResponse.Code, createdResponse.Body.String())
	}
	createdJSON := bytes.Clone(createdResponse.Body.Bytes())
	var created appAttachmentFlowOrderDTO
	if err := json.Unmarshal(createdJSON, &created); err != nil || created.ID == "" {
		t.Fatalf("decode created order: %+v %v", created, err)
	}
	replay := appAttachmentFlowRequest(application.Handler(), operatorToken, http.MethodPost, "/api/v1/orders", "application/json", "pln-0007-create", createBody)
	if replay.Code != http.StatusCreated || !bytes.Equal(replay.Body.Bytes(), createdJSON) {
		t.Fatalf("create replay = %d %s", replay.Code, replay.Body.String())
	}

	listResponse := appAttachmentFlowRequest(application.Handler(), viewerToken, http.MethodGet, "/api/v1/orders?q=PLN-0007", "", "", nil)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("order list = %d %s", listResponse.Code, listResponse.Body.String())
	}
	var list struct {
		Items []appAttachmentFlowOrderDTO `json:"items"`
		Total int64                       `json:"total"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&list); err != nil || list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != created.ID || list.Items[0].AttachmentCount != 1 {
		t.Fatalf("order list = %+v error=%v", list, err)
	}

	detailResponse := appAttachmentFlowRequest(application.Handler(), viewerToken, http.MethodGet, "/api/v1/orders/"+created.ID, "", "", nil)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("order detail = %d %s", detailResponse.Code, detailResponse.Body.String())
	}
	var detail appAttachmentFlowOrderDTO
	if err := json.NewDecoder(detailResponse.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	wantItem := appAttachmentFlowItemDTO{SKU: "PLN-0007-A", Name: "Required item", Quantity: 2, UnitPrice: 1250}
	if detail.ID != created.ID || detail.CustomerName != "PLN-0007 Required Flow" || detail.Status != "DRAFT" || detail.Currency != "CNY" || detail.TotalAmount != 2500 || detail.AttachmentCount != 1 || len(detail.Items) != 1 || detail.Items[0] != wantItem || len(detail.Attachments) != 1 {
		t.Fatalf("order detail = %+v", detail)
	}
	boundSummary := detail.Attachments[0]
	if boundSummary.ID != uploaded[0].ID || boundSummary.FileName != uploaded[0].FileName || boundSummary.ContentType != uploaded[0].ContentType || boundSummary.SizeBytes != uploaded[0].SizeBytes || boundSummary.SHA256 != uploaded[0].SHA256 || boundSummary.CreatedAt != uploaded[0].CreatedAt {
		t.Fatalf("bound attachment summary = %+v upload=%+v", boundSummary, uploaded[0])
	}

	appAttachmentFlowAssertDownload(t, application.Handler(), viewerToken, attachmentIDs[0], fileNames[0], pdfs[0])
	deleted := appAttachmentFlowRequest(application.Handler(), operatorToken, http.MethodDelete, "/api/v1/attachments/"+attachmentIDs[1], "", "", nil)
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("delete unbound attachment = %d %s", deleted.Code, deleted.Body.String())
	}
	if _, err := os.Stat(filepath.Join(base.DataDir, "attachments", "content", attachmentIDs[1])); !os.IsNotExist(err) {
		t.Fatalf("deleted local attachment still exists: %v", err)
	}

	application.Close()
	application = nil
	reopened, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, logger)
	if err != nil {
		t.Fatal(err)
	}
	application = reopened
	appAttachmentFlowAssertDownload(t, application.Handler(), viewerToken, attachmentIDs[0], fileNames[0], pdfs[0])

	application.Close()
	application = nil
	disabledDependencies := dependencies
	disabledDependencies.DisableAttachmentRoutes = true
	disabled, err := app.NewAuthenticatedAPIWithDependencies(configuration, disabledDependencies, logger)
	if err != nil {
		t.Fatal(err)
	}
	application = disabled
	if response := appAttachmentFlowRequest(application.Handler(), viewerToken, http.MethodGet, "/api/v1/orders/"+created.ID, "", "", nil); response.Code != http.StatusOK {
		t.Fatalf("orders with attachment routes disabled = %d %s", response.Code, response.Body.String())
	}
	if response := appAttachmentFlowRequest(application.Handler(), viewerToken, http.MethodGet, "/api/v1/attachments/"+attachmentIDs[0], "", "", nil); response.Code != http.StatusNotFound {
		t.Fatalf("disabled attachment route = %d %s", response.Code, response.Body.String())
	}
}

type appAttachmentFlowDTO struct {
	ID          string `json:"id"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	SHA256      string `json:"sha256"`
	CreatedAt   string `json:"createdAt"`
	ExpiresAt   string `json:"expiresAt"`
}

type appAttachmentFlowItemDTO struct {
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unitPrice"`
}

type appAttachmentFlowOrderDTO struct {
	ID              string                     `json:"id"`
	CustomerName    string                     `json:"customerName"`
	Status          string                     `json:"status"`
	Currency        string                     `json:"currency"`
	TotalAmount     int64                      `json:"totalAmount"`
	AttachmentCount int64                      `json:"attachmentCount"`
	Items           []appAttachmentFlowItemDTO `json:"items"`
	Attachments     []appAttachmentFlowDTO     `json:"attachments"`
}

func appAttachmentFlowLogin(t *testing.T, handler http.Handler, username string) string {
	t.Helper()
	response := loginResponse(handler, username, "123456789012", "attachment-flow-login-"+username)
	if response.Code != http.StatusOK {
		t.Fatalf("%s login = %d %s", username, response.Code, response.Body.String())
	}
	var result struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil || result.AccessToken == "" {
		t.Fatalf("decode %s login: %v", username, err)
	}
	return result.AccessToken
}

func appAttachmentFlowRequest(handler http.Handler, token, method, path, contentType, idempotencyKey string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func appAttachmentFlowUpload(t *testing.T, handler http.Handler, token, fileName string, content []byte) appAttachmentFlowDTO {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	response := appAttachmentFlowRequest(handler, token, http.MethodPost, "/api/v1/attachments", writer.FormDataContentType(), "", body.Bytes())
	if response.Code != http.StatusCreated {
		t.Fatalf("upload %q = %d %s", fileName, response.Code, response.Body.String())
	}
	var result appAttachmentFlowDTO
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func appAttachmentFlowAssertDownload(t *testing.T, handler http.Handler, token, attachmentID, fileName string, content []byte) {
	t.Helper()
	response := appAttachmentFlowRequest(handler, token, http.MethodGet, "/api/v1/attachments/"+attachmentID, "", "", nil)
	disposition, parameters, dispositionErr := mime.ParseMediaType(response.Header().Get("Content-Disposition"))
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), content) || response.Header().Get("Content-Type") != "application/pdf" || response.Header().Get("Content-Length") != strconv.Itoa(len(content)) || response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Cache-Control") != "private, no-store" || dispositionErr != nil || disposition != "attachment" || parameters["filename"] != fileName {
		t.Fatalf("download = %d headers=%v body=%q dispositionError=%v", response.Code, response.Header(), response.Body.Bytes(), dispositionErr)
	}
}

func appAttachmentFlowPDF(label string) []byte {
	var content bytes.Buffer
	content.WriteString("%PDF-1.4\n")
	offsets := make([]int, 5)
	writeObject := func(number int, value string) {
		offsets[number] = content.Len()
		fmt.Fprintf(&content, "%d 0 obj\n%s\nendobj\n", number, value)
	}
	writeObject(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObject(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObject(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R >>")
	stream := "% " + label + "\n"
	writeObject(4, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream))
	xref := content.Len()
	content.WriteString("xref\n0 5\n0000000000 65535 f \n")
	for number := 1; number <= 4; number++ {
		fmt.Fprintf(&content, "%010d 00000 n \n", offsets[number])
	}
	fmt.Fprintf(&content, "trailer\n<< /Size 5 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xref)
	return content.Bytes()
}

func TestAuthenticatedAPIAttachmentLocalStoreFailureReleasesResources(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	base, err := config.LoadBase(mapLookup(map[string]string{"DATA_DIR": t.TempDir()}))
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, base.DatabasePath, store.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Migrate(ctx); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	attachmentPath := filepath.Join(base.DataDir, "attachments")
	if err := os.WriteFile(attachmentPath, []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	configuration := config.APIConfig{Config: base, JWTSigningKey: []byte("12345678901234567890123456789012")}
	dependencies := app.AuthDependencies{Clock: func() time.Time { return now }}
	if application, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil))); err == nil {
		application.Close()
		t.Fatal("attachment local store initialization error = nil")
	}
	if err := os.Remove(attachmentPath); err != nil {
		t.Fatal(err)
	}

	application, err := app.NewAuthenticatedAPIWithDependencies(configuration, dependencies, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("retry after attachment initialization failure: %v", err)
	}
	defer application.Close()
	for _, directory := range []string{"temp", "content"} {
		info, err := os.Stat(filepath.Join(attachmentPath, directory))
		if err != nil || !info.IsDir() {
			t.Fatalf("local attachment directory %q: info=%v error=%v", directory, info, err)
		}
	}
}

type appAttachmentFiles struct {
	storageKey string
	fileName   string
	content    []byte
}

func (files *appAttachmentFiles) Write(storageKey, fileName string, content []byte) (order.StoredAttachmentFile, error) {
	files.storageKey = storageKey
	files.fileName = fileName
	files.content = bytes.Clone(content)
	return order.StoredAttachmentFile{FileName: fileName, ContentType: "application/pdf", SizeBytes: int64(len(content)), SHA256: sha256.Sum256(content)}, nil
}

func (*appAttachmentFiles) Read(string) ([]byte, error) { return nil, errors.New("unexpected read") }
func (*appAttachmentFiles) Delete(string) error         { return nil }
func (*appAttachmentFiles) DeleteResidual(string) error { return nil }
func (*appAttachmentFiles) ListResiduals(time.Time) ([]string, error) {
	return nil, nil
}

func TestAPINotReadyDatabaseStatesKeepHealthLive(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, config.Config)
	}{
		{name: "missing"},
		{name: "uninitialized", prepare: func(t *testing.T, configuration config.Config) {
			database, err := store.Open(context.Background(), configuration.DatabasePath, store.OpenCreate)
			if err != nil {
				t.Fatal(err)
			}
			database.Close()
		}},
		{name: "corrupt", prepare: func(t *testing.T, configuration config.Config) {
			if err := os.WriteFile(configuration.DatabasePath, []byte("not a database"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "too new", prepare: func(t *testing.T, configuration config.Config) {
			database, err := store.Open(context.Background(), configuration.DatabasePath, store.OpenCreate)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := database.SQL().Exec("PRAGMA user_version = " + strconv.Itoa(store.LatestSchemaVersion()+1)); err != nil {
				database.Close()
				t.Fatal(err)
			}
			database.Close()
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration, err := config.Load(mapLookup(map[string]string{"DATA_DIR": t.TempDir()}))
			if err != nil {
				t.Fatal(err)
			}
			if test.prepare != nil {
				test.prepare(t, configuration)
			}
			application, err := app.NewAPI(configuration, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			if err != nil {
				t.Fatal(err)
			}
			defer application.Close()
			if status := requestStatus(application.Handler(), "/healthz"); status != http.StatusOK {
				t.Fatalf("health status = %d", status)
			}
			if status := requestStatus(application.Handler(), "/readyz"); status != http.StatusServiceUnavailable {
				t.Fatalf("ready status = %d", status)
			}
		})
	}
}

func requestStatus(handler http.Handler, path string) int {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
