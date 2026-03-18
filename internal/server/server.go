package server

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/store"
)

// Server is the Outpost HTTP server.
type Server struct {
	cfg   *config.Config
	store *store.Store
	mux   *http.ServeMux
}

// New creates a Server with all routes registered.
func New(cfg *config.Config, st *store.Store) *Server {
	s := &Server{
		cfg:   cfg,
		store: st,
		mux:   http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("POST /handoff", s.requireAuth(s.handleHandoff))
	s.mux.HandleFunc("GET /runs", s.requireAuth(s.handleListRuns))
	s.mux.HandleFunc("GET /runs/{id}", s.requireAuth(s.handleGetRun))
	s.mux.HandleFunc("GET /runs/{id}/patch", s.requireAuth(s.handleGetPatch))
	s.mux.HandleFunc("GET /runs/{id}/log", s.requireAuth(s.handleGetLog))
	s.mux.HandleFunc("DELETE /runs/{id}", s.requireAuth(s.handleDeleteRun))
	s.mux.HandleFunc("POST /runs/{id}/cleanup", s.requireAuth(s.handleCleanup))

	return s
}

// ListenAndServe starts the HTTP server on the configured port.
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Server.Port)
	log.Printf("outpost listening on %s", addr)

	srv := &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return srv.ListenAndServe()
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		expected := "Bearer " + s.cfg.Server.Token

		if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encoding json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
