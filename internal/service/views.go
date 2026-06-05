package service

import (
	"context"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// TaskSummary is a compact task row for lists and graph nodes.
type TaskSummary struct {
	ID         domain.TaskID     `json:"id"`
	Title      string            `json:"title"`
	Type       string            `json:"type"`
	Status     domain.TaskStatus `json:"status"`
	AssigneeID domain.ActorID    `json:"assignee_id"`
	UpdatedAt  string            `json:"updated_at"`
}

func summarize(t domain.Task) TaskSummary {
	return TaskSummary{
		ID: t.ID, Title: t.Title, Type: t.Type, Status: t.Status,
		AssigneeID: t.AssigneeID, UpdatedAt: t.UpdatedAt.Format("2006-01-02 15:04"),
	}
}

// StepView describes the current vertex of the workflow.
type StepView struct {
	Key         string            `json:"key"`
	Kind        workflow.StepKind `json:"kind"`
	GateKind    workflow.GateKind `json:"gate_kind,omitempty"`
	Title       string            `json:"title"`
	EntersState domain.TaskStatus `json:"enters_state"`
}

// TaskDetail is the full assembled view for the detail page.
type TaskDetail struct {
	Task             domain.Task             `json:"task"`
	Run              *workflow.WorkflowRun   `json:"run"`
	Definition       *workflow.WorkflowDefinition `json:"definition,omitempty"`
	CurrentStep      *StepView               `json:"current_step,omitempty"`
	AvailableActions []string                `json:"available_actions"`
	Events           []workflow.Event        `json:"events"`
	Links            []domain.TaskLink       `json:"links"`
	Nodes            []TaskSummary           `json:"nodes"`
	Approvals        []domain.ApprovalRecord `json:"approvals"`
}

// ListTasks returns task summaries for the dashboard.
func (s *Service) ListTasks(ctx context.Context) ([]TaskSummary, error) {
	tasks, err := s.Store.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, summarize(t))
	}
	return out, nil
}

// GetTaskDetail assembles task + run + timeline + graph links + approvals.
func (s *Service) GetTaskDetail(ctx context.Context, taskID domain.TaskID) (TaskDetail, error) {
	task, err := s.Store.GetTask(ctx, taskID)
	if err != nil {
		return TaskDetail{}, err
	}
	d := TaskDetail{Task: task, AvailableActions: []string{}, Events: []workflow.Event{}, Links: []domain.TaskLink{}, Nodes: []TaskSummary{}, Approvals: []domain.ApprovalRecord{}}

	if task.WorkflowRunID != nil {
		run, err := s.Store.LoadRun(ctx, *task.WorkflowRunID)
		if err != nil {
			return TaskDetail{}, err
		}
		d.Run = &run
		if def, ok := s.Registry.Get(run.DefinitionKey, run.DefinitionVer); ok {
			d.Definition = &def
			if step, ok := def.Step(run.CurrentStep); ok {
				d.CurrentStep = &StepView{
					Key: step.Key, Kind: step.Kind, GateKind: step.GateKind,
					Title: step.Title, EntersState: step.EntersState,
				}
				d.AvailableActions = availableActions(run, step)
			}
		}
		events, err := s.Store.LoadByTask(ctx, taskID)
		if err != nil {
			return TaskDetail{}, err
		}
		if events != nil {
			d.Events = events
		}
	}

	links, err := s.Store.ListLinksForTask(ctx, taskID)
	if err != nil {
		return TaskDetail{}, err
	}
	if links != nil {
		d.Links = links
	}

	// Graph nodes: this task plus every linked task.
	seen := map[domain.TaskID]bool{taskID: true}
	d.Nodes = append(d.Nodes, summarize(task))
	for _, l := range links {
		other := l.TargetID
		if other == taskID {
			other = l.SourceID
		}
		if seen[other] {
			continue
		}
		seen[other] = true
		if ot, err := s.Store.GetTask(ctx, other); err == nil {
			d.Nodes = append(d.Nodes, summarize(ot))
		}
	}

	approvals, err := s.Store.ListApprovalsByTask(ctx, taskID)
	if err != nil {
		return TaskDetail{}, err
	}
	if approvals != nil {
		d.Approvals = approvals
	}
	return d, nil
}

// availableActions derives the contextual buttons the UI should offer, from the
// run status and the current step (docs/07 — confirm vs approve vs trigger).
func availableActions(run workflow.WorkflowRun, step workflow.StepDef) []string {
	switch run.Status {
	case workflow.RunRunning:
		return []string{"advance"}
	case workflow.RunWaiting:
		if step.Kind == workflow.KindHumanGate {
			switch step.GateKind {
			case workflow.GateElicitation:
				return []string{"elicit"}
			case workflow.GateApproval:
				return []string{"awaiting_approval"}
			case workflow.GateTrigger:
				return []string{"settle"}
			}
		}
	}
	return []string{}
}
