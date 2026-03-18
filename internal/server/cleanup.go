package server

import (
	"net/http"
	"os"

	"github.com/wesgrimes/outpost/internal/store"
)

func (s *Server) handleCleanup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	run, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	// Don't clean up active runs.
	if run.Status == store.StatusRunning || run.Status == store.StatusPending {
		writeError(w, http.StatusConflict, "run is still active")
		return
	}

	// Remove run directory from disk.
	if run.Dir != "" {
		if err := os.RemoveAll(run.Dir); err != nil {
			writeError(w, http.StatusInternalServerError, "removing run directory: "+err.Error())
			return
		}
	}

	// Keep metadata in store but mark as cleaned up.
	s.store.Update(id, func(r *store.Run) {
		r.PatchReady = false
		r.Dir = ""
	})

	writeResponse(w, r, http.StatusOK, map[string]string{
		"id":     id,
		"status": "cleaned up",
	})
}
