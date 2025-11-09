package set

import (
	"context"
	"errors"
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
		return 0, errors.New("could not type coerce the set id in into int64")
	}
	return setID, nil
}

func HandlerDeleteSet(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// set id is stored in the context with a generic key
		setID, err := retrieveParseIDFromContext(r.Context())
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// delete the set
		if _, err := db.DeleteSet(r.Context(), setID); err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "not found", nil)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
