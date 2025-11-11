package session

import (
	"net/http"

	"github.com/CTSDM/gogym/internal/api/exlog"
	"github.com/CTSDM/gogym/internal/api/set"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func HandlerGetSession(db *database.Queries) http.HandlerFunc {
	type setItem struct {
		set.SetRes
		Logs []exlog.LogRes `json:"logs"`
	}
	type res struct {
		sessionRes
		Sets []setItem `json:"sets"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Get session id from the context
		sessionID, err := retrieveParseUUIDFromContext(r.Context())
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// Fetch the session
		sessionRow, err := db.GetSession(r.Context(), sessionID)
		if err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "session not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// Fetch the sets related to the session
		setRows, err := db.GetSetsBySessionIDs(r.Context(), []uuid.UUID{sessionID})
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// collect set IDs
		setIDs := make([]int64, len(setRows))
		for i, s := range setRows {
			setIDs[i] = s.ID
		}

		// Fetch logs for these sets
		var logs []database.Log
		if len(setIDs) > 0 {
			logs, err = db.GetLogsBySetIDs(r.Context(), setIDs)
			if err != nil {
				util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
				return
			}
		}

		// build response structure
		logsBySetID := make(map[int64][]exlog.LogRes)
		for _, log := range logs {
			logsBySetID[log.SetID] = append(logsBySetID[log.SetID], exlog.LogRes{
				ID:    log.ID,
				SetID: log.SetID,
				LogReq: exlog.LogReq{
					ExerciseID: log.ExerciseID,
					Weight:     log.Weight.Float64,
					Reps:       log.Reps,
					Order:      log.LogsOrder,
				},
			})
		}

		setsBySessionID := make(map[string][]setItem)
		for _, s := range setRows {
			sessionID := s.SessionID.String()
			setsBySessionID[sessionID] = append(setsBySessionID[sessionID], setItem{
				SetRes: set.SetRes{
					ID:        s.ID,
					SessionID: sessionID,
					SetReq: set.SetReq{
						ExerciseID: s.ExerciseID,
						SetOrder:   s.SetOrder,
						RestTime:   s.RestTime.Int32,
					},
				},
				Logs: logsBySetID[s.ID],
			})
		}

		resParams := res{
			sessionRes: sessionRes{
				ID: sessionID.String(),
				sessionReq: sessionReq{
					Name:            sessionRow.Name,
					Date:            sessionRow.Date.Time.Format(apiconstants.DATE_LAYOUT),
					StartTimestamp:  sessionRow.StartTimestamp.Time.Unix(),
					DurationMinutes: int(sessionRow.DurationMinutes.Int16),
				},
			},
			Sets: setsBySessionID[sessionID.String()],
		}

		util.RespondWithJSON(w, http.StatusOK, resParams)
	}
}
