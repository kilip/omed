package config

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
