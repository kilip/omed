package authz

import (
	"embed"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	stringadapter "github.com/qiangmzsx/string-adapter/v2"
)

//go:embed model.conf
var modelFS embed.FS

//go:embed policy.csv
var policyFS embed.FS

func NewEnforcer() (*casbin.Enforcer, error) {
	modelData, err := modelFS.ReadFile("model.conf")
	if err != nil {
		return nil, err
	}
	policyData, err := policyFS.ReadFile("policy.csv")
	if err != nil {
		return nil, err
	}

	m, err := model.NewModelFromString(string(modelData))
	if err != nil {
		return nil, err
	}
	adapter := stringadapter.NewAdapter(string(policyData))
	return casbin.NewEnforcer(m, adapter)
}
