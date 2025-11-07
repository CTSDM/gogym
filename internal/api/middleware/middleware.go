package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Middleware struct {
	db         *database.Queries
	authConfig *auth.Config
}

func NewMiddleware(db *database.Queries, authConfig *auth.Config) *Middleware {
	return &Middleware{
		db:         db,
		authConfig: authConfig,
	}
}

type contextKey int

const (
	_ contextKey = iota
	userKey
)

func ContextWithUser(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userKey, userID)
}

func UserFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userKey).(uuid.UUID)
	return userID, ok
}

func Authentication(db *database.Queries, authConfig *auth.Config) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, errToken := auth.GetHeaderValueToken(r.Header, "Auth")
			refreshTokenString, errRefreshToken := auth.GetHeaderValueToken(r.Header, "X-Refresh-Token")

			ctx := r.Context()

			if errToken == nil {
				userIDString, err := auth.ValidateJWT(tokenString, authConfig.JWTsecret)
				if err == nil {
					userID, err := uuid.Parse(userIDString)
					if err == nil {
						ctx = ContextWithUser(ctx, userID)
						r = r.WithContext(ctx)
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			if errRefreshToken == nil {
				refreshToken, err := db.GetRefreshToken(ctx, refreshTokenString)
				if err != nil || !refreshToken.UserID.Valid {
					util.RespondWithError(w, http.StatusUnauthorized, "Invalid JWT and/or refresh token", err)
					return
				}
				if refreshToken.ExpiresAt.Time.Before(time.Now()) {
					util.RespondWithError(w, http.StatusUnauthorized, "Invalid JWT and/or refresh token", err)
					return
				}
				// generate new jwt and attach it to the header
				newTokenString, err := auth.MakeJWT(refreshToken.UserID.String(), authConfig.JWTsecret, authConfig.JWTDuration)
				if err != nil {
					util.RespondWithError(w, http.StatusInternalServerError, "Could not generate the JWT", err)
					return
				}
				w.Header().Set("Auth", "Bearer "+newTokenString)

				ctx = ContextWithUser(ctx, refreshToken.UserID.Bytes)
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}

			util.RespondWithError(w, http.StatusUnauthorized, "JWT and refresh token not found in the headers", nil)
		})
	}
}

func AdminOnly(db *database.Queries) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID, _ := UserFromContext(ctx)

			user, err := db.GetUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
			switch err {
			}
			if err == pgx.ErrNoRows {
				util.RespondWithError(w, http.StatusUnauthorized, "could not retrieve user from the database", err)
				return
			} else if err != nil {
				util.RespondWithError(w, http.StatusInternalServerError, "could not fetch the user from the database", err)
				return
			}

			if !user.IsAdmin.Valid || !user.IsAdmin.Bool {
				util.RespondWithError(w, http.StatusForbidden, "admin access required", nil)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}
