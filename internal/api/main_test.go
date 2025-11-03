package api

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

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
		log.Fatalf("could not start the container %s", err.Error())
	}
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()

	connURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	dbPool, err = pgxpool.New(context.Background(), connURL)
	if err != nil {
		log.Fatalf("Could not ping the db...: %s", err)
	}

	// obtain *sql.DB from *pgxpool.Pool
	db := stdlib.OpenDBFromPool(dbPool)
	if err := goose.UpContext(ctx, db, "../../sql/schema"); err != nil {
		log.Fatalf("Something went wrong while migrating the database...: %s", err)
	}

	res := m.Run()
	os.Exit(res)
}

func cleanup(tableTarget string) error {
	tables := map[string]struct{}{
		"users":          {},
		"exercises":      {},
		"logs":           {},
		"sessions":       {},
		"sets":           {},
		"refresh_tokens": {},
	}
	if tableTarget == "" {
		for _, table := range tables {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, err := dbPool.Exec(ctx, fmt.Sprintf("DELETE FROM %s;", table)); err != nil {
				return fmt.Errorf("could not delete table %s: %s", table, err.Error())
			}
		}
		return nil
	}
	if _, ok := tables[tableTarget]; !ok {
		return fmt.Errorf("could not find table %s", tableTarget)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := dbPool.Exec(ctx, fmt.Sprintf("DELETE FROM %s;", tableTarget)); err != nil {
		return fmt.Errorf("could not delete table %s: %s", tableTarget, err.Error())
	}

	return nil
}
