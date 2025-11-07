package exlog

import (
	"log"
	"net/http"
	"strconv"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func HandlerUpdateLog(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// log id must be a valid number
		logID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			util.RespondWithError(w, http.StatusNotFound, "set ID not found", err)
			return
		}

		// decode and validate
		reqParams, problems, err := validation.DecodeValid[*logReq](r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Get user from the context
		userID, ok := middleware.UserFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("user not found in context after auth middleware")
			return
		}

		// Check the user ownership before updating
		ownerID, err := db.GetLogOwnerID(r.Context(), logID)
		if err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "log not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError,
				"could not fetch the log", err)
			return
		}
		if ownerID.Bytes != userID {
			util.RespondWithError(w, http.StatusForbidden, "user is not the owner of the log", nil)
			return
		}

		// Update the entry
		dbParams := database.UpdateLogParams{
			Weight:    pgtype.Float8{Float64: reqParams.Weight, Valid: true},
			Reps:      reqParams.Reps,
			LogsOrder: reqParams.Order,
			ID:        int64(logID),
		}
		updatedLog, err := db.UpdateLog(r.Context(), dbParams)
		if err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "log not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError,
				"something went wrong while updating the log", err)
			return
		}

		util.RespondWithJSON(w, http.StatusOK, logRes{
			ID:    updatedLog.ID,
			SetID: updatedLog.SetID,
			logReq: logReq{
				ExerciseID: updatedLog.ExerciseID,
				Weight:     updatedLog.Weight.Float64,
				Reps:       updatedLog.Reps,
				Order:      updatedLog.LogsOrder,
			},
		})
	}
}
