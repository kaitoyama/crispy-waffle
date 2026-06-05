package workflow

import (
	"encoding/json"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

// RunState is the projection produced by folding the event ledger. It is the
// single source of truth for "where is this task now"; the persisted
// WorkflowRun / Task.Status rows are caches of this (docs/03 §1).
type RunState struct {
	Started     bool
	Completed   bool
	Failed      bool
	CurrentStep string
	TaskStatus  domain.TaskStatus

	// Accumulated typed payload (amount, route, pre_approval_id, settled, ...).
	Context map[string]any
	// Latest decision / answer per gate step.
	Approvals     map[string]domain.Decision
	ElicitAnswers map[string]bool
	Triggers      map[string]bool

	// WaitingOn is set while the current gate is parked, cleared on transition.
	WaitingOn *WaitingOn

	// Entry-scoped flags for the *current* step (reset on every transition).
	// They let the engine tell "fresh entry" from "already executed / already
	// woken", which makes loops (e.g. elicitation "No" → back to price check)
	// behave correctly.
	CurExecuted      bool
	CurGateRequested bool
	CurGateSatisfied bool

	// invokedKeys indexes recorded tool invocations by idempotency key so the
	// engine can skip re-running a side effect (effectively exactly-once).
	invokedKeys map[string]map[string]any
}

// Invoked returns the recorded output for an idempotency key, if the tool was
// already invoked in this run's ledger.
func (s RunState) Invoked(idempotencyKey string) (map[string]any, bool) {
	out, ok := s.invokedKeys[idempotencyKey]
	return out, ok
}

// Status derives the run status from the folded state.
func (s RunState) Status() RunStatus {
	switch {
	case s.Completed:
		return RunCompleted
	case s.Failed:
		return RunFailed
	case s.WaitingOn != nil && !s.CurGateSatisfied:
		return RunWaiting
	default:
		return RunRunning
	}
}

// Fold reconstructs the current state from the ordered event ledger
// (current_state = fold(events), docs/03 §1). Pure: no I/O, fully testable.
func Fold(def WorkflowDefinition, events []Event) RunState {
	s := RunState{
		CurrentStep:   def.EntryStep,
		Context:       map[string]any{},
		Approvals:     map[string]domain.Decision{},
		ElicitAnswers: map[string]bool{},
		Triggers:      map[string]bool{},
		invokedKeys:   map[string]map[string]any{},
	}
	if entry, ok := def.Step(def.EntryStep); ok {
		s.TaskStatus = entry.EntersState
	}

	resetCurrent := func() {
		s.CurExecuted = false
		s.CurGateRequested = false
		s.CurGateSatisfied = false
		s.WaitingOn = nil
	}

	for _, e := range events {
		switch e.Type {
		case EvtTaskCreated:
			s.Started = true
			// The creation event carries the task's initial typed context
			// (e.g. route_from/route_to) so the ledger is self-contained.
			var init map[string]any
			if json.Unmarshal(e.Payload, &init) == nil {
				for k, v := range init {
					s.Context[k] = v
				}
			}

		case EvtStepStarted:
			// informational only

		case EvtToolInvoked:
			var p toolInvokedPayload
			_ = json.Unmarshal(e.Payload, &p)
			for k, v := range p.Output {
				s.Context[k] = v
			}
			if p.IdempotencyKey != "" {
				s.invokedKeys[p.IdempotencyKey] = p.Output
			}

		case EvtStepCompleted:
			var p transitionPayload // reuse {from,...}; we store step in From
			_ = json.Unmarshal(e.Payload, &p)
			if p.From == s.CurrentStep {
				s.CurExecuted = true
			}

		case EvtStepFailed:
			// failure handled by run.failed; keep informational here

		case EvtElicitAnswered:
			var p elicitAnsweredPayload
			_ = json.Unmarshal(e.Payload, &p)
			s.ElicitAnswers[p.Step] = p.Yes
			if p.Step == s.CurrentStep {
				s.CurGateSatisfied = true
			}

		case EvtTriggerFired:
			var p elicitAnsweredPayload // {step,...}
			_ = json.Unmarshal(e.Payload, &p)
			s.Triggers[p.Step] = true
			if p.Step == s.CurrentStep {
				s.CurGateSatisfied = true
			}

		case EvtApprovalRequested:
			var p approvalDecidedPayload
			_ = json.Unmarshal(e.Payload, &p)
			if p.Step == s.CurrentStep {
				s.CurGateRequested = true
			}

		case EvtApprovalDecided:
			var p approvalDecidedPayload
			_ = json.Unmarshal(e.Payload, &p)
			s.Approvals[p.Step] = p.Decision
			if p.Step == s.CurrentStep {
				s.CurGateSatisfied = true
			}

		case EvtRunWaiting:
			var p waitingPayload
			_ = json.Unmarshal(e.Payload, &p)
			s.CurGateRequested = true
			s.WaitingOn = &WaitingOn{Kind: p.Kind, Ref: p.Ref}

		case EvtStateTransitioned:
			var p transitionPayload
			_ = json.Unmarshal(e.Payload, &p)
			s.CurrentStep = p.To
			if st, ok := def.Step(p.To); ok {
				s.TaskStatus = st.EntersState
			}
			resetCurrent()

		case EvtRunCompleted:
			s.Completed = true

		case EvtRunFailed:
			s.Failed = true
		}
	}
	return s
}
