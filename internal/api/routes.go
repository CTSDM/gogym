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
)

func NewServer(db *database.Queries, authConfig *auth.Config) http.Handler {
	serveMux := http.NewServeMux()
	addRoutes(serveMux, db, authConfig)
	return serveMux
}

func addRoutes(mux *http.ServeMux, db *database.Queries, authConfig *auth.Config) {
	// login endpoint
	mux.HandleFunc("POST /api/v1/login", user.HandlerLogin(db, authConfig))

	// users endpoints
	mux.HandleFunc("POST /api/v1/users", user.HandlerCreateUser(db))
	mux.HandleFunc("GET /api/v1/users/{id}",
		middleware.Authentication(db, authConfig)(user.HandlerGetUser(db)))
	mux.HandleFunc("GET /api/v1/users",
		middleware.Authentication(db, authConfig)(
			(middleware.AdminOnly(db))(user.HandlerGetUsers(db)),
		),
	)

	// sessions endpoints
	mux.HandleFunc("POST /api/v1/sessions",
		middleware.Authentication(db, authConfig)(session.HandlerCreateSession(db)))

	// sets endpoints
	mux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets",
		middleware.Authentication(db, authConfig)(set.HandlerCreateSet(db)))

	// logs endpoints
	mux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets/{setID}/logs",
		middleware.Authentication(db, authConfig)(exlog.HandlerCreateLog(db)))

	// exercises endpoints
	mux.HandleFunc("GET /api/v1/exercises/{id}",
		middleware.Authentication(db, authConfig)(exercise.HandlerGetExercise(db)))
	mux.HandleFunc("GET /api/v1/exercises",
		middleware.Authentication(db, authConfig)(exercise.HandlerGetExercises(db)))
}
