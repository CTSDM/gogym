package testutil

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func SetupTestDB(ctx context.Context) (*pgxpool.Pool, func(), error) {
	dbName := "testdb"
	dbUser := "user"
	dbPassword := "password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not start the container: %w", err)
	}

	connURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	dbPool, err := pgxpool.New(context.Background(), connURL)
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to db: %w", err)
	}

	db := stdlib.OpenDBFromPool(dbPool)

	migrationsPath := findMigrationsPath()
	if err := goose.UpContext(ctx, db, migrationsPath); err != nil {
		return nil, nil, fmt.Errorf("could not run migrations: %w", err)
	}

	cleanup := func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	return dbPool, cleanup, nil
}

func Cleanup(dbPool *pgxpool.Pool, tableTarget string) error {
	tables := []string{
		"users",
		"exercises",
		"logs",
		"sessions",
		"sets",
		"refresh_tokens",
	}

	if tableTarget == "" {
		for _, table := range tables {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, err := dbPool.Exec(ctx, fmt.Sprintf("DELETE FROM %s;", table)); err != nil {
				return fmt.Errorf("could not delete table %s: %w", table, err)
			}
		}
		return nil
	}

	if !slices.Contains(tables, tableTarget) {
		return fmt.Errorf("could not find table %s", tableTarget)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := dbPool.Exec(ctx, fmt.Sprintf("DELETE FROM %s;", tableTarget)); err != nil {
		return fmt.Errorf("could not delete table %s: %w", tableTarget, err)
	}

	return nil
}

func CreateTokensDBHelperTest(t testing.TB, db *database.Queries, authConfig *auth.Config, userID uuid.UUID) (string, string) {
	refreshToken, err := auth.MakeRefreshToken()
	require.NoError(t, err)
	jwt, err := auth.MakeJWT(userID.String(), authConfig.JWTsecret, authConfig.JWTDuration)
	require.NoError(t, err)
	_, err = db.CreateRefreshToken(context.Background(),
		database.CreateRefreshTokenParams{
			Token:     refreshToken,
			ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(authConfig.RefreshTokenDuration), Valid: true},
			UserID:    userID,
		})
	require.NoError(t, err)

	return jwt, refreshToken
}

func CreateUserDBTestHelper(t testing.TB, db *database.Queries, username, password string, hasBirthay bool) database.User {
	hashedPassword, err := auth.HashPassword(password)
	require.NoError(t, err)

	dbParams := database.CreateUserParams{
		Username:       username,
		HashedPassword: hashedPassword,
	}

	if hasBirthay {
		dbParams.Birthday = pgtype.Date{Time: time.Now().AddDate(-50, 0, 0), Valid: true}
	}

	user, err := db.CreateUser(context.Background(), dbParams)
	require.NoError(t, err)

	return user
}

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	sb := strings.Builder{}
	sb.Grow(length)
	for range length {
		sb.WriteByte(charset[rand.Intn(len(charset))]) // #nosec G404
	}
	return sb.String()
}

func CreateSessionDBTestHelper(t testing.TB, db *database.Queries, name string, userID uuid.UUID) uuid.UUID {
	// generate random string and random password to be used in the database
	session, err := db.CreateSession(context.Background(),
		database.CreateSessionParams{
			Name:   name,
			Date:   pgtype.Date{Time: time.Now(), Valid: true},
			UserID: userID,
		})
	require.NoError(t, err)

	return session.ID
}

func CreateSetDBTestHelper(t testing.TB, db *database.Queries, sessionID uuid.UUID, exerciseID int32) int64 {
	set, err := db.CreateSet(context.Background(),
		database.CreateSetParams{
			SetOrder:   1,
			RestTime:   pgtype.Int4{Int32: 90, Valid: true},
			SessionID:  sessionID,
			ExerciseID: exerciseID,
		})
	require.NoError(t, err)

	return set.ID
}

func CreateExerciseDBTestHelper(t testing.TB, db *database.Queries, name string) int32 {
	exercise, err := db.CreateExercise(context.Background(), database.CreateExerciseParams{
		Name:        name,
		Description: pgtype.Text{String: "", Valid: true},
	})
	require.NoError(t, err)
	return exercise.ID
}

func CreateExerciseWithDescDBTestHelper(t testing.TB, db *database.Queries, name, description string) int32 {
	exercise, err := db.CreateExercise(context.Background(), database.CreateExerciseParams{
		Name:        name,
		Description: pgtype.Text{String: description, Valid: true},
	})
	require.NoError(t, err)
	return exercise.ID
}

func CreateLogExerciseDBTestHelper(
	t testing.TB,
	db *database.Queries,
	reps, order, exerciseID int32,
	setID int64,
	weight float64) int64 {
	newLog, err := db.CreateLog(context.Background(), database.CreateLogParams{
		SetID:      setID,
		ExerciseID: exerciseID,
		Weight:     pgtype.Float8{Float64: weight, Valid: true},
		Reps:       reps,
		LogsOrder:  order,
	})
	require.NoError(t, err)
	return newLog.ID
}

func findMigrationsPath() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get working directory: %s", err)
	}

	for {
		migrationsPath := filepath.Join(wd, "sql", "schema")
		if _, err := os.Stat(migrationsPath); err == nil {
			return migrationsPath
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			log.Fatal("could not find sql/schema directory")
		}
		wd = parent
	}
}
