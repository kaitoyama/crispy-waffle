// Package authz is the thin authorization-plane seam: a tool invocation is
// allowed only within the intersection of the step's ToolBinding and the
// actor's Capability for that tool (docs/02 §11-2, docs/05). Delegated agents
// hold only a subset of the delegator's permissions (attenuation).
package authz

import (
	"fmt"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// Decision is the result of an authorization check.
type Decision struct {
	Allowed        bool
	Reason         string
	CapabilityUsed string // human-readable record of the exercised permission
}

// Check verifies that actor may invoke toolKey with the given numeric args,
// bounded by the step's binding for that tool. amountArgs carries the values to
// compare against scope caps (e.g. {"amount": 1280} vs cap {"amount_cap": 20000}).
func Check(step workflow.StepDef, actor domain.Actor, toolKey string, amountArgs map[string]float64) Decision {
	binding, hasBinding := bindingFor(step, toolKey)
	if !hasBinding {
		return Decision{Reason: fmt.Sprintf("step %q is not bound to tool %q", step.Key, toolKey)}
	}
	cap, hasCap := actor.Capability(toolKey)
	if !hasCap {
		return Decision{Reason: fmt.Sprintf("actor %q lacks capability for tool %q", actor.ID, toolKey)}
	}

	// Effective cap = min(binding amount_cap, capability amount_cap).
	effCap, hasCapLimit := minCap(binding.Scope, cap.Scope, "amount_cap")
	if hasCapLimit {
		if amt, ok := amountArgs["amount"]; ok && amt > effCap {
			return Decision{Reason: fmt.Sprintf("amount %.0f exceeds cap %.0f", amt, effCap)}
		}
	}

	used := toolKey
	if hasCapLimit {
		used = fmt.Sprintf("%s@{amount_cap:%.0f}", toolKey, effCap)
	}
	return Decision{Allowed: true, CapabilityUsed: used}
}

func bindingFor(step workflow.StepDef, toolKey string) (workflow.ToolBinding, bool) {
	for _, b := range step.ToolBindings {
		if b.ToolKey == toolKey {
			return b, true
		}
	}
	return workflow.ToolBinding{}, false
}

// minCap returns the tighter of two scope caps for key, if either is present.
func minCap(a, b map[string]any, key string) (float64, bool) {
	av, aok := scopeFloat(a, key)
	bv, bok := scopeFloat(b, key)
	switch {
	case aok && bok:
		if av < bv {
			return av, true
		}
		return bv, true
	case aok:
		return av, true
	case bok:
		return bv, true
	}
	return 0, false
}

func scopeFloat(m map[string]any, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	switch v := m[key].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}
