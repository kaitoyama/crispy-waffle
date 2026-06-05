package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

// --- ports the engine depends on (implemented by store/exec) ---

// EventStore is the append-only ledger.
type EventStore interface {
	LoadByRun(ctx context.Context, runID domain.RunID) ([]Event, error)
	// Append persists events, assigning Seq (monotonic per run) and returning them.
	Append(ctx context.Context, events []Event) ([]Event, error)
}

// RunStore loads and saves the run projection.
type RunStore interface {
	LoadRun(ctx context.Context, runID domain.RunID) (WorkflowRun, error)
	SaveRun(ctx context.Context, run WorkflowRun) error
}

// TaskStore loads a task and updates its status projection.
type TaskStore interface {
	LoadTask(ctx context.Context, taskID domain.TaskID) (domain.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID domain.TaskID, status domain.TaskStatus) error
}

// ActorResolver resolves an actor by id (for capabilities/roles).
type ActorResolver interface {
	ResolveActor(ctx context.Context, id domain.ActorID) (domain.Actor, error)
}

// ExecInput is what a step executor receives.
type ExecInput struct {
	Task           domain.Task
	Actor          domain.Actor
	Step           StepDef
	Context        map[string]any
	IdempotencyKey string
}

// ExecResult is the outcome of executing one step's tool.
type ExecResult struct {
	Tool           string
	CapabilityUsed string
	Output         map[string]any
	Denied         bool
	DenyReason     string
}

// StepExecutor runs agent_action / system_action steps. The deterministic mock
// and a future real-LLM executor implement the same interface — agents are not
// privileged, they are just one executor (user requirement / docs/01).
type StepExecutor interface {
	Execute(ctx context.Context, in ExecInput) (ExecResult, error)
}

// Engine drives a run forward until it waits or completes.
type Engine struct {
	Events   EventStore
	Runs     RunStore
	Tasks    TaskStore
	Actors   ActorResolver
	Registry *Registry
	Executor StepExecutor
}

const maxIterations = 200

var ErrNoTransition = errors.New("workflow: no satisfied transition from step")

// Advance loops the run forward, executing automatic steps and parking at human
// gates, until the run reaches a waiting/terminal state (docs/03 §3). It is
// re-entrant: calling it again after a signal resumes from the folded state.
func (e *Engine) Advance(ctx context.Context, runID domain.RunID) error {
	for i := 0; i < maxIterations; i++ {
		run, err := e.Runs.LoadRun(ctx, runID)
		if err != nil {
			return err
		}
		def, ok := e.Registry.Get(run.DefinitionKey, run.DefinitionVer)
		if !ok {
			return fmt.Errorf("workflow: definition %s@%d not registered", run.DefinitionKey, run.DefinitionVer)
		}
		events, err := e.Events.LoadByRun(ctx, runID)
		if err != nil {
			return err
		}
		st := Fold(def, events)

		if st.Completed || st.Failed {
			return e.persist(ctx, run, st)
		}

		step, ok := def.Step(st.CurrentStep)
		if !ok {
			return fmt.Errorf("workflow: current step %q not in definition", st.CurrentStep)
		}

		task, err := e.Tasks.LoadTask(ctx, run.TaskID)
		if err != nil {
			return err
		}
		actor, err := e.Actors.ResolveActor(ctx, task.AssigneeID)
		if err != nil {
			return err
		}

		if step.Terminal {
			if _, err := e.emit(ctx, run, actor, "", EvtRunCompleted, nil); err != nil {
				return err
			}
			continue
		}

		switch step.Kind {
		case KindAgentAction, KindSystemAction:
			if !st.CurExecuted {
				if err := e.executeStep(ctx, run, task, actor, step, st); err != nil {
					return err
				}
				continue // reload: CurExecuted now true
			}
			next, err := e.firstTransition(step, st, actor)
			if err != nil {
				if _, e2 := e.emit(ctx, run, actor, "", EvtRunFailed, map[string]any{"reason": err.Error()}); e2 != nil {
					return e2
				}
				continue
			}
			if err := e.transition(ctx, run, actor, step.Key, next); err != nil {
				return err
			}

		case KindHumanGate:
			if st.CurGateSatisfied {
				next, err := e.firstTransition(step, st, actor)
				if err != nil {
					if _, e2 := e.emit(ctx, run, actor, "", EvtRunFailed, map[string]any{"reason": err.Error()}); e2 != nil {
						return e2
					}
					continue
				}
				if err := e.transition(ctx, run, actor, step.Key, next); err != nil {
					return err
				}
				continue
			}
			if !st.CurGateRequested {
				if err := e.enterWait(ctx, run, actor, step); err != nil {
					return err
				}
				continue // reload: will fall through to persist+return
			}
			return e.persist(ctx, run, st)

		case KindSubWorkflow:
			// MVP: pass-through (a full child-run fan-out is future work).
			next, err := e.firstTransition(step, st, actor)
			if err != nil {
				if _, e2 := e.emit(ctx, run, actor, "", EvtRunFailed, map[string]any{"reason": err.Error()}); e2 != nil {
					return e2
				}
				continue
			}
			if err := e.transition(ctx, run, actor, step.Key, next); err != nil {
				return err
			}

		default:
			return fmt.Errorf("workflow: unknown step kind %q", step.Kind)
		}
	}
	return fmt.Errorf("workflow: advance exceeded %d iterations (possible cycle)", maxIterations)
}

