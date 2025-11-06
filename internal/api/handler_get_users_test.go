package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHandlerGetUser(t *testing.T) {
	apiState := NewState(database.New(dbPool), &auth.Config{})
	username := "username"
	password := "password"

	testCases := []struct {
		name       string
		userID     string
		statusCode int
		validateFn func(*testing.T, *httptest.ResponseRecorder, database.User)
	}{
		{
			name:       "happy path",
			statusCode: http.StatusOK,
			validateFn: func(t *testing.T, rr *httptest.ResponseRecorder, user database.User) {
				var response User
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err, "failed to unmarshal response")
				require.Equal(t, user.ID.String(), response.ID)
				require.Equal(t, user.Username, response.Username)
				require.Empty(t, response.Country)
				require.NotEmpty(t, response.CreatedAt)
				require.NotEmpty(t, response.Birthday)
			},
		},
		{
			name:       "invalid user id",
			userID:     "test",
			statusCode: http.StatusNotFound,
			validateFn: func(t *testing.T, rr *httptest.ResponseRecorder, user database.User) {},
		},
		{
			name:       "user id not in the database",
			userID:     uuid.NewString(),
			statusCode: http.StatusNotFound,
			validateFn: func(t *testing.T, rr *httptest.ResponseRecorder, user database.User) {},
		},
	}

	user := createUserDBTestHelper(t, apiState, username, password, true)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/test", bytes.NewReader([]byte{}))
			require.NoError(t, err, "unexpected error while setting up the request")
			req.SetPathValue("id", user.ID.String())
			if tc.userID != "" {
				req.SetPathValue("id", tc.userID)
			}

			rr := httptest.NewRecorder()
			apiState.HandlerGetUser(rr, req)
			require.Equal(t, tc.statusCode, rr.Code, "wrong status code")
			tc.validateFn(t, rr, user)
		})
	}

	t.Run("user without birthday", func(t *testing.T) {
		userNoBirthday := createUserDBTestHelper(t, apiState, "user_no_birthday", password, false)

		req, err := http.NewRequest("GET", "/test", bytes.NewReader([]byte{}))
		require.NoError(t, err, "unexpected error while setting up the request")
		req.SetPathValue("id", userNoBirthday.ID.String())

		rr := httptest.NewRecorder()
		apiState.HandlerGetUser(rr, req)
		require.Equal(t, http.StatusOK, rr.Code, "wrong status code")

		var response User
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")
		require.Equal(t, userNoBirthday.ID.String(), response.ID)
		require.Equal(t, userNoBirthday.Username, response.Username)
		require.Empty(t, response.Birthday)
	})
}

func TestHandlerGetUsers(t *testing.T) {
	apiState := NewState(database.New(dbPool), &auth.Config{})

	t.Run("empty database", func(t *testing.T) {
		require.NoError(t, cleanup("users"))

		req, err := http.NewRequest("GET", "/test", bytes.NewReader([]byte{}))
		require.NoError(t, err, "unexpected error while setting up the request")

		rr := httptest.NewRecorder()
		apiState.HandlerGetUsers(rr, req)
		require.Equal(t, http.StatusOK, rr.Code, "wrong status code")

		var response getUsersResponse
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")
		require.Len(t, response.Users, 0, "expected empty users list")
	})

	t.Run("multiple users", func(t *testing.T) {
		require.NoError(t, cleanup("users"))

		user1 := createUserDBTestHelper(t, apiState, "user1", "password1", true)
		user2 := createUserDBTestHelper(t, apiState, "user2", "password2", false)
		user3 := createUserDBTestHelper(t, apiState, "user3", "password3", true)

		req, err := http.NewRequest("GET", "/test", bytes.NewReader([]byte{}))
		require.NoError(t, err, "unexpected error while setting up the request")

		rr := httptest.NewRecorder()
		apiState.HandlerGetUsers(rr, req)
		require.Equal(t, http.StatusOK, rr.Code, "wrong status code")

		var response getUsersResponse
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")
		require.GreaterOrEqual(t, len(response.Users), 3, "expected at least 3 users")

		userMap := make(map[string]User)
		for _, u := range response.Users {
			userMap[u.ID] = u
		}

		require.Contains(t, userMap, user1.ID.String())
		require.Equal(t, user1.Username, userMap[user1.ID.String()].Username)
		require.NotEmpty(t, userMap[user1.ID.String()].Birthday)

		require.Contains(t, userMap, user2.ID.String())
		require.Equal(t, user2.Username, userMap[user2.ID.String()].Username)
		require.Empty(t, userMap[user2.ID.String()].Birthday)

		require.Contains(t, userMap, user3.ID.String())
		require.Equal(t, user3.Username, userMap[user3.ID.String()].Username)
		require.NotEmpty(t, userMap[user3.ID.String()].Birthday)
	})

	t.Run("response structure validation", func(t *testing.T) {
		require.NoError(t, cleanup("users"))

		user := createUserDBTestHelper(t, apiState, "user_structure_test", "password", true)

		req, err := http.NewRequest("GET", "/test", bytes.NewReader([]byte{}))
		require.NoError(t, err, "unexpected error while setting up the request")

		rr := httptest.NewRecorder()
		apiState.HandlerGetUsers(rr, req)
		require.Equal(t, http.StatusOK, rr.Code, "wrong status code")

		var response getUsersResponse
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")

		found := false
		for _, u := range response.Users {
			if u.ID == user.ID.String() {
				found = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.Username)
				require.NotEmpty(t, u.CreatedAt)
				require.Empty(t, u.Country)
				if user.Birthday.Valid {
					require.NotEmpty(t, u.Birthday)
				}
				break
			}
		}
		require.True(t, found, "created user not found in response")
	})
}
