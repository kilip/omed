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

func setDefaults(v *viper.Viper){
	v.SetDefault("db.type", "postgres")
	v.SetDefault("db.host", "10.0.0.5")
	v.SetDefault("db.port", 5432)
	v.SetDefault("db.username", "omed")
	v.SetDefault("db.password", "omed")
	v.SetDefault("db.database", "omed")
	v.SetDefault("db.pool.idle", 10)
	v.SetDefault("db.pool.max", 100)
	v.SetDefault("db.pool.lifetime", 300)
	
	v.SetDefault("log.level", "debug")
	v.SetDefault("web.prefork", false)
	v.SetDefault("web.port", 3000)
}

func NewConfig() *OmedConfig {
	var v = viper.New()

	setDefaults(v)

	godotenv.Load()
	v.AutomaticEnv()
	v.SetEnvPrefix("omed")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var c OmedConfig
	
	err := v.Unmarshal(&c)
	if err != nil {
		panic(err)
	}
	return &c
}
