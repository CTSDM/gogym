package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
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

	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.LookupEnv); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, w io.Writer, fCheckEnv func(string) (string, bool)) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	logHandler := slog.NewTextHandler(w, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})
	logger := slog.New(logHandler)

	env, err := loadEnvConfig(fCheckEnv)
	if err != nil {
		logger.Error("missing env parameter", slog.String("error", err.Error()))
		return err
	}

	logger.Info("connecting to database",
		slog.String("host", env.dbHostPort),
		slog.String("database", env.database),
	)
	dbPool, dbQueries, err := getDB(
		ctx,
		env.dbUsername,
		env.dbPassword,
		env.dbHostPort,
		env.database,
		env.devFlag,
	)
	if err != nil {
		logger.Error("could not connect to the database", slog.String("error", err.Error()))
		return fmt.Errorf("could not connect to the database: %w", err)
	}
	logger.Info("database connected")
	defer dbPool.Close()

	if err := initialSetup(ctx, dbQueries, env.adminUsername, env.adminPassword, logger); err != nil {
		logger.Error("could not finish the initial set up", slog.String("error", err.Error()))
		return fmt.Errorf("could not set the initial set up: %w", err)
	}
	authConfig, err := getAuthConfig(env.jwtSecret, env.jwtDuration, env.refreshTokenDuration)
	if err != nil {
		logger.Error("could not set up the auth config", slog.String("error", err.Error()))
		return fmt.Errorf("could not set up the auth config: %w", err)
	}
	server := api.NewServer(dbPool, dbQueries, authConfig, logger)

	httpServer := &http.Server{
		Addr:        ":" + env.serverPort,
		Handler:     server,
		ReadTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("starting server...", slog.String("port", env.serverPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("error while trying to listen and serve",
				slog.String("error", err.Error()),
				slog.String("addr", httpServer.Addr),
			)
			cancel()
		}
	}()

	var wg sync.WaitGroup
	wg.Go(func() {
		<-ctx.Done()
		timeout := 10 * time.Second
		logger.Info("received signal to cancel server, shutting down server...")
		shutDownCtx := context.Background()
		shutDownCtx, cancel := context.WithTimeout(shutDownCtx, timeout)
		defer cancel()
		if err := httpServer.Shutdown(shutDownCtx); err != nil {
			logger.Error("could not gracefully shut down server",
				slog.String("error", err.Error()),
				slog.Duration("timeout", timeout))
			return
		}
		logger.Info("server shut down gracefully...")
	})

	wg.Wait()
	return nil
}

type envConfig struct {
	jwtSecret            string
	jwtDuration          int
	refreshTokenDuration int
	adminUsername        string
	adminPassword        string
	devFlag              string
	dbUsername           string
	dbPassword           string
	dbHostPort           string
	database             string
	serverPort           string
}

