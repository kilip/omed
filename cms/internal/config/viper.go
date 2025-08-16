package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func setDefaults(config *viper.Viper){
  config.SetDefault("admin.email", "admin@example.com")
  config.SetDefault("admin.password", "admin")
  config.SetDefault("admin.name", "Omed Admin User")

	config.SetDefault("db.driver", "sqlite")
	config.SetDefault("db.pool.idle", 10)
	config.SetDefault("db.pool.max", 100)
	config.SetDefault("db.pool.lifetime", 300)
	config.SetDefault("db.driver", "sqlite")

	// postgres
	config.SetDefault("postgres.host", "localhost")
	config.SetDefault("postgres.port", 5432)
	config.SetDefault("postgres.username", "omed")
	config.SetDefault("postgres.password", "omed")
	config.SetDefault("postgres.database", "omed")


	config.SetDefault("sqlite.path", "/data/omed.db")

	// mysql
	config.SetDefault("mysql.host", "localhost")
	config.SetDefault("mysql.port", 5432)
	config.SetDefault("mysql.username", "omed")
	config.SetDefault("mysql.password", "omed")
	config.SetDefault("mysql.database", "omed")

	config.SetDefault("log.level", "debug")
	config.SetDefault("web.prefork", false)
	config.SetDefault("web.port", 3000)
}

func loadEnv(config *viper.Viper){
	env := config.GetString("app.env")
	// ignore dotenv
	if env == "production" {
		return
	}

	_, path, _, _ :=runtime.Caller(1)
	root := filepath.Join(filepath.Dir(path), "../..");
	files := []string{
		root + "/.env",
		root + "/.env.local",
		root + "/.env." + env,
		root + "/.env." + env + ".local",
	}

	// configures app root
	config.Set("app.root", root)

	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			godotenv.Load(file)
		}
	}
}

func NewConfig() *viper.Viper {
	env := os.Getenv("OMED_APP_ENV")
	if env == "" {
		os.Setenv("OMED_APP_ENV", "dev")
		env = "dev"
	}

	var config = viper.New()

	config.Set("app.env", env)

	loadEnv(config)
	setDefaults(config)


	config.AutomaticEnv()
	config.SetEnvPrefix("omed")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// override sqlite path during development
	if config.Get("app.env") != "production" {
		path := fmt.Sprintf("%s/data/omed.%s.db", config.GetString("app.root"), config.GetString("app.env"))
		config.Set("sqlite.path", path)
	}

	return config
}
