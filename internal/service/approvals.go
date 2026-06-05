package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// ApprovalRequest is the minimal-context approval surface (docs/07 §3.1): the
// approver sees what they need to decide, not a conversation.
type ApprovalRequest struct {
	TaskID    domain.TaskID  `json:"task_id"`
	RunID     domain.RunID   `json:"run_id"`
	StepKey   string         `json:"step_key"`
	Title     string         `json:"title"`
	Type      string         `json:"type"`
	Requester domain.ActorID `json:"requester_id"`
	Context   map[string]any `json:"context"`
}

// ListPendingApprovals returns every task currently parked on an approval gate.
func (s *Service) ListPendingApprovals(ctx context.Context) ([]ApprovalRequest, error) {
	runs, err := s.Store.ListWaitingApprovalRuns(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ApprovalRequest, 0, len(runs))
	for _, r := range runs {
		task, err := s.Store.GetTask(ctx, r.TaskID)
		if err != nil {
			continue
		}
		var c map[string]any
		_ = json.Unmarshal(task.Context, &c)
		step := r.CurrentStep
		if r.WaitingOn != nil {
			step = r.WaitingOn.Ref
		}
		out = append(out, ApprovalRequest{
			TaskID: task.ID, RunID: r.ID, StepKey: step, Title: task.Title,
			Type: task.Type, Requester: task.RequesterID, Context: c,
		})
	}
	return out, nil
}

// SubmitDecision records an ApprovalRecord and signals the run to resume
// (docs/03 §3, docs/07 §3). The approval-request id is the task id.
func (s *Service) SubmitDecision(ctx context.Context, taskID domain.TaskID, approverID domain.ActorID, decision domain.Decision, rationale string, conditions json.RawMessage) (TaskDetail, error) {
	if !domain.ValidDecision(decision) {
		return TaskDetail{}, fmt.Errorf("invalid decision %q", decision)
	}
	run, step, err := s.currentGate(ctx, taskID, workflow.GateApproval)
	if err != nil {
		return TaskDetail{}, err
	}
	approver, err := s.Store.GetActor(ctx, approverID)
	if err != nil {
		return TaskDetail{}, err
	}
	rec := domain.ApprovalRecord{
		ID: domain.NewApprovalID(), TargetTaskID: taskID, TargetStepID: step,
		ApproverID: approverID, Decision: decision, Conditions: conditions,
		Rationale: rationale, DecidedAt: time.Now().UTC(),
	}
	if err := s.Store.CreateApproval(ctx, rec); err != nil {
		return TaskDetail{}, err
	}
	if err := s.Engine.SignalApproval(ctx, run.ID, approver, step, rec.ID, decision); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTaskDetail(ctx, taskID)
}
