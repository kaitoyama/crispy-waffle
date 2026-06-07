// Package tools defines the extension point for capabilities a workflow step
// can invoke. A tool is a small, self-describing unit authored in code:
// implement the Tool interface and register it once (see internal/tools/builtin).
// Everything else — the catalog API, the flow-builder tool picker, authorization
// and execution — is derived from that single registration. There is no
// data-driven/HTTP tool builder; new tools are added by writing Go.
package tools

import "context"

// SideEffectClass classifies a tool's blast radius (docs/04 §3). It governs
// authorization matching and is advisory to the flow designer (irreversible
// tools should be preceded by a confirmation/approval gate).
type SideEffectClass string

const (
	ReadOnly             SideEffectClass = "read_only"
	ReversibleWrite      SideEffectClass = "reversible_write"
	IrreversibleMonetary SideEffectClass = "irreversible_or_monetary"
)

// Field documents one declared input or output of a tool, for validation and UI.
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string | number | boolean
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// Spec is a tool's self-description. It is the single source of truth surfaced
// to the catalog API and the flow builder.
type Spec struct {
	Key             string          `json:"key"`
	DisplayName     string          `json:"display_name"`
	SideEffectClass SideEffectClass `json:"side_effect_class"`
	Description     string          `json:"description,omitempty"`
	Inputs          []Field         `json:"inputs,omitempty"`
	Outputs         []Field         `json:"outputs,omitempty"`
	// ScopeDimensions are the context keys the authorization gate bounds against
	// capability scope caps (e.g. "amount" vs capability {amount_cap: 20000}).
	ScopeDimensions []string `json:"scope_dimensions,omitempty"`
}

// TaskMeta is the read-only task context handed to a tool (kept free of the
// domain package so tools stay dependency-light).
type TaskMeta struct {
	ID    string
	Type  string
	Title string
}

// Input is what a tool receives when invoked.
type Input struct {
	Context        map[string]any // accumulated task context (route, amount, ...)
	Task           TaskMeta
	IdempotencyKey string // stable per (task, step); use for idempotent effects
}

// Tool is the unit of extension. Implement Spec() (metadata) and Execute()
// (the effect), then register the value in internal/tools/builtin.
type Tool interface {
	Spec() Spec
	Execute(ctx context.Context, in Input) (map[string]any, error)
}
