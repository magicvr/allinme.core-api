package order

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
)

const (
	OrderIDPrefix = "ord_"
	ItemIDPrefix  = "itm_"
)

var (
	validOrderID = regexp.MustCompile(`^ord_[0-9a-f]{32}$`)
	validItemID  = regexp.MustCompile(`^itm_[0-9a-f]{32}$`)
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

func ValidOrderID(id string) bool {
	return validOrderID.MatchString(id)
}

func ValidItemID(id string) bool {
	return validItemID.MatchString(id)
}

func randomID(reader io.Reader, prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", fmt.Errorf("generate secure order identifier: %w", err)
	}
	return prefix + hex.EncodeToString(value), nil
}
