package exlog

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
)

func HandlerDeleteLog(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		// log id is stored in the context with a generic key
		logID, _ := retrieveParseIDFromContext(r.Context())
		reqLogger = reqLogger.With(slog.Int64("log_id", logID))

		// delete the log
		if _, err := db.DeleteLog(r.Context(), logID); err == pgx.ErrNoRows {
			reqLogger.Warn("delete log failed - log id not in database", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", nil)
			return
		} else if err != nil {
			reqLogger.Error("delete log failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		reqLogger.Info("delete log success")
		w.WriteHeader(http.StatusNoContent)
	}
}

func retrieveParseIDFromContext(ctx context.Context) (int64, error) {
	// pull the resource from the context
	resourceID, ok := middleware.ResourceIDFromContext(ctx)
	if !ok {
		return 0, errors.New("could not find the log id")
	}
	// coerce the resource id into int
	logID, ok := resourceID.(int64)
	if !ok {
		return 0, errors.New("could not type coerce the log id into int64")
	}
	return logID, nil
}
