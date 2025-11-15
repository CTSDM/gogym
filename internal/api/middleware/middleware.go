package middleware

import (
	"context"
	"fmt"
	"log/slog"
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
	requestIDKey
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

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok
}

func RequestID(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}
		ctx := ContextWithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func BasicReqLogger(logger *slog.Logger, r *http.Request) *slog.Logger {
	requestID, _ := RequestIDFromContext(r.Context())
	reqLogger := logger.With(
		slog.String("request_id", requestID),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("remote_addr", r.RemoteAddr),
	)
	return reqLogger.With(slog.String("request_id", requestID))
}

func Authentication(
	db *database.Queries,
	authConfig *auth.Config,
	logger *slog.Logger,
) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqLogger := BasicReqLogger(logger, r)

			tokenString, errJWT := auth.GetHeaderValueToken(r.Header, "Auth")
			refreshTokenString, errRefreshToken := auth.GetHeaderValueToken(r.Header, "X-Refresh-Token")

			ctx := r.Context()

			if errJWT == nil {
				userIDString, err := auth.ValidateJWT(tokenString, authConfig.JWTsecret)
				if err == nil {
					userID, err := uuid.Parse(userIDString)
					if err == nil {
						ctx = ContextWithUser(ctx, userID)
						r = r.WithContext(ctx)
						next.ServeHTTP(w, r)
						return
					}
				} else {
					reqLogger.Debug("authentication failed - jwt validation error",
						slog.String("error", err.Error()),
					)
				}
			} else {
				reqLogger.Debug("jwt not found")
			}

			if errRefreshToken == nil {
				refreshToken, err := db.GetRefreshToken(ctx, refreshTokenString)
				if err == pgx.ErrNoRows {
					reqLogger.Debug("authentication failed - refresh token not found on database",
						slog.String("refresh_token", refreshTokenString),
					)
					util.RespondWithError(w, http.StatusUnauthorized, "Invalid JWT and/or refresh token", err)
					return
				} else if err != nil {
					reqLogger.Error("authentication failed - database error",
						slog.String("error", err.Error()),
					)
					util.RespondWithError(w, http.StatusUnauthorized, "Invalid JWT and/or refresh token", err)
					return
				}
				if refreshToken.ExpiresAt.Time.Before(time.Now().UTC()) {
					reqLogger.Debug("authentication failed - expired refresh token",
						slog.String("user_id", refreshToken.UserID.String()),
						slog.Time("expiration", refreshToken.ExpiresAt.Time),
					)
					util.RespondWithError(w, http.StatusUnauthorized, "Invalid JWT and/or refresh token", err)
					return
				}
				// generate new jwt and attach it to the header
				newTokenString, err := auth.MakeJWT(refreshToken.UserID.String(), authConfig.JWTsecret, authConfig.JWTDuration)
				if err != nil {
					reqLogger.Error("authentication failed - JWT creation error",
						slog.String("user_id", refreshToken.UserID.String()),
						slog.String("error", err.Error()),
					)
					util.RespondWithError(w, http.StatusInternalServerError, "Could not generate the JWT", err)
					return
				}

				w.Header().Set("Auth", "Bearer "+newTokenString)
				ctx = ContextWithUser(ctx, refreshToken.UserID)
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}

			reqLogger.Warn("authentication failed - no valid credentials")
			util.RespondWithError(w, http.StatusUnauthorized, "JWT and refresh token not found in the headers", nil)
		})
	}
}

func AdminOnly(db *database.Queries, logger *slog.Logger) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			requestID, ok := RequestIDFromContext(r.Context())
			if !ok {
				logger.Error("request id not found in the context")
			}
			reqLogger := logger.With(
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)

			ctx := r.Context()
			userID, ok := UserFromContext(ctx)
			if !ok {
				reqLogger.Error("admin authentication failed - user not found in context")
				util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", nil)
				return
			}

			reqLogger = reqLogger.With(slog.String("user_id", userID.String()))
			user, err := db.GetUser(ctx, userID)
			switch err {
			}
			if err == pgx.ErrNoRows {
				reqLogger.Info("admin authentication failed - user not found in the database")
				util.RespondWithError(w, http.StatusUnauthorized, "could not retrieve user from the database", err)
				return
			} else if err != nil {
				reqLogger.Error("admin authentication failed - database error",
					slog.String("error", err.Error()),
				)
				util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
				return
			}

			if !user.IsAdmin.Valid || !user.IsAdmin.Bool {
				reqLogger.Warn("admin authentication failed - user is not admin")
				util.RespondWithError(w, http.StatusForbidden, "admin access required", nil)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}

func Ownership[T any](
	pathKey string,
	fn func(ctx context.Context, v T) (uuid.UUID, error),
	logger *slog.Logger,
) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			reqLogger := BasicReqLogger(logger, r)

			ctx := r.Context()
			userID, ok := UserFromContext(ctx)
			if !ok {
				reqLogger.Error("ownership check failed - user id not found in the context")
				util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", nil)
				return
			}

			reqLogger.With(slog.String("user_id", userID.String()))
			idStr := r.PathValue(pathKey)
			var id any
			var zero T

			switch any(zero).(type) {
			case int64:
				parsed, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil {
					reqLogger.Warn("ownership check failed - invalid format",
						slog.String("path_key", pathKey),
						slog.String("type", "uuid"),
						slog.String("value", idStr),
					)
					util.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s format", pathKey), nil)
					return
				}
				id = parsed
			case uuid.UUID:
				parsed, err := uuid.Parse(idStr)
				if err != nil {
					reqLogger.Warn("ownership check failed - invalid format",
						slog.String("path_key", pathKey),
						slog.String("type", "uuid"),
						slog.String("value", idStr),
					)
					util.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s format", pathKey), nil)
					return
				}
				id = parsed
			default:
				reqLogger.Warn("ownership check failed - invalid format",
					slog.String("path_key", pathKey),
					slog.String("value", idStr),
				)
				err := fmt.Errorf("could not recognize the type for %s", pathKey)
				util.RespondWithError(w, http.StatusInternalServerError, "could not process the request", err)
				return
			}

			reqLogger = reqLogger.With(slog.String("item_id", idStr))
			ownerID, err := fn(ctx, id.(T))
			if err == pgx.ErrNoRows {
				reqLogger.Warn("ownership check failed - user or item not found in the database",
					slog.String("error", err.Error()),
				)
				util.RespondWithError(w, http.StatusNotFound, "not found", err)
				return
			} else if err != nil {
				reqLogger.Error("ownership check failed - fetching item database error",
					slog.String("error", err.Error()),
				)
				util.RespondWithError(w, http.StatusInternalServerError, "something went wrong", err)
				return
			}
			if ownerID != userID {
				reqLogger.Warn("ownership check failed - user is not owner")
				util.RespondWithError(w, http.StatusForbidden, "user is not owner", nil)
				return
			}
			ctx = ContextWithResourceID(ctx, id)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}
	}
}
