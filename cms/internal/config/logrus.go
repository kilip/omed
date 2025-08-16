package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewLogger(config *viper.Viper) *logrus.Logger {
	log := logrus.New()

	log.SetLevel(logrus.Level(logrus.WarnLevel))
	log.SetFormatter(&logrus.JSONFormatter{})

	return log
}
