package user

import (
	"net/http"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type getUsersResponse struct {
	Users []User `json:"Users"`
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Country   string `json:"country"`
	CreatedAt string `json:"created_at"`
	Birthday  string `json:"birthday,omitempty"`
}

func HandlerGetUsers(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := db.GetUsers(r.Context())
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "could not retrieve users from the database", err)
			return
		}

		responseVals := getUsersResponse{Users: make([]User, len(users))}
		for i, user := range users {
			responseVals.Users[i].ID = user.ID.String()
			responseVals.Users[i].Username = user.Username
			responseVals.Users[i].Country = user.Country.String
			responseVals.Users[i].CreatedAt = user.CreatedAt.Time.Format(apiconstants.DATE_LAYOUT)
			if user.Birthday.Valid {
				responseVals.Users[i].Birthday = user.Birthday.Time.Format(apiconstants.DATE_LAYOUT)
			}
		}
		util.RespondWithJSON(w, http.StatusOK, responseVals)
	}
}

func HandlerGetUser(db *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// user id must be a valid uuid
		userID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			util.RespondWithError(w, http.StatusNotFound, "user not found", err)
			return
		}

		// get the userDB from the database
		userDB, err := db.GetUser(r.Context(), userID)
		if err == pgx.ErrNoRows {
			util.RespondWithError(w, http.StatusNotFound, "user not found", err)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError,
				"something went wrong while fetching the user from the database", err)
			return
		}

		// Create response
		user := User{
			ID:        userDB.ID.String(),
			Username:  userDB.Username,
			Country:   userDB.Country.String,
			CreatedAt: userDB.CreatedAt.Time.Format(apiconstants.DATE_LAYOUT),
		}
		// Only add the birthday if it has been defined
		if userDB.Birthday.Valid {
			user.Birthday = userDB.Birthday.Time.Format(apiconstants.DATE_LAYOUT)
		}

		util.RespondWithJSON(w, http.StatusOK, user)
	}
}
