package api

import (
	"context"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/auth"
	"github.com/google/uuid"
)

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

func (s *State) HandlerMiddlewareLogin(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, errToken := auth.GetHeaderValueToken(r.Header, "Auth")
		refreshTokenString, errRefreshToken := auth.GetHeaderValueToken(r.Header, "X-Refresh-Token")

		ctx := r.Context()

		if errToken == nil {
			userIDString, err := auth.ValidateJWT(tokenString, s.authConfig.JWTsecret)
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
			refreshToken, err := s.db.GetRefreshToken(ctx, refreshTokenString)
			if err != nil || !refreshToken.UserID.Valid {
				respondWithError(w, http.StatusUnauthorized, "Invalid JWT and/or refresh token", err)
				return
			}
			if refreshToken.ExpiresAt.Time.Before(time.Now()) {
				respondWithError(w, http.StatusUnauthorized, "Invalid JWT and/or refresh token", err)
				return
			}
			// generate new jwt and attach it to the header
			newTokenString, err := auth.MakeJWT(refreshToken.UserID.String(), s.authConfig.JWTsecret, s.authConfig.JWTDuration)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Could not generate the JWT", err)
				return
			}
			w.Header().Set("Auth", "Bearer "+newTokenString)

			ctx = ContextWithUser(ctx, refreshToken.UserID.Bytes)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			return
		}

		respondWithError(w, http.StatusUnauthorized, "JWT and refresh token not found in the headers", nil)
	})
}
