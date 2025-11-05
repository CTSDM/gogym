package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checkContextNext(t *testing.T, expectedUserID uuid.UUID) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserFromContext(r.Context())
		require.True(t, ok)
		assert.Equal(t, expectedUserID, userID)
		w.WriteHeader(http.StatusOK)
	}
}

func TestHandlerMiddlewareLogin(t *testing.T) {
	apiState := NewState(
		database.New(dbPool),
		&auth.Config{
			JWTsecret:            "somerandomsecret",
			RefreshTokenDuration: time.Hour,
			JWTDuration:          time.Minute,
		})

	testCases := []struct {
		name                  string
		hasHeaderJWT          bool
		hasValidJWT           bool
		hasHeaderRefreshToken bool
		hasValidRefreshToken  bool
		statusCode            int
		jwtString             string
		refreshTokenString    string
		errMessage            string
		receivesNewJWT        bool
	}{
		{
			name:                  "happy path: has header and valid jwt/refresh token",
			statusCode:            http.StatusOK,
			hasHeaderJWT:          true,
			hasValidJWT:           true,
			hasHeaderRefreshToken: true,
			hasValidRefreshToken:  true,
		},
		{
			name:                  "has a valid refresh token",
			statusCode:            http.StatusOK,
			hasHeaderRefreshToken: true,
			hasValidRefreshToken:  true,
			receivesNewJWT:        true,
		},
		{
			name:                  "has headers but they are invalid",
			statusCode:            http.StatusUnauthorized,
			hasHeaderJWT:          true,
			jwtString:             "jwt",
			hasHeaderRefreshToken: true,
			refreshTokenString:    "refresh",
			errMessage:            "Invalid JWT and/or refresh token",
		},
		{
			name:         "has a valid jwt header",
			statusCode:   http.StatusOK,
			hasHeaderJWT: true,
			hasValidJWT:  true,
		},
		{
			name:       "no headers",
			statusCode: http.StatusUnauthorized,
			errMessage: "JWT and refresh token not found in the headers",
		},
	}

	// populate the userDB with a single user
	reqBodyStruct := createUserRequest{
		Username: "username",
		Password: "password",
	}
	reqBody, err := json.Marshal(reqBodyStruct)
	require.NoError(t, err, "could not marshal the request body for the created user")
	reqBodyReader := bytes.NewReader(reqBody)
	reqHelper := httptest.NewRequestWithContext(context.Background(), "GET", "/test", reqBodyReader)
	reqHelper.Header.Set("Content-Type", "application/json")
	defer reqHelper.Body.Close()
	reqHelper.Body = &Repeat{reader: reqBodyReader, offset: 0}
	rrCreateUser := httptest.NewRecorder()
	apiState.HandlerCreateUser(rrCreateUser, reqHelper)
	require.Equal(t, http.StatusCreated, rrCreateUser.Code, "could not create the user")

	// login to get the jwt and the refresh token
	reqHelper.Body.(*Repeat).Reset()
	rrLogin := httptest.NewRecorder()
	apiState.HandlerLogin(rrLogin, reqHelper)
	require.Equal(t, http.StatusOK, rrLogin.Code, "could not log in")

	// decode the login user response
	var resBody loginRes
	decoder := json.NewDecoder(rrLogin.Body)
	require.NoError(t, decoder.Decode(&resBody), "could not decode the JSON response")
	userID, err := uuid.Parse(resBody.UserID)
	require.NoError(t, err, "the user id obtained from the login response is not valid")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", bytes.NewReader([]byte{}))
			rr := httptest.NewRecorder()

			// set up the headers
			if tc.hasValidJWT {
				tc.jwtString = resBody.Token
			}
			if tc.hasHeaderJWT {
				req.Header.Set("Auth", "Bearer "+tc.jwtString)
			}
			if tc.hasValidRefreshToken {
				tc.refreshTokenString = resBody.RefreshToken
			}
			if tc.hasHeaderRefreshToken {
				req.Header.Set("X-Refresh-Token", "Token "+tc.refreshTokenString)
			}

			handler := apiState.HandlerMiddlewareLogin(checkContextNext(t, userID))
			handler.ServeHTTP(rr, req)
			require.Equal(t, tc.statusCode, rr.Code)

			// check for error message
			if tc.errMessage != "" {
				var valsResponse errorResponse
				decoder := json.NewDecoder(rr.Body)
				decoder.DisallowUnknownFields()
				require.NoError(t, decoder.Decode(&valsResponse))
				assert.Equal(t, tc.errMessage, valsResponse.Error)
			}

			if tc.receivesNewJWT {
				// the old jwt should not be valid
				_, err := auth.ValidateJWT(tc.jwtString, apiState.authConfig.JWTsecret)
				require.Error(t, err)

				// the new jwt should exist and be valid
				gotJWT, err := auth.GetHeaderValueToken(rr.Result().Header, "Auth")
				require.NoError(t, err)
				_, err = auth.ValidateJWT(gotJWT, apiState.authConfig.JWTsecret)
				assert.NoError(t, err)
			}
		})
	}
}
