package api

import (
	"context"
	"testing"
	"time"

	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createTokensDBHelperTest(t testing.TB, userID uuid.UUID, s *State) (string, string) {
	refreshToken, err := auth.MakeRefreshToken()
	require.NoError(t, err)
	jwt, err := auth.MakeJWT(userID.String(), s.authConfig.JWTsecret, s.authConfig.JWTDuration)
	require.NoError(t, err)
	_, err = s.db.CreateRefreshToken(context.Background(),
		database.CreateRefreshTokenParams{
			Token:     refreshToken,
			ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(s.authConfig.RefreshTokenDuration), Valid: true},
			UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		})
	require.NoError(t, err)

	return jwt, refreshToken
}

func createUserDBTestHelper(t testing.TB, s *State, username, password string) uuid.UUID {
	hashedPassword, err := auth.HashPassword(password)
	require.NoError(t, err)

	user, err := s.db.CreateUser(context.Background(),
		database.CreateUserParams{
			Username:       username,
			HashedPassword: hashedPassword,
		})

	require.NoError(t, err)
	userID := uuid.MustParse(user.ID.String())
	return userID
}
