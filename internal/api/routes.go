package api

import (
	"log/slog"
	"net/http"

	"github.com/CTSDM/gogym/internal/api/exercise"
	"github.com/CTSDM/gogym/internal/api/exlog"
	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/session"
	"github.com/CTSDM/gogym/internal/api/set"
	"github.com/CTSDM/gogym/internal/api/user"
	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewServer(
	pool *pgxpool.Pool,
	db *database.Queries,
	authConfig *auth.Config,
	logger *slog.Logger,
) http.Handler {
	serveMux := http.NewServeMux()
	addRoutes(pool, serveMux, db, authConfig, logger)
	var handler http.Handler = serveMux
	handler = middleware.RequestID(handler)
	return handler
}

func addRoutes(
	pool *pgxpool.Pool,
	mux *http.ServeMux,
	db *database.Queries,
	authConfig *auth.Config,
	logger *slog.Logger,
) {
	// middleware declaration
	authentication := middleware.Authentication(db, authConfig, logger)
	admin := middleware.AdminOnly(db, logger)

	// login endpoint
	mux.HandleFunc("POST /api/v1/login", user.HandlerLogin(db, authConfig, logger))

	// users endpoints
	mux.HandleFunc("POST /api/v1/users", user.HandlerCreateUser(db, logger))
	mux.HandleFunc("GET /api/v1/users/{id}",
		middleware.Chain(user.HandlerGetUser(db, logger), admin, authentication))
	mux.HandleFunc("GET /api/v1/users", middleware.Chain(
		user.HandlerGetUsers(db, logger),
		admin,
		authentication),
	)

	// sessions endpoints
	mux.HandleFunc("POST /api/v1/sessions", authentication(session.HandlerCreateSession(db, logger)))
	mux.HandleFunc("GET /api/v1/sessions", authentication(session.HandlerGetSessions(db, logger)))
	mux.HandleFunc("GET /api/v1/sessions/{id}", middleware.Chain(
		session.HandlerGetSession(db, logger),
		middleware.Ownership("id", db.GetSessionOwnerID, logger),
		authentication))
	mux.HandleFunc("PUT /api/v1/sessions/{id}", middleware.Chain(
		session.HandlerUpdateSession(db, logger),
		middleware.Ownership("id", db.GetSessionOwnerID, logger),
		authentication))
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", middleware.Chain(
		session.HandlerDeleteSession(db, logger),
		middleware.Ownership("id", db.GetSessionOwnerID, logger),
		authentication))

	// sets endpoints
	mux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets", authentication(set.HandlerCreateSet(db, logger)))
	mux.HandleFunc("DELETE /api/v1/sets/{id}", middleware.Chain(
		set.HandlerDeleteSet(db, logger),
		middleware.Ownership("id", db.GetSetOwnerID, logger),
		authentication))
	mux.HandleFunc("GET /api/v1/sets/{id}", middleware.Chain(
		set.HandlerGetSet(db, logger),
		middleware.Ownership("id", db.GetSetOwnerID, logger),
		authentication))
	mux.HandleFunc("PUT /api/v1/sets/{id}", middleware.Chain(
		set.HandlerUpdateSet(pool, db, logger),
		middleware.Ownership("id", db.GetSetOwnerID, logger),
		authentication))

	// logs endpoints
	mux.HandleFunc("GET /api/v1/logs/", authentication(exlog.HandlerGetLogs(db)))
	mux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets/{setID}/logs",
		authentication(exlog.HandlerCreateLog(db)))
	mux.HandleFunc("PUT /api/v1/logs/{id}", middleware.Chain(
		exlog.HandlerUpdateLog(db),
		middleware.Ownership("id", db.GetLogOwnerID, logger),
		authentication))
	mux.HandleFunc("DELETE /api/v1/logs/{id}", middleware.Chain(
		exlog.HandlerDeleteLog(db),
		middleware.Ownership("id", db.GetLogOwnerID, logger),
		authentication))

	// exercises endpoints
	mux.HandleFunc("GET /api/v1/exercises/{id}", authentication(exercise.HandlerGetExercise(db)))
	mux.HandleFunc("GET /api/v1/exercises", authentication(exercise.HandlerGetExercises(db)))

	// health endpoint
	mux.HandleFunc("GET /health", handlerHealth(pool))
}
