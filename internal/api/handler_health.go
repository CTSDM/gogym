package api

import (
	"log"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

func handlerHealth(pool *pgxpool.Pool) http.HandlerFunc {
	type res struct {
		Status    string `json:"status"`
		Database  string `json:"database"`
		Timestamp string `json:"timestamp"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			log.Printf("could not reach the database: %s", err.Error())
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
