package exlog

import (
	"context"
	"errors"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
)

func HandlerDeleteLog(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// log id is stored in the context with a generic key
		logID, err := retrieveParseIDFromContext(r.Context())
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// delete the log
		if _, err := db.DeleteLog(r.Context(), logID); err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "not found", nil)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
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
		return 0, errors.New("could not type coerce the log id in into int64")
	}
	return logID, nil
}
