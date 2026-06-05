package domain

import (
	"encoding/json"
	"time"
)

// TaskStatus is the per-type state-machine state, mirrored from the folded run
// state onto the Task row as a fast-query projection. Values are workflow
// specific (e.g. "Drafting", "PreApproved", "AwaitingSettlement", "Settled").
type TaskStatus string

// Origin records how a task came to exist.
const (
	OriginManual    = "manual"
	OriginRecurring = "recurring"
	OriginTriggered = "triggered"
)

// Task is the central durable aggregate: one business intent, addressable and
// linkable like a Git issue (docs/02 §2).
type Task struct {
	ID            TaskID          `json:"id"`
	Title         string          `json:"title"`
	Intent        string          `json:"intent"`
	Type          string          `json:"type"` // TaskType.key, e.g. "expense.transport"
	Status        TaskStatus      `json:"status"`
	RequesterID   ActorID         `json:"requester_id"`
	AssigneeID    ActorID         `json:"assignee_id"`
	ParentID      *TaskID         `json:"parent_id,omitempty"`
	WorkflowRunID *RunID          `json:"workflow_run_id,omitempty"`
	Context       json.RawMessage `json:"context"`
	Priority      *int            `json:"priority,omitempty"`
	DueAt         *time.Time      `json:"due_at,omitempty"`
	Origin        string          `json:"origin"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// TaskType is the business template. Adding a new kind of work means adding a
// new TaskType + WorkflowDefinition (+ tools), per docs/02 §3.
type TaskType struct {
	Key                string          `json:"key"`
	DisplayName        string          `json:"display_name"`
	ContextSchema      json.RawMessage `json:"context_schema"`
	DefaultWorkflowKey string          `json:"default_workflow_key"`
	DefaultWorkflowVer int             `json:"default_workflow_ver"`
}
