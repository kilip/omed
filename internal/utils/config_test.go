package utils_test

import (
	"testing"

	"github.com/kilip/omed/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigValue(t *testing.T) {
	conf := utils.NewConfig()
	assert.Equal(t, 3000, conf.Http.Port)
	assert.Equal(t, 5, conf.Http.Context.Timeout)
}
