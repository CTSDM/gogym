package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/CTSDM/gogym/internal/api"
	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("could not load env variables: %s", err.Error())
	}
	dbQueries, err := getDB()
	if err != nil {
		log.Fatalf("could not connect to the database: %s", err.Error())
	}
	if err := initialSetup(dbQueries); err != nil {
		log.Fatalf("could not set the initial setup: %s", err)
	}
	authConfig, err := getAuthConfig()
	if err != nil {
		log.Fatalf("could not set up the auth config: %s", err.Error())
	}
	apiState := api.NewState(dbQueries, authConfig)
	log.Fatalf("something went wrong while setting up the server: %s", apiState.SetupServer().Error())
}

func getAuthConfig() (*auth.Config, error) {
	jwtSecret, ok := os.LookupEnv("JWT_SECRET")
	if !ok {
		return nil, fmt.Errorf("JWT secret was not found on the env file")
	}
	jwtDurationStr, ok := os.LookupEnv("JWT_DURATION")
	if !ok {
		return nil, fmt.Errorf("JWT duration was not found on the env file")
	}
	jwtDurationInt, err := strconv.Atoi(jwtDurationStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse JWT duration into an integer: %s", jwtDurationStr)
	}
	refreshTokenDurationStr, ok := os.LookupEnv("REFRESH_TOKEN_DURATION")
	if !ok {
		return nil, fmt.Errorf("refresh token duration was not found on the env file")
	}
	refreshTokenDurationInt, err := strconv.Atoi(refreshTokenDurationStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse JWT duration into an integer: %s", refreshTokenDurationStr)
	}
	return &auth.Config{
		JWTsecret:            jwtSecret,
		JWTDuration:          time.Duration(jwtDurationInt) * time.Second,
		RefreshTokenDuration: time.Duration(refreshTokenDurationInt) * time.Second,
	}, nil
}

func dbSetup(db *database.Queries, username, password string) error {
	// check if the admin exists
	user, err := db.GetUserByUsername(context.Background(), username)
	switch err {
	case nil:
		if user.IsAdmin.Bool {
			log.Println("Admin found on the database")
			return nil
		} else {
			return errors.New("found user instead of admin when setting up the admin")
		}
	case pgx.ErrNoRows:
	default:
		return fmt.Errorf("something went wrong while fetching the user from the database: %s", err)
	}

	// create the admin
	hashed, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("could not generate the hashed password: %s", err)
	}
	if _, err := db.CreateAdmin(context.Background(), database.CreateAdminParams{
		Username:       username,
		HashedPassword: hashed,
	}); err != nil {
		return fmt.Errorf("something went wrong while creating the admin: %s", err)
	}

	return nil
}

func initialSetup(db *database.Queries) error {
	adminUsername, ok := os.LookupEnv("ADMIN_USERNAME")
	if !ok {
		return errors.New("failed to obtain the admin username")
	}
	adminPassword, ok := os.LookupEnv("ADMIN_PASSWORD")
	if !ok {
		return errors.New("failed to obtain the admin password")
	}

	return dbSetup(db, adminUsername, adminPassword)

}

func getDB() (*database.Queries, error) {
	dbConnURL, err := getDBConnURL()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain the database url: %w", err)
	}
	dbPool, err := startDB(dbConnURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}
	return database.New(dbPool), nil
}

func startDB(connPath string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), connPath)
	if err != nil {
		return nil, fmt.Errorf("could not open the database: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("could not ping the database: %w", err)
	}

	return pool, nil
}

func getDBConnURL() (string, error) {
	devFlag, ok := os.LookupEnv("DEV")
	if !ok {
		return "", errors.New("dev flag not found on the .env file")
	}
	dbUsername, ok := os.LookupEnv("POSTGRES_USER")
	if !ok {
		return "", errors.New("username for the database connection was not found on the .env file")
	}
	dbPassword, ok := os.LookupEnv("POSTGRES_PASSWORD")
	if !ok {
		return "", errors.New("password for the database connection was not found on the .env file")
	}
	dbHostPort, ok := os.LookupEnv("POSTGRES_HOST_PORT")
	if !ok {
		return "", errors.New("host or/and port for the database connection was not found on the .env file")
	}
	database, ok := os.LookupEnv("POSTGRES_DB")
	if !ok {
		return "", errors.New("database name for the database connection was not found on the .env file")
	}

	url := fmt.Sprintf("postgresql://%s:%s@%s/%s", dbUsername, dbPassword, dbHostPort, database)
	if devFlag == "1" {
		url += "?sslmode=disable"
	}
	return url, nil
}
