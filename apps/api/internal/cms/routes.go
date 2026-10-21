package cms

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/platform/authz"
)

func initRoutes(mod CmsModule, r fiber.Router, az *authz.Authorizer) {
	// CMS routes will be registered here as controllers are implemented.
}
