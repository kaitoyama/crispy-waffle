package domain

import (
	"encoding/json"
	"time"
)

// Decision is the outcome of an approval gate (docs/02 §7).
type Decision string

const (
	DecisionApproved         Decision = "approved"
	DecisionRejected         Decision = "rejected"
	DecisionChangesRequested Decision = "changes_requested"
)

func ValidDecision(d Decision) bool {
	switch d {
	case DecisionApproved, DecisionRejected, DecisionChangesRequested:
		return true
	}
	return false
}

// ApprovalRecord is the structured, standalone decision produced at a
// human_gate. It is NOT a chat message: the approver sees only minimal context
// (docs/02 §7, docs/07 §3.1).
type ApprovalRecord struct {
	ID           ApprovalID      `json:"id"`
	TargetTaskID TaskID          `json:"target_task_id"`
	TargetStepID string          `json:"target_step_id"` // step key being gated
	ApproverID   ActorID         `json:"approver_id"`
	Decision     Decision        `json:"decision"`
	Conditions   json.RawMessage `json:"conditions,omitempty"`
	Rationale    string          `json:"rationale"`
	DecidedAt    time.Time       `json:"decided_at"`
}
