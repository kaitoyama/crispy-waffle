// Package service orchestrates the domain, workflow engine and store. It is the
// only layer that creates tasks/runs and drives signals; the api layer calls it.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/store"
	"github.com/kaitoyama/crispy-waffle/internal/tools"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// Service holds the wired dependencies.
type Service struct {
	Store    *store.Store
	Engine   *workflow.Engine
	Registry *workflow.Registry
	// Catalog bounds which tools a UI-registered flow may bind (optional).
	Catalog *tools.Catalog
	// DefaultAssignee executes agent/system steps when a request omits one.
	DefaultAssignee domain.ActorID
}

func New(st *store.Store, eng *workflow.Engine, reg *workflow.Registry, defaultAssignee domain.ActorID) *Service {
	return &Service{Store: st, Engine: eng, Registry: reg, DefaultAssignee: defaultAssignee}
}

// CreateTaskRequest is the input to CreateTask.
type CreateTaskRequest struct {
	Type       string         `json:"type"`
	Title      string         `json:"title"`
	Intent     string         `json:"intent"`
	AssigneeID domain.ActorID `json:"assignee_id"`
	Context    map[string]any `json:"context"`
}

// CreateTask creates a task, starts its workflow run, records task.created
// (carrying the initial context), and advances to the first wait/terminal.
func (s *Service) CreateTask(ctx context.Context, requester domain.ActorID, req CreateTaskRequest) (TaskDetail, error) {
	tt, err := s.Store.GetTaskType(ctx, req.Type)
	if err != nil {
		return TaskDetail{}, fmt.Errorf("unknown task type %q: %w", req.Type, err)
	}
	def, ok := s.Registry.Get(tt.DefaultWorkflowKey, tt.DefaultWorkflowVer)
	if !ok {
		return TaskDetail{}, fmt.Errorf("workflow %s@%d not registered", tt.DefaultWorkflowKey, tt.DefaultWorkflowVer)
	}
	entry, ok := def.Step(def.EntryStep)
	if !ok {
		return TaskDetail{}, fmt.Errorf("definition has no entry step")
	}

	assignee := req.AssigneeID
	if assignee == "" {
		assignee = s.DefaultAssignee
	}
	if req.Context == nil {
		req.Context = map[string]any{}
	}
	ctxJSON, _ := json.Marshal(req.Context)
	now := time.Now().UTC()

	taskID := domain.NewTaskID()
	runID := domain.NewRunID()

	task := domain.Task{
		ID: taskID, Title: req.Title, Intent: req.Intent, Type: req.Type,
		Status: entry.EntersState, RequesterID: requester, AssigneeID: assignee,
		WorkflowRunID: &runID, Context: ctxJSON, Origin: domain.OriginManual,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Store.CreateTask(ctx, task); err != nil {
		return TaskDetail{}, err
	}
	run := workflow.WorkflowRun{
		ID: runID, TaskID: taskID, DefinitionKey: def.Key, DefinitionVer: def.Version,
		Status: workflow.RunRunning, CurrentStep: def.EntryStep,
	}
	if err := s.Store.CreateRun(ctx, run); err != nil {
		return TaskDetail{}, err
	}

	// task.created carries the initial context so the ledger is self-contained.
	if _, err := s.Store.Append(ctx, []workflow.Event{{
		ID: domain.NewEventID(), TaskID: taskID, RunID: runID,
		Type: workflow.EvtTaskCreated, ActorID: requester, Payload: ctxJSON, At: now,
	}}); err != nil {
		return TaskDetail{}, err
	}

	if err := s.Engine.Advance(ctx, runID); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTaskDetail(ctx, taskID)
}

// AdvanceTask drives the engine one push (for agent/system steps that are
// awaiting a manual advance).
func (s *Service) AdvanceTask(ctx context.Context, taskID domain.TaskID) (TaskDetail, error) {
	task, err := s.Store.GetTask(ctx, taskID)
	if err != nil {
		return TaskDetail{}, err
	}
	if task.WorkflowRunID == nil {
		return TaskDetail{}, fmt.Errorf("task has no run")
	}
	if err := s.Engine.Advance(ctx, *task.WorkflowRunID); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTaskDetail(ctx, taskID)
}

// AnswerElicitation signals a Yes/No at an elicitation gate.
func (s *Service) AnswerElicitation(ctx context.Context, taskID domain.TaskID, actorID domain.ActorID, yes bool) (TaskDetail, error) {
	run, step, err := s.currentGate(ctx, taskID, workflow.GateElicitation)
	if err != nil {
		return TaskDetail{}, err
	}
	actor, err := s.Store.GetActor(ctx, actorID)
	if err != nil {
		return TaskDetail{}, err
	}
	if err := s.Engine.SignalElicitation(ctx, run.ID, actor, step, yes); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTaskDetail(ctx, taskID)
}

// Settle signals the awaiting-settlement trigger gate.
func (s *Service) Settle(ctx context.Context, taskID domain.TaskID, actorID domain.ActorID) (TaskDetail, error) {
	run, step, err := s.currentGate(ctx, taskID, workflow.GateTrigger)
	if err != nil {
		return TaskDetail{}, err
	}
	actor, err := s.Store.GetActor(ctx, actorID)
	if err != nil {
		return TaskDetail{}, err
	}
	if err := s.Engine.SignalTrigger(ctx, run.ID, actor, step); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTaskDetail(ctx, taskID)
}

// currentGate returns the run and current step key if the task is parked on a
// gate of the wanted kind.
func (s *Service) currentGate(ctx context.Context, taskID domain.TaskID, want workflow.GateKind) (workflow.WorkflowRun, string, error) {
	task, err := s.Store.GetTask(ctx, taskID)
	if err != nil {
		return workflow.WorkflowRun{}, "", err
	}
	if task.WorkflowRunID == nil {
		return workflow.WorkflowRun{}, "", fmt.Errorf("task has no run")
	}
	run, err := s.Store.LoadRun(ctx, *task.WorkflowRunID)
	if err != nil {
		return workflow.WorkflowRun{}, "", err
	}
	def, ok := s.Registry.Get(run.DefinitionKey, run.DefinitionVer)
	if !ok {
		return workflow.WorkflowRun{}, "", fmt.Errorf("definition not registered")
	}
	step, ok := def.Step(run.CurrentStep)
	if !ok || step.Kind != workflow.KindHumanGate || step.GateKind != want {
		return workflow.WorkflowRun{}, "", fmt.Errorf("task is not awaiting a %s gate (current: %s)", want, run.CurrentStep)
	}
	if run.Status != workflow.RunWaiting {
		return workflow.WorkflowRun{}, "", fmt.Errorf("run is not waiting (status: %s)", run.Status)
	}
	return run, run.CurrentStep, nil
}
