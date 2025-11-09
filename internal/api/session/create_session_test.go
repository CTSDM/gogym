package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerCreateSession(t *testing.T) {
	testCases := []struct {
		testName        string
		statusCode      int
		errMsg          []string
		userNotFound    bool
		missingContext  bool
		hasJSON         bool
		hasEmptyJSON    bool
		name            string
		date            string
		startTimestamp  int64
		durationMinutes int
	}{
		{
			testName:       "user not found in context",
			missingContext: true,
			statusCode:     http.StatusInternalServerError,
			errMsg:         []string{"Could not find user id in request context"},
		},
		{
			testName:   "no JSON sent",
			statusCode: http.StatusBadRequest,
			errMsg:     []string{"invalid payload"},
		},
		{
			testName:     "empty JSON",
			statusCode:   http.StatusCreated,
			hasJSON:      true,
			hasEmptyJSON: true,
		},
		{
			testName:   "JSON with valid name",
			name:       "name",
			statusCode: http.StatusCreated,
			hasJSON:    true,
		},
		{
			testName:   "JSON with valid date",
			date:       "2025-10-10",
			statusCode: http.StatusCreated,
			hasJSON:    true,
		},
		{
			testName:   "JSON with invalid date",
			date:       "name",
			statusCode: http.StatusBadRequest,
			hasJSON:    true,
			errMsg:     []string{"invalid date"},
		},
		{
			testName:       "JSON with invalid time start",
			startTimestamp: -10,
			statusCode:     http.StatusBadRequest,
			hasJSON:        true,
			errMsg:         []string{"invalid start_timestamp"},
		},
		{
			testName:        "JSON with invalid duration",
			durationMinutes: -3,
			statusCode:      http.StatusBadRequest,
			hasJSON:         true,
			errMsg:          []string{"invalid duration_minutes"},
		},
		{
			testName:        "json filled with valid values",
			name:            "morning workout",
			date:            "2025-11-04",
			startTimestamp:  1762294649,
			durationMinutes: 120,
			statusCode:      http.StatusCreated,
			hasJSON:         true,
		},
		{
			testName:     "user in context but not in the database",
			statusCode:   http.StatusUnauthorized,
			userNotFound: true,
		},
	}

	testutil.Cleanup(dbPool, "users")
	db := database.New(dbPool)
	userID := testutil.CreateUserDBTestHelper(t, db, "usertest", "passwordtest", false).ID

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Set up the response recorder and the request
			reader := &bytes.Reader{}
			if tc.hasEmptyJSON {
				reader = bytes.NewReader([]byte("{}"))
			} else if tc.hasJSON {
				reqParams := sessionReq{}
				if tc.name != "" {
					reqParams.Name = tc.name
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
			}

			if tc.userNotFound {
				// generate another uuid that is not on the database
				userID = uuid.New()
			}

			ctx := middleware.ContextWithUser(context.Background(), userID)
			if tc.missingContext {
				ctx = context.Background()
			}
			req, err := http.NewRequestWithContext(ctx, "POST", "/test", reader)
			require.NoError(t, err, "unexpected error while building the request")
			rr := httptest.NewRecorder()

			// Call the function
			HandlerCreateSession(db).ServeHTTP(rr, req)
			if tc.statusCode != rr.Code {
				assert.Equal(t, tc.statusCode, rr.Code)
				t.Logf("Response body: %s", rr.Body.String())
				return
			}
			if tc.statusCode > 399 {
				for _, message := range tc.errMsg {
					assert.Contains(t, rr.Body.String(), message)
				}
				return
			} else if tc.statusCode == http.StatusCreated {
				var resParams sessionRes
				decoder := json.NewDecoder(rr.Body)
				require.NoError(t, decoder.Decode(&resParams))
				assert.NotEmpty(t, resParams.ID)
				if tc.name != "" {
					assert.Equal(t, tc.name, resParams.Name)
				}
				if tc.date != "" {
					assert.Equal(t, tc.date, resParams.Date)
				} else {
					assert.Equal(t, time.Now().Format(apiconstants.DATE_LAYOUT), resParams.Date)
				}
				if tc.startTimestamp != 0 {
					assert.Equal(t, tc.startTimestamp, resParams.StartTimestamp)
				}
				if tc.durationMinutes != 0 {
					assert.Equal(t, tc.durationMinutes, resParams.DurationMinutes)
				} else {
					assert.Equal(t, 0, resParams.DurationMinutes)
				}
			}
		})
	}
}

