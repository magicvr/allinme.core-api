package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/app"
	"github.com/magicvr/allinme.core-api/internal/config"
)

func TestWalletHTTPIntegration(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DB.SQLitePath = t.TempDir() + "/wallets.db"
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Auth.JWTTTL = time.Hour
	application, err := app.New(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close() })

	admin := walletTestToken(t, application.Handler, "admin")
	operator := walletTestToken(t, application.Handler, "operator")
	viewer := walletTestToken(t, application.Handler, "viewer")

	assertWalletStatus(t, application.Handler, http.MethodGet, "/v1/wallets", nil, "", http.StatusUnauthorized)
	assertWalletStatus(t, application.Handler, http.MethodPost, "/v1/wallets", []byte(`{"accountNo":"VIEWER","ownerName":"Viewer"}`), viewer, http.StatusForbidden)

	seeded := requestWallet(t, application.Handler, http.MethodGet, "/v1/wallets?page=1&pageSize=20", nil, viewer, http.StatusOK)
	var seededEnv struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"list"`
			Total int `json:"total"`
		} `json:"data"`
	}
	decodeWalletBody(t, seeded, &seededEnv)
	if seededEnv.Code != 0 || seededEnv.Data.Total != 2 || len(seededEnv.Data.List) != 2 {
		t.Fatalf("seeded wallet envelope = %s", seeded.Body.String())
	}

	created := requestWallet(t, application.Handler, http.MethodPost, "/v1/wallets", []byte(`{"accountNo":"WAL-HTTP","ownerName":"HTTP Owner","balanceCents":1999,"currency":"usd"}`), operator, http.StatusOK)
	var createEnv struct {
		Code int `json:"code"`
		Data struct {
			ID           string `json:"id"`
			AccountNo    string `json:"accountNo"`
			OwnerName    string `json:"ownerName"`
			BalanceCents int64  `json:"balanceCents"`
			Currency     string `json:"currency"`
			Status       string `json:"status"`
			Version      int64  `json:"version"`
		} `json:"data"`
	}
	decodeWalletBody(t, created, &createEnv)
	if createEnv.Code != 0 || createEnv.Data.ID == "" || createEnv.Data.AccountNo != "WAL-HTTP" || createEnv.Data.OwnerName != "HTTP Owner" || createEnv.Data.BalanceCents != 1999 || createEnv.Data.Currency != "USD" || createEnv.Data.Status != "active" || createEnv.Data.Version != 1 {
		t.Fatalf("create wallet envelope = %s", created.Body.String())
	}

	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets", []byte(`{"accountNo":"WAL-HTTP","ownerName":"Duplicate"}`), admin, http.StatusConflict, "account_no_conflict")
	list := requestWallet(t, application.Handler, http.MethodGet, "/v1/wallets?status=active&q=HTTP&page=1&pageSize=20", nil, viewer, http.StatusOK)
	var listEnv struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				ID string `json:"id"`
			} `json:"list"`
			Total int `json:"total"`
		} `json:"data"`
	}
	decodeWalletBody(t, list, &listEnv)
	if listEnv.Code != 0 || listEnv.Data.Total != 1 || len(listEnv.Data.List) != 1 || listEnv.Data.List[0].ID != createEnv.Data.ID {
		t.Fatalf("list wallet envelope = %s", list.Body.String())
	}

	detail := requestWallet(t, application.Handler, http.MethodGet, "/v1/wallets/"+createEnv.Data.ID, nil, viewer, http.StatusOK)
	var detailEnv struct {
		Data struct {
			ID           string `json:"id"`
			BalanceCents int64  `json:"balanceCents"`
		} `json:"data"`
	}
	decodeWalletBody(t, detail, &detailEnv)
	if detailEnv.Data.ID != createEnv.Data.ID || detailEnv.Data.BalanceCents != 1999 {
		t.Fatalf("wallet detail envelope = %s", detail.Body.String())
	}
	assertWalletError(t, application.Handler, http.MethodGet, "/v1/wallets/does-not-exist", nil, viewer, http.StatusNotFound, "wallet_not_found")

	assertWalletError(t, application.Handler, http.MethodPut, "/v1/wallets/"+createEnv.Data.ID, []byte(`{"version":1,"ownerName":"Illegal","balanceCents":0}`), admin, http.StatusBadRequest, "bad_request")
	updated := requestWallet(t, application.Handler, http.MethodPut, "/v1/wallets/"+createEnv.Data.ID, []byte(`{"version":1,"ownerName":"Updated Owner"}`), admin, http.StatusOK)
	var updateEnv struct {
		Data struct {
			OwnerName    string `json:"ownerName"`
			BalanceCents int64  `json:"balanceCents"`
			Currency     string `json:"currency"`
			Status       string `json:"status"`
			Version      int64  `json:"version"`
		} `json:"data"`
	}
	decodeWalletBody(t, updated, &updateEnv)
	if updateEnv.Data.OwnerName != "Updated Owner" || updateEnv.Data.BalanceCents != 1999 || updateEnv.Data.Currency != "USD" || updateEnv.Data.Status != "active" || updateEnv.Data.Version != 2 {
		t.Fatalf("update wallet envelope = %s", updated.Body.String())
	}
	assertWalletError(t, application.Handler, http.MethodPut, "/v1/wallets/"+createEnv.Data.ID, []byte(`{"version":1,"ownerName":"Stale"}`), operator, http.StatusConflict, "version_conflict")
	frozenOwnerUpdate := requestWallet(t, application.Handler, http.MethodPut, "/v1/wallets/wal_seed_frozen", []byte(`{"version":1,"ownerName":"Frozen Updated"}`), operator, http.StatusOK)
	var frozenOwnerEnv struct {
		Data struct {
			OwnerName    string `json:"ownerName"`
			BalanceCents int64  `json:"balanceCents"`
			Currency     string `json:"currency"`
			Status       string `json:"status"`
			Version      int64  `json:"version"`
		} `json:"data"`
	}
	decodeWalletBody(t, frozenOwnerUpdate, &frozenOwnerEnv)
	if frozenOwnerEnv.Data.OwnerName != "Frozen Updated" || frozenOwnerEnv.Data.BalanceCents != 8800 || frozenOwnerEnv.Data.Currency != "USD" || frozenOwnerEnv.Data.Status != "frozen" || frozenOwnerEnv.Data.Version != 2 {
		t.Fatalf("frozen owner update envelope = %s", frozenOwnerUpdate.Body.String())
	}

	frozen := requestWallet(t, application.Handler, http.MethodPost, "/v1/wallets/"+createEnv.Data.ID+"/freeze", []byte(`{"version":2}`), operator, http.StatusOK)
	var freezeEnv struct {
		Data struct {
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeWalletBody(t, frozen, &freezeEnv)
	if freezeEnv.Data.Status != "frozen" || freezeEnv.Data.Version != 3 {
		t.Fatalf("freeze wallet envelope = %s", frozen.Body.String())
	}
	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets/"+createEnv.Data.ID+"/freeze", []byte(`{"version":3}`), admin, http.StatusConflict, "invalid_state")
	unfrozen := requestWallet(t, application.Handler, http.MethodPost, "/v1/wallets/"+createEnv.Data.ID+"/unfreeze", []byte(`{"version":3}`), admin, http.StatusOK)
	var unfreezeEnv struct {
		Data struct {
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeWalletBody(t, unfrozen, &unfreezeEnv)
	if unfreezeEnv.Data.Status != "active" || unfreezeEnv.Data.Version != 4 {
		t.Fatalf("unfreeze wallet envelope = %s", unfrozen.Body.String())
	}
	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets/"+createEnv.Data.ID+"/freeze", []byte(`{"version":3}`), operator, http.StatusConflict, "version_conflict")
	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets/"+createEnv.Data.ID+"/unfreeze", []byte(`{"version":4}`), operator, http.StatusConflict, "invalid_state")

	batchOne := createWalletForBatch(t, application.Handler, operator, "WAL-BATCH-1")
	batchTwo := createWalletForBatch(t, application.Handler, admin, "WAL-BATCH-2")
	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets/batch-freeze", []byte(`{"ids":["`+batchOne+`","wal_seed_frozen"]}`), admin, http.StatusConflict, "invalid_state")
	assertWalletWalletStatus(t, application.Handler, viewer, batchOne, "active")
	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets/batch-freeze", []byte(`{"ids":["`+batchOne+`","missing"]}`), admin, http.StatusNotFound, "wallet_not_found")
	assertWalletWalletStatus(t, application.Handler, viewer, batchOne, "active")
	batch := requestWallet(t, application.Handler, http.MethodPost, "/v1/wallets/batch-freeze", []byte(`{"ids":["`+batchOne+`","`+batchTwo+`"]}`), operator, http.StatusOK)
	var batchEnv struct {
		Data struct {
			Frozen int `json:"frozen"`
		} `json:"data"`
	}
	decodeWalletBody(t, batch, &batchEnv)
	if batchEnv.Data.Frozen != 2 {
		t.Fatalf("batch wallet envelope = %s", batch.Body.String())
	}
	assertWalletWalletStatus(t, application.Handler, viewer, batchOne, "frozen")
	assertWalletWalletStatus(t, application.Handler, viewer, batchTwo, "frozen")
	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets/batch-freeze", []byte(`{"ids":["`+batchOne+`","`+batchOne+`"]}`), admin, http.StatusBadRequest, "bad_request")

	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets", []byte(`{"accountNo":"WAL-BAD","unknown":true}`), admin, http.StatusBadRequest, "bad_request")
	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets", []byte(`{"accountNo":"WAL-TRAIL","ownerName":"Trail"}{}`), admin, http.StatusBadRequest, "bad_request")
	assertWalletError(t, application.Handler, http.MethodGet, "/v1/wallets?status=closed", nil, viewer, http.StatusBadRequest, "bad_request")
	assertWalletError(t, application.Handler, http.MethodGet, "/v1/wallets?page=9223372036854775807&pageSize=100", nil, viewer, http.StatusBadRequest, "bad_request")
	oversized := []byte(`{"accountNo":"WAL-LARGE","ownerName":"` + strings.Repeat("x", (1<<20)+1) + `"}`)
	assertWalletError(t, application.Handler, http.MethodPost, "/v1/wallets", oversized, admin, http.StatusBadRequest, "bad_request")
}

func createWalletForBatch(t *testing.T, handler http.Handler, token, accountNo string) string {
	t.Helper()
	rr := requestWallet(t, handler, http.MethodPost, "/v1/wallets", []byte(`{"accountNo":"`+accountNo+`","ownerName":"Batch Owner"}`), token, http.StatusOK)
	var envelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeWalletBody(t, rr, &envelope)
	if envelope.Data.ID == "" {
		t.Fatalf("missing wallet ID: %s", rr.Body.String())
	}
	return envelope.Data.ID
}

func assertWalletWalletStatus(t *testing.T, handler http.Handler, token, id, wantStatus string) {
	t.Helper()
	rr := requestWallet(t, handler, http.MethodGet, "/v1/wallets/"+id, nil, token, http.StatusOK)
	var envelope struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	decodeWalletBody(t, rr, &envelope)
	if envelope.Data.Status != wantStatus {
		t.Fatalf("wallet %s status=%q want=%q", id, envelope.Data.Status, wantStatus)
	}
}

func walletTestToken(t *testing.T, handler http.Handler, username string) string {
	t.Helper()
	rr := requestWallet(t, handler, http.MethodPost, "/v1/auth/login", []byte(`{"username":"`+username+`","password":"Demo@1234"}`), "", http.StatusOK)
	var envelope struct {
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	decodeWalletBody(t, rr, &envelope)
	if envelope.Data.AccessToken == "" {
		t.Fatalf("missing token for %s", username)
	}
	return envelope.Data.AccessToken
}

func assertWalletError(t *testing.T, handler http.Handler, method, path string, body []byte, token string, wantStatus int, wantCode string) *httptest.ResponseRecorder {
	t.Helper()
	rr := requestWallet(t, handler, method, path, body, token, wantStatus)
	var envelope struct {
		Code string `json:"code"`
	}
	decodeWalletBody(t, rr, &envelope)
	if envelope.Code != wantCode {
		t.Fatalf("%s %s error code=%q want=%q", method, path, envelope.Code, wantCode)
	}
	return rr
}

func assertWalletStatus(t *testing.T, handler http.Handler, method, path string, body []byte, token string, want int) {
	t.Helper()
	requestWallet(t, handler, method, path, body, token, want)
}

func requestWallet(t *testing.T, handler http.Handler, method, path string, body []byte, token string, want int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != want {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, rr.Code, want, rr.Body.String())
	}
	return rr
}

func decodeWalletBody(t *testing.T, rr *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}
