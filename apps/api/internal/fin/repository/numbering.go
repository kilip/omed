// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/entry"
	"github.com/kilip/omed/api/ent/transaction"
	"github.com/kilip/omed/api/internal/base"
)

// GenerateTransactionNumber generates the next sequential transaction number (TRX-YYYY-000001) for the workspace.
// This function must be executed inside an active transaction (tx).
func GenerateTransactionNumber(ctx context.Context, tx *ent.Tx, year int) (string, error) {
	if !base.SkipWorkspaceFilterFromContext(ctx) {
		wsID := base.WorkspaceIDFromContext(ctx)
		if wsID <= 0 {
			return "", base.ErrWorkspaceRequired
		}
	}

	prefix := fmt.Sprintf("TRX-%04d-", year)

	wsID := base.WorkspaceIDFromContext(ctx)
	q := tx.Transaction.Query().
		Where(
			transaction.NumberHasPrefix(prefix),
		)
	if wsID > 0 {
		q = q.Where(transaction.WorkspaceIDEQ(wsID))
	}

	last, err := q.
		Order(ent.Desc(transaction.FieldNumber)).
		Limit(1).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return "", err
	}

	nextSeq := 1
	if last != nil {
		suffix := strings.TrimPrefix(last.Number, prefix)
		if n, parseErr := strconv.Atoi(suffix); parseErr == nil {
			nextSeq = n + 1
		}
	}

	return fmt.Sprintf("%s%06d", prefix, nextSeq), nil
}

// GenerateEntryNumber generates the next sequential journal entry number (JE-YYYY-000001) for the workspace.
// This function must be executed inside an active transaction (tx).
func GenerateEntryNumber(ctx context.Context, tx *ent.Tx, year int) (string, error) {
	if !base.SkipWorkspaceFilterFromContext(ctx) {
		wsID := base.WorkspaceIDFromContext(ctx)
		if wsID <= 0 {
			return "", base.ErrWorkspaceRequired
		}
	}

	prefix := fmt.Sprintf("JE-%04d-", year)

	wsID := base.WorkspaceIDFromContext(ctx)
	q := tx.Entry.Query().
		Where(
			entry.NumberHasPrefix(prefix),
		)
	if wsID > 0 {
		q = q.Where(entry.WorkspaceIDEQ(wsID))
	}

	last, err := q.
		Order(ent.Desc(entry.FieldNumber)).
		Limit(1).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return "", err
	}

	nextSeq := 1
	if last != nil {
		suffix := strings.TrimPrefix(last.Number, prefix)
		if n, parseErr := strconv.Atoi(suffix); parseErr == nil {
			nextSeq = n + 1
		}
	}

	return fmt.Sprintf("%s%06d", prefix, nextSeq), nil
}

func isLockOrConflictError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "locked") || strings.Contains(s, "busy")
}

