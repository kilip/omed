package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/kilip/omed/finance/ent"
)

// WithTx runs your database operations within a managed transaction.
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

type searchCursor struct {
	Offset int `json:"offset"`
}

func encodeCursor(offset int) string {
	b, _ := json.Marshal(searchCursor{Offset: offset})
	return base64.URLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor: %w", err)
	}
	var c searchCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return 0, fmt.Errorf("invalid cursor: %w", err)
	}
	return c.Offset, nil
}
