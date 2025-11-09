package exlog

import (
	"errors"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func HandlerUpdateLog(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resourceID, ok := middleware.ResourceIDFromContext(r.Context())
		if !ok {
			err := errors.New("expected log id to be in the context")
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		logID, ok := resourceID.(int64)
		if !ok {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", nil)
			return
		}
		// decode and validate
		reqParams, problems, err := validation.DecodeValid[*LogReq](r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
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
			util.RespondWithError(w, http.StatusNotFound, "log not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		util.RespondWithJSON(w, http.StatusOK, LogRes{
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
