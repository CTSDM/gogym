package auth

import "time"

type Config struct {
	JWTsecret            string
	RefreshTokenDuration time.Duration
	JWTDuration          time.Duration
}
