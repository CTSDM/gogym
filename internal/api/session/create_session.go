package session

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type sessionReq struct {
	Name            string `json:"name"`
	Date            string `json:"date"`
	StartTimestamp  int64  `json:"start_timestamp"` // UTC
	DurationMinutes int    `json:"duration_minutes"`

	date            time.Time
	startTimestamp  time.Time
	durationMinutes int16
}

type sessionRes struct {
	ID string `json:"id"`
	sessionReq
}

// This method also populates with default values
func (r *sessionReq) Valid(ctx context.Context) map[string]string {
	r.populate()
	problems := make(map[string]string)

	// session name validation
	if r.Name == "" {
		problems["name"] = "invalid name: name cannot be empty"
	}
	if err := validation.String(r.Name, apiconstants.MinSessionNameLength, apiconstants.MaxSessionNameLength); err != nil {
		problems["name"] = "invalid name: " + err.Error()
	}

	// date validation
	date, err := validation.Date(r.Date, apiconstants.DATE_LAYOUT, nil, nil)
	if err != nil {
		problems["date"] = "invalid date: " + err.Error()
	}
	r.date = date

	// start timestamp validation
	if r.StartTimestamp < 0 {
		problems["start_timestamp"] = "invalid start_timestamp: start_timestamp must be greater than UNIX epoch"
	}
	r.startTimestamp = time.Unix(r.StartTimestamp, 0).UTC()

	// duration minutes validation
	if r.DurationMinutes < 0 {
		problems["duration_minutes"] = fmt.Sprintf("invalid duration_minutes: duration_minutes must be between 1 and %d minutes", math.MaxInt16)
	} else {
		if r.DurationMinutes > math.MaxInt16 {
			problems["duration_minutes"] = fmt.Sprintf(
				"invalid duration_minutes: duration_minutes must be between 1 and %d minutes",
				math.MaxInt16)
		}
		r.durationMinutes = int16(r.DurationMinutes)
	}

	return problems
}

func HandlerCreateSession(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get userID from the context
		userID, ok := middleware.UserFromContext(r.Context())
		if !ok {
			util.RespondWithError(w, http.StatusInternalServerError, "Could not find user id in request context", nil)
			return
		}

		// Check userID against database
		if _, err := db.GetUser(r.Context(), userID); err != nil {
			util.RespondWithError(w, http.StatusUnauthorized, "Invalid credentials",
				fmt.Errorf("could not find the userID provied by the JWT in the database: %w", err))
			return
		}

		// Populate and validate the incoming request
		reqParams, problems, err := validation.DecodeValid[*sessionReq](r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// Fill the database object and create the session
		dbParams := database.CreateSessionParams{
			Name:            reqParams.Name,
			Date:            pgtype.Date{Time: reqParams.date, Valid: true},
			UserID:          userID,
			StartTimestamp:  pgtype.Timestamp{Time: reqParams.startTimestamp, Valid: true},
			DurationMinutes: pgtype.Int2{Int16: reqParams.durationMinutes, Valid: true},
		}

		session, err := db.CreateSession(r.Context(), dbParams)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Could not create the session", err)
			return
		}

		util.RespondWithJSON(w, http.StatusCreated,
			sessionRes{
				ID: session.ID.String(),
				sessionReq: sessionReq{
					Name:            session.Name,
					Date:            session.Date.Time.Format(apiconstants.DATE_LAYOUT),
					StartTimestamp:  session.StartTimestamp.Time.Unix(),
					DurationMinutes: int(session.DurationMinutes.Int16),
				},
			})
	}
}

// Populate needed empty fields: name and date
func (r *sessionReq) populate() {
	if r.Name == "" {
		now := time.Now()
		r.Name = now.Format(apiconstants.DATE_TIME_LAYOUT)
	}

	if r.Date == "" {
		now := time.Now()
		r.Date = now.Format(apiconstants.DATE_LAYOUT)
	}
}
