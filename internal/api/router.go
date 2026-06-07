// Package api is the HTTP transport. It maps requests to the service layer and
// never touches the store directly. Stub auth: the acting actor comes from the
// X-Actor-Id header (set by the frontend ActorSwitcher).
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/service"
	"github.com/kaitoyama/crispy-waffle/internal/tools"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

func errNotFound(what string) error { return fmt.Errorf("not found: %s", what) }

// Server holds dependencies for the handlers.
type Server struct {
	Svc      *service.Service
	Tools    *tools.Registry
	Registry *workflow.Registry
}

// Handler builds the routed, middleware-wrapped HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /api/tasks", s.listTasks)
	mux.HandleFunc("POST /api/tasks", s.createTask)
	mux.HandleFunc("GET /api/tasks/{id}", s.getTask)
	mux.HandleFunc("POST /api/tasks/{id}/advance", s.advanceTask)
	mux.HandleFunc("POST /api/tasks/{id}/elicitation", s.elicitation)
	mux.HandleFunc("POST /api/tasks/{id}/settle", s.settle)
	mux.HandleFunc("POST /api/tasks/{id}/links", s.createLink)

	mux.HandleFunc("GET /api/approvals", s.listApprovals)
	mux.HandleFunc("POST /api/approvals/{id}/decision", s.submitDecision)

	mux.HandleFunc("GET /api/tools", s.listTools)
	mux.HandleFunc("GET /api/actors", s.listActors)
	mux.HandleFunc("GET /api/task-types", s.listTaskTypes)
	mux.HandleFunc("GET /api/workflow-definitions", s.listDefinitions)
	mux.HandleFunc("POST /api/workflow-definitions", s.registerFlow)
	mux.HandleFunc("GET /api/workflow-definitions/{key}", s.getDefinition)

	return cors(mux)
}

// --- middleware & helpers ---

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,X-Actor-Id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// actorID extracts the acting actor from the X-Actor-Id header, defaulting to
// tanaka (the demo requester).
func actorID(r *http.Request) domain.ActorID {
	if v := r.Header.Get("X-Actor-Id"); v != "" {
		return domain.ActorID(v)
	}
	return domain.ActorID("tanaka")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
