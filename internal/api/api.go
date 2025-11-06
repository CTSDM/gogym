package api

import (
	"log"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/auth"
	"github.com/CTSDM/gogym/internal/database"
)

type State struct {
	db         *database.Queries
	authConfig *auth.Config
}

func NewState(db *database.Queries, auth *auth.Config) *State {
	return &State{
		db:         db,
		authConfig: auth,
	}
}

func (s *State) SetupServer() error {
	serveMux := http.NewServeMux()

	// login endpoint
	serveMux.HandleFunc("POST /api/v1/login", s.HandlerLogin)

	// users endpoints
	serveMux.HandleFunc("POST /api/v1/users", s.HandlerCreateUser)
	serveMux.HandleFunc("GET /api/v1/users/{id}", s.HandlerMiddlewareAuthentication(s.HandlerGetUser))
	serveMux.HandleFunc("GET /api/v1/users", s.HandlerMiddlewareAdminOnly(s.HandlerGetUsers))

	// sessions endpoints
	serveMux.HandleFunc("POST /api/v1/sessions", s.HandlerMiddlewareAuthentication(s.HandlerCreateSession))

	// sets endpoints
	serveMux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets", s.HandlerMiddlewareAuthentication(s.HandlerCreateSet))

	// logs endpoints
	serveMux.HandleFunc("POST /api/v1/sessions/{sessionID}/sets/{setID}/logs", s.HandlerMiddlewareAuthentication(s.HandlerCreateLog))

	// exercises endpoints
	serveMux.HandleFunc("GET /api/v1/exercises/{id}", s.HandlerGetExercise)
	serveMux.HandleFunc("GET /api/v1/exercises", s.HandlerMiddlewareAdminOnly(s.HandlerGetExercises))

	// server setup
	server := &http.Server{
		Addr:        ":" + "8080",
		Handler:     serveMux,
		ReadTimeout: 10 * time.Second,
	}

	log.Printf("Serving on port: %s \n", "8080")
	return server.ListenAndServe()
}
