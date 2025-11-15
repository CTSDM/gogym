package user

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/api/middleware"
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

func HandlerLogin(db *database.Queries, authConfig *auth.Config, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)

		reqParams, problems, err := validation.DecodeValid[loginReq](r)
		if len(problems) > 0 {
			reqLogger.Debug("login failed - validation errors",
				slog.Any("problems", problems),
			)
			util.RespondWithJSON(w, r, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			reqLogger.Debug("login failed - invalid payload",
				slog.String("error", err.Error()),
			)
			util.RespondWithError(w, r, http.StatusBadRequest, "invalid payload", err)
			return
		}

		reqLogger = reqLogger.With(slog.String("username", reqParams.Username))
		// find the user in the database
		user, err := db.GetUserByUsername(r.Context(), reqParams.Username)
		if err == pgx.ErrNoRows {
			reqLogger.Warn("login failed - user not found")
			util.RespondWithJSON(w, r, http.StatusOK, util.ErrorResponse{
				Error: "Incorrect username/password",
			})
			return
		} else if err != nil {
			reqLogger.Error("login failed - database error", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// verify the password
		if err := auth.CheckPasswordHash(reqParams.Password, user.HashedPassword); err == bcrypt.ErrMismatchedHashAndPassword {
			reqLogger.Warn("login failed - incorrect password")
			util.RespondWithJSON(w, r, http.StatusOK, util.ErrorResponse{
				Error: "Incorrect username/password",
			})
			return
		} else if err != nil {
			reqLogger.Error("login failed - password verification error", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// Generate refresh Token, JWT
		refreshToken, err := auth.MakeRefreshToken()
		if err != nil {
			reqLogger.Error("login failed - refresh token creation error",
				slog.String("error", err.Error()),
			)
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		jwtString, err := auth.MakeJWT(user.ID.String(), authConfig.JWTsecret, authConfig.JWTDuration)
		if err != nil {
			reqLogger.Error("login failed - JWT creation error",
				slog.String("error", err.Error()),
			)
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// Store the refresh token at the database
		if _, err := db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			Token:     refreshToken,
			ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(authConfig.RefreshTokenDuration).UTC(), Valid: true},
			UserID:    user.ID,
		}); err != nil {
			reqLogger.Error("login failed - refresh token storage error",
				slog.String("error", err.Error()),
			)
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		// Return the payload
		reqLogger.Info("login successful", slog.String("user_id", user.ID.String()))
		util.RespondWithJSON(w, r, http.StatusOK, loginRes{
			Username:     user.Username,
			UserID:       user.ID.String(),
			Token:        jwtString,
			RefreshToken: refreshToken,
		})
	}
}
