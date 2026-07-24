package domain

import "time"

// WalletStatus is the operational state of a wallet.
type WalletStatus string

const (
	WalletStatusActive WalletStatus = "active"
	WalletStatusFrozen WalletStatus = "frozen"
)

// Wallet is an API-safe wallet aggregate.
type Wallet struct {
	ID           string       `json:"id"`
	AccountNo    string       `json:"accountNo"`
	OwnerName    string       `json:"ownerName"`
	BalanceCents int64        `json:"balanceCents"`
	Currency     string       `json:"currency"`
	Status       WalletStatus `json:"status"`
	Version      int64        `json:"version"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

// IsKnownWalletStatus reports whether status is part of the wallet lifecycle.
func IsKnownWalletStatus(status WalletStatus) bool {
	switch status {
	case WalletStatusActive, WalletStatusFrozen:
		return true
	default:
		return false
	}
}
