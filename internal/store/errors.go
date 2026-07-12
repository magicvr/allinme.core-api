package store

import (
	"context"
	"errors"

	"github.com/magicvr/allinme.core-api/internal/order"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func classifyOrderError(err error) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, order.ErrNotFound) || errors.Is(err, order.ErrVersionConflict) || errors.Is(err, order.ErrStateConflict) || errors.Is(err, order.ErrInternal) || errors.Is(err, order.ErrUnavailable) {
		return err
	}
	var sqliteError *sqlite.Error
	if errors.As(err, &sqliteError) && unavailableSQLiteCode(sqliteError.Code()) {
		return order.Unavailable(err)
	}
	return order.Internal(err)
}

func unavailableSQLiteCode(code int) bool {
	switch code & 0xff {
	case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
		return true
	default:
		return false
	}
}
