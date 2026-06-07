// Package exec implements workflow.StepExecutor. It is the seam where a step's
// bound tool is actually run: load the tool from the registry, check
// authorization (binding ∩ capability), then execute. Per the project's framing
// an "agent" is not privileged — it is just one executor, and every agent/system
// step is driven uniformly through here. Tools are added in code
// (internal/tools/builtin), not via a data-driven builder.
package exec

import (
	"context"
	"fmt"

	"github.com/kaitoyama/crispy-waffle/internal/authz"
	"github.com/kaitoyama/crispy-waffle/internal/tools"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// Executor runs a step's single bound tool from the registry.
type Executor struct {
	registry *tools.Registry
}

// NewExecutor wires the executor to the tool registry.
func NewExecutor(registry *tools.Registry) *Executor {
	return &Executor{registry: registry}
}

// Execute satisfies workflow.StepExecutor.
func (e *Executor) Execute(ctx context.Context, in workflow.ExecInput) (workflow.ExecResult, error) {
	if len(in.Step.ToolBindings) == 0 {
		return workflow.ExecResult{}, fmt.Errorf("exec: step %q has no tool binding", in.Step.Key)
	}
	// MVP: one tool per agent/system step.
	toolKey := in.Step.ToolBindings[0].ToolKey

	tool, ok := e.registry.Get(toolKey)
	if !ok {
		return workflow.ExecResult{}, fmt.Errorf("exec: no tool registered for %q", toolKey)
	}
	spec := tool.Spec()

	// Authorization: binding ∩ capability, bounding the tool's scope dimensions
	// (e.g. amount) against the actor's capability caps.
	dec := authz.Check(in.Step, in.Actor, toolKey, scopeArgs(spec, in.Context))
	if !dec.Allowed {
		return workflow.ExecResult{Tool: toolKey, Denied: true, DenyReason: dec.Reason}, nil
	}

	out, err := tool.Execute(ctx, tools.Input{
		Context: in.Context,
		Task: tools.TaskMeta{
			ID: string(in.Task.ID), Type: in.Task.Type, Title: in.Task.Title,
		},
		IdempotencyKey: in.IdempotencyKey,
	})
	if err != nil {
		return workflow.ExecResult{}, err
	}
	return workflow.ExecResult{Tool: toolKey, CapabilityUsed: dec.CapabilityUsed, Output: out}, nil
}

// scopeArgs reads the numeric context values named by the tool's scope
// dimensions so the authz gate can bound them against capability caps.
func scopeArgs(spec tools.Spec, c map[string]any) map[string]float64 {
	args := map[string]float64{}
	for _, dim := range spec.ScopeDimensions {
		if v, ok := toFloat(c[dim]); ok {
			args[dim] = v
		}
	}
	return args
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
