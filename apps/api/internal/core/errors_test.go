package core_test

import (
	"errors"
	"testing"

	"github.com/kilip/omed/api/internal/core"
	"github.com/stretchr/testify/suite"
)

type ErrorMapSuite struct {
	suite.Suite
}

func (s *ErrorMapSuite) Test_Map_Successfully() {
	item := core.MapError(core.ErrInvalidID)
	s.Equal(core.ErrInvalidID.Error(), item.Code)
}

func (s *ErrorMapSuite) Test_Map_Unregistered() {
	item := core.MapError(errors.New("unregistered"))
	s.Equal(core.ErrUnknown.Error(), item.Code)
}

func TestErrorMap(t *testing.T) {
	suite.Run(t, &ErrorMapSuite{})
}
