package set

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type createSetReq struct {
	ExerciseID int32 `json:"exercise_id"`
	SetOrder   int32 `json:"set_order"`
	RestTime   int32 `json:"rest_time"`
}

type createSetRes struct {
	ID        int    `json:"id"`
	SessionID string `json:"session_id"`
	createSetReq
}

func HandlerCreateSet(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// session id must be a valid uuid
		sessionID, err := uuid.Parse(r.PathValue("sessionID"))
		if err != nil {
			util.RespondWithError(w, http.StatusNotFound, "session ID not found", err)
			return
		}
		// Decode the incoming json
		var requestParams createSetReq
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

		// Check session id against database
		if _, err := db.GetSession(r.Context(), pgtype.UUID{Bytes: sessionID, Valid: true}); err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "session ID not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong while fetching the session", err)
			return
		}

		// Record the set into the database
		dbParams := database.CreateSetParams{
			SessionID:  pgtype.UUID{Bytes: sessionID, Valid: true},
			SetOrder:   requestParams.SetOrder,
			ExerciseID: requestParams.ExerciseID,
		}
		// negative rest time values will be considered to be null
		if requestParams.RestTime >= 0 {
			dbParams.RestTime = pgtype.Int4{Int32: int32(requestParams.RestTime), Valid: true}
		}

		set, err := db.CreateSet(r.Context(), dbParams)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				util.RespondWithError(w, http.StatusNotFound, "exercise id not found", err)
				return
			}
			util.RespondWithError(w, http.StatusInternalServerError, "could not create the set", err)
			return
		}

		util.RespondWithJSON(w, http.StatusCreated,
			createSetRes{
				ID:        int(set.ID),
				SessionID: set.SessionID.String(),
				createSetReq: createSetReq{
					SetOrder: set.SetOrder,
					RestTime: set.RestTime.Int32,
				},
			})
	}
}

func (r *createSetReq) validate() error {
	// set order validation
	if r.SetOrder < 0 {
		return fmt.Errorf("set order must be greater than 1")
	}

	// rest time validation
	if r.RestTime > apiconstants.MaxRestTimeSeconds {
		return fmt.Errorf("rest time in seconds must be less than %d seconds", apiconstants.MaxRestTimeSeconds)
	}

	return nil
}
