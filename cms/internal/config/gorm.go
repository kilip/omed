package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kilip/omed/cms/internal/entity"
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

// Generate postgres connection
func genPostgres(conf *viper.Viper, gormConfig *gorm.Config) (*gorm.DB, error){
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		conf.Get("postgres.host"),
		conf.GetInt("postgres.port"),
		conf.Get("postgres.username"),
		conf.Get("postgres.password"),
		conf.Get("postgres.database"),
		conf.Get("postgres.ssl"),
	)

	return gorm.Open(postgres.Open(dsn), gormConfig)
}

func genSqlite(conf *viper.Viper, gormConfig *gorm.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("file:%s", conf.GetString("sqlite.path"))
	return gorm.Open(sqlite.Open(dsn), gormConfig)
}

func genDB(conf *viper.Viper, log *logrus.Logger)(*gorm.DB, error){
	driver := conf.GetString("db.driver")
	gormConfig := &gorm.Config{
		Logger: logger.New(&logrusWriter{Logger: log}, logger.Config{
			SlowThreshold: time.Second * 5,
			Colorful: false,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries: true,
			LogLevel: logger.Info,
		}),
	}
	switch driver {
		case "postgres":
			return genPostgres(conf, gormConfig)
		case "sqlite":
			return genSqlite(conf, gormConfig)
	}
	return nil, errors.New("Unsupported database driver: " + driver)
}

func NewDatabase(conf *viper.Viper, log *logrus.Logger) *gorm.DB {

	db, err := genDB(conf, log)

	if err != nil {
		log.Fatalf("failed to connect database %v", err)
	}

	connection, err := db.DB()
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// connection.SetMaxIdleConns()
	connection.SetMaxIdleConns(conf.GetInt("db.pool.idle"))
	connection.SetMaxOpenConns(conf.GetInt("db.pool.max"))
	connection.SetConnMaxLifetime(time.Second * time.Duration(conf.GetInt("db.pool.lifetime")))

  // TODO: move this into migration
  db.AutoMigrate(&entity.User{}, &entity.UserToken{})

	return db
}
