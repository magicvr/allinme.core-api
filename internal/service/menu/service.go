package menu

import (
	"context"

	"github.com/magicvr/allinme.core-api/internal/domain"
)

// Item is a menu entry for Admin shell routing.
type Item struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Path   string   `json:"path"`
	PageID string   `json:"pageId"`
	Roles  []string `json:"-"` // allowed roles; not always exposed
}

// PublicItem is the API shape for menu entries.
type PublicItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Path   string `json:"path"`
	PageID string `json:"pageId"`
}

// Service filters the static Admin menu by user roles (RBAC D-009).
type Service struct {
	catalog []Item
}

// New constructs a menu Service with the MVP catalog.
func New() *Service {
	return &Service{catalog: defaultCatalog()}
}

// ForUser returns menu items the user may see.
func (s *Service) ForUser(_ context.Context, user domain.User) []PublicItem {
	out := make([]PublicItem, 0, len(s.catalog))
	for _, it := range s.catalog {
		if user.HasAnyRole(it.Roles...) {
			out = append(out, PublicItem{
				ID: it.ID, Title: it.Title, Path: it.Path, PageID: it.PageID,
			})
		}
	}
	return out
}

func defaultCatalog() []Item {
	// admin: full; operator + viewer: business menus (viewer still sees entries; write blocked later by permissions)
	allBiz := []string{"admin", "operator", "viewer"}
	adminOnly := []string{"admin"}
	return []Item{
		{ID: "dashboard", Title: "仪表盘", Path: "/dashboard", PageID: "dashboard", Roles: allBiz},
		{ID: "orders", Title: "订单", Path: "/orders", PageID: "order_list", Roles: allBiz},
		{ID: "wallets", Title: "钱包", Path: "/wallets", PageID: "wallet_list", Roles: allBiz},
		{ID: "notifications", Title: "通知", Path: "/notifications", PageID: "notification_list", Roles: allBiz},
		{ID: "users", Title: "用户", Path: "/users", PageID: "user_list", Roles: adminOnly},
	}
}
