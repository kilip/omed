// Package entx provides utilities and lifecycle wrappers for database transactions using the Ent ORM.
package entx

import (
	"context"
	"fmt"

	"github.com/kilip/omed/api/ent"
)

// WithTx executes the provided function within a managed Ent database transaction.
// If the function returns an error, the transaction is automatically rolled back.
// If a panic occurs, the transaction is rolled back before re-panicking.
// If the function completes successfully, the transaction is committed.
func WithTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback() // Rollback on panic
			panic(v)          // Re-panic after safety rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("%w: rolling back transaction: %v", err, rerr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
