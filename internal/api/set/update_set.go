package set

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func HandlerUpdateSet(pool *pgxpool.Pool, db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		setID, _ := retrieveParseIDFromContext(r.Context())
		reqLogger = reqLogger.With(slog.Int64("set_id", setID))

		// decode and validate
		reqParams, problems, err := validation.DecodeValid[*SetReq](r)
		if len(problems) > 0 {
			reqLogger.Debug("update set failed - validation failed", slog.Any("problems", problems))
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			reqLogger.Debug("update set failed - invalid payload", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Begin transaction
		tx, err := pool.Begin(r.Context())
		if err != nil {
			reqLogger.Error("update set failed - transaction start error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// Defer a rollback if anything goes wrong
		txQueries := db.WithTx(tx)
		defer tx.Rollback(r.Context())

		// Get the current set to check if there is a change in exercise id
		// If exercise id changes, logs belonging to the session need to update their information
		setDB, err := txQueries.GetSet(r.Context(), setID)
		if err == pgx.ErrNoRows {
			reqLogger.Error(
				"update set failed - set not found after ownership check",
				slog.String("error", err.Error()),
			)
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		} else if err != nil {
			reqLogger.Error("update set failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		if reqParams.ExerciseID != setDB.ExerciseID {
			// update the logs information
			if err := txQueries.UpdateLogsExerciseIDBySetID(r.Context(), database.UpdateLogsExerciseIDBySetIDParams{
				ExerciseID: reqParams.ExerciseID,
				SetID:      setID,
			}); err != nil {
				reqLogger.Error(
					"update set failed - update logs database error",
					slog.String("error", err.Error()),
				)
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
			reqLogger.Error("update set failed - update set database error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			reqLogger.Error("update set failed - transaction commit error", slog.String("error", err.Error()))
			err = fmt.Errorf("could not commit the transaction: %w", err)
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		reqLogger.Info("update set success")
		w.WriteHeader(http.StatusNoContent)
	}
}
