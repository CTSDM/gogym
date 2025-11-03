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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerLogin(t *testing.T) {
	apiState := NewState(database.New(dbPool))

	type testCase struct {
		name     string
		username string
		password string
		code     int
	}

	t.Run("user already on the database - test http codes", func(t *testing.T) {
		t.Cleanup(func() { cleanup("") })
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
		// Prepare request body
		reqBodyCreateUserStruct := createUserRequest{
			Username: username,
			Password: password,
		}
		reqBodyCreateUser, err := json.Marshal(reqBodyCreateUserStruct)
		require.NoError(t, err, "unexpected error while marshaling the request body")

		// Set up request
		reqBodyCreateUserReader := bytes.NewReader(reqBodyCreateUser)
		reqCreateUser, err := http.NewRequest("POST", "/test", reqBodyCreateUserReader)
		require.NoError(t, err, "unexpected error while creating the request object")
		reqCreateUser.Header.Set("Content-Type", "application/json")

		rrCreateUser := httptest.NewRecorder()
		apiState.HandlerCreateUser(rrCreateUser, reqCreateUser)
		require.Equal(t, 201, rrCreateUser.Code, "unexpected error while creating the user")

		for _, tc := range testCases {
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
			reqBodyReader := bytes.NewReader(reqBody)
			req, err := http.NewRequest("POST", "/test", reqBodyReader)
			require.NoError(t, err, "unexpected error while creating the request object")
			req.Header.Set("Content-Type", "application/json")

			rrLogin := httptest.NewRecorder()
			apiState.HandlerLogin(rrLogin, req)
			require.Equal(t, tc.code, rrLogin.Code, "mismatch in http status code")

			if tc.hasInvalidPayload == true {
				continue
			}

			// Check payload
			if tc.hasErrInPayload == true {
				var errRes errorResponse
				decoder := json.NewDecoder(rrLogin.Body)
				decoder.DisallowUnknownFields()
				assert.NoError(t, decoder.Decode(&errRes), "failed to decode the res login json")
			} else {
				var resBody loginRes
				decoder := json.NewDecoder(rrLogin.Body)
				decoder.DisallowUnknownFields()
				assert.NoError(t, decoder.Decode(&resBody), "failed to decode the res login json")
			}
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

		t.Cleanup(func() {
			cleanup("users")
			cleanup("refresh_tokens")
		})
		require.NoError(t, cleanup("refresh_tokens"), "could not clean the database")
		require.NoError(t, cleanup("users"), "could not clean the database")

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Prepare request body
				reqBodyStruct := loginReq{
					Username: tc.username,
					Password: tc.password,
				}
				reqBody, err := json.Marshal(reqBodyStruct)
				require.NoError(t, err, "unexpected error while marshaling the request body")

				// Set up request
				reqBodyReader := bytes.NewReader(reqBody)
				req, err := http.NewRequest("POST", "/test", reqBodyReader)
				require.NoError(t, err, "unexpected error while creating the request object")
				req.Header.Set("Content-Type", "application/json")
				req.Body = &Repeat{reader: reqBodyReader, offset: 0} // allows to use same request multiple times

				// Set up recorders
				rrCreateUser := httptest.NewRecorder()
				rrLogin := httptest.NewRecorder()

				// Call first the create user handler
				apiState.HandlerCreateUser(rrCreateUser, req)
				require.Equal(t, 201, rrCreateUser.Code, "unexpected error while creating the user")

				// Handler Login
				req.Body.(*Repeat).Reset()
				apiState.HandlerLogin(rrLogin, req)
				require.Equal(t, tc.code, rrLogin.Code, "mismatch in http status code")

				if tc.code > 200 {
					return
				}

				// Unmarshal the response body
				var resBody loginRes
				decoder := json.NewDecoder(rrLogin.Body)
				require.NoError(t, decoder.Decode(&resBody), "could not decode the JSON response")

				// call the database to get the user and refresh token information
				user, err := apiState.db.GetUserByUsername(context.Background(), tc.username)
				require.NoError(t, err, "unexpected error while retrieving the user from the database")
				refreshToken, err := apiState.db.GetRefreshTokenByUserID(
					context.Background(),
					pgtype.UUID{Bytes: user.ID.Bytes, Valid: true},
				)
				require.NoError(t, err, "unexpected error while retrieving the refresh token from the database")

				// Checks
				assert.Equal(t, user.ID.String(), resBody.UserID, "mismatch in user id")
				assert.Equal(t, user.Username, resBody.Username, "mismatch in username")
				assert.Equal(t, refreshToken.Token, resBody.RefreshToken, "mismatch in refresh token")
				id, err := auth.ValidateJWT(resBody.Token, "secret")
				assert.NoError(t, err, "invalid JWT in the response body")
				assert.Equal(t, user.ID.String(), id, "id obtained from JWT does not match the user id")
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
	}{
		{
			name:     "invalid username",
			username: "",
			password: "somepassword",
			hasError: true,
		},
		{
			name:     "invalid password",
			username: "someusername",
			password: "",
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
			loginReqParams := loginReq{
				Username: tc.username,
				Password: tc.password,
			}
			err := validateLogin(loginReqParams)
			if tc.hasError == true {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
