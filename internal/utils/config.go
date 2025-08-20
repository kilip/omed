package utils

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App struct {
		Path   string `mapstructure:"path"`
		Secret string `mapstructure:"secret"`
	} `mapstructure:"app"`
	Api struct {
		Host    string `mapstructure:"host" json:"host"`
		Port    int    `mapstructure:"port" json:"port"`
		Context struct {
			Timeout int `mapstructure:"timeout" json:"timeout"`
		} `mapstructure:"context" json:"context"`
	} `mapstructure:"http" json:"http"`
	Postgres struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Database string `mapstructure:"database"`
		SSLMode  string `mapstructure:"sslmode"`
	} `mapstructure:"postgres" json:"postgres"`
}

func NewConfig() Config {
	conf := NewViper()
	var cfg Config
	if err := conf.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	return cfg
}

func NewViper() *viper.Viper {
	conf := viper.New()

	// set root path
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Join(b, "../../../")
	conf.Set("app.path", basepath)

	conf.SetConfigName("config")
	conf.SetConfigType("json")
	conf.AddConfigPath(basepath + "/config")

	conf.ReadInConfig()

	conf.AutomaticEnv()
	conf.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	conf.SetEnvPrefix("omed")

	return conf
}
