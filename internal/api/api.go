package api

import (
	"log"
	"net/http"
	"time"

	"github.com/CTSDM/gogym/internal/database"
)

type State struct {
	db *database.Queries
}

func NewState(db *database.Queries) *State {
	return &State{
		db: db,
	}
}

func (s *State) SetupServer() error {
	serveMux := http.NewServeMux()

	// handler functions
	serveMux.HandleFunc("GET /api/v1/users", s.HandlerGetUsers)
	serveMux.HandleFunc("POST /api/v1/login", s.HandlerLogin)
	serveMux.HandleFunc("POST /api/v1/users", s.HandlerCreateUser)

	// server setup
	server := &http.Server{
		Addr:        ":" + "8080",
		Handler:     serveMux,
		ReadTimeout: 10 * time.Second,
	}

	log.Printf("Serving on port: %s \n", "8080")

	return server.ListenAndServe()
}
