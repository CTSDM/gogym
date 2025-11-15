package exlog

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type LogReq struct {
	ExerciseID int32   `json:"exercise_id"`
	Weight     float64 `json:"weight"`
	Reps       int32   `json:"reps"`
	Order      int32   `json:"order"`
}

type LogRes struct {
	ID    int64 `json:"id"`
	SetID int64 `json:"set_id"`
	LogReq
}

func (r *LogReq) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)
	// weight validation
	// negative values are mapped to 0
	if r.Weight < 0 {
		r.Weight = 0
	}

	// order validation
	if r.Order < 0 {
		problems["order"] = "invalid order: log order must be positive"
	}

	// reps validation
	if r.Reps <= 0 {
		problems["reps"] = "invalid reps: reps must be positive"
	}

	return problems
}

func HandlerCreateLog(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		// set id must be a valid number
		setID, err := strconv.ParseInt(r.PathValue("setID"), 10, 64)
		if err != nil {
			util.RespondWithError(w, r, http.StatusNotFound, "set ID not found", err)
			return
		}
		reqLogger = reqLogger.With(slog.Int64("set_id", setID))

		reqParams, problems, err := validation.DecodeValid[*LogReq](r)
		if len(problems) > 0 {
			reqLogger.Debug("create log failed - validation failed", slog.Any("problems", problems))
			util.RespondWithJSON(w, r, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			reqLogger.Debug("create log failed - invalid payload", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Record the log into the database
		dbParams := database.CreateLogParams{
			Weight:     pgtype.Float8{Float64: reqParams.Weight, Valid: true},
			Reps:       reqParams.Reps,
			LogsOrder:  reqParams.Order,
			SetID:      setID,
			ExerciseID: reqParams.ExerciseID,
		}
		newLog, err := db.CreateLog(r.Context(), dbParams)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				if strings.Contains(err.Error(), "set") {
					reqLogger.Warn("create log failed - set not found")
					util.RespondWithError(w, r, http.StatusNotFound, "set ID not found", err)
				} else if strings.Contains(err.Error(), "exercise") {
					reqLogger.Warn("create log failed - exercise not found",
						slog.Int64("exercise_id", int64(reqParams.ExerciseID)))
					util.RespondWithError(w, r, http.StatusNotFound, "exercise ID not found", err)
				} else {
					reqLogger.Error("create log failed - unknown FK violation",
						slog.String("error", err.Error()))
					util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
				}
				return
			}
			reqLogger.Error("create log failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		reqLogger.Info("create log success", slog.Int64("log_id", newLog.ID))
		util.RespondWithJSON(w, r, http.StatusCreated, LogRes{
			ID:    newLog.ID,
			SetID: setID,
			LogReq: LogReq{
				ExerciseID: newLog.ExerciseID,
				Weight:     newLog.Weight.Float64,
				Reps:       newLog.Reps,
				Order:      newLog.LogsOrder,
			},
		})
	}
}
