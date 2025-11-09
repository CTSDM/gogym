package session

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

func TestHandlerUpdateSession(t *testing.T) {
	testCases := []struct {
		name            string
		statusCode      int
		errMsg          []string
		sessionName     string
		date            string
		startTimestamp  int64
		durationMinutes int
		userID          uuid.UUID
		hasSessionID    bool
		sessionID       int64
	}{
		{
			name:         "happy path",
			statusCode:   http.StatusOK,
			sessionName:  "updated session",
			date:         "2025-11-08",
			hasSessionID: true,
		},
		{
			name:       "session id not found in the context",
			statusCode: http.StatusInternalServerError,
			errMsg:     []string{"something went wrong"},
		},
		{
			name:         "session does not exist",
			statusCode:   http.StatusNotFound,
			sessionName:  "updated session",
			date:         "2025-11-08",
			hasSessionID: true,
			sessionID:    -1,
			errMsg:       []string{"session not found"},
		},
		{
			name:         "invalid date",
			statusCode:   http.StatusBadRequest,
			sessionName:  "updated session",
			date:         "invalid-date",
			hasSessionID: true,
			errMsg:       []string{"invalid date"},
		},
		{
			name:           "invalid start timestamp",
			statusCode:     http.StatusBadRequest,
			sessionName:    "updated session",
			date:           "2025-11-08",
			startTimestamp: -100,
			hasSessionID:   true,
			errMsg:         []string{"invalid start_timestamp"},
		},
		{
			name:            "invalid duration minutes",
			statusCode:      http.StatusBadRequest,
			sessionName:     "updated session",
			date:            "2025-11-08",
			durationMinutes: -5,
			hasSessionID:    true,
			errMsg:          []string{"invalid duration_minutes"},
		},
	}

	require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
	require.NoError(t, testutil.Cleanup(dbPool, "users"))
	db := database.New(dbPool)
	user := testutil.CreateUserDBTestHelper(t, db, "testuser", "testpassword", false)
	sessionID := testutil.CreateSessionDBTestHelper(t, db, "test session", user.ID)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &bytes.Reader{}

			reqParams := sessionReq{}
			if tc.sessionName != "" {
				reqParams.Name = tc.sessionName
			}
			if tc.date != "" {
				reqParams.Date = tc.date
			}
			if tc.startTimestamp != 0 {
				reqParams.StartTimestamp = tc.startTimestamp
			}
			if tc.durationMinutes != 0 {
				reqParams.DurationMinutes = tc.durationMinutes
			}
			body, err := json.Marshal(reqParams)
			require.NoError(t, err, "unexpected JSON marshal error")
			reader = bytes.NewReader(body)

			req, err := http.NewRequest("PUT", "/test", reader)
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

			HandlerUpdateSession(db).ServeHTTP(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("mismatch in status code, want %d, gots %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}
			if tc.statusCode > 399 {
				for _, message := range tc.errMsg {
					assert.Contains(t, rr.Body.String(), message)
				}
				return
			} else {
				var resParams sessionRes
				decoder := json.NewDecoder(rr.Body)
				require.NoError(t, decoder.Decode(&resParams))
				assert.Equal(t, sessionID.String(), resParams.ID)
				if tc.sessionName != "" {
					assert.Equal(t, tc.sessionName, resParams.Name)
				}
				if tc.date != "" {
					assert.Equal(t, tc.date, resParams.Date)
				}
				if tc.startTimestamp > 0 {
					assert.Equal(t, tc.startTimestamp, resParams.StartTimestamp)
				}
				if tc.durationMinutes > 0 {
					assert.Equal(t, tc.durationMinutes, resParams.DurationMinutes)
				}
			}
		})
	}
}
