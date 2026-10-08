package seed_coa

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed freelancer.en.json
//go:embed freelancer.id.json
var coaFS embed.FS

type COAAccount struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	ParentCode *string `json:"parentCode"`
}

func Load(profile, lang string) ([]COAAccount, error) {
	filename := fmt.Sprintf("%s.%s.json", profile, lang)
	data, err := coaFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("seed profile not found: %s: %w", filename, err)
	}

	var accounts []COAAccount
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, fmt.Errorf("invalid seed data: %w", err)
	}
	return accounts, nil
}
