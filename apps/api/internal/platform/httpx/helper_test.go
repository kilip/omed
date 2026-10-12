package httpx_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HelperSuite struct {
	suite.Suite
}

func (s *HelperSuite) Test_GetByID() {
	s.Equal("hello", "hello")
}

func TestHelperSuite(t *testing.T) {
	suite.Run(t, &HelperSuite{})
}
