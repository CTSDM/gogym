package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func HandlerDeleteSession(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		userID, ok := util.UserFromContext(r.Context())
		if !ok {
			err := errors.New("could not find user id in the context")
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		// session id is stored in the context with a generic key
		sessionID, _ := retrieveParseUUIDFromContext(r.Context())
		reqLogger.With(slog.String("user_id", userID.String()), slog.String("session_id", sessionID.String()))

		// delete the resource
		if _, err := db.DeleteSession(r.Context(), database.DeleteSessionParams{
			UserID: userID,
			ID:     sessionID,
		}); err == pgx.ErrNoRows {
			reqLogger.Error("delete session failed - session not found", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusNotFound, "not found", nil)
			return
		} else if err != nil {
			reqLogger.Error("delete session failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		reqLogger.Info("delete session success")
		w.WriteHeader(http.StatusNoContent)
	}
}

func retrieveParseUUIDFromContext(ctx context.Context) (uuid.UUID, error) {
	// Pull the resource from the context.
	// If the resource is not found or cannot be parsed is an error, as it should have not happened.
	resourceID, ok := util.ResourceIDFromContext(ctx)
	if !ok {
		return uuid.UUID{}, errors.New("could not find resource id from the context")
	}
	// coerce the resource into uuid
	sessionID, ok := resourceID.(uuid.UUID)
	if !ok {
		err := fmt.Errorf("could not coerce the resource id, %v, into an uuid", resourceID)
		return uuid.UUID{}, err
	}
	return sessionID, nil
}
