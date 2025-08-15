package config

import "github.com/sirupsen/logrus"

func NewLogger(config *OmedConfig) *logrus.Logger {
	log := logrus.New()

	log.SetLevel(logrus.Level(logrus.DebugLevel))
	log.SetFormatter(&logrus.JSONFormatter{})

	return log
}
