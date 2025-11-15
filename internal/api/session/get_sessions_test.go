package session

import (
	"encoding/json"
	"fmt"
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

func TestHandlerGetSessions(t *testing.T) {
	testCases := []struct {
		name          string
		setupSessions int
		setupSets     int
		setupLogs     int
		offset        string
		limit         string
		expectedCount int
		statusCode    int
		errMsg        []string
		noUserContext bool
	}{
		{
			name:          "happy path: no sessions",
			expectedCount: 0,
			statusCode:    http.StatusOK,
		},
		{
			name:          "happy path: single session with set and log",
			setupSessions: 1,
			setupSets:     1,
			setupLogs:     1,
			expectedCount: 1,
			statusCode:    http.StatusOK,
		},
		{
			name:          "happy path: multiple sessions",
			setupSessions: 3,
			setupSets:     2,
			setupLogs:     2,
			expectedCount: 3,
			statusCode:    http.StatusOK,
		},
		{
			name:          "happy path: session with multiple sets",
			setupSessions: 1,
			setupSets:     3,
			setupLogs:     2,
			expectedCount: 1,
			statusCode:    http.StatusOK,
		},
		{
			name:          "happy path: with valid offset",
			setupSessions: 5,
			setupSets:     1,
			setupLogs:     1,
			offset:        "2",
			expectedCount: 3,
			statusCode:    http.StatusOK,
		},
		{
			name:          "happy path: with valid limit",
			setupSessions: 5,
			setupSets:     1,
			setupLogs:     1,
			limit:         "3",
			expectedCount: 3,
			statusCode:    http.StatusOK,
		},
		{
			name:          "happy path: with offset and limit",
			setupSessions: 10,
			setupSets:     1,
			setupLogs:     1,
			offset:        "5",
			limit:         "3",
			expectedCount: 3,
			statusCode:    http.StatusOK,
		},
		{
			name:          "invalid offset format",
			setupSessions: 1,
			setupSets:     1,
			setupLogs:     1,
			offset:        "invalid",
			statusCode:    http.StatusBadRequest,
			errMsg:        []string{"invalid offset format"},
		},
		{
			name:          "invalid limit format",
			setupSessions: 1,
			setupSets:     1,
			setupLogs:     1,
			limit:         "invalid",
			statusCode:    http.StatusBadRequest,
			errMsg:        []string{"invalid limit format"},
		},
		{
			name:          "negative offset",
			setupSessions: 1,
			setupSets:     1,
			setupLogs:     1,
			offset:        "-1",
			statusCode:    http.StatusBadRequest,
			errMsg:        []string{"invalid offset value"},
		},
		{
			name:          "negative limit",
			setupSessions: 1,
			setupSets:     1,
			setupLogs:     1,
			limit:         "-1",
			statusCode:    http.StatusBadRequest,
			errMsg:        []string{"invalid limit value"},
		},
		{
			name:          "limit exceeds max",
			setupSessions: 1,
			setupSets:     1,
			setupLogs:     1,
			limit:         "21",
			statusCode:    http.StatusBadRequest,
			errMsg:        []string{"invalid limit value"},
		},
		{
			name:          "user not in context",
			setupSessions: 1,
			setupSets:     1,
			setupLogs:     1,
			noUserContext: true,
			statusCode:    http.StatusInternalServerError,
			errMsg:        []string{"something went wrong"},
		},
	}

	require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
	require.NoError(t, testutil.Cleanup(dbPool, "sets"))
	require.NoError(t, testutil.Cleanup(dbPool, "logs"))
	require.NoError(t, testutil.Cleanup(dbPool, "users"))

	db := database.New(dbPool)
	user := testutil.CreateUserDBTestHelper(t, db, "testuser", "testpassword", false)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
			require.NoError(t, testutil.Cleanup(dbPool, "sets"))
			require.NoError(t, testutil.Cleanup(dbPool, "logs"))

			for i := 0; i < tc.setupSessions; i++ {
				sessionID := testutil.CreateSessionDBTestHelper(
					t,
					db,
					fmt.Sprintf("session-%d", i),
					user.ID,
				)
				exerciseID := testutil.CreateExerciseDBTestHelper(t, db, "squats")

				for j := 0; j < tc.setupSets; j++ {
					setID := testutil.CreateSetDBTestHelper(t, db, sessionID, exerciseID)

					for k := 0; k < tc.setupLogs; k++ {
						testutil.CreateLogExerciseDBTestHelper(t, db, int32(k+1), int32(k+1), exerciseID, setID, float64(100+k*10))
					}
				}
			}

			url := "/test"
			if tc.offset != "" || tc.limit != "" {
				url += "?"
				if tc.offset != "" {
					url += fmt.Sprintf("offset=%s", tc.offset)
				}
				if tc.limit != "" {
					if tc.offset != "" {
						url += "&"
					}
					url += fmt.Sprintf("limit=%s", tc.limit)
				}
			}

			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err, "unexpected error while creating the request")

			if !tc.noUserContext {
				ctx := util.ContextWithUser(req.Context(), user.ID)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()

			handler := HandlerGetSessions(db, logger)
			middleware.RequestID(handler).ServeHTTP(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("Status code do not match, want %d, got %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}

			if tc.statusCode > 399 {
				for _, message := range tc.errMsg {
					assert.Contains(t, rr.Body.String(), message)
				}
				return
			}

			type setItem struct {
				ID        int64  `json:"id"`
				SessionID string `json:"session_id"`
				Logs      []struct {
					ID    int64 `json:"id"`
					SetID int64 `json:"set_id"`
				} `json:"logs"`
			}
			type sessionItem struct {
				ID   string    `json:"id"`
				Sets []setItem `json:"sets"`
			}
			type res struct {
				Sessions []sessionItem `json:"sessions"`
				Total    int           `json:"total"`
			}

			var resParams res
			decoder := json.NewDecoder(rr.Body)
			require.NoError(t, decoder.Decode(&resParams))
			assert.Equal(t, tc.expectedCount, len(resParams.Sessions))
			assert.Equal(t, tc.setupSessions, resParams.Total)

			if tc.setupSessions > 0 {
				for _, session := range resParams.Sessions {
					assert.NotEmpty(t, session.ID)
					if tc.setupSets > 0 {
						assert.Equal(t, tc.setupSets, len(session.Sets))
						for _, set := range session.Sets {
							assert.NotZero(t, set.ID)
							assert.Equal(t, session.ID, set.SessionID)
							if tc.setupLogs > 0 {
								assert.Equal(t, tc.setupLogs, len(set.Logs))
								for _, log := range set.Logs {
									assert.NotZero(t, log.ID)
									assert.Equal(t, set.ID, log.SetID)
								}
							}
						}
					}
				}
			}
		})
	}
}
