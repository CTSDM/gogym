package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	testCases := []struct {
		testname       string
		username       string
		password       string
		birthday       string
		country        string
		invalidPayload bool
		statusCode     int
	}{
		{
			testname:   "adding new user",
			username:   "user",
			password:   "passwordtest",
			birthday:   "2000-07-28",
			statusCode: 201,
			country:    "Spain",
		},
		{
			testname:   "adding duplicated user",
			username:   "user",
			password:   "passwordtest",
			statusCode: 409,
		},
		{
			testname:   "empty username",
			username:   "",
			password:   "passwordtest",
			statusCode: 400,
		},
		{
			testname:   "country field value is too short",
			username:   "anotheruser",
			password:   "passwordtest",
			country:    "1",
			statusCode: 400,
		},
		{
			testname:   "invalid date value",
			username:   "anotheruser",
			password:   "passwordtest",
			birthday:   "2020-01-01",
			statusCode: 400,
		},
		{
			testname:   "adding another use",
			username:   "철수",
			password:   "passwordtest",
			birthday:   "2000-07-28",
			statusCode: 201,
			country:    "South Korea",
		},
		{
			testname:       "invalid payload",
			statusCode:     400,
			invalidPayload: true,
		},
	}

	done := make(chan struct{})
	timeoutDuration := 10 * time.Second
	ticker := time.NewTicker(timeoutDuration)
	testutil.Cleanup(dbPool, "users")
	db := database.New(dbPool)

	go func() {
		defer close(done)
		for _, tc := range testCases {
			t.Run(tc.testname, func(t *testing.T) {
				// Create the request body
				reqBodyStruct := createUserRequest{
					Username: tc.username,
					Password: tc.password,
					Birthday: tc.birthday,
					Country:  tc.country,
				}
				reqBody, err := json.Marshal(reqBodyStruct)
				require.NoError(t, err, "could not marshal the request body")

				// Setup request and response recorder
				if tc.invalidPayload == true {
					reqBody = []byte("invalid")
				}
				req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", bytes.NewReader(reqBody))
				req.Header.Set("Content-Type", "application/json")
				defer req.Body.Close()
				rr := httptest.NewRecorder()

				// Call the handler to test
				handler := HandlerCreateUser(db, logger)
				middleware.RequestID(handler).ServeHTTP(rr, req)

				// Checks the response
				assert.Equal(t, tc.statusCode, rr.Code, "mismatch in http status")
				if tc.statusCode == 201 {
					// Unmarshal the body and get the id
					var resBody User
					decoder := json.NewDecoder(rr.Body)
					require.NoError(t, decoder.Decode(&resBody), "could not decode the JSON response")
					userID, err := uuid.Parse(resBody.ID)
					require.NoError(t, err, "could not parse the user id into UUID")

					// Obtain the user information from the database
					user, err := db.GetUser(context.Background(), userID)
					assert.NotErrorIs(t, pgx.ErrNoRows, err, "user not found on the database")
					require.NoError(t, err, "unexpected error when querying the database")

					// Validate sent response is the same as the one stored in the db
					assert.Equal(t, user.Username, resBody.Username)
					assert.Equal(t, user.Birthday.Time.Format(apiconstants.DATE_LAYOUT), resBody.Birthday)
					assert.Equal(t, user.Country.String, resBody.Country)
					assert.Equal(t, user.CreatedAt.Time.String(), resBody.CreatedAt)
				}
			})
		}

	}()

	select {
	case <-ticker.C:
		t.Fatalf("tests timed out after %.3f seconds", timeoutDuration.Seconds())
	case <-done:
		// All test have finished
	}
}
