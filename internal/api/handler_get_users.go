package api

import (
	"net/http"
)

type getUsersResponse struct {
	Users []User `json:"Users"`
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Country   string `json:"country"`
	CreatedAt string `json:"created_at"`
	Birthday  string `json:"birthday"`
}

func (s *State) HandlerGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.GetUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not retrieve users from the database", err)
		return
	}

	responseVals := getUsersResponse{Users: make([]User, len(users))}
	for i := range users {
		responseVals.Users[i].ID = users[i].ID.String()
		responseVals.Users[i].Username = users[i].Username
		responseVals.Users[i].Country = users[i].Country.String
		responseVals.Users[i].CreatedAt = users[i].CreatedAt.Time.Format(DATE_LAYOUT)
		responseVals.Users[i].Birthday = users[i].Birthday.Time.Format(DATE_LAYOUT)
	}
	respondWithJSON(w, http.StatusOK, responseVals)
}
