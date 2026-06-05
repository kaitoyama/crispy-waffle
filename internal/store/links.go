package store

import (
	"context"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

func (s *Store) CreateLink(ctx context.Context, l domain.TaskLink) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO task_links(id,source_id,target_id,link_type,created_by,created_at)
		VALUES(?,?,?,?,?,?)`,
		l.ID, l.SourceID, l.TargetID, l.LinkType, l.CreatedBy, l.CreatedAt.Format(rfc))
	return err
}

// ListLinksForTask returns links where the task is either source or target, so
// the graph view can render both incoming and outgoing edges (docs/02 §4).
func (s *Store) ListLinksForTask(ctx context.Context, taskID domain.TaskID) ([]domain.TaskLink, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,source_id,target_id,link_type,created_by,created_at
		FROM task_links WHERE source_id=? OR target_id=? ORDER BY created_at ASC`, taskID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TaskLink
	for rows.Next() {
		var l domain.TaskLink
		var created string
		if err := rows.Scan(&l.ID, &l.SourceID, &l.TargetID, &l.LinkType, &l.CreatedBy, &created); err != nil {
			return nil, err
		}
		l.CreatedAt, _ = time.Parse(rfc, created)
		out = append(out, l)
	}
	return out, rows.Err()
}
