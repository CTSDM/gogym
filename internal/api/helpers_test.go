package api

import (
	"context"
	"math/rand"
	"strings"
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

	return user.ID.Bytes
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	sb := strings.Builder{}
	sb.Grow(length)
	for range length {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

func createSessionDBTestHelper(t testing.TB, s *State, name string) uuid.UUID {
	// generate random string and random password to be used in the database
	userID, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		Username:       randomString(10),
		HashedPassword: randomString(10),
	})
	require.NoError(t, err)
	session, err := s.db.CreateSession(context.Background(),
		database.CreateSessionParams{
			Name:   name,
			Date:   pgtype.Date{Time: time.Now(), Valid: true},
			UserID: pgtype.UUID{Bytes: userID.ID.Bytes, Valid: true},
		})
	require.NoError(t, err)

	return session.ID.Bytes
}