func TestValidateCreateSession(t *testing.T) {
	testCases := []struct {
		testName        string
		name            string
		date            string
		startTimestamp  int64
		durationMinutes int
		hasError        bool
		errMap          map[string]string
		populate        bool
	}{
		{
			testName: "empty structure should be ok",
			populate: true,
		},
		{
			testName: "valid name and date",
			name:     "name",
			date:     "2025-10-10",
		},
		{
			testName: "empty name",
		},
		{
			testName: "very large name",
			name:     string(make([]byte, apiconstants.MaxSessionNameLength+10)),
			errMap: map[string]string{
				"name": "invalid name",
			},
			hasError: true,
		},
		{
			testName: "valid name but wrong date",
			name:     "name",
			date:     "2025-31-01",
			errMap: map[string]string{
				"date": "invalid date",
			},
			hasError: true,
		},
		{
			testName: "empty date",
			name:     "name",
			date:     "",
		},
		{
			testName:        "invalid duration minutes > MaxInt16",
			durationMinutes: math.MaxInt16 + 1,
			errMap: map[string]string{
				"duration_minutes": "invalid duration_minutes",
			},
			hasError: true,
			populate: true,
		},
		{
			testName:        "max duration in minutes",
			durationMinutes: math.MaxInt16,
			populate:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			reqParams := sessionReq{
				Name:            tc.name,
				Date:            tc.date,
				StartTimestamp:  tc.startTimestamp,
				DurationMinutes: tc.durationMinutes,
			}
			if tc.populate {
				reqParams.populate()
			}
			problems := reqParams.Valid(context.Background())
			if tc.hasError {
				require.Greater(t, len(problems), 0)
				for key, value := range tc.errMap {
					got, ok := problems[key]
					if !ok {
						t.Errorf("key not found: %s", key)
					} else {
						assert.Contains(t, got, value)
					}
				}
			} else {
				require.Equal(t, 0, len(problems))
			}
		})
	}
}

func TestPopulateCreateSession(t *testing.T) {
	testCases := []struct {
		testName        string
		name            string
		date            string
		timeStart       int
		durationMinutes int
	}{
		{
			testName: "name and date should be filled",
		},
		{
			testName: "name should be filled",
			date:     "last year",
		},
		{
			testName: "date should be filled",
			name:     "morning",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			req := sessionReq{
				Name: tc.name,
				Date: tc.date,
			}
			req.populate()

			if tc.name == "" {
				assert.NotEqual(t, tc.name, req.Name, "name was not populated")
				// check that the format is correct
				_, err := time.Parse(apiconstants.DATE_TIME_LAYOUT, req.Name)
				assert.NoError(t, err,
					fmt.Sprintf("generated name does not follow the required format %s",
						apiconstants.DATE_TIME_LAYOUT))
			} else {
				assert.Equal(t, tc.name, req.Name, "name was populated")
			}

			if tc.date == "" {
				assert.NotEqual(t, tc.date, req.Date, "date was not populated")
				_, err := time.Parse(apiconstants.DATE_LAYOUT, req.Date)
				assert.NoError(t, err,
					fmt.Sprintf("generated name does not follow the required format %s",
						apiconstants.DATE_TIME_LAYOUT))
			} else {
				assert.Equal(t, tc.date, req.Date, "date was populated")
			}

			assert.Empty(t, tc.timeStart, "time start was populated")
			assert.Empty(t, tc.durationMinutes, "duration minutes was populated")
		})
	}
}
