package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRes struct {
	Username     string `json:"username"`
	UserID       string `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
	Token        string `json:"token"`
}

func HandlerLogin(db *database.Queries, authConfig auth.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := loginReq{}
		defer r.Body.Close()

		// decode the json
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&params); err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "Could not decode the request body into a JSON.", err)
			return
		}

		// validate the inputs
		if err := validateLogin(params); err != nil {
			message := fmt.Sprintf("invalid login %s", err.Error())
			util.RespondWithError(w, http.StatusBadRequest, message, nil)
			return
		}

		// find the user in the database
		user, err := db.GetUserByUsername(r.Context(), params.Username)
		if err == pgx.ErrNoRows {
			util.RespondWithJSON(w, http.StatusOK, util.ErrorResponse{
				Error: "Incorrect username/password",
			})
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Something went wrong while processing the login", err)
			return
		}

		// validate the password
		if err := auth.CheckPasswordHash(params.Password, user.HashedPassword); err == bcrypt.ErrMismatchedHashAndPassword {
			util.RespondWithJSON(w, http.StatusOK, util.ErrorResponse{
				Error: "Incorrect username/password",
			})
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Something went wrong while verifying the password", err)
			return
		}

		// Generate refresh Token, JWT
		refreshToken, err := auth.MakeRefreshToken()
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Something went wrong while creating the refresh token", err)
			return
		}
		jwtString, err := auth.MakeJWT(user.ID.String(), authConfig.JWTsecret, authConfig.JWTDuration)
		if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Something went wrong while creating the JWT", err)
			return
		}

		// Store the refresh token at the database
		if _, err := db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			Token:     refreshToken,
			ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(time.Hour), Valid: true},
			UserID:    pgtype.UUID{Bytes: user.ID.Bytes, Valid: true},
		}); err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Something went wrong while saving the refresh token", err)
			return
		}

		// Return the payload
		util.RespondWithJSON(w, http.StatusOK, loginRes{
			Username:     user.Username,
			UserID:       user.ID.String(),
			Token:        jwtString,
			RefreshToken: refreshToken,
		})
	}
}

func validateLogin(req loginReq) error {
	// validate username
	if req.Username == "" {
		return errors.New("username cannot be empty")
	}

	// validate password
	if req.Password == "" {
		return errors.New("password cannot be empty")
	}

	return nil
}
