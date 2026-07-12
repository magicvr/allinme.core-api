package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
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
	if crossOriginResponse.Code != http.StatusOK || crossOriginResponse.Header().Get("Access-Control-Allow-Origin") != configuration.CORSAllowedOrigin || crossOriginResponse.Header().Get("Access-Control-Expose-Headers") != "X-Request-ID" {
		t.Fatalf("cross origin orders = %d headers=%v body=%s", crossOriginResponse.Code, crossOriginResponse.Header(), crossOriginResponse.Body.String())
	}
	if response := requestWithToken(http.MethodGet, "/api/v1/orders/ord_00000000000000000000000000000001"); response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"items":[{"id":"itm_`)) {
		t.Fatalf("order detail = %d %s", response.Code, response.Body.String())
	}
	if response := requestWithToken(http.MethodHead, "/api/v1/orders"); response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
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
