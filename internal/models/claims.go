package models

import (
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Email string `json:"email"`
	//has standard jwt field issued at, issued by etc
	jwt.RegisteredClaims
}
