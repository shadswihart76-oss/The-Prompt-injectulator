package handler

import (
	"net/http"

	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/assessment"
)

// assessmentsHandler returns an HTTP handler that validates the request and
// returns a structured prompt-injection assessment for the calling user.
// Requires authentication; no token budget is consumed (deterministic local
// generation, zero cost).
func (s *Server) Assess(gen assessment.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		claims, ok := s.requireAuth(w, r)
		if !ok {
			return
		}
		_ = claims // authenticated; assessment content does not depend on identity

		var req assessment.Request
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := assessment.Validate(req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		result, err := gen.Generate(req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not generate assessment")
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}