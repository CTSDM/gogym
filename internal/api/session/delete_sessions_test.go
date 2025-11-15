package session

import (
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

func TestHandlerDeleteSession(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		errMsg       []string
		userID       uuid.UUID
		hasSessionID bool
		sessionID    int64
	}{
		{
			name:         "happy path",
			statusCode:   http.StatusNoContent,
			hasSessionID: true,
		},
		{
			name:         "session does not exist",
			statusCode:   http.StatusNotFound,
			hasSessionID: true,
			sessionID:    -1,
			errMsg:       []string{"not found"},
		},
		{
			name:         "user does not own session",
			statusCode:   http.StatusNotFound,
			hasSessionID: true,
			userID:       uuid.New(),
			errMsg:       []string{"not found"},
		},
	}

	require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
	require.NoError(t, testutil.Cleanup(dbPool, "users"))
	db := database.New(dbPool)
	user := testutil.CreateUserDBTestHelper(t, db, "testuser", "testpassword", false)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sessionID := testutil.CreateSessionDBTestHelper(t, db, "test session", user.ID)

			req, err := http.NewRequest("DELETE", "/test", nil)
			require.NoError(t, err, "unexpected error while creating the request")

			ctx := req.Context()
			if tc.userID != [16]byte{} {
				ctx = middleware.ContextWithUser(ctx, tc.userID)
			} else {
				ctx = middleware.ContextWithUser(ctx, user.ID)
			}

			if tc.hasSessionID {
				var sid uuid.UUID
				if tc.sessionID != 0 {
					sid = uuid.New()
				} else {
					sid = sessionID
				}
				ctx = middleware.ContextWithResourceID(ctx, sid)
			}

			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler := HandlerDeleteSession(db, logger)
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

			_, err = db.GetSession(ctx, sessionID)
			require.Error(t, err, "expected session to be deleted")
		})
	}
}
