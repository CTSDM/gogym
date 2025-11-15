package set

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type SetReq struct {
	ExerciseID int32 `json:"exercise_id"`
	SetOrder   int32 `json:"set_order"`
	RestTime   int32 `json:"rest_time"`
}

type SetRes struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	SetReq
}

func (r *SetReq) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)

	// set order validation
	if r.SetOrder < 0 {
		problems["order"] = "invalid order: set order must be positive"
	}

	// rest time validation
	if r.RestTime > apiconstants.MaxRestTimeSeconds {
		msg := fmt.Sprintf("rest time in seconds must be less than %d seconds", apiconstants.MaxRestTimeSeconds)
		problems["rest_time"] = "invalid rest_time: " + msg
	} else if r.RestTime < 0 { // negative rest times are mapped to 0
		r.RestTime = 0
	}

	return problems
}

func HandlerCreateSet(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		// session id must be a valid uuid
		sessionID, err := uuid.Parse(r.PathValue("sessionID"))
		if err != nil {
			reqLogger.Debug("create set failed - invalid session id",
				slog.String("error", err.Error()),
				slog.String("session_id", r.PathValue("sessionID")),
			)
			util.RespondWithError(w, r, http.StatusBadRequest, "invalid session ID format", err)
			return
		}

		reqLogger = reqLogger.With(slog.String("session_id", sessionID.String()))
		reqParams, problems, err := validation.DecodeValid[*SetReq](r)
		if len(problems) > 0 {
			reqLogger.Debug("create set failed - validation failed", slog.Any("problems", problems))
			util.RespondWithJSON(w, r, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			reqLogger.Debug("create set failed - invalid payload", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Record the set into the database
		dbParams := database.CreateSetParams{
			SessionID:  sessionID,
			SetOrder:   reqParams.SetOrder,
			ExerciseID: reqParams.ExerciseID,
			RestTime:   pgtype.Int4{Int32: reqParams.RestTime, Valid: true},
		}

		set, err := db.CreateSet(r.Context(), dbParams)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				if strings.Contains(err.Error(), "session") {
					reqLogger.Warn("create set failed - session not found")
					util.RespondWithError(w, r, http.StatusNotFound, "session ID not found", err)
				} else if strings.Contains(err.Error(), "exercise") {
					reqLogger.Warn(
						"create set failed - exercise not found",
						slog.Int64("exercise_id", int64(reqParams.ExerciseID)),
					)
					util.RespondWithError(w, r, http.StatusNotFound, "exercise ID not found", err)
				} else {
					reqLogger.Error(
						"create set failed - unknown foreign key violation",
						slog.String("error", err.Error()),
					)
					util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
				}
				return
			}
			reqLogger.Error("create set failed - create set database error", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		reqLogger.Info("create set success", slog.Int64("set_id", set.ID))
		util.RespondWithJSON(w, r, http.StatusCreated,
			SetRes{
				ID:        set.ID,
				SessionID: set.SessionID.String(),
				SetReq: SetReq{
					SetOrder: set.SetOrder,
					RestTime: set.RestTime.Int32,
				},
			})
	}
}
