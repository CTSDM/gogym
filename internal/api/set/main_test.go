package set

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	var cleanup func()
	var err error
	dbPool, cleanup, err = testutil.SetupTestDB(context.Background())
	if err != nil {
		log.Fatalf("could not set up test containers: %s", err.Error())
	}

	defer cleanup()
	os.Exit(m.Run())
}
