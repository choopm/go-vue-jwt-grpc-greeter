package jwthelper

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/metadata"
)

type CustomClaims struct {
	jwt.StandardClaims
	Username string `json:"username,omitempty"`
}

func CreateJWT(jwtSecret, username string, expires int64) (string, error) {
	claims := &CustomClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expires,
			IssuedAt:  time.Now().Unix(),
			Issuer:    "greeter",
		},
		Username: username,
	}
	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}
	return string(t), nil
}

func GetClaimsFromContext(jwtSecret string, ctx context.Context) *CustomClaims {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	authHeader := md["authorization"]
	if len(authHeader) == 0 {
		return nil
	}
	bearerToken := authHeader[0]
	tokenStr := strings.Join(strings.Split(bearerToken, " ")[1:], " ")

	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		log.Println(err)
		return nil
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims

	} else {
		log.Println(err)
		return nil
	}
}
