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

func createUserDBTestHelper(t testing.TB, s *State, username, password string, hasBirthay bool) database.User {
	hashedPassword, err := auth.HashPassword(password)
	require.NoError(t, err)

	dbParams := database.CreateUserParams{
		Username:       username,
		HashedPassword: hashedPassword,
	}

	if hasBirthay {
		dbParams.Birthday = pgtype.Date{Time: time.Now().AddDate(-50, 0, 0), Valid: true}
	}

	user, err := s.db.CreateUser(context.Background(), dbParams)
	require.NoError(t, err)

	return user
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

func createSetDBTestHelper(t testing.TB, s *State, sessionID uuid.UUID) int64 {
	set, err := s.db.CreateSet(context.Background(),
		database.CreateSetParams{
			SetOrder:  1,
			RestTime:  pgtype.Int4{Int32: 90, Valid: true},
			SessionID: pgtype.UUID{Bytes: sessionID, Valid: true},
		})
	require.NoError(t, err)

	return set.ID
}

func createExerciseDBTestHelper(t testing.TB, s *State, name string) int64 {
	exercise, err := s.db.CreateExercise(context.Background(), database.CreateExerciseParams{
		Name:        name,
		Description: pgtype.Text{String: "", Valid: true},
	})
	require.NoError(t, err)
	return int64(exercise.ID)
}

func createExerciseWithDescDBTestHelper(t testing.TB, s *State, name, description string) int64 {
	exercise, err := s.db.CreateExercise(context.Background(), database.CreateExerciseParams{
		Name:        name,
		Description: pgtype.Text{String: description, Valid: true},
	})
	require.NoError(t, err)
	return int64(exercise.ID)
}
