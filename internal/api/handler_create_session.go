package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type createSessionReq struct {
	Name            string `json:"name"`
	Date            string `json:"date"`
	StartTimestamp  int64  `json:"start_timestamp"` // UTC
	DurationMinutes int    `json:"duration_minutes"`
}

type createSessionRes struct {
	ID string `json:"id"`
	createSessionReq
}

func (s *State) HandlerCreateSession(w http.ResponseWriter, r *http.Request) {
	// Get userID from the context
	userID, ok := UserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusInternalServerError, "Could not find user id in request context", nil)
		return
	}

	// Check userID against database
	if _, err := s.db.GetUser(r.Context(), pgtype.UUID{Bytes: userID, Valid: true}); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials",
			fmt.Errorf("could not find the userID provied by the JWT in the database: %w", err))
		return
	}

	// Decode the incoming json
	var requestParams createSessionReq
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestParams); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload", err)
		return
	}

	// Populate and validate the incoming request
	requestParams.populate()
	date, startTimestamp, err := requestParams.validate()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Fill the database object and create the session
	dbParams := database.CreateSessionParams{
		Name:   requestParams.Name,
		Date:   pgtype.Date{Time: date, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	}
	if requestParams.StartTimestamp > 0 {
		dbParams.StartTimestamp.Time = startTimestamp
		dbParams.StartTimestamp.Valid = true
	}
	if requestParams.DurationMinutes > 0 {
		if requestParams.DurationMinutes <= math.MaxInt16 {
			dbParams.DurationMinutes.Int16 = int16(requestParams.DurationMinutes)
			dbParams.DurationMinutes.Valid = true
		}
	}

	session, err := s.db.CreateSession(r.Context(), dbParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create the session", err)
		return
	}

	respondWithJSON(w, http.StatusCreated,
		createSessionRes{
			ID: session.ID.String(),
			createSessionReq: createSessionReq{
				Name:            session.Name,
				Date:            session.Date.Time.Format(DATE_LAYOUT),
				StartTimestamp:  session.StartTimestamp.Time.Unix(),
				DurationMinutes: int(session.DurationMinutes.Int16),
			},
		})
}

// Validate needed fields: name and date
func (r *createSessionReq) validate() (date time.Time, timeStart time.Time, err error) {
	// session name validation
	if r.Name == "" {
		return time.Time{}, time.Time{}, errors.New("name cannot be empty")
	}
	if err := validateString(r.Name, minSessionNameLength, maxSessionNameLength); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("could not validate the name: %w", err)
	}

	// date validation
	if r.Date == "" {
		return time.Time{}, time.Time{}, errors.New("date cannot be empty")
	}
	date, err = validateDate(r.Date, DATE_LAYOUT, nil, nil)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("could not validate the date: %w", err)
	}

	// start timestamp validation
	if r.StartTimestamp < 0 {
		return time.Time{}, time.Time{}, errors.New("start timestamp must be greater than UNIX epoch")
	}
	timeStart = time.Unix(r.StartTimestamp, 0).UTC()

	// duration minutes validation
	if r.DurationMinutes < 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("workout duration must be between 1 and %d minutes", math.MaxInt16)
	} else {
		if r.DurationMinutes > math.MaxInt16 {
			return time.Time{}, time.Time{}, fmt.Errorf("workout duration must be between 1 and %d minutes", math.MaxInt16)
		}
	}
	return date, timeStart, nil
}

// Populate needed empty fields: name and date
func (r *createSessionReq) populate() {
	if r.Name == "" {
		now := time.Now()
		r.Name = now.Format(DATE_TIME_LAYOUT)
	}

	if r.Date == "" {
		now := time.Now()
		r.Date = now.Format(DATE_LAYOUT)
	}
}
