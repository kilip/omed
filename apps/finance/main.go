package main

import (
	"log"

	"github.com/MicahParks/keyfunc/v3"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func main(){
	app := fiber.New()

	jwksURL := "http://localhost:3000/api/auth/jwks"
	
	// Create the JWKS keyfunc manually
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		// INSTEAD OF PANICKING: Log the error and fall back or retry
		log.Printf("WARNING: Failed to fetch JWKS on startup: %v. Server will still start.", err)
	}

	app.Use(jwtware.New(jwtware.Config{
		KeyFunc: func(token *jwt.Token) (interface{}, error) {
			if jwks == nil {
				return nil, fiber.NewError(fiber.StatusServiceUnavailable, "Authentication service unavailable")
			}
			return jwks.Keyfunc(token)
		},
	}))
	
	app.Get("/hello", func(c fiber.Ctx) error {
		user := jwtware.FromContext(c)
		claims := user.Claims.(jwt.MapClaims)

		return c.JSON(claims)
	})
	log.Fatal(app.Listen(":4123"))
}