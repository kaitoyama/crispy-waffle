package domain

// ActorKind unifies humans, agents and programs as equal participants: what
// advances a task is the *system*; an actor is merely whoever/whatever a step
// is satisfied by (docs/01, docs/05 §1).
type ActorKind string

const (
	ActorHuman ActorKind = "human"
	ActorAgent ActorKind = "agent"
)

// Capability is a held permission: a tool key plus a scope that bounds it
// (e.g. payment.execute @ {amount_cap: 20000}). Delegation may only narrow a
// capability, never widen it (docs/05 attenuation).
type Capability struct {
	ToolKey string         `json:"tool_key"`
	Scope   map[string]any `json:"scope,omitempty"`
}

type Actor struct {
	ID            ActorID      `json:"id"`
	DisplayName   string       `json:"display_name"`
	Kind          ActorKind    `json:"kind"`
	IdentityRef   string       `json:"identity_ref"`
	Roles         []string     `json:"roles"`
	DelegatedFrom *ActorID     `json:"delegated_from,omitempty"`
	Capabilities  []Capability `json:"capabilities"`
}

// HasRole reports whether the actor holds the given role.
func (a Actor) HasRole(role string) bool {
	for _, r := range a.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Capability returns the actor's capability for a tool, if any.
func (a Actor) Capability(toolKey string) (Capability, bool) {
	for _, c := range a.Capabilities {
		if c.ToolKey == toolKey {
			return c, true
		}
	}
	return Capability{}, false
}
