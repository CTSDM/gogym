package api

import (
	"log"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/api/exercise"
	"github.com/CTSDM/gogym/internal/api/exlog"
	"github.com/CTSDM/gogym/internal/api/middleware"
	"github.com/CTSDM/gogym/internal/api/session"
	"github.com/CTSDM/gogym/internal/api/set"
	"github.com/CTSDM/gogym/internal/api/user"
	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
)

type State struct {
	db         *database.Queries
	authConfig auth.Config
}

func NewState(db *database.Queries, auth auth.Config) *State {
	return &State{
		db:         db,
		authConfig: auth,
	}
}

func (s *State) SetupServer() error {
	serveMux := http.NewServeMux()

	// login endpoint
	serveMux.HandleFunc("POST /api/v1/login", user.HandlerLogin(s.db, s.authConfig))

	// users endpoints
	serveMux.HandleFunc("POST /api/v1/users", user.HandlerCreateUser(s.db))
	serveMux.HandleFunc("GET /api/v1/users/{id}",
		middleware.Authentication(s.db, s.authConfig)(user.HandlerGetUser(s.db)))
	serveMux.HandleFunc("GET /api/v1/users",
		middleware.Authentication(s.db, s.authConfig)(
			(middleware.AdminOnly(s.db, s.authConfig))(user.HandlerGetUsers(s.db)),
		),
	)

	// sessions endpoints
	serveMux.HandleFunc("POST /api/v1/sessions",
		middleware.Authentication(s.db, s.authConfig)(session.HandlerCreateSession(s.db)))

	// sets endpoints
	serveMux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets",
		middleware.Authentication(s.db, s.authConfig)(set.HandlerCreateSet(s.db)))

	// logs endpoints
	serveMux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets/{setID}/logs",
		middleware.Authentication(s.db, s.authConfig)(exlog.HandlerCreateLog(s.db)))

	// exercises endpoints
	serveMux.HandleFunc("GET /api/v1/exercises/{id}",
		middleware.Authentication(s.db, s.authConfig)(exercise.HandlerGetExercise(s.db)))
	serveMux.HandleFunc("GET /api/v1/exercises",
		middleware.Authentication(s.db, s.authConfig)(exercise.HandlerGetExercises(s.db)))

	// server setup
	server := &http.Server{
		Addr:        ":" + "8080",
		Handler:     serveMux,
		ReadTimeout: 10 * time.Second,
	}

	log.Printf("Serving on port: %s \n", "8080")
	return server.ListenAndServe()
}
