package exlog

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
)

const (
	DEFAULT_OFFSET int32 = 0
	DEFAULT_LIMIT  int32 = 50
	MAX_LIMIT      int32 = 200
)

func HandlerGetLogs(db *database.Queries) http.HandlerFunc {
	type logItem struct {
		Log  LogRes
		Date string `json:"date"`
	}
	type res struct {
		Logs []logItem
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
		// Get user from the context
		userID, ok := middleware.UserFromContext(r.Context())
		if !ok {
			err := errors.New("could not find user in the context")
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		offset, limit, problems := validateQueryParams(r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		}

		// Get logs from the database
		rows, err := db.GetLogsByUserID(r.Context(), database.GetLogsByUserIDParams{
			UserID: userID,
			Offset: offset,
			Limit:  limit,
		})
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// build the response
		resParams := res{Logs: make([]logItem, len(rows))}
		for i, row := range rows {
			resParams.Logs[i] = logItem{
				Date: row.Date.Time.Format(apiconstants.DATE_LAYOUT),
				Log: LogRes{
					ID:    row.ID,
					SetID: row.SetID,
					LogReq: LogReq{
						ExerciseID: row.ExerciseID,
						Weight:     row.Weight.Float64,
						Reps:       row.Reps,
						Order:      row.LogsOrder,
					},
				},
			}
		}

		util.RespondWithJSON(w, http.StatusOK, resParams)
	}
}
