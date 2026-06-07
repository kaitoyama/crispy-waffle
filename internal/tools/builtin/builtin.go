// Package builtin holds the in-tree tool implementations and registers them.
//
// To add a new tool:
//  1. Create a file in this package with a type implementing tools.Tool
//     (a Spec() describing it and an Execute() performing the effect).
//  2. Add one line to Register below.
//
// That is the entire extension surface — the catalog API, the flow-builder tool
// picker, authorization and execution all derive from the registered Spec.
package builtin

import (
	"fmt"
	"hash/fnv"

	"github.com/kaitoyama/crispy-waffle/internal/tools"
)

// Register installs every built-in tool into the registry.
func Register(r *tools.Registry) {
	r.Register(FareLookup{})
	r.Register(PreApplicationSubmit{})
	r.Register(PaymentExecute{})
	r.Register(NotifySend{})
	// ↑ add new tools here
}

// --- shared helpers for tool implementations ---

func ctxString(c map[string]any, key string) string {
	if s, ok := c[key].(string); ok {
		return s
	}
	return ""
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

// shortHash returns a stable short token from s (for deterministic ids).
func shortHash(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%08x", h.Sum32())
}
