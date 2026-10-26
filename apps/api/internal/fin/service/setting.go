// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"

	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
)

// SettingRepository defines persistence operations for workspace financial settings.
type SettingRepository interface {
	Get(c context.Context, workspaceID int64) (domain.FinSetting, error)
	Save(c context.Context, setting domain.FinSetting) (domain.FinSetting, error)
}

// SettingService coordinates financial settings retrieval and updates.
type SettingService struct {
	settingRepo SettingRepository
	accountRepo AccountRepository
}

// NewSettingService constructs a new SettingService instance.
func NewSettingService(settingRepo SettingRepository, accountRepo AccountRepository) SettingService {
	return SettingService{
		settingRepo: settingRepo,
		accountRepo: accountRepo,
	}
}

// GetFXAccounts returns the foreign exchange accounts configured for a workspace.
func (s SettingService) GetFXAccounts(c context.Context, workspaceID int64) (model.FXAccountsSettingResponse, error) {
	setting, err := s.settingRepo.Get(c, workspaceID)
	if err != nil {
		if err == domain.ErrFinSettingNotFound {
			return model.FXAccountsSettingResponse{}, nil
		}
		return model.FXAccountsSettingResponse{}, err
	}

	return model.FXAccountsSettingResponse{
		FXGainAccountID:           setting.FXGainAccountID,
		FXLossAccountID:           setting.FXLossAccountID,
		FXUnrealizedGainAccountID: setting.FXUnrealizedGainAccountID,
		FXUnrealizedLossAccountID: setting.FXUnrealizedLossAccountID,
		FXRoundingAccountID:       setting.FXRoundingAccountID,
	}, nil
}

// UpdateFXAccounts updates the foreign exchange accounts mapping for a workspace after validation.
func (s SettingService) UpdateFXAccounts(c context.Context, workspaceID int64, req model.UpdateFXAccountsRequest) (model.FXAccountsSettingResponse, error) {
	accountIDs := []*int64{
		req.FXGainAccountID,
		req.FXLossAccountID,
		req.FXUnrealizedGainAccountID,
		req.FXUnrealizedLossAccountID,
		req.FXRoundingAccountID,
	}

	for _, accID := range accountIDs {
		if accID == nil {
			continue
		}
		acc, err := s.accountRepo.GetByID(c, *accID)
		if err != nil {
			return model.FXAccountsSettingResponse{}, domain.ErrAccountNotFound
		}
		if !acc.IsActive || !acc.IsPostable {
			return model.FXAccountsSettingResponse{}, domain.ErrFXAccountInvalid
		}
	}

	setting, err := s.settingRepo.Get(c, workspaceID)
	if err != nil && err != domain.ErrFinSettingNotFound {
		return model.FXAccountsSettingResponse{}, err
	}
	if err == domain.ErrFinSettingNotFound {
		setting = domain.FinSetting{WorkspaceID: workspaceID}
	}

	setting.FXGainAccountID = req.FXGainAccountID
	setting.FXLossAccountID = req.FXLossAccountID
	setting.FXUnrealizedGainAccountID = req.FXUnrealizedGainAccountID
	setting.FXUnrealizedLossAccountID = req.FXUnrealizedLossAccountID
	setting.FXRoundingAccountID = req.FXRoundingAccountID

	saved, err := s.settingRepo.Save(c, setting)
	if err != nil {
		return model.FXAccountsSettingResponse{}, err
	}

	return model.FXAccountsSettingResponse{
		FXGainAccountID:           saved.FXGainAccountID,
		FXLossAccountID:           saved.FXLossAccountID,
		FXUnrealizedGainAccountID: saved.FXUnrealizedGainAccountID,
		FXUnrealizedLossAccountID: saved.FXUnrealizedLossAccountID,
		FXRoundingAccountID:       saved.FXRoundingAccountID,
	}, nil
}
