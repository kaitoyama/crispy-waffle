package workflow

import "github.com/kaitoyama/crispy-waffle/internal/domain"

type RunStatus string

const (
	RunRunning   RunStatus = "running"
	RunWaiting   RunStatus = "waiting"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
	RunCancelled RunStatus = "cancelled"
)

type StepRunStatus string

const (
	StepPending StepRunStatus = "pending"
	StepRunning StepRunStatus = "running"
	StepWaiting StepRunStatus = "waiting"
	StepDone    StepRunStatus = "done"
	StepFailed  StepRunStatus = "failed"
)

// WaitingOn describes why a run is paused, so a signal can be matched to it
// (docs/03 §3).
type WaitingOn struct {
	Kind string `json:"kind"` // "approval" | "elicitation" | "trigger" | "timer"
	Ref  string `json:"ref"`  // step key (or approval id) the signal must target
}

// WorkflowRun is the execution instance bound to a task. Its Status / CurrentStep
// / WaitingOn are projections of the event fold (state.go) cached for fast query.
type WorkflowRun struct {
	ID            domain.RunID  `json:"id"`
	TaskID        domain.TaskID `json:"task_id"`
	DefinitionKey string        `json:"definition_key"`
	DefinitionVer int           `json:"definition_ver"`
	Status        RunStatus     `json:"status"`
	CurrentStep   string        `json:"current_step"`
	WaitingOn     *WaitingOn    `json:"waiting_on,omitempty"`
}
