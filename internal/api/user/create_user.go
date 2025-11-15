package user

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/api/validation"
	"github.com/CTSDM/gogym/internal/apiconstants"
	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Birthday string `json:"birthday"` // represented as YYYY-MM-DD (ISO 8601)
	Country  string `json:"country"`

	bdate time.Time
}

func (r *createUserRequest) Valid(ctx context.Context) map[string]string {
	// username validation
	problems := make(map[string]string)
	if err := validation.String(r.Username, apiconstants.MinUsernameLength, apiconstants.MaxUsernameLength); err != nil {
		problems["username"] = fmt.Sprintf("invalid username: %s", err.Error())
	}

	// password validation
	if err := validation.String(r.Password, apiconstants.MinPasswordLength, apiconstants.MaxPasswordLength); err != nil {
		problems["password"] = fmt.Sprintf("invalid password: %s", err.Error())
	}

	// date validation, it is an optional parameter
	if r.Birthday != "" {
		date, err := validation.Date(r.Birthday, apiconstants.DATE_LAYOUT, &apiconstants.MinBirthDate, &apiconstants.MaxBirthDate)
		if err != nil {
			problems["birthday"] = fmt.Sprintf("invalid birthday: %s", err.Error())
		}
		r.bdate = date
	}

	// country validation, it is an optional parameter
	if r.Country != "" {
		if err := validation.String(r.Country, apiconstants.MinCountryLength, apiconstants.MaxCountryLength); err != nil {
			problems["country"] = fmt.Sprintf("invalid country: %s", err.Error())
		}
	}

	return problems
}

func HandlerCreateUser(db *database.Queries, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqLogger := middleware.BasicReqLogger(logger, r)
		// validate the json
		reqParams, problems, err := validation.DecodeValid[*createUserRequest](r)
		if len(problems) > 0 {
			reqLogger.Debug("create user failed - invalid parameters", slog.Any("problems", problems))
			util.RespondWithJSON(w, r, http.StatusBadRequest, problems)
			return
		} else if err != nil {
			reqLogger.Debug("create user failed - invalid payload", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusBadRequest, "invalid payload", err)
			return
		}

		// generate hashed password
		hashed, err := auth.HashPassword(reqParams.Password)
		if err != nil {
			reqLogger.Error(
				"create user failed - error hashing the password",
				slog.String("error", err.Error()),
			)
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}

		reqLogger = reqLogger.With(slog.String("username", reqParams.Username))
		// check if the user exists in the database
		// insert new user into the database
		user, err := db.CreateUser(r.Context(), database.CreateUserParams{
			Username:       reqParams.Username,
			Birthday:       pgtype.Date{Time: reqParams.bdate, Valid: true},
			Country:        pgtype.Text{String: reqParams.Country, Valid: true},
			HashedPassword: hashed,
		})
		if err != nil {
			if strings.Contains(err.Error(), "23505") {
				reqLogger.Warn(
					"create user failed - username already taken",
					slog.String("error", err.Error()),
				)
				util.RespondWithError(w, r, http.StatusConflict, "Username is already in use", err)
				return
			}
			reqLogger.Error("create user failed - user creation error", slog.String("error", err.Error()))
			util.RespondWithError(w, r, http.StatusInternalServerError, "something went wrong", err)
			return
		}
		reqLogger.Info("create user success", slog.String("user_id", user.ID.String()))
		util.RespondWithJSON(w, r, http.StatusCreated, User{
			ID:        user.ID.String(),
			Username:  user.Username,
			Country:   user.Country.String,
			CreatedAt: user.CreatedAt.Time.String(),
			Birthday:  user.Birthday.Time.Format(apiconstants.DATE_LAYOUT),
		})
	}
}
