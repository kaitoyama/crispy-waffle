// Package exec implements workflow.StepExecutor. It is the pluggable seam where
// a step's bound tool is actually run. The built-in executor is deterministic
// (no API keys, fully offline); a real-LLM executor would implement the same
// interface. Per the project's framing, an "agent" is not privileged — it is
// just one executor, and every step is driven uniformly through here.
package exec

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/kaitoyama/crispy-waffle/internal/authz"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// toolFunc computes a deterministic output from the step context.
type toolFunc func(in workflow.ExecInput) (map[string]any, error)

// Executor dispatches a step's single bound tool through the authz gate, then
// runs the deterministic tool implementation.
type Executor struct {
	tools map[string]toolFunc
}

// NewExecutor wires the built-in deterministic tool implementations.
func NewExecutor() *Executor {
	e := &Executor{tools: map[string]toolFunc{}}
	e.tools["fare.lookup"] = fareLookup
	e.tools["pre_application.submit"] = preApplicationSubmit
	e.tools["payment.execute"] = paymentExecute
	return e
}

// Execute satisfies workflow.StepExecutor.
func (e *Executor) Execute(ctx context.Context, in workflow.ExecInput) (workflow.ExecResult, error) {
	if len(in.Step.ToolBindings) == 0 {
		return workflow.ExecResult{}, fmt.Errorf("exec: step %q has no tool binding", in.Step.Key)
	}
	// MVP: one tool per agent/system step.
	toolKey := in.Step.ToolBindings[0].ToolKey

	// Authorization: binding ∩ capability, with amount bound where relevant.
	dec := authz.Check(in.Step, in.Actor, toolKey, amountArgs(in.Context))
	if !dec.Allowed {
		return workflow.ExecResult{Tool: toolKey, Denied: true, DenyReason: dec.Reason}, nil
	}

	fn, ok := e.tools[toolKey]
	if !ok {
		return workflow.ExecResult{}, fmt.Errorf("exec: no implementation for tool %q", toolKey)
	}
	out, err := fn(in)
	if err != nil {
		return workflow.ExecResult{}, err
	}
	return workflow.ExecResult{Tool: toolKey, CapabilityUsed: dec.CapabilityUsed, Output: out}, nil
}

// amountArgs pulls numeric args the authz gate compares against scope caps.
func amountArgs(ctx map[string]any) map[string]float64 {
	args := map[string]float64{}
	if v, ok := toFloat(ctx["amount"]); ok {
		args["amount"] = v
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

func ctxString(ctx map[string]any, key string) string {
	if s, ok := ctx[key].(string); ok {
		return s
	}
	return ""
}

// shortHash returns a stable short token derived from s (for deterministic ids).
func shortHash(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%08x", h.Sum32())
}