// executeStep runs one agent/system step idempotently and records the result.
func (e *Engine) executeStep(ctx context.Context, run WorkflowRun, task domain.Task, actor domain.Actor, step StepDef, st RunState) error {
	key := idempotencyKey(task.ID, step.Key)

	if _, done := st.Invoked(key); done {
		// Side effect already recorded in the ledger: skip re-execution
		// (effectively exactly-once) and just mark the step completed.
		_, err := e.emit(ctx, run, actor, "", EvtStepCompleted, transitionPayload{From: step.Key})
		return err
	}

	if _, err := e.emit(ctx, run, actor, "", EvtStepStarted, transitionPayload{From: step.Key}); err != nil {
		return err
	}

	res, err := e.Executor.Execute(ctx, ExecInput{
		Task: task, Actor: actor, Step: step, Context: st.Context, IdempotencyKey: key,
	})
	if err != nil {
		_, _ = e.emit(ctx, run, actor, "", EvtStepFailed, map[string]any{"step": step.Key, "error": err.Error()})
		_, e2 := e.emit(ctx, run, actor, "", EvtRunFailed, map[string]any{"reason": err.Error()})
		return e2
	}
	if res.Denied {
		_, _ = e.emit(ctx, run, actor, "", EvtToolDenied, map[string]any{"tool": res.Tool, "reason": res.DenyReason})
		_, _ = e.emit(ctx, run, actor, "", EvtStepFailed, map[string]any{"step": step.Key})
		_, e2 := e.emit(ctx, run, actor, "", EvtRunFailed, map[string]any{"reason": "authorization denied: " + res.DenyReason})
		return e2
	}

	if _, err := e.emitCap(ctx, run, actor, res.CapabilityUsed, EvtToolInvoked, toolInvokedPayload{
		Tool: res.Tool, IdempotencyKey: key, Output: res.Output,
	}); err != nil {
		return err
	}
	_, err = e.emit(ctx, run, actor, "", EvtStepCompleted, transitionPayload{From: step.Key})
	return err
}

func (e *Engine) enterWait(ctx context.Context, run WorkflowRun, actor domain.Actor, step StepDef) error {
	if _, err := e.emit(ctx, run, actor, "", EvtStepStarted, transitionPayload{From: step.Key}); err != nil {
		return err
	}
	if step.GateKind == GateApproval {
		if _, err := e.emit(ctx, run, actor, "", EvtApprovalRequested, approvalDecidedPayload{Step: step.Key}); err != nil {
			return err
		}
	}
	kind := string(step.GateKind)
	if kind == "" {
		kind = "trigger"
	}
	_, err := e.emit(ctx, run, actor, "", EvtRunWaiting, waitingPayload{Kind: kind, Ref: step.Key})
	return err
}

func (e *Engine) transition(ctx context.Context, run WorkflowRun, actor domain.Actor, from, to string) error {
	_, err := e.emit(ctx, run, actor, "", EvtStateTransitioned, transitionPayload{From: from, To: to})
	return err
}

// firstTransition returns the target of the first transition whose guard holds.
func (e *Engine) firstTransition(step StepDef, st RunState, actor domain.Actor) (string, error) {
	for _, t := range step.Transitions {
		if EvalGuard(t.Guard, st, actor) {
			return t.To, nil
		}
	}
	return "", fmt.Errorf("%w: step=%s", ErrNoTransition, step.Key)
}

func (e *Engine) persist(ctx context.Context, run WorkflowRun, st RunState) error {
	run.Status = st.Status()
	run.CurrentStep = st.CurrentStep
	run.WaitingOn = st.WaitingOn
	if err := e.Runs.SaveRun(ctx, run); err != nil {
		return err
	}
	return e.Tasks.UpdateTaskStatus(ctx, run.TaskID, st.TaskStatus)
}

func (e *Engine) emit(ctx context.Context, run WorkflowRun, actor domain.Actor, cap string, t EventType, payload any) ([]Event, error) {
	return e.emitCap(ctx, run, actor, cap, t, payload)
}

func (e *Engine) emitCap(ctx context.Context, run WorkflowRun, actor domain.Actor, cap string, t EventType, payload any) ([]Event, error) {
	ev := Event{
		ID:             domain.NewEventID(),
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Type:           t,
		ActorID:        actor.ID,
		CapabilityUsed: cap,
		At:             time.Now().UTC(),
	}
	if payload != nil {
		ev.Payload = mustJSON(payload)
	}
	return e.Events.Append(ctx, []Event{ev})
}

func idempotencyKey(taskID domain.TaskID, stepKey string) string {
	return fmt.Sprintf("%s:%s", taskID, stepKey)
}
