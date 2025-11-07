package exercise

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type createExerciseReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createExerciseRes struct {
	ID int32 `json:"id"`
	createExerciseReq
}

type exerciseItem struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type exercisesRes struct {
	Exercises []exerciseItem `json:"exercises"`
}

func (r createExerciseReq) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)
	// name validation
	if err := validation.String(r.Name, 0, apiconstants.MaxExerciseLength); err != nil {
		problems["name"] = fmt.Sprintf("invalid name: %s", err.Error())
	}

	// description validation; it is an optinal field so there is no minimum length
	if err := validation.String(r.Description, -1, apiconstants.MaxDescriptionLength); err != nil {
		problems["description"] = fmt.Sprintf("invalid description: %s", err.Error())
	}

	return problems
}

func HandlerCreateExercise(db *database.Queries) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		// Decode json into the expected structure
		reqParams, problems, err := validation.DecodeValid[createExerciseReq](r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Create the entry in the database
		exercise, err := db.CreateExercise(r.Context(), database.CreateExerciseParams{
			Name:        reqParams.Name,
			Description: pgtype.Text{String: reqParams.Description, Valid: true},
		})
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong while creating the exercise", err)
			return
		}

		util.RespondWithJSON(w, http.StatusCreated, createExerciseRes{
			ID: exercise.ID,
			createExerciseReq: createExerciseReq{
				Name:        exercise.Name,
				Description: exercise.Description.String,
			},
		})
	}
}

func HandlerGetExercises(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exercisesDB, err := db.GetExercises(r.Context())
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong while retrieving the exercises", err)
			return
		}
		resParams := exercisesRes{Exercises: make([]exerciseItem, len(exercisesDB))}
		for i, e := range exercisesDB {
			resParams.Exercises[i].Description = e.Description.String
			resParams.Exercises[i].Name = e.Name
			resParams.Exercises[i].ID = e.ID
		}
		util.RespondWithJSON(w, http.StatusOK, resParams)
	}
}

func HandlerGetExercise(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exerciseIDString := r.PathValue("id")
		exerciseID, err := strconv.ParseInt(exerciseIDString, 10, 32)
		if err != nil {
			util.RespondWithError(w, http.StatusNotFound, "exercise id not found", err)
			return
		}
		exerciseDB, err := db.GetExercise(r.Context(), int32(exerciseID))
		if err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "exercise id not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong while retrieving the exercise", err)
			return
		}

		util.RespondWithJSON(w, http.StatusOK, exerciseItem{
			ID:          exerciseDB.ID,
			Name:        exerciseDB.Name,
			Description: exerciseDB.Description.String,
		})
	}
}
