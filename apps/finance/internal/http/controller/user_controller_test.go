package controller_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/http/controller"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPing(t *testing.T) {
	tester := testutil.NewApiTester()

	tester.Execute(t, testutil.ApiRequest{
		Method: fiber.MethodGet,
		Path:   "/ping",
	})

	tester.OK(t)
	var resp model.WebResponse[controller.PingResponse]
	testutil.DecodeResponse(t, tester.Response, &resp)
	assert.Equal(t, "Test User", resp.Data.User.Name)
	assert.NotEmpty(t, resp.Meta)
	assert.Empty(t, resp.Errors)
}
