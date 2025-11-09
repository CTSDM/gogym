package exlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerUpdateLog(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		errMsg       []string
		hasJSON      bool
		hasEmptyJSON bool
		setID        int64
		exerciseID   int32
		weight       float64
		reps         int32
		order        int32
		logID        int64
		userID       uuid.UUID
		hasLogID     bool
	}{
		{
			name:       "happy path",
			statusCode: http.StatusOK,
			hasJSON:    true,
			exerciseID: 1,
			weight:     100.5,
			reps:       10,
			order:      1,
			hasLogID:   true,
		},
		{
			name:       "log id does not exist",
			statusCode: http.StatusNotFound,
			hasJSON:    true,
			weight:     100,
			reps:       10,
			order:      1,
			setID:      99999,
			logID:      -5, // ids are always positive
			hasLogID:   true,
			errMsg:     []string{"not found"},
		},
		{
			name:       "log id not found in the context",
			statusCode: http.StatusInternalServerError,
			hasJSON:    true,
			weight:     100,
			reps:       10,
			order:      1,
			setID:      99999,
			errMsg:     []string{"something went wrong"},
		},
	}

	require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
	require.NoError(t, testutil.Cleanup(dbPool, "sets"))
	require.NoError(t, testutil.Cleanup(dbPool, "exercises"))
	db := database.New(dbPool)
	user := testutil.CreateUserDBTestHelper(t, db, "testuser", "testpassword", false)
	sessionID := testutil.CreateSessionDBTestHelper(t, db, "test session", user.ID.Bytes)
	exerciseID := testutil.CreateExerciseDBTestHelper(t, db, "pull ups")
	setID := testutil.CreateSetDBTestHelper(t, db, sessionID, exerciseID)
	// a log is created with negative values
	logID := testutil.CreateLogExerciseDBTestHelper(t, db, -10, -5, exerciseID, setID, -100)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &bytes.Reader{}
			if tc.hasEmptyJSON {
				reader = bytes.NewReader([]byte("{}"))
			} else if tc.hasJSON {
				reqParams := LogReq{
					ExerciseID: exerciseID,
					Weight:     tc.weight,
					Reps:       tc.reps,
					Order:      tc.order,
				}
				if tc.exerciseID != 0 {
					reqParams.ExerciseID = tc.exerciseID
				}
				body, err := json.Marshal(reqParams)
				require.NoError(t, err, "unexpected JSON marshal error")
				reader = bytes.NewReader(body)
			}

			req, err := http.NewRequest("POST", "/test", reader)
			require.NoError(t, err, "unexpected error while creating the request")

			// set up the user id as the owner
			ctx := req.Context()
			if tc.userID != [16]byte{} {
				ctx = middleware.ContextWithUser(ctx, tc.userID)
			} else {
				ctx = middleware.ContextWithUser(ctx, user.ID.Bytes)
			}
			// set up the log id in the context using ContextWithResource
			if tc.hasLogID {
				if tc.logID != 0 {
					ctx = middleware.ContextWithResourceID(ctx, tc.logID)
				} else {
					ctx = middleware.ContextWithResourceID(ctx, logID)
				}
			}

			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			HandlerUpdateLog(db).ServeHTTP(rr, req)
			if tc.statusCode != rr.Code {
				t.Fatalf("Body response: %s", rr.Body.String())
			}
			if tc.statusCode > 399 {
				for _, message := range tc.errMsg {
					assert.Contains(t, rr.Body.String(), message)
				}
				return
			} else {
				var resParams LogRes
				decoder := json.NewDecoder(rr.Body)
				require.NoError(t, decoder.Decode(&resParams))
				assert.Equal(t, setID, resParams.SetID)
				assert.Equal(t, int32(exerciseID), resParams.ExerciseID)
				if tc.weight > 0 {
					assert.Equal(t, tc.weight, resParams.Weight)
				} else {
					assert.Equal(t, 0.0, resParams.Weight)
				}
				assert.Equal(t, tc.reps, resParams.Reps)
				assert.Equal(t, tc.order, resParams.Order)
			}
		})
	}
}
