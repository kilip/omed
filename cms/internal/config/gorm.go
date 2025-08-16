package config

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)


type logrusWriter struct {
	Logger *logrus.Logger
}

func (l *logrusWriter) Printf(message string, args ...interface{}) {
	l.Logger.Tracef(message, args...)
}

func NewDatabase(c *viper.Viper, log *logrus.Logger) *gorm.DB {
	
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		c.Get("db.host"),
		c.Get("db.username"),
		c.Get("db.password"),
		c.Get("db.database"),
		c.Get("db.port"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(&logrusWriter{Logger: log}, logger.Config{
			SlowThreshold: time.Second * 5,
			Colorful: false,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries: true,
			LogLevel: logger.Info,
		}),
	})

	if err != nil {
		log.Fatalf("failed to connect database %v", err)
	}

	connection, err := db.DB()
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// connection.SetMaxIdleConns()
	connection.SetMaxIdleConns(c.GetInt("db.pool.idle"))
	connection.SetMaxOpenConns(c.GetInt("db.pool.max"))
	connection.SetConnMaxLifetime(time.Second * time.Duration(c.GetInt("db.pool.lifetime")))

	return db
}
