package config

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type OmedConfig struct {
	AppName string `mapstructure:"name"`
	DB struct{
		Type string `mapstructure:"type"`
		Host string `mapstructure:"host"`
		Port int `mapstructure:"port"`
		Database string `mapstructure:"database"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Pool struct {
			Idle int `mapstructure:"idle"`
			Max int `mapstructure:"max"`
			Lifetime int `mapstructure:"lifetime"`
		} `mapstructure:"pool"`
	} `mapstructure:"db"`
	LogLevel string `mapstructure:"log_level"`
	Web struct {
		Prefork bool `mapstructure:"prefork"`
		Port int `mapstructure:"port"`
	} `mapstructure:"web"`
	

}

func setDefaults(config *viper.Viper){
	config.SetDefault("db.type", "postgres")
	config.SetDefault("db.host", "10.0.0.5")
	config.SetDefault("db.port", 5432)
	config.SetDefault("db.username", "omed")
	config.SetDefault("db.password", "omed")
	config.SetDefault("db.database", "omed")
	config.SetDefault("db.pool.idle", 10)
	config.SetDefault("db.pool.max", 100)
	config.SetDefault("db.pool.lifetime", 300)
	
	config.SetDefault("log.level", "debug")
	config.SetDefault("web.prefork", false)
	config.SetDefault("web.port", 3000)
}

func NewConfig() *viper.Viper {
	var config = viper.New()

	setDefaults(config)

	godotenv.Load()
	config.AutomaticEnv()
	config.SetEnvPrefix("omed")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	return config
}
