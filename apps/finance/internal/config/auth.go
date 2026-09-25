package config

import (
	"context"
	"encoding/json"
	"log"

	"github.com/casbin/casbin/v2"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/authz"
	"github.com/kilip/omed/finance/internal/model"
)

func InjectUser(c fiber.Ctx) error {
	token := jwtware.FromContext(c)
	claims := token.Claims.(jwt.MapClaims)
	b, err := json.Marshal(claims)
	if err != nil {
		return fiber.ErrUnauthorized
	}

	var user model.AuthenticatedUser
	if err := json.Unmarshal(b, &user); err != nil {
		return fiber.ErrInternalServerError
	}

	c.Locals("user", &user)
	return c.Next()
}

func CasbinMiddleware(enforcer *casbin.Enforcer) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := fiber.Locals[*model.AuthenticatedUser](c, "user")
		obj := c.Route().Path
		act := c.Method()

		for _, role := range user.Roles {
			if ok, _ := enforcer.Enforce(role, user.WorkspaceID, string(obj), string(act)); ok {
				return c.Next()
			}

			log.Printf("sub=%v dom=%v obj=%v act=%v", role, user.WorkspaceID, obj, act)
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
}

func ReadinessGate(manager *JWKSManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !manager.Ready() {
			return c.Status(fiber.StatusServiceUnavailable).
				JSON(fiber.Map{"error": "auth not ready"})
		}
		return c.Next()
	}
}

func ReadyzHandler(manager *JWKSManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !manager.Ready() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not_ready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	}
}

func configureAuth(cfg BootstrapConfig) {
	manager := NewJWKSManager(cfg.Config.App.JWKSUrl)
	manager.Start(context.Background())

	cfg.Fiber.Get("/readyz", ReadyzHandler(manager)) // daftar SEBELUM gate, biar selalu accessible

	cfg.Fiber.Use(ReadinessGate(manager))
	cfg.Fiber.Use(jwtware.New(jwtware.Config{
		KeyFunc:   manager.Keyfunc(),
		Extractor: extractors.FromAuthHeader("Bearer"),
	}))
	cfg.Fiber.Use(InjectUser)
}

func GetEnforcer() *casbin.Enforcer {
	enforcer, err := authz.NewEnforcer()
	if err != nil {
		log.Fatalf("Failed creating enforce: %v", err)
	}

	return enforcer
}
