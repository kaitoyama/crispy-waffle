package domain

import "time"

// LinkType is the typed directed edge between tasks (docs/02 §4). Links are
// first-class entities (not embedded arrays) so the graph can be traversed in
// both directions and edges can carry their own metadata.
type LinkType string

const (
	LinkSubtaskOf   LinkType = "subtask_of"
	LinkBlocks      LinkType = "blocks"
	LinkBlockedBy   LinkType = "blocked_by"
	LinkRelatesTo   LinkType = "relates_to"
	LinkTriggeredBy LinkType = "triggered_by"
	LinkApprovalFor LinkType = "approval_for"
	LinkSupersedes  LinkType = "supersedes"
)

// ValidLinkType reports whether t is one of the known link types.
func ValidLinkType(t LinkType) bool {
	switch t {
	case LinkSubtaskOf, LinkBlocks, LinkBlockedBy, LinkRelatesTo,
		LinkTriggeredBy, LinkApprovalFor, LinkSupersedes:
		return true
	}
	return false
}

type TaskLink struct {
	ID        LinkID    `json:"id"`
	SourceID  TaskID    `json:"source_id"`
	TargetID  TaskID    `json:"target_id"`
	LinkType  LinkType  `json:"link_type"`
	CreatedBy ActorID   `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}
