package workflow

import "fmt"

// Registry holds workflow definitions keyed by (key, version). Runs pin a
// version so a definition update never changes the interpretation of an
// in-flight instance (docs/02 §11-6).
type Registry struct {
	defs map[string]WorkflowDefinition
}

func NewRegistry() *Registry {
	return &Registry{defs: map[string]WorkflowDefinition{}}
}

func regKey(key string, ver int) string {
	return fmt.Sprintf("%s@%d", key, ver)
}

func (r *Registry) Register(def WorkflowDefinition) {
	r.defs[regKey(def.Key, def.Version)] = def
}

func (r *Registry) Get(key string, ver int) (WorkflowDefinition, bool) {
	d, ok := r.defs[regKey(key, ver)]
	return d, ok
}

// Latest returns the highest-version definition registered under key.
func (r *Registry) Latest(key string) (WorkflowDefinition, bool) {
	var best WorkflowDefinition
	found := false
	for _, d := range r.defs {
		if d.Key == key && (!found || d.Version > best.Version) {
			best, found = d, true
		}
	}
	return best, found
}

// All returns every registered definition.
func (r *Registry) All() []WorkflowDefinition {
	out := make([]WorkflowDefinition, 0, len(r.defs))
	for _, d := range r.defs {
		out = append(out, d)
	}
	return out
}
