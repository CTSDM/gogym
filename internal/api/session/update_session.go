package session

import (
	"log/slog"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func HandlerUpdateSession(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		sessionID, _ := retrieveParseUUIDFromContext(r.Context())
		reqLogger = reqLogger.With(slog.String("session_id", sessionID.String()))

		// decode and validate
		reqParams, problems, err := validation.DecodeValid[*sessionReq](r)
		if len(problems) > 0 {
			reqLogger.Debug("update session failed - validation errors", slog.Any("problems", problems))
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			reqLogger.Debug("update session failed - invalid payload", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Update the entry
		dbParams := database.UpdateSessionParams{
			ID:              sessionID,
			Name:            reqParams.Name,
			Date:            pgtype.Date{Time: reqParams.date, Valid: true},
			StartTimestamp:  pgtype.Timestamp{Time: reqParams.startTimestamp, Valid: true},
			DurationMinutes: pgtype.Int2{Int16: reqParams.durationMinutes, Valid: true},
		}
		updatedSession, err := db.UpdateSession(r.Context(), dbParams)
		if err == pgx.ErrNoRows {
			reqLogger.Error("update session failed - missing session", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusNotFound, "session not found", err)
			return
		} else if err != nil {
			reqLogger.Error("update session failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		reqLogger.Info("update session success")
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
