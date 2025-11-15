package set

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerGetSet(t *testing.T) {
	testCases := []struct {
		name         string
		setID        int64
		statusCode   int
		errMessage   string
		skipSetup    bool
		expectedLogs int
	}{
		{
			name:         "happy path with logs",
			statusCode:   http.StatusOK,
			expectedLogs: 3,
		},
		{
			name:         "happy path without logs",
			statusCode:   http.StatusOK,
			expectedLogs: 0,
		},
		{
			name:       "set not found - invalid id",
			setID:      99999,
			statusCode: http.StatusNotFound,
			errMessage: "not found",
		},
		{
			name:       "set not found - negative id",
			skipSetup:  true,
			setID:      -1,
			statusCode: http.StatusNotFound,
			errMessage: "not found",
		},
	}

	db := database.New(dbPool)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, testutil.Cleanup(dbPool, "users"))
			require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
			require.NoError(t, testutil.Cleanup(dbPool, "sets"))
			require.NoError(t, testutil.Cleanup(dbPool, "exercises"))

			var setID int64
			if !tc.skipSetup {
				user := testutil.CreateUserDBTestHelper(t, db, "testuser", "testpassword", false)
				sessionID := testutil.CreateSessionDBTestHelper(t, db, "test session", user.ID)
				exerciseID := testutil.CreateExerciseDBTestHelper(t, db, "pull ups")
				setID = testutil.CreateSetDBTestHelper(t, db, sessionID, exerciseID)

				for i := 0; i < tc.expectedLogs; i++ {
					testutil.CreateLogExerciseDBTestHelper(t, db, int32(10), int32(i+1), exerciseID, setID, float64(100+i*10))
				}
			}

			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err, "unexpected error while creating the request")
			rr := httptest.NewRecorder()

			var ctx context.Context
			if tc.setID != 0 {
				ctx = util.ContextWithResourceID(req.Context(), tc.setID)
			} else {
				ctx = util.ContextWithResourceID(req.Context(), setID)
			}
			req = req.WithContext(ctx)
			handler := HandlerGetSet(db, logger)
			middleware.RequestID(handler).ServeHTTP(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("Status code do not match, want %d, got %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}

			if tc.statusCode > 399 {
				assert.Contains(t, rr.Body.String(), tc.errMessage)
				return
			}

			var resParams struct {
				SetRes
				Logs []struct {
					ID         int64   `json:"id"`
					SetID      int64   `json:"set_id"`
					ExerciseID int32   `json:"exercise_id"`
					Weight     float64 `json:"weight"`
					Reps       int32   `json:"reps"`
					Order      int32   `json:"order"`
				} `json:"logs"`
			}
			decoder := json.NewDecoder(rr.Body)
			require.NoError(t, decoder.Decode(&resParams))
			assert.Equal(t, setID, resParams.ID)
			assert.Equal(t, tc.expectedLogs, len(resParams.Logs))

			for i, log := range resParams.Logs {
				assert.NotZero(t, log.ID)
				assert.Equal(t, setID, log.SetID)
				assert.Equal(t, int32(i+1), log.Order)
				assert.Equal(t, int32(10), log.Reps)
				assert.Equal(t, float64(100+i*10), log.Weight)
			}
		})
	}
}