func loadEnvConfig(fn func(string) (string, bool)) (*envConfig, error) {
	// Admin info
	adminUsername, ok := fn("ADMIN_USERNAME")
	if !ok {
		return nil, errors.New("failed to obtain the admin username")
	}
	adminPassword, ok := fn("ADMIN_PASSWORD")
	if !ok {
		return nil, errors.New("failed to obtain the admin password")
	}

	// Database variables
	devFlag, ok := fn("DEV")
	if !ok {
		return nil, errors.New("dev flag not found on the .env file")
	}
	dbUsername, ok := fn("POSTGRES_USER")
	if !ok {
		return nil, errors.New("username for the database connection was not found on the .env file")
	}
	dbPassword, ok := fn("POSTGRES_PASSWORD")
	if !ok {
		return nil, errors.New("password for the database connection was not found on the .env file")
	}
	dbHostPort, ok := fn("POSTGRES_HOST_PORT")
	if !ok {
		return nil, errors.New("host or/and port for the database connection was not found on the .env file")
	}
	database, ok := fn("POSTGRES_DB")
	if !ok {
		return nil, errors.New("database name for the database connection was not found on the .env file")
	}

	// Token variables
	jwtSecret, ok := fn("JWT_SECRET")
	if !ok {
		return nil, fmt.Errorf("JWT secret was not found on the env file")
	}
	jwtDurationStr, ok := fn("JWT_DURATION")
	if !ok {
		return nil, fmt.Errorf("JWT duration was not found on the env file")
	}
	jwtDurationInt, err := strconv.Atoi(jwtDurationStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse JWT duration into an integer: %s", jwtDurationStr)
	}
	refreshTokenDurationStr, ok := fn("REFRESH_TOKEN_DURATION")
	if !ok {
		return nil, fmt.Errorf("refresh token duration was not found on the env file")
	}
	refreshTokenDurationInt, err := strconv.Atoi(refreshTokenDurationStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse JWT duration into an integer: %s", refreshTokenDurationStr)
	}

	serverPort, ok := fn("SERVER_PORT")
	if !ok {
		return nil, fmt.Errorf("server port was not found on the env file")
	}

	return &envConfig{
		adminUsername:        adminUsername,
		adminPassword:        adminPassword,
		devFlag:              devFlag,
		dbUsername:           dbUsername,
		dbPassword:           dbPassword,
		dbHostPort:           dbHostPort,
		database:             database,
		jwtSecret:            jwtSecret,
		jwtDuration:          jwtDurationInt,
		refreshTokenDuration: refreshTokenDurationInt,
		serverPort:           serverPort,
	}, nil

}

func getAuthConfig(jwtSecret string, jwtDuration, refreshTokenDuration int) (*auth.Config, error) {
	return &auth.Config{
		JWTsecret:            jwtSecret,
		JWTDuration:          time.Duration(jwtDuration) * time.Second,
		RefreshTokenDuration: time.Duration(refreshTokenDuration) * time.Second,
	}, nil
}

func dbSetup(
	ctx context.Context,
	db *database.Queries,
	username, password string,
	logger *slog.Logger,
) error {
	// check if the admin exists
	user, err := db.GetUserByUsername(ctx, username)
	switch err {
	case nil:
		if user.IsAdmin.Bool {
			logger.Info("admin found on the database")
			return nil
		} else {
			return errors.New("found user instead of admin when setting up the admin")
		}
	case pgx.ErrNoRows:
	default:
		return fmt.Errorf("something went wrong while trying to fetch the admin from the database: %s", err)
	}

	// create the admin
	logger.Info("creating user admin...",
		slog.String("username", username),
		slog.String("password", "REDACTED"),
	)
	hashed, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("could not generate the hashed password for the admin user: %s", err)
	}
	if _, err := db.CreateAdmin(ctx, database.CreateAdminParams{
		Username:       username,
		HashedPassword: hashed,
	}); err != nil {
		return fmt.Errorf("something went wrong while creating the admin: %s", err)
	}

	logger.Info("admin successfully created")
	return nil
}

// Creates an admin in the database in case it does not exist.
func initialSetup(
	ctx context.Context,
	db *database.Queries,
	username, password string,
	logger *slog.Logger,
) error {
	return dbSetup(ctx, db, username, password, logger)
}

func getDB(
	ctx context.Context,
	dbUsername, dbPassword, dbHostPort, dbName, devFlag string,
) (*pgxpool.Pool, *database.Queries, error) {
	dbConnURL, err := getDBConnURL(dbUsername, dbPassword, dbHostPort, dbName, devFlag)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to obtain the database url: %w", err)
	}
	dbPool, err := startDB(ctx, dbConnURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to the database: %w", err)
	}
	return dbPool, database.New(dbPool), nil
}

func startDB(ctx context.Context, connPath string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connPath)
	if err != nil {
		return nil, fmt.Errorf("could not open the database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("could not ping the database: %w", err)
	}

	return pool, nil
}

func getDBConnURL(dbUsername, dbPassword, dbHostPort, dbName, devFlag string) (string, error) {
	url := fmt.Sprintf("postgresql://%s:%s@%s/%s", dbUsername, dbPassword, dbHostPort, dbName)
	if devFlag == "1" {
		url += "?sslmode=disable"
	}
	return url, nil
}
