package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (s *State) HandlerGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.GetUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not retrieve users from the database", err)
		return
	}

	responseVals := getUsersResponse{Users: make([]User, len(users))}
	for i, user := range users {
		responseVals.Users[i].ID = user.ID.String()
		responseVals.Users[i].Username = user.Username
		responseVals.Users[i].Country = user.Country.String
		responseVals.Users[i].CreatedAt = user.CreatedAt.Time.Format(DATE_LAYOUT)
		if user.Birthday.Valid {
			responseVals.Users[i].Birthday = user.Birthday.Time.Format(DATE_LAYOUT)
		}
	}
	respondWithJSON(w, http.StatusOK, responseVals)
}

func (s *State) HandlerGetUser(w http.ResponseWriter, r *http.Request) {
	// user id must be a valid uuid
	userID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user not found", err)
		return
	}

	// get the userDB from the database
	userDB, err := s.db.GetUser(r.Context(), pgtype.UUID{Bytes: userID, Valid: true})
	if err == pgx.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "user not found", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			"something went wrong while fetching the user from the database", err)
		return
	}

	// Create response
	user := User{
		ID:        userDB.ID.String(),
		Username:  userDB.Username,
		Country:   userDB.Country.String,
		CreatedAt: userDB.CreatedAt.Time.Format(DATE_LAYOUT),
	}
	// Only add the birthday if it has been defined
	if userDB.Birthday.Valid {
		user.Birthday = userDB.Birthday.Time.Format(DATE_LAYOUT)
	}

	respondWithJSON(w, http.StatusOK, user)
}
