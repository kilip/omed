package main

import (
	"fmt"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"
	"github.com/kilip/omed/internal/entity"
	"github.com/kilip/omed/internal/infra/database"
	"github.com/kilip/omed/internal/utils"
)

// atlas loader
func main() {
	conf := utils.NewConfig()
	gdb := database.NewGormDB(conf)
	stmts, err := gormschema.New("postgres", gormschema.WithConfig(gdb.Config)).Load(
		&entity.User{},
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(stmts)
}
