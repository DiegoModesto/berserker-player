package nativeapi

import (
	"net/http"
	"os"
)

// candidatePaths para localizar o contrato OpenAPI (fonte da verdade na raiz do monorepo).
var openAPICandidates = []string{
	"openapi.yaml",
	"../openapi.yaml",
	"../../openapi.yaml",
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, _ *http.Request) {
	for _, p := range openAPICandidates {
		if b, err := os.ReadFile(p); err == nil {
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write(b)
			return
		}
	}
	writeError(w, http.StatusNotFound, "openapi.yaml não encontrado")
}
