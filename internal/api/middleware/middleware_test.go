package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CTSDM/gogym/internal/api/testutil"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	authConfig := &auth.Config{
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

			userID := user.ID
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

			handler := AdminOnly(db, logger)(dummyHandler)
			RequestID(handler).ServeHTTP(rr, req)

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
	require.NoError(t, testutil.Cleanup(dbPool, "refresh_tokens"), "failed to clean the database")

	db := database.New(dbPool)
	authConfig := &auth.Config{
		JWTsecret:            "somerandomsecret",
		RefreshTokenDuration: time.Hour,
		JWTDuration:          time.Minute,
	}
	userID := testutil.CreateUserDBTestHelper(t, db, "usertest", "passwordtest", false).ID
	token, refreshToken := testutil.CreateTokensDBHelperTest(t, db, authConfig, userID)

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

			handler := Authentication(db, authConfig, logger)(checkContextNext(t, userID))
			RequestID(handler).ServeHTTP(rr, req)
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

func TestOwnershipInt64(t *testing.T) {
	user1ID := uuid.New()
	user2ID := uuid.New()

	testCases := []struct {
		name       string
		statusCode int
		errMessage string
		pathKey    string
		pathValue  string
		userID     uuid.UUID
		ownerFn    func(ctx context.Context, id int64) (uuid.UUID, error)
	}{
		{
			name:       "happy path: user is owner",
			statusCode: http.StatusOK,
			pathKey:    "id",
			pathValue:  "123",
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id int64) (uuid.UUID, error) {
				return user1ID, nil
			},
		},
		{
			name:       "user is not owner",
			statusCode: http.StatusForbidden,
			errMessage: "user is not owner",
			pathKey:    "id",
			pathValue:  "123",
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id int64) (uuid.UUID, error) {
				return user2ID, nil
			},
		},
		{
			name:       "invalid path value format",
			statusCode: http.StatusBadRequest,
			errMessage: "invalid id format",
			pathKey:    "id",
			pathValue:  "invalid",
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id int64) (uuid.UUID, error) {
				return user1ID, nil
			},
		},
		{
			name:       "resource not found",
			statusCode: http.StatusNotFound,
			errMessage: "not found",
			pathKey:    "id",
			pathValue:  "999999",
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id int64) (uuid.UUID, error) {
				return uuid.UUID{}, pgx.ErrNoRows
			},
		},
		{
			name:       "database error",
			statusCode: http.StatusInternalServerError,
			errMessage: "something went wrong",
			pathKey:    "id",
			pathValue:  "123",
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id int64) (uuid.UUID, error) {
				return uuid.UUID{}, fmt.Errorf("database error")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/test/"+tc.pathValue, nil)
			req.SetPathValue(tc.pathKey, tc.pathValue)

			ctx := ContextWithUser(req.Context(), tc.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resourceID, ok := ResourceIDFromContext(r.Context())
				require.True(t, ok)
				require.Equal(t, int64(123), resourceID.(int64))
				w.WriteHeader(http.StatusOK)
			})

			handler := Ownership(tc.pathKey, tc.ownerFn, logger)(dummyHandler)
			RequestID(handler).ServeHTTP(rr, req)
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

func TestOwnershipUUID(t *testing.T) {
	user1ID := uuid.New()
	user2ID := uuid.New()
	resourceID := uuid.New()

	testCases := []struct {
		name       string
		statusCode int
		errMessage string
		pathKey    string
		pathValue  string
		userID     uuid.UUID
		ownerFn    func(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	}{
		{
			name:       "happy path: user is owner",
			statusCode: http.StatusOK,
			pathKey:    "sessionID",
			pathValue:  resourceID.String(),
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return user1ID, nil
			},
		},
		{
			name:       "user is not owner",
			statusCode: http.StatusForbidden,
			errMessage: "user is not owner",
			pathKey:    "sessionID",
			pathValue:  resourceID.String(),
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return user2ID, nil
			},
		},
		{
			name:       "invalid uuid format",
			statusCode: http.StatusBadRequest,
			errMessage: "invalid sessionID format",
			pathKey:    "sessionID",
			pathValue:  "invalid-uuid",
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return user1ID, nil
			},
		},
		{
			name:       "resource not found",
			statusCode: http.StatusNotFound,
			errMessage: "not found",
			pathKey:    "sessionID",
			pathValue:  uuid.New().String(),
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.UUID{}, pgx.ErrNoRows
			},
		},
		{
			name:       "database error",
			statusCode: http.StatusInternalServerError,
			errMessage: "something went wrong",
			pathKey:    "sessionID",
			pathValue:  resourceID.String(),
			userID:     user1ID,
			ownerFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.UUID{}, fmt.Errorf("database error")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test/"+tc.pathValue, nil)
			req.SetPathValue(tc.pathKey, tc.pathValue)

			ctx := ContextWithUser(req.Context(), tc.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rid, ok := ResourceIDFromContext(r.Context())
				require.True(t, ok)
				require.Equal(t, resourceID, rid.(uuid.UUID))
				w.WriteHeader(http.StatusOK)
			})

			handler := Ownership(tc.pathKey, tc.ownerFn, logger)(dummyHandler)
			RequestID(handler).ServeHTTP(rr, req)
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
