package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Birthday string `json:"birthday"` // represented as YYYY-MM-DD (ISO 8601)
	Country  string `json:"country"`
}

func (s *State) HandlerCreateUser(w http.ResponseWriter, r *http.Request) {
	params := createUserRequest{}
	defer r.Body.Close()

	// decode inc json body
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		msg := "Could not decode the request body into a JSON."
		respondWithError(w, http.StatusBadRequest, msg, err)
		return
	}

	// validate the json
	birthday, err := validateCreateUser(params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// generate hashed password
	hashed, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong while hashing the password", err)
		return
	}

	// check if the user exists in the database
	// insert new user into the database
	user, err := s.db.CreateUser(r.Context(), database.CreateUserParams{
		Username:       params.Username,
		Birthday:       pgtype.Date{Time: birthday, Valid: true},
		Country:        pgtype.Text{String: params.Country, Valid: true},
		HashedPassword: hashed,
	})
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			respondWithError(w, http.StatusConflict, "Username is already in use", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Something went wrong while creating the user", err)
		return
	}
	respondWithJSON(w, http.StatusCreated, User{
		ID:        user.ID.String(),
		Username:  user.Username,
		Country:   user.Country.String,
		CreatedAt: user.CreatedAt.Time.String(),
		Birthday:  user.Birthday.Time.Format(DATE_LAYOUT),
	})
}

func validateCreateUser(req createUserRequest) (birthday time.Time, err error) {
	// username validation
	if err = validateString(req.Username, minUsernameLength, maxUsernameLength); err != nil {
		return birthday, fmt.Errorf("could not validate the username: %w", err)
	}
	// password validation
	if err = validateString(req.Password, minPasswordLength, maxPasswordLength); err != nil {
		return birthday, fmt.Errorf("could not validate the password: %w", err)
	}
	// date validation, it is an optional parameter
	if req.Birthday != "" {
		date, err := validateDate(req.Birthday, DATE_LAYOUT, &minBirthDate, &maxBirthDate)
		if err != nil {
			return birthday, fmt.Errorf("could not validate the date: %w", err)
		}
		birthday = date
	}
	// country validation, it is an optional parameter
	if req.Country != "" {
		if err := validateString(req.Country, minCountryLength, maxCountryLength); err != nil {
			return birthday, fmt.Errorf("could not validate the country: %w", err)
		}
	}

	return birthday, err
}
