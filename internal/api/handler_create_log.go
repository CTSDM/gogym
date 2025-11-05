package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type createLogReq struct {
	SetID      int64   `json:"set_id"`
	ExerciseID int32   `json:"exercise_id"`
	Weight     float64 `json:"weight"`
	Reps       int32   `json:"reps"`
	Order      int32   `json:"order"`
}

type createLogRes struct {
	ID int64 `json:"id"`
	createLogReq
}

func (s *State) HandlerCreateLog(w http.ResponseWriter, r *http.Request) {
	// Decode the incoming json
	var requestParams createLogReq
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestParams); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload", err)
		return
	}

	// Validat request
	err := requestParams.validate()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Check set id against database
	if _, err := s.db.GetSet(r.Context(), int64(requestParams.SetID)); err != nil {
		respondWithError(w, http.StatusNotFound, "set id does not exist", err)
		return
	}
	// Check exercise id against database
	if _, err := s.db.GetExercise(r.Context(), requestParams.ExerciseID); err != nil {
		respondWithError(w, http.StatusNotFound, "exercise id does not exist", err)
		return
	}

	// Record the log into the database
	dbParams := database.CreateLogParams{
		Weight:     pgtype.Float8{Float64: requestParams.Weight, Valid: true},
		Reps:       requestParams.Reps,
		LogsOrder:  requestParams.Order,
		SetID:      requestParams.SetID,
		ExerciseID: requestParams.ExerciseID,
	}
	newLog, err := s.db.CreateLog(r.Context(), dbParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create the log exercise", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, createLogRes{
		ID: newLog.ID,
		createLogReq: createLogReq{
			SetID:      requestParams.SetID,
			ExerciseID: requestParams.ExerciseID,
			Weight:     requestParams.Weight,
			Reps:       requestParams.Reps,
			Order:      requestParams.Order,
		},
	})
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
