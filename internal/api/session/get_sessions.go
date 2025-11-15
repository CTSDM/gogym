package session

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/CTSDM/gogym/internal/api/exlog"
	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/set"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	MAX_LIMIT      int32 = 20
	DEFAULT_LIMIT  int32 = 10
	DEFAULT_OFFSET int32 = 0
)

func HandlerGetSessions(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	type setItem struct {
		set.SetRes
		Logs []exlog.LogRes `json:"logs"`
	}
	type sessionItem struct {
		sessionRes
		Sets []setItem `json:"sets"`
	}
	type res struct {
		Sessions []sessionItem `json:"sessions"`
		Limit    int32
		Offset   int32
		Total    int `json:"total"` // total number of sessions for a given user
	}

	validateQueryParams := func(r *http.Request) (int32, int32, map[string]string) {
		problems := map[string]string{}
		var offset int32
		var limit int32

		// validate and offset
		if r.URL.Query().Has("offset") {
			parsed, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32)
			if err != nil {
				problems["offset"] = "invalid offset format"
			} else if parsed < 0 {
				problems["offset"] = "invalid offset value, must be positive"
			} else {
				offset = int32(parsed)
			}
		} else {
			offset = DEFAULT_OFFSET
		}

		// validate and limit
		if r.URL.Query().Has("limit") {
			parsed, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
			if err != nil {
				problems["limit"] = "invalid limit format"
			} else if parsed < 0 {
				problems["limit"] = "invalid limit value, must be positive"
			} else if int32(parsed) > MAX_LIMIT {
				problems["limit"] = fmt.Sprintf("invalid limit value, must be less than %d", MAX_LIMIT)
			} else {
				limit = int32(parsed)
			}
		} else {
			limit = DEFAULT_LIMIT
		}

		return offset, limit, problems
	}

	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		// retrieve user from context
		userID, ok := middleware.UserFromContext(r.Context())
		if !ok {
			reqLogger.Error("get sessions failed - could not find user in context")
			err := errors.New("could not find user in context")
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		reqLogger = reqLogger.With(slog.String("user_id", userID.String()))
		// Get offset and limit from the query parameters
		offset, limit, problems := validateQueryParams(r)
		if len(problems) > 0 {
			reqLogger.Debug("get sessions failed - validation error", slog.Any("problem", problems))
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		}

		// Get total number of sessions
		sessionsCount, err := db.GetNumberSessionsByUserID(r.Context(), userID)
		if err == pgx.ErrNoRows {
			// early return with empty structure
			util.RespondWithJSON(w, http.StatusOK, res{Sessions: make([]sessionItem, 0), Total: 0})
			return
		} else if err != nil {
			reqLogger.Error("get sessions failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// fetch sessions with pagination and offset
		sessions, err := db.GetSessionsPaginated(r.Context(), database.GetSessionsPaginatedParams{
			UserID: userID,
			Offset: offset,
			Limit:  limit,
		})
		if err != nil {
			reqLogger.Error("get sessions failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		if len(sessions) == 0 {
			util.RespondWithJSON(w, http.StatusOK, res{Sessions: []sessionItem{}, Total: int(sessionsCount)})
			return
		}

		// collect session IDs
		sessionIDs := make([]uuid.UUID, len(sessions))
		for i, s := range sessions {
			sessionIDs[i] = s.ID
		}

		// fetch sets for these sessions
		sets, err := db.GetSetsBySessionIDs(r.Context(), sessionIDs)
		if err != nil {
			reqLogger.Error("get sessions failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// collect set IDs
		setIDs := make([]int64, len(sets))
		for i, s := range sets {
			setIDs[i] = s.ID
		}

		// fetch logs for these sets
		var logs []database.Log
		if len(setIDs) > 0 {
			logs, err = db.GetLogsBySetIDs(r.Context(), setIDs)
			if err != nil {
				reqLogger.Error(
					"get sessions failed - database error",
					slog.String("error", err.Error()),
				)
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
		for _, s := range sets {
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

		result := make([]sessionItem, 0, len(sessions))
		for _, s := range sessions {
			sessionID := s.ID.String()
			result = append(result, sessionItem{
				sessionRes: sessionRes{
					ID: sessionID,
					sessionReq: sessionReq{
						Name:            s.Name,
						Date:            s.Date.Time.Format(apiconstants.DATE_LAYOUT),
						StartTimestamp:  s.StartTimestamp.Time.Unix(),
						DurationMinutes: int(s.DurationMinutes.Int16),
					},
				},
				Sets: setsBySessionID[sessionID],
			})
		}

		util.RespondWithJSON(w, http.StatusOK, res{
			Sessions: result,
			Total:    int(sessionsCount),
			Limit:    limit,
			Offset:   offset,
		})
	}
}
