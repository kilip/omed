package config

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase(c *OmedConfig, log *logrus.Logger) *gorm.DB {
	
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		c.DB.Host,
		c.DB.Username,
		c.DB.Password,
		c.DB.Database,
		c.DB.Port,
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
	connection.SetMaxIdleConns(c.DB.Pool.Idle)
	connection.SetMaxOpenConns(c.DB.Pool.Max)
	connection.SetConnMaxLifetime(time.Second * time.Duration(c.DB.Pool.Lifetime))

	return db
}

type logrusWriter struct {
	Logger *logrus.Logger
}

func (l *logrusWriter) Printf(message string, args ...interface{}) {
	l.Logger.Tracef(message, args...)
}
