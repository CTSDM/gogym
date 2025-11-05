package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCreateExercise(t *testing.T) {
	testCases := []struct {
		exerciseName string
		description  string
		hasError     bool
	}{
		{
			exerciseName: "valid name",
			description:  "valid description",
		},
		{
			exerciseName: "valid name without description",
		},
		{
			exerciseName: randomString(MaxExerciseLength),
			description:  randomString(MaxDescriptionLength),
		},
		{
			exerciseName: randomString(MaxExerciseLength + 1),
			description:  "description",
			hasError:     true,
		},
		{
			exerciseName: "name",
			description:  randomString(MaxDescriptionLength + 1),
			hasError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("name length %d, description length %d", len(tc.exerciseName), len(tc.description)), func(t *testing.T) {
			req := createExerciseReq{
				Name:        tc.exerciseName,
				Description: tc.description,
			}
			err := req.validate()
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandlerCreateExercise(t *testing.T) {
	apiState := NewState(database.New(dbPool), &auth.Config{})
	testCases := []struct {
		name         string
		exerciseName string
		description  string
		statusCode   int
		errMessage   string
		hasEmptyJSON bool
	}{
		{
			name:         "happy path",
			exerciseName: "Bench Press",
			description:  "Chest exercise",
			statusCode:   http.StatusCreated,
		},
		{
			name:         "happy path without description",
			exerciseName: "Squat",
			statusCode:   http.StatusCreated,
		},
		{
			name:         "happy path with empty name and description",
			exerciseName: "",
			description:  "",
			statusCode:   http.StatusCreated,
		},
		{
			name:         "name too long",
			exerciseName: randomString(MaxExerciseLength + 1),
			description:  "description",
			statusCode:   http.StatusBadRequest,
			errMessage:   "could not validate the name",
		},
		{
			name:         "description too long",
			exerciseName: "Valid Name",
			description:  randomString(MaxDescriptionLength + 1),
			statusCode:   http.StatusBadRequest,
			errMessage:   "could not validate the name",
		},
		{
			name:         "invalid JSON",
			hasEmptyJSON: true,
			statusCode:   http.StatusBadRequest,
			errMessage:   "could not parse JSON",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup("exercises")
			reader := &bytes.Reader{}
			if tc.hasEmptyJSON {
				reader = bytes.NewReader([]byte("{invalid json}"))
			} else {
				reqParams := createExerciseReq{
					Name:        tc.exerciseName,
					Description: tc.description,
				}
				body, err := json.Marshal(reqParams)
				require.NoError(t, err, "unexpected JSON marshal error")
				reader = bytes.NewReader(body)
			}

			req, err := http.NewRequest("POST", "/test", reader)
			require.NoError(t, err, "unexpected error while creating the request")
			rr := httptest.NewRecorder()

			apiState.HandlerCreateExercise(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("Status code do not match, want %d, got %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}
			if tc.statusCode > 399 {
				assert.Contains(t, rr.Body.String(), tc.errMessage)
				return
			} else {
				var resParams createExerciseRes
				decoder := json.NewDecoder(rr.Body)
				require.NoError(t, decoder.Decode(&resParams))
				assert.Equal(t, tc.exerciseName, resParams.Name)
				assert.Equal(t, tc.description, resParams.Description)
				_, err := apiState.db.GetExercise(context.Background(), resParams.ID)
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandlerGetExercises(t *testing.T) {
	apiState := NewState(database.New(dbPool), &auth.Config{})
	testCases := []struct {
		name           string
		setupExercises []struct {
			name        string
			description string
		}
		expectedCount int
		statusCode    int
	}{
		{
			name:          "no exercises",
			expectedCount: 0,
			statusCode:    http.StatusOK,
		},
		{
			name: "single exercise",
			setupExercises: []struct {
				name        string
				description string
			}{
				{name: "Bench Press", description: "Chest exercise"},
			},
			expectedCount: 1,
			statusCode:    http.StatusOK,
		},
		{
			name: "multiple exercises",
			setupExercises: []struct {
				name        string
				description string
			}{
				{name: "Bench Press", description: "Chest exercise"},
				{name: "Squat", description: "Leg exercise"},
				{name: "Deadlift", description: "Back exercise"},
			},
			expectedCount: 3,
			statusCode:    http.StatusOK,
		},
		{
			name: "exercises without descriptions",
			setupExercises: []struct {
				name        string
				description string
			}{
				{name: "Pull-ups", description: ""},
				{name: "Push-ups", description: ""},
			},
			expectedCount: 2,
			statusCode:    http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup("exercises")
			for _, ex := range tc.setupExercises {
				createExerciseDBTestHelper(t, apiState, ex.name)
			}

			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err, "unexpected error while creating the request")
			rr := httptest.NewRecorder()

			apiState.HandlerGetExercises(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("Status code do not match, want %d, got %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}

			var resParams exercisesRes
			decoder := json.NewDecoder(rr.Body)
			require.NoError(t, decoder.Decode(&resParams))
			assert.Equal(t, tc.expectedCount, len(resParams.Exercises))

			for i, ex := range tc.setupExercises {
				assert.Equal(t, ex.name, resParams.Exercises[i].Name)
				assert.NotZero(t, resParams.Exercises[i].ID)
			}
		})
	}
}

func TestHandlerGetExercise(t *testing.T) {
	apiState := NewState(database.New(dbPool), &auth.Config{})
	testCases := []struct {
		name       string
		setupName  string
		setupDesc  string
		exerciseID string
		statusCode int
		errMessage string
		skipSetup  bool
	}{
		{
			name:       "happy path",
			setupName:  "Bench Press",
			setupDesc:  "Chest exercise",
			statusCode: http.StatusOK,
		},
		{
			name:       "happy path without description",
			setupName:  "Squat",
			statusCode: http.StatusOK,
		},
		{
			name:       "exercise not found - invalid id",
			setupName:  "Deadlift",
			exerciseID: "99999",
			statusCode: http.StatusNotFound,
			errMessage: "exercise id not found",
		},
		{
			name:       "exercise not found - non-numeric id",
			setupName:  "Pull-ups",
			exerciseID: "abc",
			statusCode: http.StatusNotFound,
			errMessage: "exercise id not found",
		},
		{
			name:       "exercise not found - negative id",
			skipSetup:  true,
			exerciseID: "-1",
			statusCode: http.StatusNotFound,
			errMessage: "exercise id not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup("exercises")
			var exerciseID int32
			if !tc.skipSetup {
				if tc.setupDesc != "" {
					exerciseID = int32(createExerciseWithDescDBTestHelper(t, apiState, tc.setupName, tc.setupDesc))
				} else {
					exerciseID = int32(createExerciseDBTestHelper(t, apiState, tc.setupName))
				}
			}

			idParam := tc.exerciseID
			if idParam == "" {
				idParam = fmt.Sprintf("%d", exerciseID)
			}

			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err, "unexpected error while creating the request")
			req.SetPathValue("id", idParam)
			rr := httptest.NewRecorder()

			apiState.HandlerGetExercise(rr, req)
			if tc.statusCode != rr.Code {
				t.Logf("Status code do not match, want %d, got %d", tc.statusCode, rr.Code)
				t.Fatalf("Body response: %s", rr.Body.String())
			}

			if tc.statusCode > 399 {
				assert.Contains(t, rr.Body.String(), tc.errMessage)
				return
			}

			var resParams exerciseItem
			decoder := json.NewDecoder(rr.Body)
			require.NoError(t, decoder.Decode(&resParams))
			assert.Equal(t, tc.setupName, resParams.Name)
			assert.Equal(t, tc.setupDesc, resParams.Description)
			assert.Equal(t, exerciseID, resParams.ID)
		})
	}
}
