package api

import (
	"encoding/json"
	"net/http"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/service"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.Svc.ListTasks(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req service.CreateTaskRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	detail, err := s.Svc.CreateTask(r.Context(), actorID(r), req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	detail, err := s.Svc.GetTaskDetail(r.Context(), domain.TaskID(r.PathValue("id")))
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) advanceTask(w http.ResponseWriter, r *http.Request) {
	detail, err := s.Svc.AdvanceTask(r.Context(), domain.TaskID(r.PathValue("id")))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) elicitation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Answer string `json:"answer"` // "yes" | "no"
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	detail, err := s.Svc.AnswerElicitation(r.Context(), domain.TaskID(r.PathValue("id")), actorID(r), body.Answer == "yes")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) settle(w http.ResponseWriter, r *http.Request) {
	detail, err := s.Svc.Settle(r.Context(), domain.TaskID(r.PathValue("id")), actorID(r))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) createLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TargetID string `json:"target_id"`
		LinkType string `json:"link_type"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	link, err := s.Svc.CreateLink(r.Context(), domain.TaskID(r.PathValue("id")),
		domain.TaskID(body.TargetID), domain.LinkType(body.LinkType), actorID(r))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

func (s *Server) listApprovals(w http.ResponseWriter, r *http.Request) {
	reqs, err := s.Svc.ListPendingApprovals(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, reqs)
}

func (s *Server) submitDecision(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Decision   string          `json:"decision"`
		Rationale  string          `json:"rationale"`
		Conditions json.RawMessage `json:"conditions"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	detail, err := s.Svc.SubmitDecision(r.Context(), domain.TaskID(r.PathValue("id")), actorID(r),
		domain.Decision(body.Decision), body.Rationale, body.Conditions)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) listTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Tools.Specs())
}

func (s *Server) listActors(w http.ResponseWriter, r *http.Request) {
	actors, err := s.Svc.Store.ListActors(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, actors)
}

func (s *Server) getDefinition(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	def, ok := s.Registry.Latest(key)
	if !ok {
		writeErr(w, http.StatusNotFound, errNotFound(key))
		return
	}
	writeJSON(w, http.StatusOK, def)
}

func (s *Server) listDefinitions(w http.ResponseWriter, r *http.Request) {
	defs, err := s.Svc.ListDefinitions(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if defs == nil {
		defs = []workflow.WorkflowDefinition{}
	}
	writeJSON(w, http.StatusOK, defs)
}

func (s *Server) registerFlow(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterFlowRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	def, err := s.Svc.RegisterFlow(r.Context(), req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, def)
}

func (s *Server) listTaskTypes(w http.ResponseWriter, r *http.Request) {
	tts, err := s.Svc.ListTaskTypes(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tts)
}
