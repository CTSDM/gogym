package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/CTSDM/gogym/internal/api/util"
	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type contextKey int

const (
	_ contextKey = iota
	userKey
	resourceIDKey
)

// middlewares are applied from left from left to right
func Chain(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}

func ContextWithUser(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userKey, userID)
}

func UserFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userKey).(uuid.UUID)
	return userID, ok
}

func ContextWithResourceID(ctx context.Context, resourceID any) context.Context {
	return context.WithValue(ctx, resourceIDKey, resourceID)
}

func ResourceIDFromContext(ctx context.Context) (any, bool) {
	resourceID := ctx.Value(resourceIDKey)
	if resourceID == nil {
		return nil, false
	}
	return resourceID, true
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
				if err != nil {
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

				ctx = ContextWithUser(ctx, refreshToken.UserID)
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

			user, err := db.GetUser(ctx, userID)
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

func Ownership[T any](pathKey string, fn func(ctx context.Context, v T) (uuid.UUID, error)) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID, _ := UserFromContext(ctx)

			idStr := r.PathValue(pathKey)
			var id any
			var zero T

			switch any(zero).(type) {
			case int64:
				parsed, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil {
					util.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s format", pathKey), nil)
					return
				}
				id = parsed
			case uuid.UUID:
				parsed, err := uuid.Parse(idStr)
				if err != nil {
					util.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s format", pathKey), nil)
					return
				}
				id = parsed
			default:
				err := fmt.Errorf("could not recognize the type for %s", pathKey)
				util.RespondWithError(w, http.StatusInternalServerError, "could not process the request", err)
				return
			}

			ownerID, err := fn(r.Context(), id.(T))
			if err == pgx.ErrNoRows {
				util.RespondWithError(w, http.StatusNotFound, "not found", err)
				return
			} else if err != nil {
				util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
				return
			}
			if ownerID != userID {
				util.RespondWithError(w, http.StatusForbidden, "user is not owner", nil)
				return
			}
			ctx = ContextWithResourceID(ctx, id)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}
	}
}
