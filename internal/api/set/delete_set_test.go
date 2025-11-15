package set

import (
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

func TestHandlerDeleteSet(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		errMsg     []string
		hasSetID   bool
		setID      int64
	}{
		{
			name:       "happy path",
			statusCode: http.StatusNoContent,
			hasSetID:   true,
		},
		{
			name:       "set does not exist",
			statusCode: http.StatusNotFound,
			hasSetID:   true,
			setID:      -1,
			errMsg:     []string{"not found"},
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

			req, err := http.NewRequest("DELETE", "/test", nil)
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

			handler := HandlerDeleteSet(db, logger)
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

			_, err = db.GetSet(ctx, setID)
			require.Error(t, err, "expected set to be deleted")
		})
	}
}
