package exlog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerCreateLog(t *testing.T) {
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
		invalidSetID bool
	}{
		{
			name:       "happy path",
			statusCode: http.StatusCreated,
			hasJSON:    true,
			exerciseID: 1,
			weight:     100.5,
			reps:       10,
			order:      1,
		},
		{
			name:       "no JSON sent",
			statusCode: http.StatusBadRequest,
			errMsg:     []string{"invalid payload"},
		},
		{
			name:         "empty JSON",
			statusCode:   http.StatusBadRequest,
			hasJSON:      true,
			hasEmptyJSON: true,
			errMsg:       []string{"invalid reps"},
		},
		{
			name:       "negative weight should be set to zero",
			statusCode: http.StatusCreated,
			hasJSON:    true,
			weight:     -50,
			reps:       10,
			order:      1,
		},
		{
			name:       "negative order",
			statusCode: http.StatusBadRequest,
			hasJSON:    true,
			weight:     100,
			reps:       10,
			order:      -1,
			errMsg:     []string{"invalid order"},
		},
		{
			name:       "zero reps",
			statusCode: http.StatusBadRequest,
			hasJSON:    true,
			weight:     100,
			reps:       0,
			order:      1,
			errMsg:     []string{"invalid reps"},
		},
		{
			name:       "negative reps",
			statusCode: http.StatusBadRequest,
			hasJSON:    true,
			weight:     100,
			reps:       -5,
			order:      1,
			errMsg:     []string{"invalid reps"},
		},
		{
			name:       "set id does not exist",
			statusCode: http.StatusNotFound,
			hasJSON:    true,
			weight:     100,
			reps:       10,
			order:      1,
			setID:      99999,
			errMsg:     []string{"not found"},
		},
		{
			name:         "set id is not a valid id",
			statusCode:   http.StatusNotFound,
			hasJSON:      true,
			weight:       100,
			reps:         10,
			order:        1,
			errMsg:       []string{"not found"},
			invalidSetID: true,
		},
		{
			name:       "exercise id not found",
			statusCode: http.StatusNotFound,
			hasJSON:    true,
			weight:     100,
			reps:       10,
			order:      1,
			exerciseID: 99999,
			errMsg:     []string{"not found"},
		},
	}

	require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
	require.NoError(t, testutil.Cleanup(dbPool, "sets"))
	require.NoError(t, testutil.Cleanup(dbPool, "exercises"))
	db := database.New(dbPool)
	user := testutil.CreateUserDBTestHelper(t, db, "usertest", "passwordtest", false)
	sessionID := testutil.CreateSessionDBTestHelper(t, db, "test session", user.ID.Bytes)
	exerciseID := testutil.CreateExerciseDBTestHelper(t, db, "pull ups")
	setID := testutil.CreateSetDBTestHelper(t, db, sessionID, exerciseID)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &bytes.Reader{}
			if tc.hasEmptyJSON {
				reader = bytes.NewReader([]byte("{}"))
			} else if tc.hasJSON {
				reqParams := logReq{
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

			// set up the path value
			req.SetPathValue("setID", strconv.FormatInt(setID, 10))
			if tc.invalidSetID {
				req.SetPathValue("setID", "not an int")
			} else if tc.setID != 0 {
				req.SetPathValue("setID", strconv.FormatInt(tc.setID, 10))
			}

			rr := httptest.NewRecorder()

			HandlerCreateLog(db).ServeHTTP(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("Status code do not match, want %d, got %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}
			if tc.statusCode > 399 {
				for _, message := range tc.errMsg {
					assert.Contains(t, rr.Body.String(), message)
				}
				return
			} else {
				var resParams logRes
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
				_, err := db.GetLog(context.Background(), resParams.ID)
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCreateLog(t *testing.T) {
	testCases := []struct {
		name      string
		req       logReq
		shouldErr bool
		errKeys   map[string]string
	}{
		{
			name: "happy path",
			req: logReq{
				Weight:     100.5,
				Reps:       10,
				Order:      1,
				ExerciseID: 1,
			},
			shouldErr: false,
		},
		{
			name: "negative weight should be set to zero",
			req: logReq{
				Weight:     -50,
				Reps:       10,
				Order:      1,
				ExerciseID: 1,
			},
			shouldErr: false,
		},
		{
			name: "negative order",
			req: logReq{
				Weight:     100,
				Reps:       10,
				Order:      -1,
				ExerciseID: 1,
			},
			shouldErr: true,
			errKeys: map[string]string{
				"order": "must be positive",
			},
		},
		{
			name: "zero reps",
			req: logReq{
				Weight:     100,
				Reps:       0,
				Order:      1,
				ExerciseID: 1,
			},
			shouldErr: true,
			errKeys: map[string]string{
				"reps": "must be positive",
			},
		},
		{
			name: "negative reps",
			req: logReq{
				Weight:     100,
				Reps:       -5,
				Order:      1,
				ExerciseID: 1,
			},
			shouldErr: true,
			errKeys: map[string]string{
				"reps": "must be positive",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			problems := tc.req.Valid(context.Background())
			if tc.shouldErr {
				require.Greater(t, len(problems), 0)
				for key, value := range tc.errKeys {
					got, ok := problems[key]
					if !ok {
						t.Errorf("key not found: %s", key)
					} else {
						assert.Contains(t, got, value)
					}
				}
			} else {
				require.Equal(t, 0, len(problems))
				if tc.req.Weight < 0 {
					assert.Equal(t, 0.0, tc.req.Weight)
				}
			}
		})
	}
}
