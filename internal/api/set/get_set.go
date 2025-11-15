package set

import (
	"log/slog"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/exlog"
	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
)

func HandlerGetSet(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	type res struct {
		SetRes
		Logs []exlog.LogRes `json:"logs"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		setID, _ := retrieveParseIDFromContext(r.Context())
		reqLogger = reqLogger.With(slog.Int64("set_id", setID))

		// fetch a set by set id
		setDB, err := db.GetSet(r.Context(), setID)
		if err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "not found", err)
			return
		} else if err != nil {
			reqLogger.Error("get set failed - fetch set database error", slog.String("error", err.Error()))
			util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// fetch logs by set id
		logsDB, err := db.GetLogsBySetID(r.Context(), setDB.ID)
		if err != nil {
			reqLogger.Error("get set failed - fetch logs database error", slog.String("error", err.Error()))
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
