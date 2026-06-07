package seed_test

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"testing"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/exec"
	"github.com/kaitoyama/crispy-waffle/internal/seed"
	"github.com/kaitoyama/crispy-waffle/internal/tools"
	"github.com/kaitoyama/crispy-waffle/internal/tools/builtin"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// --- in-memory ports (no SQLite) so the engine logic is tested in isolation ---

type memStore struct {
	mu     sync.Mutex
	events map[domain.RunID][]workflow.Event
	runs   map[domain.RunID]workflow.WorkflowRun
	tasks  map[domain.TaskID]domain.Task
	actors map[domain.ActorID]domain.Actor
}

func newMemStore() *memStore {
	return &memStore{
		events: map[domain.RunID][]workflow.Event{},
		runs:   map[domain.RunID]workflow.WorkflowRun{},
		tasks:  map[domain.TaskID]domain.Task{},
		actors: map[domain.ActorID]domain.Actor{},
	}
}

func (m *memStore) LoadByRun(_ context.Context, runID domain.RunID) ([]workflow.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := append([]workflow.Event(nil), m.events[runID]...)
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out, nil
}

func (m *memStore) Append(_ context.Context, evs []workflow.Event) ([]workflow.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range evs {
		next := int64(len(m.events[evs[i].RunID])) + 1
		evs[i].Seq = next
		m.events[evs[i].RunID] = append(m.events[evs[i].RunID], evs[i])
	}
	return evs, nil
}

func (m *memStore) LoadRun(_ context.Context, id domain.RunID) (workflow.WorkflowRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runs[id], nil
}
func (m *memStore) SaveRun(_ context.Context, r workflow.WorkflowRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[r.ID] = r
	return nil
}
func (m *memStore) LoadTask(_ context.Context, id domain.TaskID) (domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tasks[id], nil
}
func (m *memStore) UpdateTaskStatus(_ context.Context, id domain.TaskID, s domain.TaskStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.tasks[id]
	t.Status = s
	m.tasks[id] = t
	return nil
}
func (m *memStore) ResolveActor(_ context.Context, id domain.ActorID) (domain.Actor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.actors[id], nil
}

func newEngine(m *memStore) *workflow.Engine {
	reg := workflow.NewRegistry()
	reg.Register(seed.ExpenseTransportDefinition())
	tr := tools.NewRegistry()
	builtin.Register(tr)
	return &workflow.Engine{
		Events: m, Runs: m, Tasks: m, Actors: m,
		Registry: reg, Executor: exec.NewExecutor(tr),
	}
}

func setup(t *testing.T) (*memStore, *workflow.Engine, domain.TaskID, domain.RunID) {
	t.Helper()
	m := newMemStore()
	for _, a := range seed.Actors() {
		m.actors[a.ID] = a
	}
	taskID := domain.NewTaskID()
	runID := domain.NewRunID()
	ctxJSON, _ := json.Marshal(map[string]any{"route_from": "北千住", "route_to": "御茶ノ水"})
	m.tasks[taskID] = domain.Task{
		ID: taskID, Title: "交通費精算", Type: "expense.transport",
		RequesterID: seed.ActorTanaka, AssigneeID: seed.ActorAccBot,
		WorkflowRunID: &runID, Context: ctxJSON, Status: seed.StDrafting,
	}
	m.runs[runID] = workflow.WorkflowRun{
		ID: runID, TaskID: taskID, DefinitionKey: "expense.transport", DefinitionVer: 1,
		Status: workflow.RunRunning, CurrentStep: "price_check",
	}
	// task.created marker (carries initial context) so Fold seeds the route
	_, _ = m.Append(context.Background(), []workflow.Event{{
		ID: domain.NewEventID(), TaskID: taskID, RunID: runID, Type: workflow.EvtTaskCreated,
		Payload: ctxJSON,
	}})
	return m, newEngine(m), taskID, runID
}

