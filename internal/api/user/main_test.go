package user

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool
var logger *slog.Logger

func TestMain(m *testing.M) {
	var cleanup func()
	var err error
	dbPool, cleanup, err = testutil.SetupTestDB(context.Background())
	if err != nil {
		log.Fatalf("could not set up test containers: %s", err.Error())
	}

	b := bytes.NewBuffer([]byte{})
	logger = slog.New(slog.NewTextHandler(b, nil))

	defer cleanup()
	os.Exit(m.Run())
}
