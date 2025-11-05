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
	"github.com/jackc/pgx/v5/pgtype"
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

	require.NoError(t, cleanup("users"), "failed to clean the database")
	userID := createUserDBTestHelper(t, apiState, "usertest", "passwordtest")
	token, refreshToken := createTokensDBHelperTest(t, userID, apiState)

	apiState.db.CreateRefreshToken(context.Background(),
		database.CreateRefreshTokenParams{
			Token:     refreshToken,
			UserID:    pgtype.UUID{Bytes: userID, Valid: true},
			ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(time.Hour), Valid: true},
		})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", bytes.NewReader([]byte{}))
			rr := httptest.NewRecorder()

			// set up the headers
			if tc.hasValidJWT {
				tc.jwtString = token
			}
			if tc.hasHeaderJWT {
				req.Header.Set("Auth", "Bearer "+tc.jwtString)
			}
			if tc.hasValidRefreshToken {
				tc.refreshTokenString = refreshToken
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
