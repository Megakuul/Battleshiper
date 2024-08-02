package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtOptions struct {
	Secret string
	TTL    time.Duration
}

type UserClaims struct {
	Id        string `json:"id"`
	Provider  string `json:"provider"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	jwt.RegisteredClaims
}

// CreateJWT generates a jwt token based on the input options.
func CreateJWT(options *JwtOptions, id, provider, username, avatarUrl string) (string, error) {
	claims := &UserClaims{
		Id:        id,
		Provider:  provider,
		Username:  username,
		AvatarURL: avatarUrl,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(options.TTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(options.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseJWT verifies the jwt string based on the provided options. It returns the user claims or an error if invalid.
func ParseJWT(options *JwtOptions, token string) (*UserClaims, error) {
	claims := &UserClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(options.Secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		return nil, err
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
