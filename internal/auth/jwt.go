package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func MakeJWT(userID string, tokenSecret string, expiresIn time.Duration) (string, error) {
	if userID == "" {
		return "", errors.New("userID cannot be empty")
	}
	if tokenSecret == "" {
		return "", errors.New("token secret cannot be empty")
	}
	if expiresIn == 0 {
		return "", errors.New("expiration time cannot be zero")
	}

	signingKey := []byte(tokenSecret)
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Issuer:    "gogym",
		Subject:   userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(signingKey)
}

func MakeRefreshToken() (string, error) {
	randomData := make([]byte, 32)
	if _, err := rand.Read(randomData); err != nil {
		return "", fmt.Errorf("could not generate the random string: %w", err)
	}
	randomString := hex.EncodeToString(randomData)
	return randomString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (string, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		})
	if err != nil {
		return "", err
	}

	userID, err := token.Claims.GetSubject()
	if err != nil {
		return "", err
	}

	return userID, nil
}

// The token values is expected to be as --> headerName: "tokenName tokenValue"
func GetHeaderValueToken(headers http.Header, headerName string) (string, error) {
	header := headers.Get(headerName)
	headerParts := strings.Fields(header)
	if len(headerParts) != 2 {
		return "", fmt.Errorf("token string for header %q not found", headerName)
	}
	return headerParts[1], nil
}
