package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/app"
	"github.com/magicvr/allinme.core-api/internal/config"
)

func TestOrderHTTPIntegration(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DB.SQLitePath = t.TempDir() + "/orders.db"
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Auth.JWTTTL = time.Hour
	application, err := app.New(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close() })

	admin := orderTestToken(t, application.Handler, "admin")
	operator := orderTestToken(t, application.Handler, "operator")
	viewer := orderTestToken(t, application.Handler, "viewer")

	assertOrderStatus(t, application.Handler, http.MethodGet, "/v1/orders", nil, "", http.StatusUnauthorized)
	assertOrderStatus(t, application.Handler, http.MethodPost, "/v1/orders", []byte(`{"orderNo":"ORD-VIEWER","customerName":"Viewer","amountCents":1}`), viewer, http.StatusForbidden)

	created := requestOrder(t, application.Handler, http.MethodPost, "/v1/orders", []byte(`{"orderNo":"ORD-HTTP","customerName":"HTTP Customer","amountCents":1999,"remark":"new"}`), operator, http.StatusOK)
	var createEnv struct {
		Code int `json:"code"`
		Data struct {
			ID       string `json:"id"`
			Version  int64  `json:"version"`
			Status   string `json:"status"`
			Currency string `json:"currency"`
		} `json:"data"`
	}
	decodeOrderBody(t, created, &createEnv)
	if createEnv.Code != 0 || createEnv.Data.ID == "" || createEnv.Data.Version != 1 || createEnv.Data.Status != "pending" || createEnv.Data.Currency != "CNY" {
		t.Fatalf("create envelope = %s", created.Body.String())
	}

	list := requestOrder(t, application.Handler, http.MethodGet, "/v1/orders?status=pending&q=HTTP&page=1&pageSize=20", nil, viewer, http.StatusOK)
	var listEnv struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				ID string `json:"id"`
			} `json:"list"`
			Total int `json:"total"`
		} `json:"data"`
	}
	decodeOrderBody(t, list, &listEnv)
	if listEnv.Code != 0 || listEnv.Data.Total != 1 || len(listEnv.Data.List) != 1 || listEnv.Data.List[0].ID != createEnv.Data.ID {
		t.Fatalf("list envelope = %s", list.Body.String())
	}
	createdDetail := requestOrder(t, application.Handler, http.MethodGet, "/v1/orders/"+createEnv.Data.ID, nil, viewer, http.StatusOK)
	var detailEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeOrderBody(t, createdDetail, &detailEnv)
	if detailEnv.Data.ID != createEnv.Data.ID {
		t.Fatalf("detail envelope = %s", createdDetail.Body.String())
	}
	createdForCancel := requestOrder(t, application.Handler, http.MethodPost, "/v1/orders", []byte(`{"orderNo":"ORD-CANCEL","customerName":"Cancel Customer","amountCents":500}`), admin, http.StatusOK)
	var cancelCreateEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeOrderBody(t, createdForCancel, &cancelCreateEnv)
	cancelled := requestOrder(t, application.Handler, http.MethodPost, "/v1/orders/"+cancelCreateEnv.Data.ID+"/cancel", []byte(`{"version":1}`), operator, http.StatusOK)
	var cancelEnv struct {
		Data struct {
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeOrderBody(t, cancelled, &cancelEnv)
	if cancelEnv.Data.Status != "cancelled" || cancelEnv.Data.Version != 2 {
		t.Fatalf("cancel envelope = %s", cancelled.Body.String())
	}

	assertOrderError(t, application.Handler, http.MethodGet, "/v1/orders/does-not-exist", nil, admin, http.StatusNotFound, "order_not_found")
	updated := requestOrder(t, application.Handler, http.MethodPut, "/v1/orders/"+createEnv.Data.ID, []byte(`{"version":1,"customerName":"Updated Customer","amountCents":2999,"currency":"USD","remark":"updated"}`), admin, http.StatusOK)
	var updateEnv struct {
		Data struct {
			Version int64 `json:"version"`
		} `json:"data"`
	}
	decodeOrderBody(t, updated, &updateEnv)
	if updateEnv.Data.Version != 2 {
		t.Fatalf("update envelope = %s", updated.Body.String())
	}
	assertOrderError(t, application.Handler, http.MethodPut, "/v1/orders/"+createEnv.Data.ID, []byte(`{"version":1,"customerName":"Stale","amountCents":1,"currency":"CNY"}`), operator, http.StatusConflict, "version_conflict")

	paid := requestOrder(t, application.Handler, http.MethodPost, "/v1/orders/"+createEnv.Data.ID+"/mark-paid", []byte(`{"version":2}`), operator, http.StatusOK)
	var paidEnv struct {
		Data struct {
			Version int64  `json:"version"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	decodeOrderBody(t, paid, &paidEnv)
	if paidEnv.Data.Status != "paid" || paidEnv.Data.Version != 3 {
		t.Fatalf("paid envelope = %s", paid.Body.String())
	}
	assertOrderError(t, application.Handler, http.MethodPost, "/v1/orders/"+createEnv.Data.ID+"/cancel", []byte(`{"version":3}`), admin, http.StatusConflict, "invalid_state")

	batch := assertOrderError(t, application.Handler, http.MethodPost, "/v1/orders/batch-delete", []byte(`{"ids":["ord_seed_pending","ord_seed_paid"]}`), admin, http.StatusConflict, "invalid_state")
	if batch.Code != http.StatusConflict {
		t.Fatalf("batch status = %d", batch.Code)
	}
	assertOrderStatus(t, application.Handler, http.MethodGet, "/v1/orders/ord_seed_pending", nil, viewer, http.StatusOK)
	deleted := requestOrder(t, application.Handler, http.MethodPost, "/v1/orders/batch-delete", []byte(`{"ids":["ord_seed_pending","ord_seed_cancelled"]}`), admin, http.StatusOK)
	var deleteEnv struct {
		Data struct {
			Deleted int `json:"deleted"`
		} `json:"data"`
	}
	decodeOrderBody(t, deleted, &deleteEnv)
	if deleteEnv.Data.Deleted != 2 {
		t.Fatalf("delete envelope = %s", deleted.Body.String())
	}

	assertOrderError(t, application.Handler, http.MethodPost, "/v1/orders", []byte(`{"orderNo":"ORD-BAD","unknown":true}`), admin, http.StatusBadRequest, "bad_request")
	assertOrderError(t, application.Handler, http.MethodGet, "/v1/orders?page=9223372036854775807&pageSize=100", nil, viewer, http.StatusBadRequest, "bad_request")

	assertOrderError(t, application.Handler, http.MethodPost, "/v1/orders", []byte(`{"orderNo":"ORD-HTTP","customerName":"Duplicate","amountCents":1}`), admin, http.StatusConflict, "order_no_conflict")
}

func orderTestToken(t *testing.T, handler http.Handler, username string) string {
	t.Helper()
	rr := requestOrder(t, handler, http.MethodPost, "/v1/auth/login", []byte(`{"username":"`+username+`","password":"Demo@1234"}`), "", http.StatusOK)
	var envelope struct {
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	decodeOrderBody(t, rr, &envelope)
	if envelope.Data.AccessToken == "" {
		t.Fatalf("missing token for %s", username)
	}
	return envelope.Data.AccessToken
}

func assertOrderError(t *testing.T, handler http.Handler, method, path string, body []byte, token string, wantStatus int, wantCode string) *httptest.ResponseRecorder {
	t.Helper()
	rr := requestOrder(t, handler, method, path, body, token, wantStatus)
	var envelope struct {
		Code string `json:"code"`
	}
	decodeOrderBody(t, rr, &envelope)
	if envelope.Code != wantCode {
		t.Fatalf("%s %s error code=%q want=%q body=%s", method, path, envelope.Code, wantCode, rr.Body.String())
	}
	return rr
}

func assertOrderStatus(t *testing.T, handler http.Handler, method, path string, body []byte, token string, want int) {
	t.Helper()
	requestOrder(t, handler, method, path, body, token, want)
}

func requestOrder(t *testing.T, handler http.Handler, method, path string, body []byte, token string, want int) *httptest.ResponseRecorder {
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

func decodeOrderBody(t *testing.T, rr *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}
