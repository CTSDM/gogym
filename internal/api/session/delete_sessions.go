package session

import (
	"context"
	"errors"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func HandlerDeleteSession(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserFromContext(r.Context())
		if !ok {
			err := errors.New("could not find user id in the context")
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		// session id is stored in the context with a generic key
		sessionID, err := retrieveParseUUIDFromContext(r.Context())
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// delete the resource
		if _, err := db.DeleteSession(r.Context(), database.DeleteSessionParams{
			UserID: pgtype.UUID{Bytes: userID, Valid: true},
			ID:     pgtype.UUID{Bytes: sessionID, Valid: true},
		}); err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "not found", nil)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func retrieveParseUUIDFromContext(ctx context.Context) (uuid.UUID, error) {
	// pull the resource from the context
	resourceID, ok := middleware.ResourceIDFromContext(ctx)
	if !ok {
		return uuid.UUID{}, errors.New("could not find user id in the context")
	}
	// coerce the resource into uuid
	sessionID, ok := resourceID.(pgtype.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("could not type coerce session id into uuid")
	}
	return sessionID.Bytes, nil
}
