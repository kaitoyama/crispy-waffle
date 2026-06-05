package workflow

import (
	"context"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

// Signals are the resume triggers for a parked run (docs/03 §3). Each appends a
// ledger event and then re-runs Advance, which re-folds and proceeds past the
// gate. The signal's actor is whoever produced it (approver / executor), not
// necessarily the task assignee.

// SignalElicitation records a synchronous Yes/No answer at an elicitation gate.
func (e *Engine) SignalElicitation(ctx context.Context, runID domain.RunID, actor domain.Actor, step string, yes bool) error {
	run, err := e.Runs.LoadRun(ctx, runID)
	if err != nil {
		return err
	}
	if _, err := e.emit(ctx, run, actor, "", EvtElicitAnswered, elicitAnsweredPayload{Step: step, Yes: yes}); err != nil {
		return err
	}
	return e.Advance(ctx, runID)
}

// SignalApproval records an approval decision (the ApprovalRecord itself is
// created by the service layer) and resumes the run.
func (e *Engine) SignalApproval(ctx context.Context, runID domain.RunID, approver domain.Actor, step string, approvalID domain.ApprovalID, decision domain.Decision) error {
	run, err := e.Runs.LoadRun(ctx, runID)
	if err != nil {
		return err
	}
	if _, err := e.emit(ctx, run, approver, "", EvtApprovalDecided, approvalDecidedPayload{
		Step: step, ApprovalID: string(approvalID), Decision: decision,
	}); err != nil {
		return err
	}
	return e.Advance(ctx, runID)
}

// SignalTrigger fires a self-trigger gate (e.g. the "settle" button).
func (e *Engine) SignalTrigger(ctx context.Context, runID domain.RunID, actor domain.Actor, step string) error {
	run, err := e.Runs.LoadRun(ctx, runID)
	if err != nil {
		return err
	}
	if _, err := e.emit(ctx, run, actor, "", EvtTriggerFired, elicitAnsweredPayload{Step: step}); err != nil {
		return err
	}
	return e.Advance(ctx, runID)
}
