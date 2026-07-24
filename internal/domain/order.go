package domain

import "time"

// OrderStatus is the lifecycle state of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRefunded  OrderStatus = "refunded"
)

// Order is an API-safe order aggregate.
type Order struct {
	ID           string      `json:"id"`
	OrderNo      string      `json:"orderNo"`
	CustomerName string      `json:"customerName"`
	Status       OrderStatus `json:"status"`
	AmountCents  int64       `json:"amountCents"`
	Currency     string      `json:"currency"`
	Remark       string      `json:"remark"`
	Version      int64       `json:"version"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

// IsKnownOrderStatus reports whether status is part of the defined order lifecycle.
func IsKnownOrderStatus(status OrderStatus) bool {
	switch status {
	case OrderStatusPending, OrderStatusPaid, OrderStatusCancelled, OrderStatusRefunded:
		return true
	default:
		return false
	}
}
