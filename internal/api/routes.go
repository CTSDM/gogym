package api

import (
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

func NewServer(pool *pgxpool.Pool, db *database.Queries, authConfig *auth.Config) http.Handler {
	serveMux := http.NewServeMux()
	addRoutes(pool, serveMux, db, authConfig)
	return serveMux
}

func addRoutes(pool *pgxpool.Pool, mux *http.ServeMux, db *database.Queries, authConfig *auth.Config) {
	// middleware declaration
	authentication := middleware.Authentication(db, authConfig)
	admin := middleware.AdminOnly(db)

	// login endpoint
	mux.HandleFunc("POST /api/v1/login", user.HandlerLogin(db, authConfig))

	// users endpoints
	mux.HandleFunc("POST /api/v1/users", user.HandlerCreateUser(db))
	mux.HandleFunc("GET /api/v1/users/{id}",
		middleware.Chain(user.HandlerGetUser(db), authentication, admin))
	mux.HandleFunc("GET /api/v1/users", middleware.Chain(user.HandlerGetUsers(db), authentication, admin))

	// sessions endpoints
	mux.HandleFunc("POST /api/v1/sessions", authentication(session.HandlerCreateSession(db)))
	mux.HandleFunc("GET /api/v1/sessions", authentication(session.HandlerGetSessions(db)))
	mux.HandleFunc("PUT /api/v1/sessions/{id}", middleware.Chain(
		session.HandlerUpdateSession(db),
		authentication,
		middleware.Ownership("id", db.GetSessionOwnerID)))
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", middleware.Chain(
		session.HandlerDeleteSession(db),
		authentication,
		middleware.Ownership("id", db.GetSessionOwnerID)))

	// sets endpoints
	mux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets", authentication(set.HandlerCreateSet(db)))
	mux.HandleFunc("DELETE /api/v1/sets/{id}", middleware.Chain(
		set.HandlerDeleteSet(db),
		authentication,
		middleware.Ownership("id", db.GetSetOwnerID)))
	mux.HandleFunc("GET /api/v1/sets/{id}", authentication(set.HandlerGetSet(db)))
	mux.HandleFunc("PUT /api/v1/sets/{id}", middleware.Chain(
		set.HandlerUpdateSet(pool, db),
		authentication,
		middleware.Ownership("id", db.GetSetOwnerID)))

	// logs endpoints
	mux.HandleFunc("GET /api/v1/logs/", authentication(exlog.HandlerGetLogs(db)))
	mux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets/{setID}/logs",
		authentication(exlog.HandlerCreateLog(db)))
	mux.HandleFunc("PUT /api/v1/logs/{id}", middleware.Chain(
		exlog.HandlerUpdateLog(db),
		authentication,
		middleware.Ownership("id", db.GetLogOwnerID)))
	mux.HandleFunc("DELETE /api/v1/logs/{id}", middleware.Chain(
		exlog.HandlerDeleteLog(db),
		authentication,
		middleware.Ownership("id", db.GetLogOwnerID)))

	// exercises endpoints
	mux.HandleFunc("GET /api/v1/exercises/{id}", authentication(exercise.HandlerGetExercise(db)))
	mux.HandleFunc("GET /api/v1/exercises", authentication(exercise.HandlerGetExercises(db)))
}
