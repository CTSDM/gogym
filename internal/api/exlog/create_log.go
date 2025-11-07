package exlog

import (
	"context"
	"net/http"
	"strconv"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type createLogReq struct {
	ExerciseID int32   `json:"exercise_id"`
	Weight     float64 `json:"weight"`
	Reps       int32   `json:"reps"`
	Order      int32   `json:"order"`
}

type createLogRes struct {
	ID    int64 `json:"id"`
	SetID int64 `json:"set_id"`
	createLogReq
}

func (r *createLogReq) Valid(ctx context.Context) map[string]string {
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

func HandlerCreateLog(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// set id must be a valid number
		setID, err := strconv.Atoi(r.PathValue("setID"))
		if err != nil {
			util.RespondWithError(w, http.StatusNotFound, "set ID not found", err)
			return
		}

		reqParams, problems, err := validation.DecodeValid[*createLogReq](r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Check set id against database
		if _, err := db.GetSet(r.Context(), int64(setID)); err != nil {
			util.RespondWithError(w, http.StatusNotFound, "set ID not found", err)
			return
		}
		// Check exercise id against database
		if _, err := db.GetExercise(r.Context(), reqParams.ExerciseID); err != nil {
			util.RespondWithError(w, http.StatusNotFound, "exercise id not found", err)
			return
		}

		// Record the log into the database
		dbParams := database.CreateLogParams{
			Weight:     pgtype.Float8{Float64: reqParams.Weight, Valid: true},
			Reps:       reqParams.Reps,
			LogsOrder:  reqParams.Order,
			SetID:      int64(setID),
			ExerciseID: reqParams.ExerciseID,
		}
		newLog, err := db.CreateLog(r.Context(), dbParams)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "could not create the log exercise", err)
			return
		}

		util.RespondWithJSON(w, http.StatusCreated, createLogRes{
			ID:    newLog.ID,
			SetID: int64(setID),
			createLogReq: createLogReq{
				ExerciseID: newLog.ExerciseID,
				Weight:     newLog.Weight.Float64,
				Reps:       newLog.Reps,
				Order:      newLog.LogsOrder,
			},
		})
	}
}
