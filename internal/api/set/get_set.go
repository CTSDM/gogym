package set

import (
	"net/http"
	"strconv"

	"github.com/CTSDM/gogym/internal/api/exlog"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
)

func HandlerGetSet(db *database.Queries) http.HandlerFunc {
	type res struct {
		SetRes
		Logs []exlog.LogRes `json:"logs"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		setID, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid set id", err)
			return
		}

		// fetch a set by set id
		setDB, err := db.GetSet(r.Context(), setID)
		if err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// fetch logs by set id
		logsDB, err := db.GetLogsBySetID(r.Context(), setDB.ID)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		logsResParams := make([]exlog.LogRes, len(logsDB))
		for i, logDB := range logsDB {
			logsResParams[i] = exlog.LogRes{
				ID:    logDB.ID,
				SetID: logDB.SetID,
				LogReq: exlog.LogReq{
					ExerciseID: logDB.ExerciseID,
					Weight:     logDB.Weight.Float64,
					Reps:       logDB.Reps,
					Order:      logDB.LogsOrder,
				},
			}
		}

		resParams := res{
			SetRes: SetRes{
				ID:        setDB.ID,
				SessionID: setDB.SessionID.String(),
				SetReq: SetReq{
					ExerciseID: setDB.ExerciseID,
					SetOrder:   setDB.SetOrder,
					RestTime:   setDB.RestTime.Int32,
				},
			},
			Logs: logsResParams,
		}

		util.RespondWithJSON(w, http.StatusOK, resParams)
	}
}
