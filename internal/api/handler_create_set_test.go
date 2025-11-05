package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSet(t *testing.T) {
	apiState := NewState(database.New(dbPool), &auth.Config{})
	testCases := []struct {
		name         string
		order        int32
		restTime     int32
		statusCode   int
		sessionIDStr string
		hasEmptyJSON bool
		errMessage   string
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
			errMessage:   "session ID not found",
		},
		{
			name:       "invalid order",
			order:      -1,
			statusCode: http.StatusBadRequest,
			errMessage: "must be greater",
		},
		{
			name:         "session id not found",
			sessionIDStr: uuid.NewString(),
			statusCode:   http.StatusNotFound,
			errMessage:   "session ID not found",
		},
		{
			name:       "negative rest time should return 0 value",
			restTime:   -1,
			statusCode: http.StatusCreated,
		},
		{
			name:       "rest time value too large",
			restTime:   maxRestTimeSeconds + 1,
			statusCode: http.StatusBadRequest,
			errMessage: "must be less than",
		},
	}

	sessionID := createSessionDBTestHelper(t, apiState, "test name")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup("sets")
			// Set up the response recorder and the request
			reader := &bytes.Reader{}
			if tc.hasEmptyJSON {
				reader = bytes.NewReader([]byte("{}"))
			} else {
				reqParams := createSetReq{
					RestTime: tc.restTime,
					SetOrder: tc.order,
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
			apiState.HandlerCreateSet(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("Status code do not match, want %d, got %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}
			if tc.statusCode > 399 {
				assert.Contains(t, rr.Body.String(), tc.errMessage)
				return
			} else {
				// check the body to make sure
				var resParams createSetRes
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
				_, err := apiState.db.GetSet(context.Background(), int64(resParams.ID))
				assert.NoError(t, err)
			}
		})
	}
}
