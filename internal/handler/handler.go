package handler

import (
	"log/slog"
	"net/http"

	"github.com/magicvr/allinme.core-api/internal/middleware"
	"github.com/magicvr/allinme.core-api/internal/service/auth"
	"github.com/magicvr/allinme.core-api/internal/service/menu"
	"github.com/magicvr/allinme.core-api/internal/service/meta"
	orderservice "github.com/magicvr/allinme.core-api/internal/service/order"
	walletservice "github.com/magicvr/allinme.core-api/internal/service/wallet"
)

// Deps are inbound adapter dependencies injected by the composition root.
type Deps struct {
	Logger *slog.Logger
	Meta   *meta.Service
	Auth   *auth.Service
	Menu   *menu.Service
	Order  *orderservice.Service
	Wallet *walletservice.Service
}

// Register mounts all HTTP routes on mux.
// Keep this as the single place Admin and future API modules plug into.
func Register(mux *http.ServeMux, deps Deps) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	// Public
	mux.Handle("GET /healthz", healthz())
	mux.Handle("GET /readyz", readyz(deps.Meta))
	mux.Handle("POST /v1/auth/login", login(deps.Auth))

	// Protected (D-007: default require auth except health/ready/login)
	require := middleware.RequireAuth(deps.Auth)
	mux.Handle("GET /v1/ping", require(ping(deps.Logger)))
	mux.Handle("GET /v1/auth/me", require(me(deps.Auth)))
	mux.Handle("GET /v1/admin/menu", require(adminMenu(deps.Menu)))
	mux.Handle("GET /v1/orders", require(listOrders(deps.Order)))
	mux.Handle("GET /v1/orders/{id}", require(getOrder(deps.Order)))

	orderWrite := func(next http.Handler) http.Handler {
		return require(middleware.RequireRoles("admin", "operator")(next))
	}
	mux.Handle("POST /v1/orders", orderWrite(createOrder(deps.Order)))
	mux.Handle("PUT /v1/orders/{id}", orderWrite(updateOrder(deps.Order)))
	mux.Handle("POST /v1/orders/{id}/mark-paid", orderWrite(markOrderPaid(deps.Order)))
	mux.Handle("POST /v1/orders/{id}/cancel", orderWrite(cancelOrder(deps.Order)))
	mux.Handle("POST /v1/orders/batch-delete", orderWrite(batchDeleteOrders(deps.Order)))

	mux.Handle("GET /v1/wallets", require(listWallets(deps.Wallet)))
	mux.Handle("GET /v1/wallets/{id}", require(getWallet(deps.Wallet)))

	walletWrite := func(next http.Handler) http.Handler {
		return require(middleware.RequireRoles("admin", "operator")(next))
	}
	mux.Handle("POST /v1/wallets", walletWrite(createWallet(deps.Wallet)))
	mux.Handle("PUT /v1/wallets/{id}", walletWrite(updateWallet(deps.Wallet)))
	mux.Handle("POST /v1/wallets/{id}/freeze", walletWrite(freezeWallet(deps.Wallet)))
	mux.Handle("POST /v1/wallets/{id}/unfreeze", walletWrite(unfreezeWallet(deps.Wallet)))
	mux.Handle("POST /v1/wallets/batch-freeze", walletWrite(batchFreezeWallets(deps.Wallet)))
}
