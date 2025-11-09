package set

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSet(t *testing.T) {
	testCases := []struct {
		name         string
		order        int32
		restTime     int32
		statusCode   int
		exerciseID   int32
		sessionIDStr string
		hasEmptyJSON bool
		errMessage   []string
	}{
		{
			name:       "happy path",
			order:      1,
			restTime:   90,
			statusCode: http.StatusCreated,
		},
		{
			name:         "invalid session id",
			sessionIDStr: "notvalid",
			statusCode:   http.StatusNotFound,
			errMessage:   []string{"session ID not found"},
		},
		{
			name:       "invalid order",
			order:      -1,
			statusCode: http.StatusBadRequest,
			errMessage: []string{"invalid order"},
		},
		{
			name:         "session id not found",
			sessionIDStr: uuid.NewString(),
			statusCode:   http.StatusNotFound,
			errMessage:   []string{"session ID not found"},
		},
		{
			name:       "negative rest time should return 0 value",
			restTime:   -1,
			statusCode: http.StatusCreated,
		},
		{
			name:       "exercise id not found",
			restTime:   -1,
			exerciseID: -100,
			statusCode: http.StatusNotFound,
		},
		{
			name:       "rest time value too large",
			restTime:   apiconstants.MaxRestTimeSeconds + 1,
			statusCode: http.StatusBadRequest,
			errMessage: []string{"must be less than"},
		},
	}

	require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
	require.NoError(t, testutil.Cleanup(dbPool, "exercises"))
	require.NoError(t, testutil.Cleanup(dbPool, "users"))
	db := database.New(dbPool)
	user := testutil.CreateUserDBTestHelper(t, db, "usertest", "passwordtest", false)
	sessionID := testutil.CreateSessionDBTestHelper(t, db, "test name", user.ID.Bytes)
	exerciseID := testutil.CreateExerciseDBTestHelper(t, db, "pull ups")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Cleanup(dbPool, "sets")
			// Set up the response recorder and the request
			reader := &bytes.Reader{}
			if tc.hasEmptyJSON {
				reader = bytes.NewReader([]byte("{}"))
			} else {
				reqParams := SetReq{
					RestTime:   tc.restTime,
					SetOrder:   tc.order,
					ExerciseID: exerciseID,
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
			req.SetPathValue("sessionID", sessionID.String())
			if tc.sessionIDStr != "" {
				req.SetPathValue("sessionID", tc.sessionIDStr)
			}
			rr := httptest.NewRecorder()

			// call the function
			HandlerCreateSet(db).ServeHTTP(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("Status code do not match, want %d, got %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}
			if tc.statusCode > 399 {
				for _, msg := range tc.errMessage {
					assert.Contains(t, rr.Body.String(), msg)
				}
				return
			} else {
				// check the body to make sure
				var resParams SetRes
				decoder := json.NewDecoder(rr.Body)
				require.NoError(t, decoder.Decode(&resParams))
				// assert values
				assert.Equal(t, req.PathValue("sessionID"), resParams.SessionID)
				if tc.restTime > 0 {
					assert.Equal(t, tc.restTime, resParams.RestTime)
				} else {
					assert.Equal(t, int32(0), resParams.RestTime)
				}
				assert.Equal(t, tc.order, resParams.SetOrder)
				// check that the created user is on the database
				_, err := db.GetSet(context.Background(), resParams.ID)
				assert.NoError(t, err)
			}
		})
	}
}
