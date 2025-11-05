package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type createSetReq struct {
	SessionID string `json:"session_id"`
	SetOrder  int32  `json:"set_order"`
	RestTime  int32  `json:"rest_time"`
}

type createSetRes struct {
	ID int `json:"id"`
	createSetReq
}

func (s *State) HandlerCreateSet(w http.ResponseWriter, r *http.Request) {
	// session id must be a valid uuid
	sessionID, err := uuid.Parse(r.PathValue("sessionID"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "session ID not found", err)
		return
	}
	// Decode the incoming json
	var requestParams createSetReq
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestParams); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload", err)
		return
	}

	// Validate request
	if err := requestParams.validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Check session id against database
	if _, err := s.db.GetSession(r.Context(), pgtype.UUID{Bytes: sessionID, Valid: true}); err != nil {
		respondWithError(w, http.StatusNotFound, "session ID not found", err)
		return
	}

	// Record the set into the database
	dbParams := database.CreateSetParams{
		SessionID: pgtype.UUID{Bytes: sessionID, Valid: true},
		SetOrder:  requestParams.SetOrder,
	}
	// negative rest time values will be considered to be null
	if requestParams.RestTime >= 0 {
		dbParams.RestTime = pgtype.Int4{Int32: int32(requestParams.RestTime), Valid: true}
	}

	set, err := s.db.CreateSet(r.Context(), dbParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create the set", err)
		return
	}

	respondWithJSON(w, http.StatusCreated,
		createSetRes{
			ID: int(set.ID),
			createSetReq: createSetReq{
				SessionID: set.SessionID.String(),
				SetOrder:  set.SetOrder,
				RestTime:  set.RestTime.Int32,
			},
		})
}

func (r *createSetReq) validate() error {
	// set order validation
	if r.SetOrder < 0 {
		return fmt.Errorf("set order must be greater than 1")
	}

	// rest time validation
	if r.RestTime > maxRestTimeSeconds {
		return fmt.Errorf("rest time in seconds must be less than %d seconds", maxRestTimeSeconds)
	}

	return nil
}
