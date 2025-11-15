package set

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerUpdateSet(t *testing.T) {
	testCases := []struct {
		name          string
		statusCode    int
		errMsg        []string
		setOrder      int32
		restTime      int32
		exerciseID    int32
		newExerciseID int32
		hasSetID      bool
		setID         int64
		createLogs    bool
	}{
		{
			name:       "happy path",
			statusCode: http.StatusNoContent,
			setOrder:   2,
			restTime:   120,
			hasSetID:   true,
		},
		{
			name:          "happy path with exercise id change",
			statusCode:    http.StatusNoContent,
			setOrder:      1,
			restTime:      90,
			newExerciseID: 0,
			hasSetID:      true,
			createLogs:    true,
		},
		{
			name:       "set id not found in the context",
			statusCode: http.StatusInternalServerError,
			errMsg:     []string{"something went wrong"},
		},
		{
			name:       "set does not exist",
			statusCode: http.StatusInternalServerError,
			setOrder:   1,
			restTime:   60,
			hasSetID:   true,
			setID:      -1,
			errMsg:     []string{"something went wrong"},
		},
		{
			name:       "invalid order",
			statusCode: http.StatusBadRequest,
			setOrder:   -1,
			restTime:   60,
			hasSetID:   true,
			errMsg:     []string{"invalid order"},
		},
		{
			name:       "invalid rest time",
			statusCode: http.StatusBadRequest,
			setOrder:   1,
			restTime:   apiconstants.MaxRestTimeSeconds + 1,
			hasSetID:   true,
			errMsg:     []string{"must be less than"},
		},
	}

	require.NoError(t, testutil.Cleanup(dbPool, "users"))
	require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
	require.NoError(t, testutil.Cleanup(dbPool, "sets"))
	require.NoError(t, testutil.Cleanup(dbPool, "exercises"))
	db := database.New(dbPool)
	user := testutil.CreateUserDBTestHelper(t, db, "testuser", "testpassword", false)
	sessionID := testutil.CreateSessionDBTestHelper(t, db, "test session", user.ID)
	exerciseID := testutil.CreateExerciseDBTestHelper(t, db, "pull ups")
	newExerciseID := testutil.CreateExerciseDBTestHelper(t, db, "push ups")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Cleanup(dbPool, "logs")
			testutil.Cleanup(dbPool, "sets")
			setID := testutil.CreateSetDBTestHelper(t, db, sessionID, exerciseID)

			if tc.createLogs {
				for i := range 3 {
					testutil.CreateLogExerciseDBTestHelper(t, db, int32(i+1), int32(10), exerciseID, setID, float64(100))
				}
			}

			reqParams := SetReq{
				SetOrder: tc.setOrder,
				RestTime: tc.restTime,
			}
			if tc.newExerciseID != 0 {
				reqParams.ExerciseID = tc.newExerciseID
			} else if tc.exerciseID != 0 {
				reqParams.ExerciseID = tc.exerciseID
			} else {
				reqParams.ExerciseID = exerciseID
			}

			if tc.name == "happy path with exercise id change" {
				reqParams.ExerciseID = newExerciseID
			}

			body, err := json.Marshal(reqParams)
			require.NoError(t, err, "unexpected JSON marshal error")
			reader := bytes.NewReader(body)

			req, err := http.NewRequest("PUT", "/test", reader)
			require.NoError(t, err, "unexpected error while creating the request")

			ctx := req.Context()
			ctx = util.ContextWithUser(ctx, user.ID)

			if tc.hasSetID {
				if tc.setID != 0 {
					ctx = util.ContextWithResourceID(ctx, tc.setID)
				} else {
					ctx = util.ContextWithResourceID(ctx, setID)
				}
			}

			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler := HandlerUpdateSet(dbPool, db, logger)
			middleware.RequestID(handler).ServeHTTP(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("mismatch in status code, want %d, got %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}
			if tc.statusCode > 399 {
				for _, message := range tc.errMsg {
					assert.Contains(t, rr.Body.String(), message)
				}
				return
			}

			updatedSet, err := db.GetSet(ctx, setID)
			require.NoError(t, err)
			assert.Equal(t, tc.setOrder, updatedSet.SetOrder)
			assert.Equal(t, tc.restTime, updatedSet.RestTime.Int32)

			if tc.createLogs {
				logs, err := db.GetLogsBySetID(ctx, setID)
				require.NoError(t, err)
				for _, log := range logs {
					assert.Equal(t, newExerciseID, log.ExerciseID)
				}
			}
		})
	}
}
