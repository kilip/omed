package utils

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kilip/omed/internal/infra/config"
	"github.com/spf13/viper"
)

func NewConfig() config.Config {
	conf := NewViper()
	var cfg config.Config
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
