package session

import (
	"errors"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func HandlerUpdateSession(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resourceID, ok := middleware.ResourceIDFromContext(r.Context())
		if !ok {
			err := errors.New("expected session id to be in the context")
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		sessionResource, ok := resourceID.(pgtype.UUID)
		if !ok {
			err := errors.New("could not type coerce session id into uuid")
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		// decode and validate
		reqParams, problems, err := validation.DecodeValid[*sessionReq](r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Update the entry
		dbParams := database.UpdateSessionParams{
			ID:             pgtype.UUID{Bytes: sessionResource.Bytes, Valid: true},
			Name:           reqParams.Name,
			Date:           pgtype.Date{Time: reqParams.date, Valid: true},
			StartTimestamp: pgtype.Timestamp{Time: reqParams.startTimestamp, Valid: true},
		}
		updatedSession, err := db.UpdateSession(r.Context(), dbParams)
		if err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "session not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		util.RespondWithJSON(w, http.StatusOK, sessionRes{
			ID: updatedSession.ID.String(),
			sessionReq: sessionReq{
				Name:            updatedSession.Name,
				Date:            updatedSession.Date.Time.Format(apiconstants.DATE_LAYOUT),
				DurationMinutes: int(updatedSession.DurationMinutes.Int16),
				StartTimestamp:  updatedSession.StartTimestamp.Time.Unix(),
			},
		})
	}
}
