package server

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

// wantsText returns true if the client prefers text/plain responses.
func wantsText(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/plain")
}

// writeResponse writes JSON or text/plain key=value depending on Accept header.
func writeResponse(w http.ResponseWriter, r *http.Request, status int, v any) {
	if wantsText(r) {
		writeText(w, status, v)
		return
	}

	writeJSON(w, status, v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encoding json response: %v", err)
	}
}

func writeText(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)

	var b strings.Builder

	switch val := v.(type) {
	case map[string]string:
		for k, v := range val {
			fmt.Fprintf(&b, "%s=%s\n", k, v)
		}
	case *store.Run:
		writeRunText(&b, val)
	case []*store.Run:
		for i, run := range val {
			if i > 0 {
				b.WriteString("---\n")
			}

			writeRunText(&b, run)
		}
	default:
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Printf("encoding fallback json response: %v", err)
		}

		return
	}

	_, _ = io.WriteString(w, b.String())
}

func writeRunText(b *strings.Builder, r *store.Run) {
	fmt.Fprintf(b, "id=%s\n", r.ID)
	fmt.Fprintf(b, "name=%s\n", r.Name)
	fmt.Fprintf(b, "mode=%s\n", r.Mode)
	fmt.Fprintf(b, "status=%s\n", r.Status)
	fmt.Fprintf(b, "base_sha=%s\n", r.BaseSHA)
	fmt.Fprintf(b, "created_at=%s\n", r.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(b, "attach=%s\n", r.Attach)
	fmt.Fprintf(b, "patch_ready=%t\n", r.PatchReady)

	if r.FinalSHA != "" {
		fmt.Fprintf(b, "final_sha=%s\n", r.FinalSHA)
	}

	if r.FinishedAt != nil {
		fmt.Fprintf(b, "finished_at=%s\n", r.FinishedAt.Format(time.RFC3339))
	}

	if r.Subdir != "" {
		fmt.Fprintf(b, "subdir=%s\n", r.Subdir)
	}

	if r.LogTail != "" {
		fmt.Fprintf(b, "log_tail=%s\n", r.LogTail)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
