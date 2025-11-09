package set

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type SetReq struct {
	ExerciseID int32 `json:"exercise_id"`
	SetOrder   int32 `json:"set_order"`
	RestTime   int32 `json:"rest_time"`
}

type SetRes struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	SetReq
}

func (r *SetReq) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)

	// set order validation
	if r.SetOrder < 0 {
		problems["order"] = "invalid order: set order must be positive"
	}

	// rest time validation
	if r.RestTime > apiconstants.MaxRestTimeSeconds {
		msg := fmt.Sprintf("rest time in seconds must be less than %d seconds", apiconstants.MaxRestTimeSeconds)
		problems["rest_time"] = "invalid rest_time: " + msg
	} else if r.RestTime < 0 { // negative rest times are mapped to 0
		r.RestTime = 0
	}

	return problems
}

func HandlerCreateSet(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// session id must be a valid uuid
		sessionID, err := uuid.Parse(r.PathValue("sessionID"))
		if err != nil {
			util.RespondWithError(w, http.StatusNotFound, "session ID not found", err)
			return
		}

		reqParams, problems, err := validation.DecodeValid[*SetReq](r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Check session id against database
		if _, err := db.GetSession(r.Context(), sessionID); err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "session ID not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong while fetching the session", err)
			return
		}

		// Record the set into the database
		dbParams := database.CreateSetParams{
			SessionID:  sessionID,
			SetOrder:   reqParams.SetOrder,
			ExerciseID: reqParams.ExerciseID,
			RestTime:   pgtype.Int4{Int32: reqParams.RestTime, Valid: true},
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
			SetRes{
				ID:        set.ID,
				SessionID: set.SessionID.String(),
				SetReq: SetReq{
					SetOrder: set.SetOrder,
					RestTime: set.RestTime.Int32,
				},
			})
	}
}
