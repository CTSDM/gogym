package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestHandlerHealth(t *testing.T) {
	t.Run("database is connected and working", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", bytes.NewReader([]byte{}))
		handler := handlerHealth(dbPool, logger)
		middleware.RequestID(handler).ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code, fmt.Sprintf("want %d, got %d", http.StatusOK, rr.Code))

		// unmarshal the response
		raw, err := io.ReadAll(rr.Body)
		if err != nil {
			log.Fatal(err)
		}
		require.Contains(t, string(raw), `"status":"ok"`)
		require.Contains(t, string(raw), `"database":"connected"`)
	})

	t.Run("database is not connected", func(t *testing.T) {
		badPool, _ := pgxpool.New(context.Background(), "postgres://invalid:5432/fake")
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", bytes.NewReader([]byte{}))
		handler := handlerHealth(badPool, logger)
		middleware.RequestID(handler).ServeHTTP(rr, req)

		require.Equal(t, http.StatusServiceUnavailable, rr.Code,
			fmt.Sprintf("want %d, got %d", http.StatusServiceUnavailable, rr.Code))

		// unmarshal the response
		raw, err := io.ReadAll(rr.Body)
		if err != nil {
			log.Fatal(err)
		}
		require.Contains(t, string(raw), `"status":"database unreachable"`)
		require.Contains(t, string(raw), `"database":"unavailable"`)
	})
}
