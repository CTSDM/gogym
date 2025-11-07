package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerLogin(t *testing.T) {
	type testCase struct {
		name     string
		username string
		password string
		code     int
	}

	db := database.New(dbPool)
	authConfig := &auth.Config{
		JWTsecret:            "testSecret",
		JWTDuration:          time.Minute,
		RefreshTokenDuration: time.Hour,
	}

	t.Run("user already on the database - test http codes", func(t *testing.T) {
		username := "username"
		password := "password"
		testCases := []struct {
			testCase
			hasInvalidPayload bool
			hasErrInPayload   bool
		}{
			{
				testCase: testCase{
					name:     "correct log in",
					username: username,
					password: password,
					code:     200,
				},
			},
			{
				testCase: testCase{
					name:     "username does not exist",
					username: "userdoesnotexist",
					password: password,
					code:     200,
				},
				hasErrInPayload: true,
			},
			{
				testCase: testCase{
					name:     "invalid password",
					username: username,
					password: "invalidpassword",
					code:     200,
				},
				hasErrInPayload: true,
			},
			{
				testCase: testCase{
					code: 400,
				},
				hasInvalidPayload: true,
			},
		}

		require.NoError(t, testutil.Cleanup(dbPool, "users"), "failed to clean the database")
		testutil.CreateUserDBTestHelper(t, db, username, password, false)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {

				// set up request for each login
				reqBodyStruct := loginReq{
					Username: tc.username,
					Password: tc.password,
				}
				reqBody, err := json.Marshal(reqBodyStruct)
				require.NoError(t, err, "unexpected error while marshaling the request body")
				if tc.hasInvalidPayload == true {
					reqBody, _ = json.Marshal(struct{}{})
				}

				// Set up request for the login
				req, err := http.NewRequest("POST", "/test", bytes.NewReader(reqBody))
				require.NoError(t, err, "unexpected error while creating the request object")
				req.Header.Set("Content-Type", "application/json")

				rrLogin := httptest.NewRecorder()
				HandlerLogin(db, authConfig).ServeHTTP(rrLogin, req)
				require.Equal(t, tc.code, rrLogin.Code, "mismatch in http status code")

				if tc.hasInvalidPayload == true {
					return
				}

				// Check payload
				if tc.hasErrInPayload == true {
					var errRes util.ErrorResponse
					decoder := json.NewDecoder(rrLogin.Body)
					decoder.DisallowUnknownFields()
					assert.NoError(t, decoder.Decode(&errRes), "failed to decode the res login json")
				} else {
					var resBody loginRes
					decoder := json.NewDecoder(rrLogin.Body)
					decoder.DisallowUnknownFields()
					assert.NoError(t, decoder.Decode(&resBody), "failed to decode the res login json")
				}
			})
		}
	})

	t.Run("happy path", func(t *testing.T) {
		testCases := []testCase{
			{
				name:     "correct log in",
				username: "username",
				password: "password",
				code:     200,
			},
			{
				name:     "another correct log in",
				username: "usernameeee",
				password: "passwordddd",
				code:     200,
			},
		}

		require.NoError(t, testutil.Cleanup(dbPool, "users"), "failed to clean the users table")
		require.NoError(t, testutil.Cleanup(dbPool, "refresh_tokens"), "failed to clean refresh tokens table")

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				userIDDB := testutil.CreateUserDBTestHelper(t, db, tc.username, tc.password, false).ID
				userID := userIDDB.Bytes
				userIDString := userIDDB.String()
				testutil.CreateTokensDBHelperTest(t, db, authConfig, userID)
				// Prepare request body
				reqBodyStruct := loginReq{
					Username: tc.username,
					Password: tc.password,
				}
				reqBody, err := json.Marshal(reqBodyStruct)
				require.NoError(t, err, "unexpected error while marshaling the request body")

				// Set up request
				req, err := http.NewRequest("POST", "/test", bytes.NewReader(reqBody))
				require.NoError(t, err, "unexpected error while creating the request object")
				req.Header.Set("Content-Type", "application/json")

				// Set up recorders
				rr := httptest.NewRecorder()

				// Handler Login
				HandlerLogin(db, authConfig).ServeHTTP(rr, req)
				require.Equal(t, tc.code, rr.Code, "mismatch in http status code")

				if tc.code > 200 {
					return
				}

				// Unmarshal the response body
				var resBody loginRes
				decoder := json.NewDecoder(rr.Body)
				require.NoError(t, decoder.Decode(&resBody), "could not decode the JSON response")

				// call the database to get the user and refresh token information
				gotRefreshToken, err := db.GetRefreshTokenByUserID(
					context.Background(),
					pgtype.UUID{Bytes: userID, Valid: true},
				)
				require.NoError(t, err, "unexpected error while retrieving the refresh token from the database")

				// Checks
				assert.Equal(t, userIDString, resBody.UserID, "mismatch in user id")
				assert.Equal(t, tc.username, resBody.Username, "mismatch in username")
				assert.Equal(t, gotRefreshToken.Token, resBody.RefreshToken, "mismatch in refresh token")
				id, err := auth.ValidateJWT(resBody.Token, authConfig.JWTsecret)
				assert.NoError(t, err, "invalid JWT in the response body")
				assert.Equal(t, userIDString, id, "id obtained from JWT does not match the user id")
			})
		}
	})

}

func TestValidateLogin(t *testing.T) {
	testCases := []struct {
		name     string
		username string
		password string
		hasError bool
		errMap   map[string]string
	}{
		{
			name:     "invalid username",
			username: "",
			password: "somepassword",
			errMap: map[string]string{
				"username": "invalid username",
			},
			hasError: true,
		},
		{
			name:     "invalid password",
			username: "someusername",
			password: "",
			errMap: map[string]string{
				"password": "invalid password",
			},
			hasError: true,
		},
		{
			name:     "valid input",
			username: "someusername",
			password: "somepassword",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqParams := loginReq{
				Username: tc.username,
				Password: tc.password,
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
