package middleware

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

func TestAdminOnly(t *testing.T) {
	testCases := []struct {
		name       string
		isAdmin    bool
		deleteUser bool
		statusCode int
		errMessage string
	}{
		{
			name:       "admin user can access",
			isAdmin:    true,
			statusCode: http.StatusOK,
		},
		{
			name:       "non-admin user is forbidden",
			isAdmin:    false,
			statusCode: http.StatusForbidden,
			errMessage: "admin access required",
		},
		{
			name:       "user not found in database",
			deleteUser: true,
			statusCode: http.StatusUnauthorized,
			errMessage: "could not retrieve user from the database",
		},
	}

	db := database.New(dbPool)
	authConfig := auth.Config{
		JWTsecret:            "somerandomsecret",
		RefreshTokenDuration: time.Hour,
		JWTDuration:          time.Minute,
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, testutil.Cleanup(dbPool, "users"))
			require.NoError(t, testutil.Cleanup(dbPool, "refresh_tokens"))

			username := "usertest"
			password := "passwordtest"
			var user database.User
			var err error
			if tc.isAdmin {
				user, err = db.CreateAdmin(context.Background(), database.CreateAdminParams{
					Username:       username,
					HashedPassword: password,
				})
				require.NoError(t, err)
				t.Logf("%+v", user)
			} else {
				user, err = db.CreateUser(context.Background(), database.CreateUserParams{
					Username:       username,
					HashedPassword: password,
				})
				require.NoError(t, err)
			}

			userID := user.ID.Bytes
			token, _ := testutil.CreateTokensDBHelperTest(t, db, authConfig, userID)

			if tc.deleteUser {
				_, err = db.DeleteUser(context.Background(), user.ID)
				require.NoError(t, err)
			}

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Auth", "Bearer "+token)
			rr := httptest.NewRecorder()

			dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Admin function expects, on the happy path, to have a user on the context
			ctx := ContextWithUser(req.Context(), userID)
			req = req.WithContext(ctx)

			handler := AdminOnly(db, authConfig)(dummyHandler)
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.statusCode, rr.Code)

			if tc.errMessage != "" {
				var errRes util.ErrorResponse
				decoder := json.NewDecoder(rr.Body)
				decoder.DisallowUnknownFields()
				require.NoError(t, decoder.Decode(&errRes))
				assert.Equal(t, tc.errMessage, errRes.Error)
			}

		})
	}
}

func TestHandlerMiddlewareAuthentication(t *testing.T) {
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

	require.NoError(t, testutil.Cleanup(dbPool, "users"), "failed to clean the database")

	db := database.New(dbPool)
	authConfig := auth.Config{
		JWTsecret:            "somerandomsecret",
		RefreshTokenDuration: time.Hour,
		JWTDuration:          time.Minute,
	}
	userID := testutil.CreateUserDBTestHelper(t, db, "usertest", "passwordtest", false).ID.Bytes
	token, refreshToken := testutil.CreateTokensDBHelperTest(t, db, authConfig, userID)

	db.CreateRefreshToken(context.Background(),
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

			handler := Authentication(db, authConfig)(checkContextNext(t, userID))
			handler.ServeHTTP(rr, req)
			require.Equal(t, tc.statusCode, rr.Code)

			// check for error message
			if tc.errMessage != "" {
				var valsResponse util.ErrorResponse
				decoder := json.NewDecoder(rr.Body)
				decoder.DisallowUnknownFields()
				require.NoError(t, decoder.Decode(&valsResponse))
				assert.Equal(t, tc.errMessage, valsResponse.Error)
			}

			if tc.receivesNewJWT {
				// the old jwt should not be valid
				_, err := auth.ValidateJWT(tc.jwtString, authConfig.JWTsecret)
				require.Error(t, err)

				// the new jwt should exist and be valid
				gotJWT, err := auth.GetHeaderValueToken(rr.Result().Header, "Auth")
				require.NoError(t, err)
				_, err = auth.ValidateJWT(gotJWT, authConfig.JWTsecret)
				assert.NoError(t, err)
			}
		})
	}
}
