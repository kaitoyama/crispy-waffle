package tools

import "sort"

// Registry holds the registered tools by key. It is the single catalog used for
// execution (exec), flow validation (service) and the catalog API.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Register adds a tool. A duplicate key panics: keys are compile-time-known
// constants, so a collision is a programming error worth catching at startup.
func (r *Registry) Register(t Tool) {
	key := t.Spec().Key
	if _, dup := r.tools[key]; dup {
		panic("tools: duplicate tool key " + key)
	}
	r.tools[key] = t
}

func (r *Registry) Get(key string) (Tool, bool) {
	t, ok := r.tools[key]
	return t, ok
}

// Spec returns one tool's spec.
func (r *Registry) Spec(key string) (Spec, bool) {
	t, ok := r.tools[key]
	if !ok {
		return Spec{}, false
	}
	return t.Spec(), true
}

// Specs returns every tool's spec, sorted by key (stable catalog ordering).
func (r *Registry) Specs() []Spec {
	out := make([]Spec, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Spec())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
