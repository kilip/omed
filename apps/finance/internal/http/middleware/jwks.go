package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/shared"
)

func JWKSReadinessGate(manager *shared.JWKSManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !manager.Ready() {
			return c.Status(fiber.StatusServiceUnavailable).
				JSON(fiber.Map{"error": "auth not ready"})
		}
		return c.Next()
	}
}

func JWKSReadyzHandler(manager *shared.JWKSManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !manager.Ready() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not_ready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	}
}
