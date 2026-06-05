package nativeapi

import (
	"context"
	"net/http"
)

func (s *Server) handleTriggerScan(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, http.StatusForbidden, "requer admin")
		return
	}
	if s.scanner.IsScanning() {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "already_running"})
		return
	}
	go func() {
		_, _ = s.scanner.Scan(context.Background())
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (s *Server) handleScanStatus(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, http.StatusForbidden, "requer admin")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"scanning": s.scanner.IsScanning(),
		"last":     s.scanner.LastResult(),
	})
}
