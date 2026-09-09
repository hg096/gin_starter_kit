package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

// WithNamedLockTx serializes a workflow across processes using one MySQL connection.
// The named lock is held until commit/rollback completes, then released before pooling.
func (db *DB) WithNamedLockTx(ctx context.Context, name string, fn func(*sql.Tx) error) error {
	if fn == nil {
		return fmt.Errorf("transaction callback is nil")
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 10)", name).Scan(&acquired); err != nil {
		_ = conn.Raw(func(interface{}) error { return driver.ErrBadConn })
		return err
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return fmt.Errorf("could not acquire transaction lock")
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var released sql.NullInt64
		if err := conn.QueryRowContext(releaseCtx, "SELECT RELEASE_LOCK(?)", name).Scan(&released); err != nil || !released.Valid || released.Int64 != 1 {
			_ = conn.Raw(func(interface{}) error { return driver.ErrBadConn })
		}
	}()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer RollbackTx(tx)
	if err := fn(tx); err != nil {
		return err
	}
	return CommitTx(tx)
}
