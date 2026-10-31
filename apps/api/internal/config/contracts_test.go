package config_test

import (
	"testing"

	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/internal/fin"
)

var _ config.HasMCP = (*fin.FinModule)(nil)

func Test_Contracts_FinModuleImplementsHasMCP(t *testing.T) {
	var m any = (*fin.FinModule)(nil)
	_, ok := m.(config.HasMCP)
	if !ok {
		t.Fatal("expected *fin.FinModule to implement config.HasMCP")
	}
}
