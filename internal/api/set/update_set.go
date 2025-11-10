package set

import (
	"fmt"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func HandlerUpdateSet(pool *pgxpool.Pool, db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setID, err := retrieveParseIDFromContext(r.Context())
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// decode and validate
		reqParams, problems, err := validation.DecodeValid[*SetReq](r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Begin transaction
		// If exercise id changes, logs belonging to the session need to update their information
		tx, err := pool.Begin(r.Context())
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// Defer a rollback if anything goes wrong
		txQueries := db.WithTx(tx)
		defer tx.Rollback(r.Context())

		// Get the current session to check if there is a change in exercise id
		setDB, err := txQueries.GetSet(r.Context(), setID)
		if err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		if reqParams.ExerciseID != setDB.ExerciseID {
			// update the logs information
			if err := txQueries.UpdateLogsExerciseIDBySetID(r.Context(), database.UpdateLogsExerciseIDBySetIDParams{
				ExerciseID: reqParams.ExerciseID,
				SetID:      setID,
			}); err != nil {
				util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
				return
			}
		}
		// Update the set information
		if _, err := txQueries.UpdateSet(r.Context(), database.UpdateSetParams{
			SetOrder:   reqParams.SetOrder,
			RestTime:   pgtype.Int4{Int32: reqParams.RestTime, Valid: true},
			ExerciseID: reqParams.ExerciseID,
			ID:         setID,
		}); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			err = fmt.Errorf("could not commit the transaction: %w", err)
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
