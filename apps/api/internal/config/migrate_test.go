package config_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/kilip/omed/api/internal/config"
	"github.com/stretchr/testify/suite"
)

type MigrateSuite struct {
	suite.Suite
}

func (s *MigrateSuite) Test_RunMigrations_EmptyURL() {
	err := config.RunMigrations(context.Background(), "")
	s.Require().Error(err)
	s.Contains(err.Error(), "database URL is required")
}

func (s *MigrateSuite) Test_RunMigrationsFromFS_InvalidChecksum() {
	fakeFS := fstest.MapFS{
		"20260101000000_init.sql": &fstest.MapFile{
			Data: []byte("CREATE SCHEMA \"core\";\n"),
		},
		"atlas.sum": &fstest.MapFile{
			Data: []byte("h1:invalidchecksum=\n20260101000000_init.sql h1:wrong=\n"),
		},
	}

	err := config.RunMigrationsFromFS(context.Background(), "postgres://localhost:5432/unused", fakeFS)
	s.Require().Error(err)
	s.Contains(err.Error(), "validating migrations checksum")
}

func (s *MigrateSuite) Test_RunMigrations_InvalidConnection() {
	err := config.RunMigrations(context.Background(), "postgres://omed:omed@127.0.0.1:1/omed?sslmode=disable&connect_timeout=1")
	s.Require().Error(err)
	s.Contains(err.Error(), "pinging database")
}

func TestMigrateSuite(t *testing.T) {
	suite.Run(t, new(MigrateSuite))
}

