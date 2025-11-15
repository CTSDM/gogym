package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

func handlerHealth(pool *pgxpool.Pool, logger *slog.Logger) http.HandlerFunc {
	type res struct {
		Status    string `json:"status"`
		Database  string `json:"database"`
		Timestamp string `json:"timestamp"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		if err := pool.Ping(r.Context()); err != nil {
			reqLogger.Error("health check failed - database ping failed", slog.String("error", err.Error()))
			util.RespondWithJSON(w, http.StatusServiceUnavailable, res{
				Status:    "database unreachable",
				Database:  "unavailable",
				Timestamp: time.Now().UTC().Format("2006-01-02 15:04:05"),
			})
			return
		}

		util.RespondWithJSON(w, http.StatusOK, res{
			Status:    "ok",
			Database:  "connected",
			Timestamp: time.Now().UTC().Format("2006-01-02 15:04:05"),
		})
	}
}
