package set

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
)

func retrieveParseIDFromContext(ctx context.Context) (int64, error) {
	// pull the resource from the context
	resourceID, ok := middleware.ResourceIDFromContext(ctx)
	if !ok {
		return 0, errors.New("could not find the set id")
	}
	// coerce the resource id into int
	setID, ok := resourceID.(int64)
	if !ok {
		return 0, fmt.Errorf("could not type coerce the set id, %v, into int64", resourceID)
	}
	return setID, nil
}

func HandlerDeleteSet(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		// set id is stored in the context with a generic key
		setID, _ := retrieveParseIDFromContext(r.Context())
		reqLogger = reqLogger.With(slog.Int64("set_id", setID))

		// delete the set
		if _, err := db.DeleteSet(r.Context(), setID); err == pgx.ErrNoRows {
			reqLogger.Warn("delete set failed - set not found")
			util.RespondWithError(w, http.StatusNotFound, "set not found", nil)
			return
		} else if err != nil {
			reqLogger.Error("delete set failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		reqLogger.Info("delete set success")
		w.WriteHeader(http.StatusNoContent)
	}
}
