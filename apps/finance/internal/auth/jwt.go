package auth

import (
	"context"
	"log"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	jwt.RegisteredClaims
}

func ValidateJWT(ctx context.Context, input string){
	jwksUrl := "http://localhost:3000/api/auth/jwks"

	k, err := keyfunc.NewDefaultCtx(ctx, []string{jwksUrl})
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	var claims JWTClaims
	token, err := jwt.ParseWithClaims(input, claims, k.Keyfunc)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	if token.Valid {
		log.Printf("Token valid %s", claims.ID)
	}else{
		log.Printf("Token invalid")
	}
	

}