// Package repository provides database access implementations for user, account, session, and OTP entities using Ent.
package repository

import (
	"context"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/loginotp"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// OTPRepository defines persistence operations for login one-time passwords.
type OTPRepository interface {
	Issue(ctx context.Context, o domain.OTP) (domain.OTP, error)
	LastIssuedAt(ctx context.Context, email string) (time.Time, bool, error)
	CountSince(ctx context.Context, email string, since time.Time) (int, error)
	FindActive(ctx context.Context, email string, now time.Time) (domain.OTP, error)
	IncrementAttempts(ctx context.Context, id int64, max int) (int, error)
	Consume(ctx context.Context, id int64, now time.Time) (bool, error)
	DeleteOlderThan(ctx context.Context, email string, before time.Time) error
}

type otpRepository struct {
	cli *ent.Client
}

// NewOTPRepository creates a new OTPRepository backed by the Ent client.
func NewOTPRepository(cli *ent.Client) OTPRepository {
	return &otpRepository{cli: cli}
}

// Issue marks older unconsumed rows for the email as consumed and inserts the new OTP record within a single transaction.
func (r *otpRepository) Issue(ctx context.Context, o domain.OTP) (domain.OTP, error) {
	var result domain.OTP
	err := entx.WithTx(ctx, r.cli, func(tx *ent.Tx) error {
		now := time.Now()
		if err := tx.LoginOTP.Update().
			Where(
				loginotp.EmailEQ(o.Email),
				loginotp.ConsumedAtIsNil(),
			).
			SetConsumedAt(now).
			Exec(ctx); err != nil {
			return err
		}

		builder := tx.LoginOTP.Create().
			SetEmail(o.Email).
			SetCodeHash(o.CodeHash).
			SetExpiresAt(o.ExpiresAt)

		if o.ID != 0 {
			builder.SetID(o.ID)
		}
		if o.Attempts > 0 {
			builder.SetAttempts(o.Attempts)
		}
		if o.ConsumedAt != nil {
			builder.SetConsumedAt(*o.ConsumedAt)
		}
		if o.IPAddress != "" {
			builder.SetIPAddress(o.IPAddress)
		}
		if !o.CreatedAt.IsZero() {
			builder.SetCreatedAt(o.CreatedAt)
		}

		created, err := builder.Save(ctx)
		if err != nil {
			return err
		}

		result = toOTPDomain(created)
		return nil
	})
	if err != nil {
		return domain.OTP{}, err
	}
	return result, nil
}

// LastIssuedAt returns the creation time of the most recently issued OTP for the given email.
// If no OTP exists for the email, it returns zero time and false.
func (r *otpRepository) LastIssuedAt(ctx context.Context, email string) (time.Time, bool, error) {
	record, err := r.cli.LoginOTP.Query().
		Where(loginotp.EmailEQ(email)).
		Order(ent.Desc(loginotp.FieldCreatedAt), ent.Desc(loginotp.FieldID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return record.CreatedAt, true, nil
}

// CountSince returns the number of OTPs issued for the given email at or after the specified time.
func (r *otpRepository) CountSince(ctx context.Context, email string, since time.Time) (int, error) {
	count, err := r.cli.LoginOTP.Query().
		Where(
			loginotp.EmailEQ(email),
			loginotp.CreatedAtGTE(since),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// FindActive retrieves the active (unconsumed and unexpired) OTP for the given email.
// Returns base.ErrNotFound if no active OTP exists.
func (r *otpRepository) FindActive(ctx context.Context, email string, now time.Time) (domain.OTP, error) {
	record, err := r.cli.LoginOTP.Query().
		Where(
			loginotp.EmailEQ(email),
			loginotp.ConsumedAtIsNil(),
			loginotp.ExpiresAtGT(now),
		).
		Order(ent.Desc(loginotp.FieldCreatedAt), ent.Desc(loginotp.FieldID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.OTP{}, base.ErrNotFound
		}
		return domain.OTP{}, err
	}
	return toOTPDomain(record), nil
}

// IncrementAttempts atomically increments the attempt count for an OTP if it is unconsumed and attempts < max.
// Returns the new attempt count, or base.ErrNotFound if 0 rows were affected.
func (r *otpRepository) IncrementAttempts(ctx context.Context, id int64, max int) (int, error) {
	affected, err := r.cli.LoginOTP.Update().
		Where(
			loginotp.IDEQ(id),
			loginotp.ConsumedAtIsNil(),
			loginotp.AttemptsLT(max),
		).
		AddAttempts(1).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		return 0, base.ErrNotFound
	}

	record, err := r.cli.LoginOTP.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, base.ErrNotFound
		}
		return 0, err
	}
	return record.Attempts, nil
}

// Consume conditionally marks an OTP as consumed if consumed_at is currently NULL.
// Returns true if the record was updated, or false if it was already consumed or not found.
func (r *otpRepository) Consume(ctx context.Context, id int64, now time.Time) (bool, error) {
	affected, err := r.cli.LoginOTP.Update().
		Where(
			loginotp.IDEQ(id),
			loginotp.ConsumedAtIsNil(),
		).
		SetConsumedAt(now).
		Save(ctx)
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// DeleteOlderThan removes all OTP records for the specified email created before the given cutoff time.
func (r *otpRepository) DeleteOlderThan(ctx context.Context, email string, before time.Time) error {
	_, err := r.cli.LoginOTP.Delete().
		Where(
			loginotp.EmailEQ(email),
			loginotp.CreatedAtLT(before),
		).
		Exec(ctx)
	return err
}

func toOTPDomain(rec *ent.LoginOTP) domain.OTP {
	return domain.OTP{
		ID:         rec.ID,
		Email:      rec.Email,
		CodeHash:   rec.CodeHash,
		ExpiresAt:  rec.ExpiresAt,
		Attempts:   rec.Attempts,
		ConsumedAt: rec.ConsumedAt,
		IPAddress:  rec.IPAddress,
		CreatedAt:  rec.CreatedAt,
	}
}