func TestExpenseFlow_HappyPath(t *testing.T) {
	ctx := context.Background()
	m, eng, taskID, runID := setup(t)

	// 1) Advance: price_check (fare.lookup) runs, then parks at confirm_pre.
	if err := eng.Advance(ctx, runID); err != nil {
		t.Fatalf("advance: %v", err)
	}
	run, _ := m.LoadRun(ctx, runID)
	if run.Status != workflow.RunWaiting || run.CurrentStep != "confirm_pre" {
		t.Fatalf("expected waiting at confirm_pre, got %s/%s", run.Status, run.CurrentStep)
	}
	if got := m.tasks[taskID].Status; got != seed.StConfirmPre {
		t.Fatalf("task status = %s, want ConfirmPre", got)
	}

	// 2) Elicitation Yes -> submit_pre runs -> parks at await_approval.
	if err := eng.SignalElicitation(ctx, runID, m.actors[seed.ActorTanaka], "confirm_pre", true); err != nil {
		t.Fatalf("elicit: %v", err)
	}
	run, _ = m.LoadRun(ctx, runID)
	if run.Status != workflow.RunWaiting || run.CurrentStep != "await_approval" {
		t.Fatalf("expected waiting at await_approval, got %s/%s", run.Status, run.CurrentStep)
	}

	// 3) Approval approved -> parks at awaiting_settlement.
	if err := eng.SignalApproval(ctx, runID, m.actors[seed.ActorAccountant], "await_approval", domain.NewApprovalID(), domain.DecisionApproved); err != nil {
		t.Fatalf("approve: %v", err)
	}
	run, _ = m.LoadRun(ctx, runID)
	if run.CurrentStep != "awaiting_settlement" {
		t.Fatalf("expected awaiting_settlement, got %s", run.CurrentStep)
	}

	// 4) Settle trigger -> payment.execute -> completed/Settled.
	if err := eng.SignalTrigger(ctx, runID, m.actors[seed.ActorTanaka], "awaiting_settlement"); err != nil {
		t.Fatalf("settle: %v", err)
	}
	run, _ = m.LoadRun(ctx, runID)
	if run.Status != workflow.RunCompleted {
		t.Fatalf("expected completed, got %s", run.Status)
	}
	if got := m.tasks[taskID].Status; got != seed.StSettled {
		t.Fatalf("task status = %s, want Settled", got)
	}

	// Idempotency: count payment.execute invocations in the ledger.
	evs, _ := m.LoadByRun(ctx, runID)
	pays := 0
	var amount float64
	for _, e := range evs {
		if e.Type == workflow.EvtToolInvoked {
			var p struct {
				Tool   string         `json:"tool"`
				Output map[string]any `json:"output"`
			}
			_ = json.Unmarshal(e.Payload, &p)
			if p.Tool == "payment.execute" {
				pays++
			}
			if p.Tool == "fare.lookup" {
				amount, _ = p.Output["amount"].(float64)
			}
		}
	}
	if pays != 1 {
		t.Fatalf("expected exactly 1 payment.execute, got %d", pays)
	}
	if amount <= 0 {
		t.Fatalf("expected positive fare amount, got %v", amount)
	}

	// Fold reproducibility: re-folding the ledger yields the terminal state.
	def := seed.ExpenseTransportDefinition()
	st := workflow.Fold(def, evs)
	if !st.Completed || st.TaskStatus != seed.StSettled {
		t.Fatalf("refold mismatch: completed=%v status=%s", st.Completed, st.TaskStatus)
	}
}

func TestExpenseFlow_Rejected(t *testing.T) {
	ctx := context.Background()
	m, eng, taskID, runID := setup(t)
	_ = eng.Advance(ctx, runID)
	_ = eng.SignalElicitation(ctx, runID, m.actors[seed.ActorTanaka], "confirm_pre", true)
	if err := eng.SignalApproval(ctx, runID, m.actors[seed.ActorAccountant], "await_approval", domain.NewApprovalID(), domain.DecisionRejected); err != nil {
		t.Fatalf("reject: %v", err)
	}
	run, _ := m.LoadRun(ctx, runID)
	if run.Status != workflow.RunCompleted || m.tasks[taskID].Status != seed.StRejected {
		t.Fatalf("expected rejected terminal, got %s/%s", run.Status, m.tasks[taskID].Status)
	}
}

func TestExpenseFlow_ElicitationNoLoops(t *testing.T) {
	ctx := context.Background()
	m, eng, _, runID := setup(t)
	_ = eng.Advance(ctx, runID)
	// Say "No": should loop back through price_check and park at confirm_pre again.
	if err := eng.SignalElicitation(ctx, runID, m.actors[seed.ActorTanaka], "confirm_pre", false); err != nil {
		t.Fatalf("elicit no: %v", err)
	}
	run, _ := m.LoadRun(ctx, runID)
	if run.Status != workflow.RunWaiting || run.CurrentStep != "confirm_pre" {
		t.Fatalf("expected re-park at confirm_pre, got %s/%s", run.Status, run.CurrentStep)
	}
	// Then "Yes" proceeds.
	_ = eng.SignalElicitation(ctx, runID, m.actors[seed.ActorTanaka], "confirm_pre", true)
	run, _ = m.LoadRun(ctx, runID)
	if run.CurrentStep != "await_approval" {
		t.Fatalf("expected await_approval after yes, got %s", run.CurrentStep)
	}
}
