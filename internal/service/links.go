package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

// CreateLink adds a typed edge between two tasks (docs/02 §4).
func (s *Service) CreateLink(ctx context.Context, sourceID, targetID domain.TaskID, linkType domain.LinkType, createdBy domain.ActorID) (domain.TaskLink, error) {
	if !domain.ValidLinkType(linkType) {
		return domain.TaskLink{}, fmt.Errorf("invalid link type %q", linkType)
	}
	if sourceID == targetID {
		return domain.TaskLink{}, fmt.Errorf("cannot link a task to itself")
	}
	if _, err := s.Store.GetTask(ctx, targetID); err != nil {
		return domain.TaskLink{}, fmt.Errorf("target task not found: %w", err)
	}
	l := domain.TaskLink{
		ID: domain.NewLinkID(), SourceID: sourceID, TargetID: targetID,
		LinkType: linkType, CreatedBy: createdBy, CreatedAt: time.Now().UTC(),
	}
	if err := s.Store.CreateLink(ctx, l); err != nil {
		return domain.TaskLink{}, err
	}
	return l, nil
}
