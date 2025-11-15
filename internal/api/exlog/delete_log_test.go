package exlog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerDeleteLog(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		errMsg     []string
		userID     uuid.UUID
		hasLogID   bool
		logID      int64
	}{
		{
			name:       "happy path",
			statusCode: http.StatusNoContent,
			hasLogID:   true,
		},
		{
			name:       "log id not found in the context",
			statusCode: http.StatusInternalServerError,
			errMsg:     []string{"something went wrong"},
		},
		{
			name:       "log does not exist",
			statusCode: http.StatusInternalServerError,
			hasLogID:   true,
			logID:      -1,
			errMsg:     []string{"something went wrong"},
		},
	}

	require.NoError(t, testutil.Cleanup(dbPool, "users"))
	require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
	require.NoError(t, testutil.Cleanup(dbPool, "sets"))
	require.NoError(t, testutil.Cleanup(dbPool, "exercises"))
	db := database.New(dbPool)
	user := testutil.CreateUserDBTestHelper(t, db, "testuser", "testpassword", false)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sessionID := testutil.CreateSessionDBTestHelper(t, db, "test session", user.ID)
			exerciseID := testutil.CreateExerciseDBTestHelper(t, db, "pull ups")
			setID := testutil.CreateSetDBTestHelper(t, db, sessionID, exerciseID)
			logID := testutil.CreateLogExerciseDBTestHelper(t, db, 10, 5, exerciseID, setID, 100)

			req, err := http.NewRequest("DELETE", "/test", nil)
			require.NoError(t, err, "unexpected error while creating the request")

			ctx := req.Context()
			if tc.userID != [16]byte{} {
				ctx = util.ContextWithUser(ctx, tc.userID)
			} else {
				ctx = util.ContextWithUser(ctx, user.ID)
			}

			if tc.hasLogID {
				if tc.logID != 0 {
					ctx = util.ContextWithResourceID(ctx, tc.logID)
				} else {
					ctx = util.ContextWithResourceID(ctx, logID)
				}
			}

			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler := HandlerDeleteLog(db, logger)
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

			_, err = db.GetLog(ctx, logID)
			require.Error(t, err, "expected log to be deleted")
		})
	}
}
