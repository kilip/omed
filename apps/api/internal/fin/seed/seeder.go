// Package seed provides chart of accounts template loading and workspace seeding capabilities.
package seed

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/finsetting"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/platform/entx"
)

//go:embed coa/*.json
var templateFS embed.FS

// AccountItem represents a single account record defined within a COA template.
type AccountItem struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	NormalBalance string `json:"normal_balance"`
	ParentCode    string `json:"parent_code,omitempty"`
	Description   string `json:"description,omitempty"`
	IsPostable    bool   `json:"is_postable"`
}

// LoadTemplate reads and parses the JSON template file matching the template name and language.
func LoadTemplate(template, lang string) ([]AccountItem, error) {
	template = strings.ToLower(strings.TrimSpace(template))
	lang = strings.ToLower(strings.TrimSpace(lang))
	if template == "" || lang == "" {
		return nil, domain.ErrAccountTemplateNotFound
	}

	fileName := fmt.Sprintf("coa/%s.%s.json", template, lang)
	data, err := templateFS.ReadFile(fileName)
	if err != nil {
		return nil, domain.ErrAccountTemplateNotFound
	}

	var items []AccountItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parsing template %s: %w", fileName, err)
	}

	return items, nil
}

// Seed executes chart of accounts population for the specified workspace.
// It fails with domain.ErrAccountWorkspaceNotEmpty if the workspace already contains accounts.
func Seed(ctx context.Context, client *ent.Client, workspaceID int64, template, lang string) (int, error) {
	items, err := LoadTemplate(template, lang)
	if err != nil {
		return 0, err
	}

	ctx = base.WithWorkspaceID(ctx, workspaceID)

	// Ensure the workspace is clean before seeding
	existingCount, err := client.Account.Query().
		Where(
			account.WorkspaceIDEQ(workspaceID),
			account.DeletedAtIsNil(),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	if existingCount > 0 {
		return 0, domain.ErrAccountWorkspaceNotEmpty
	}

	codeToID := make(map[string]int64, len(items))

	err = entx.WithTx(ctx, client, func(tx *ent.Tx) error {
		for _, item := range items {
			builder := tx.Account.Create().
				SetWorkspaceID(workspaceID).
				SetCode(item.Code).
				SetName(item.Name).
				SetType(account.Type(item.Type)).
				SetNormalBalance(account.NormalBalance(item.NormalBalance)).
				SetIsActive(true).
				SetIsPostable(item.IsPostable)

			if item.Description != "" {
				builder.SetDescription(item.Description)
			}

			if item.ParentCode != "" {
				parentID, ok := codeToID[item.ParentCode]
				if !ok {
					return fmt.Errorf("parent code %s not found for account %s", item.ParentCode, item.Code)
				}
				builder.SetParentID(parentID)
			}

			created, err := builder.Save(ctx)
			if err != nil {
				return fmt.Errorf("saving account %s (%s): %w", item.Code, item.Name, err)
			}

			codeToID[item.Code] = created.ID
		}

		// Configure workspace financial settings and automatically link FX accounts if present
		var fxGainID, fxLossID, fxUnrealizedGainID, fxUnrealizedLossID, fxRoundingID *int64
		if id, ok := codeToID["7210"]; ok {
			fxGainID = &id
		}
		if id, ok := codeToID["7310"]; ok {
			fxLossID = &id
		}
		if id, ok := codeToID["7220"]; ok {
			fxUnrealizedGainID = &id
		}
		if id, ok := codeToID["7320"]; ok {
			fxUnrealizedLossID = &id
		}
		if id, ok := codeToID["7400"]; ok {
			fxRoundingID = &id
		}

		existingSetting, err := tx.FinSetting.Query().
			Where(finsetting.WorkspaceIDEQ(workspaceID)).
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return fmt.Errorf("querying workspace fin setting: %w", err)
		}

		if existingSetting != nil {
			updateBuilder := tx.FinSetting.UpdateOne(existingSetting).
				SetOnboarding(true)
			if fxGainID != nil {
				updateBuilder.SetFxGainAccountID(*fxGainID)
			}
			if fxLossID != nil {
				updateBuilder.SetFxLossAccountID(*fxLossID)
			}
			if fxUnrealizedGainID != nil {
				updateBuilder.SetFxUnrealizedGainAccountID(*fxUnrealizedGainID)
			}
			if fxUnrealizedLossID != nil {
				updateBuilder.SetFxUnrealizedLossAccountID(*fxUnrealizedLossID)
			}
			if fxRoundingID != nil {
				updateBuilder.SetFxRoundingAccountID(*fxRoundingID)
			}
			if _, err := updateBuilder.Save(ctx); err != nil {
				return fmt.Errorf("updating workspace fin setting: %w", err)
			}
		} else {
			createBuilder := tx.FinSetting.Create().
				SetWorkspaceID(workspaceID).
				SetOnboarding(true)
			if fxGainID != nil {
				createBuilder.SetFxGainAccountID(*fxGainID)
			}
			if fxLossID != nil {
				createBuilder.SetFxLossAccountID(*fxLossID)
			}
			if fxUnrealizedGainID != nil {
				createBuilder.SetFxUnrealizedGainAccountID(*fxUnrealizedGainID)
			}
			if fxUnrealizedLossID != nil {
				createBuilder.SetFxUnrealizedLossAccountID(*fxUnrealizedLossID)
			}
			if fxRoundingID != nil {
				createBuilder.SetFxRoundingAccountID(*fxRoundingID)
			}
			if _, err := createBuilder.Save(ctx); err != nil {
				return fmt.Errorf("creating workspace fin setting: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return len(items), nil
}
