package order

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
)

const (
	OrderIDPrefix      = "ord_"
	ItemIDPrefix       = "itm_"
	RefundIDPrefix     = "rfd_"
	AttachmentIDPrefix = "att_"
)

var (
	validOrderID      = regexp.MustCompile(`^ord_[0-9a-f]{32}$`)
	validItemID       = regexp.MustCompile(`^itm_[0-9a-f]{32}$`)
	validRefundID     = regexp.MustCompile(`^rfd_[0-9a-f]{32}$`)
	validAttachmentID = regexp.MustCompile(`^att_[0-9a-f]{32}$`)
)

func NewOrderID() (string, error) {
	return NewOrderIDFrom(rand.Reader)
}

func NewOrderIDFrom(reader io.Reader) (string, error) {
	return randomID(reader, OrderIDPrefix)
}

func NewItemID() (string, error) {
	return NewItemIDFrom(rand.Reader)
}

func NewItemIDFrom(reader io.Reader) (string, error) {
	return randomID(reader, ItemIDPrefix)
}

func NewRefundID() (string, error) {
	return NewRefundIDFrom(rand.Reader)
}

func NewRefundIDFrom(reader io.Reader) (string, error) {
	return randomID(reader, RefundIDPrefix)
}

func NewAttachmentID() (string, error) {
	return NewAttachmentIDFrom(rand.Reader)
}

func NewAttachmentIDFrom(reader io.Reader) (string, error) {
	return randomID(reader, AttachmentIDPrefix)
}

func ValidOrderID(id string) bool {
	return validOrderID.MatchString(id)
}

func ValidItemID(id string) bool {
	return validItemID.MatchString(id)
}

func ValidRefundID(id string) bool {
	return validRefundID.MatchString(id)
}

func ValidAttachmentID(id string) bool {
	return validAttachmentID.MatchString(id)
}

func ValidAttachmentStorageKey(storageKey string) bool {
	return validAttachmentID.MatchString(storageKey)
}

func randomID(reader io.Reader, prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", fmt.Errorf("generate secure identifier: %w", err)
	}
	return prefix + hex.EncodeToString(value), nil
}
