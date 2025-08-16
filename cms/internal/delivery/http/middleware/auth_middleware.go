package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/model"
	"github.com/kilip/omed/cms/internal/service"
)

func AuthMiddleware(auth *service.AuthService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		path := ctx.Path()
		if strings.Contains(path, "auth") || strings.Contains(path, "register") {
			return ctx.Next()
		}
		
		request := &model.VerifyUserRequest{Token: ctx.Get("Authorization", "NOT_FOUND")}
		auth.Log.Debugf("Authorization : %s", request.Token)

		user, err := auth.Verify(ctx.UserContext(), request)
		if err != nil {
			auth.Log.Warnf("Failed find user by token : %+v", err)
			return fiber.ErrUnauthorized
		}

		auth.Log.Debugf("User : %+v", user.UserID)
		ctx.Locals("auth", user)
		return ctx.Next()
	}
}


func GetUser(ctx *fiber.Ctx) *model.AuthResponse {
	return ctx.Locals("auth").(*model.AuthResponse)
}
