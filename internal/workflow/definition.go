package workflow

import "github.com/kaitoyama/crispy-waffle/internal/domain"

// StepKind separates the four ways a vertex is satisfied (docs/03 §2). The
// distinction between agent_action and human_gate is the crux: it marks where
// the system runs automatically vs. where it pauses for a person.
type StepKind string

const (
	KindAgentAction  StepKind = "agent_action"
	KindHumanGate    StepKind = "human_gate"
	KindSystemAction StepKind = "system_action"
	KindSubWorkflow  StepKind = "sub_workflow"
)

// GateKind sub-types a human_gate (docs/08 §4.1): an asynchronous approval by
// *another* actor, a synchronous elicitation to the *executor*, or a self
// trigger (e.g. the "settle" button).
type GateKind string

const (
	GateApproval    GateKind = "approval"
	GateElicitation GateKind = "elicitation"
	GateTrigger     GateKind = "trigger"
)

// ToolBinding is the flow-designer's allowed tool range for a step. The actual
// permission is binding ∩ actor.capability, checked per invocation (docs/02 §5).
type ToolBinding struct {
	ToolKey string         `json:"tool_key"`
	Scope   map[string]any `json:"scope,omitempty"`
}

// Transition is a conditional edge to the next step. Guard is a mini-expression
// evaluated over the folded RunState (see guard.go).
type Transition struct {
	To    string `json:"to"`
	Guard string `json:"guard,omitempty"` // empty == always
}

// StepDef is a vertex in the workflow graph.
type StepDef struct {
	Key          string            `json:"key"`
	Kind         StepKind          `json:"kind"`
	Title        string            `json:"title"`
	RequiredRole string            `json:"required_role,omitempty"`
	GateKind     GateKind          `json:"gate_kind,omitempty"`
	ToolBindings []ToolBinding     `json:"tool_bindings,omitempty"`
	Transitions  []Transition      `json:"transitions,omitempty"`
	EntersState  domain.TaskStatus `json:"enters_state"`
	Terminal     bool              `json:"terminal,omitempty"`
}

// WorkflowDefinition is a reusable, versioned flow. Running instances pin the
// version for reproducibility (docs/02 §11-6).
type WorkflowDefinition struct {
	Key         string             `json:"key"`
	Version     int                `json:"version"`
	DisplayName string             `json:"display_name"`
	EntryStep   string             `json:"entry_step"`
	Steps       map[string]StepDef `json:"steps"`
}

func (d WorkflowDefinition) Step(key string) (StepDef, bool) {
	s, ok := d.Steps[key]
	return s, ok
}
