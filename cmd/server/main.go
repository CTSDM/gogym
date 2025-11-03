package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/CTSDM/gogym/internal/api"
	"github.com/CTSDM/gogym/internal/database"
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
	apiState := api.NewState(dbQueries)
	log.Fatalf("something went wrong while setting up the server: %s", apiState.SetupServer().Error())
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
