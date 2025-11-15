package exlog

import (
	"log/slog"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func HandlerUpdateLog(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		logID, _ := retrieveParseIDFromContext(r.Context())
		reqLogger = reqLogger.With(slog.Int64("log_id", logID))
		// decode and validate
		reqParams, problems, err := validation.DecodeValid[*LogReq](r)
		if len(problems) > 0 {
			reqLogger.Debug("update log failed - validation failed", slog.Any("problems", problems))
			util.RespondWithJSON(w, r, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			reqLogger.Debug("update log failed - invalid payload", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Update the entry
		dbParams := database.UpdateLogParams{
			Weight:    pgtype.Float8{Float64: reqParams.Weight, Valid: true},
			Reps:      reqParams.Reps,
			LogsOrder: reqParams.Order,
			ID:        logID,
		}
		updatedLog, err := db.UpdateLog(r.Context(), dbParams)
		if err == pgx.ErrNoRows {
			reqLogger.Error("update log failed - log not found", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		} else if err != nil {
			reqLogger.Error(
				"update log failed - database error",
				slog.String("error", err.Error()),
				slog.Any("log_parameters", dbParams),
			)
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		reqLogger.Info("update log success")
		util.RespondWithJSON(w, r, http.StatusOK, LogRes{
			ID:    updatedLog.ID,
			SetID: updatedLog.SetID,
			LogReq: LogReq{
				ExerciseID: updatedLog.ExerciseID,
				Weight:     updatedLog.Weight.Float64,
				Reps:       updatedLog.Reps,
				Order:      updatedLog.LogsOrder,
			},
		})
	}
}
