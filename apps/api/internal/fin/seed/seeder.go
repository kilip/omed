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
		return nil
	})

	if err != nil {
		return 0, err
	}

	return len(items), nil
}
