// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/finsetting"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// SettingRepository provides persistence operations for workspace financial settings.
type SettingRepository struct {
	cli *ent.Client
}

// NewSettingRepository constructs a new SettingRepository instance.
func NewSettingRepository(cli *ent.Client) SettingRepository {
	return SettingRepository{cli: cli}
}

// Get retrieves the financial settings configured for a workspace.
func (r SettingRepository) Get(c context.Context, workspaceID int64) (domain.FinSetting, error) {
	s, err := r.cli.FinSetting.Query().
		Where(finsetting.WorkspaceIDEQ(workspaceID)).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.FinSetting{WorkspaceID: workspaceID}, domain.ErrFinSettingNotFound
		}
		return domain.FinSetting{}, err
	}
	return toFinSettingDomain(s), nil
}

// Save creates or updates the financial settings for a workspace.
func (r SettingRepository) Save(c context.Context, setting domain.FinSetting) (domain.FinSetting, error) {
	var result domain.FinSetting
	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		existing, err := tx.FinSetting.Query().
			Where(finsetting.WorkspaceIDEQ(setting.WorkspaceID)).
			Only(c)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}

		if existing != nil {
			builder := tx.FinSetting.UpdateOne(existing)
			if setting.FXGainAccountID != nil {
				builder.SetFxGainAccountID(*setting.FXGainAccountID)
			} else {
				builder.ClearFxGainAccountID()
			}
			if setting.FXLossAccountID != nil {
				builder.SetFxLossAccountID(*setting.FXLossAccountID)
			} else {
				builder.ClearFxLossAccountID()
			}
			if setting.FXUnrealizedGainAccountID != nil {
				builder.SetFxUnrealizedGainAccountID(*setting.FXUnrealizedGainAccountID)
			} else {
				builder.ClearFxUnrealizedGainAccountID()
			}
			if setting.FXUnrealizedLossAccountID != nil {
				builder.SetFxUnrealizedLossAccountID(*setting.FXUnrealizedLossAccountID)
			} else {
				builder.ClearFxUnrealizedLossAccountID()
			}
			if setting.FXRoundingAccountID != nil {
				builder.SetFxRoundingAccountID(*setting.FXRoundingAccountID)
			} else {
				builder.ClearFxRoundingAccountID()
			}

			saved, err := builder.Save(c)
			if err != nil {
				return err
			}
			result = toFinSettingDomain(saved)
			return nil
		}

		builder := tx.FinSetting.Create().
			SetWorkspaceID(setting.WorkspaceID)
		if setting.FXGainAccountID != nil {
			builder.SetFxGainAccountID(*setting.FXGainAccountID)
		}
		if setting.FXLossAccountID != nil {
			builder.SetFxLossAccountID(*setting.FXLossAccountID)
		}
		if setting.FXUnrealizedGainAccountID != nil {
			builder.SetFxUnrealizedGainAccountID(*setting.FXUnrealizedGainAccountID)
		}
		if setting.FXUnrealizedLossAccountID != nil {
			builder.SetFxUnrealizedLossAccountID(*setting.FXUnrealizedLossAccountID)
		}
		if setting.FXRoundingAccountID != nil {
			builder.SetFxRoundingAccountID(*setting.FXRoundingAccountID)
		}

		saved, err := builder.Save(c)
		if err != nil {
			return err
		}
		result = toFinSettingDomain(saved)
		return nil
	})
	return result, err
}

func toFinSettingDomain(s *ent.FinSetting) domain.FinSetting {
	return domain.FinSetting{
		ID:                        s.ID,
		WorkspaceID:               s.WorkspaceID,
		FXGainAccountID:           s.FxGainAccountID,
		FXLossAccountID:           s.FxLossAccountID,
		FXUnrealizedGainAccountID: s.FxUnrealizedGainAccountID,
		FXUnrealizedLossAccountID: s.FxUnrealizedLossAccountID,
		FXRoundingAccountID:       s.FxRoundingAccountID,
		CreatedAt:                 s.CreatedAt,
		UpdatedAt:                 s.UpdatedAt,
	}
}
