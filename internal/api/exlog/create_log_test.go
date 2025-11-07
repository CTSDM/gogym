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
		errMsg       string
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
			errMsg:     "Invalid payload",
		},
		{
			name:         "empty JSON",
			statusCode:   http.StatusBadRequest,
			hasJSON:      true,
			hasEmptyJSON: true,
			errMsg:       "cannot be",
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
			errMsg:     "log order cannot be less than zero",
		},
		{
			name:       "zero reps",
			statusCode: http.StatusBadRequest,
			hasJSON:    true,
			weight:     100,
			reps:       0,
			order:      1,
			errMsg:     "reps cannot be less than zero",
		},
		{
			name:       "negative reps",
			statusCode: http.StatusBadRequest,
			hasJSON:    true,
			weight:     100,
			reps:       -5,
			order:      1,
			errMsg:     "reps cannot be less than zero",
		},
		{
			name:       "set id does not exist",
			statusCode: http.StatusNotFound,
			hasJSON:    true,
			weight:     100,
			reps:       10,
			order:      1,
			setID:      99999,
			errMsg:     "not found",
		},
		{
			name:         "set id is not a valid id",
			statusCode:   http.StatusNotFound,
			hasJSON:      true,
			weight:       100,
			reps:         10,
			order:        1,
			errMsg:       "not found",
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
			errMsg:     "exercise id does not exist",
		},
	}

	require.NoError(t, testutil.Cleanup(dbPool, "sessions"))
	require.NoError(t, testutil.Cleanup(dbPool, "sets"))
	require.NoError(t, testutil.Cleanup(dbPool, "exercises"))
	db := database.New(dbPool)
	sessionID := testutil.CreateSessionDBTestHelper(t, db, "test session")
	exerciseID := testutil.CreateExerciseDBTestHelper(t, db, "pull ups")
	setID := testutil.CreateSetDBTestHelper(t, db, sessionID, exerciseID)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &bytes.Reader{}
			if tc.hasEmptyJSON {
				reader = bytes.NewReader([]byte("{}"))
			} else if tc.hasJSON {
				reqParams := createLogReq{
					ExerciseID: int32(exerciseID),
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
				assert.Contains(t, rr.Body.String(), tc.errMsg)
				return
			} else {
				var resParams createLogRes
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
		name       string
		req        createLogReq
		shouldErr  bool
		errMessage string
	}{
		{
			name: "happy path",
			req: createLogReq{
				Weight:     100.5,
				Reps:       10,
				Order:      1,
				ExerciseID: 1,
			},
			shouldErr: false,
		},
		{
			name: "negative weight should be set to zero",
			req: createLogReq{
				Weight:     -50,
				Reps:       10,
				Order:      1,
				ExerciseID: 1,
			},
			shouldErr: false,
		},
		{
			name: "negative order",
			req: createLogReq{
				Weight:     100,
				Reps:       10,
				Order:      -1,
				ExerciseID: 1,
			},
			shouldErr:  true,
			errMessage: "log order cannot be less than zero",
		},
		{
			name: "zero reps",
			req: createLogReq{
				Weight:     100,
				Reps:       0,
				Order:      1,
				ExerciseID: 1,
			},
			shouldErr:  true,
			errMessage: "reps cannot be less than zero",
		},
		{
			name: "negative reps",
			req: createLogReq{
				Weight:     100,
				Reps:       -5,
				Order:      1,
				ExerciseID: 1,
			},
			shouldErr:  true,
			errMessage: "reps cannot be less than zero",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.validate()
			if tc.shouldErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMessage)
			} else {
				require.NoError(t, err)
				if tc.req.Weight < 0 {
					assert.Equal(t, 0.0, tc.req.Weight)
				}
			}
		})
	}
}
