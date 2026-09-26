package controller

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/testutil"
)

func TestPing(t *testing.T) {
	req := testutil.ApiTester{}
	req.WithUser(&testutil.TestUser{})
	req.Execute(t, testutil.ApiRequest{
		Method: fiber.MethodGet,
		Path:   "/ping",
	})
}
