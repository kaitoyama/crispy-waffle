package workflow

import (
	"encoding/json"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

// EventType enumerates the append-only ledger entries. The ledger is the source
// of truth; current state = Fold(events) (docs/02 §8, docs/03 §1).
type EventType string

const (
	EvtTaskCreated       EventType = "task.created"
	EvtStepStarted       EventType = "step.started"
	EvtToolInvoked       EventType = "tool.invoked"
	EvtToolDenied        EventType = "tool.denied"
	EvtElicitRequested   EventType = "elicitation.requested"
	EvtElicitAnswered    EventType = "elicitation.answered"
	EvtTriggerFired      EventType = "trigger.fired"
	EvtApprovalRequested EventType = "approval.requested"
	EvtApprovalDecided   EventType = "approval.decided"
	EvtStateTransitioned EventType = "state.transitioned"
	EvtStepCompleted     EventType = "step.completed"
	EvtStepFailed        EventType = "step.failed"
	EvtRunWaiting        EventType = "run.waiting"
	EvtRunCompleted      EventType = "run.completed"
	EvtRunFailed         EventType = "run.failed"
)

// Event is one immutable ledger entry. Seq is a monotonic per-run ordering.
type Event struct {
	ID             domain.EventID  `json:"id"`
	Seq            int64           `json:"seq"`
	TaskID         domain.TaskID   `json:"task_id"`
	RunID          domain.RunID    `json:"run_id"`
	Type           EventType       `json:"type"`
	ActorID        domain.ActorID  `json:"actor_id"`
	CapabilityUsed string          `json:"capability_used,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	At             time.Time       `json:"at"`
}

// --- typed payload helpers (encoded into Event.Payload) ---

// transitionPayload accompanies EvtStateTransitioned.
type transitionPayload struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// toolInvokedPayload accompanies EvtToolInvoked. Output is merged into the
// folded context so downstream guards can read it.
type toolInvokedPayload struct {
	Tool           string         `json:"tool"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Output         map[string]any `json:"output,omitempty"`
}

// elicitAnsweredPayload accompanies EvtElicitAnswered.
type elicitAnsweredPayload struct {
	Step string `json:"step"`
	Yes  bool   `json:"yes"`
}

// approvalDecidedPayload accompanies EvtApprovalDecided.
type approvalDecidedPayload struct {
	Step       string          `json:"step"`
	ApprovalID string          `json:"approval_id"`
	Decision   domain.Decision `json:"decision"`
}

// waitingPayload accompanies EvtRunWaiting.
type waitingPayload struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err) // payloads are internal, controlled structs
	}
	return b
}
