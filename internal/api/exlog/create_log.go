package exlog

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/CTSDM/gogym/internal/api/util"
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

func HandlerCreateLog(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// set id must be a valid number
		setID, err := strconv.Atoi(r.PathValue("setID"))
		if err != nil {
			util.RespondWithError(w, http.StatusNotFound, "set ID not found", err)
			return
		}
		// Decode the incoming json
		var requestParams createLogReq
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestParams); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Invalid payload", err)
			return
		}

		// Validate request
		if err := requestParams.validate(); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, err.Error(), nil)
			return
		}

		// Check set id against database
		if _, err := db.GetSet(r.Context(), int64(setID)); err != nil {
			util.RespondWithError(w, http.StatusNotFound, "set ID not found", err)
			return
		}
		// Check exercise id against database
		if _, err := db.GetExercise(r.Context(), requestParams.ExerciseID); err != nil {
			util.RespondWithError(w, http.StatusNotFound, "exercise id does not exist", err)
			return
		}

		// Record the log into the database
		dbParams := database.CreateLogParams{
			Weight:     pgtype.Float8{Float64: requestParams.Weight, Valid: true},
			Reps:       requestParams.Reps,
			LogsOrder:  requestParams.Order,
			SetID:      int64(setID),
			ExerciseID: requestParams.ExerciseID,
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
				ExerciseID: requestParams.ExerciseID,
				Weight:     requestParams.Weight,
				Reps:       requestParams.Reps,
				Order:      requestParams.Order,
			},
		})
	}
}

func (r *createLogReq) validate() error {
	// weight validation
	if r.Weight < 0 {
		r.Weight = 0
	}

	// order validation
	if r.Order < 0 {
		return errors.New("log order cannot be less than zero")
	}

	// reps validation
	if r.Reps <= 0 {
		return errors.New("reps cannot be less than zero")
	}

	return nil
}
