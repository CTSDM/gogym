package user

import (
	"context"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
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

// For login we only check that the username/password are not empty
func (r loginReq) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)

	// validate username
	if r.Username == "" {
		problems["username"] = "invalid username"
	}

	// validate password
	if r.Password == "" {
		problems["password"] = "invalid password"
	}

	return problems
}

func HandlerLogin(db *database.Queries, authConfig *auth.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqParams, problems, err := validation.DecodeValid[loginReq](r)
		if len(problems) > 0 {
			util.RespondWithJSON(w, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// find the user in the database
		user, err := db.GetUserByUsername(r.Context(), reqParams.Username)
		if err == pgx.ErrNoRows {
			util.RespondWithJSON(w, http.StatusOK, util.ErrorResponse{
				Error: "Incorrect username/password",
			})
			return
		} else if err != nil {
			util.RespondWithError(w, http.StatusInternalServerError, "Something went wrong while processing the login", err)
			return
		}

		// verify the password
		if err := auth.CheckPasswordHash(reqParams.Password, user.HashedPassword); err == bcrypt.ErrMismatchedHashAndPassword {
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
			ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(authConfig.RefreshTokenDuration).UTC(), Valid: true},
			UserID:    user.ID,
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
