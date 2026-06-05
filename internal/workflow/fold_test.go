package workflow

import (
	"testing"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

// miniDef is a tiny two-step workflow used to unit-test Fold in isolation.
func miniDef() WorkflowDefinition {
	return WorkflowDefinition{
		Key: "mini", Version: 1, EntryStep: "a",
		Steps: map[string]StepDef{
			"a": {Key: "a", Kind: KindAgentAction, EntersState: "A",
				Transitions: []Transition{{To: "b"}}},
			"b": {Key: "b", Kind: KindSystemAction, EntersState: "B", Terminal: true},
		},
	}
}

func TestFold_SeedsContextAndTracksStep(t *testing.T) {
	def := miniDef()
	events := []Event{
		{Seq: 1, Type: EvtTaskCreated, Payload: mustJSON(map[string]any{"amount": 1280.0})},
		{Seq: 2, Type: EvtStepStarted, Payload: mustJSON(transitionPayload{From: "a"})},
		{Seq: 3, Type: EvtToolInvoked, Payload: mustJSON(toolInvokedPayload{Tool: "x", Output: map[string]any{"checked": true}})},
		{Seq: 4, Type: EvtStepCompleted, Payload: mustJSON(transitionPayload{From: "a"})},
	}
	st := Fold(def, events)

	if st.CurrentStep != "a" || !st.CurExecuted {
		t.Fatalf("expected current=a executed, got %s exec=%v", st.CurrentStep, st.CurExecuted)
	}
	if st.Context["amount"] != 1280.0 {
		t.Fatalf("context.amount=%v want 1280", st.Context["amount"])
	}
	if st.Context["checked"] != true {
		t.Fatalf("tool output not merged into context")
	}
	if st.Status() != RunRunning {
		t.Fatalf("status=%v want running", st.Status())
	}
}

func TestFold_TransitionResetsAndCompletes(t *testing.T) {
	def := miniDef()
	events := []Event{
		{Seq: 1, Type: EvtTaskCreated},
		{Seq: 2, Type: EvtStepCompleted, Payload: mustJSON(transitionPayload{From: "a"})},
		{Seq: 3, Type: EvtStateTransitioned, Payload: mustJSON(transitionPayload{From: "a", To: "b"})},
		{Seq: 4, Type: EvtRunCompleted},
	}
	st := Fold(def, events)
	if st.CurrentStep != "b" {
		t.Fatalf("current=%s want b", st.CurrentStep)
	}
	if st.CurExecuted {
		t.Fatalf("CurExecuted should reset after transition")
	}
	if st.TaskStatus != "B" {
		t.Fatalf("task status=%s want B", st.TaskStatus)
	}
	if st.Status() != RunCompleted {
		t.Fatalf("status=%v want completed", st.Status())
	}
}

func TestEvalGuard(t *testing.T) {
	s := RunState{
		Context:       map[string]any{"amount": 1500.0},
		Approvals:     map[string]domain.Decision{"g": domain.DecisionApproved},
		ElicitAnswers: map[string]bool{"e": true},
		Triggers:      map[string]bool{"t": true},
	}
	actor := domain.Actor{Roles: []string{"accounting"}}
	cases := []struct {
		guard string
		want  bool
	}{
		{"", true},
		{"context.amount > 0", true},
		{"context.amount <= 20000", true},
		{"context.amount > 2000", false},
		{`approval_approved("g")`, true},
		{`approval_rejected("g")`, false},
		{`elicit_yes("e")`, true},
		{`elicit_no("e")`, false},
		{`trigger("t")`, true},
		{`actor.role == "accounting"`, true},
		{`actor.role == "requester"`, false},
		{`context.amount > 0 && context.amount <= 20000`, true},
	}
	for _, c := range cases {
		if got := EvalGuard(c.guard, s, actor); got != c.want {
			t.Errorf("guard %q = %v, want %v", c.guard, got, c.want)
		}
	}
}
