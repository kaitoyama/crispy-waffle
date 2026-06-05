package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// Append implements workflow.EventStore. Seq is assigned monotonically per run
// inside a transaction so the ledger stays strictly ordered (docs/02 §8).
func (s *Store) Append(ctx context.Context, evs []workflow.Event) ([]workflow.Event, error) {
	if len(evs) == 0 {
		return evs, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	for i := range evs {
		var maxSeq int64
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(seq),0) FROM events WHERE run_id=?`, evs[i].RunID).Scan(&maxSeq); err != nil {
			return nil, err
		}
		evs[i].Seq = maxSeq + 1
		if evs[i].At.IsZero() {
			evs[i].At = time.Now().UTC()
		}
		var payload any
		if len(evs[i].Payload) > 0 {
			payload = string(evs[i].Payload)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO events(id,seq,task_id,run_id,type,actor_id,capability_used,payload,at)
			VALUES(?,?,?,?,?,?,?,?,?)`,
			evs[i].ID, evs[i].Seq, evs[i].TaskID, evs[i].RunID, evs[i].Type,
			evs[i].ActorID, evs[i].CapabilityUsed, payload, evs[i].At.Format(rfc)); err != nil {
			return nil, err
		}
	}
	return evs, tx.Commit()
}

// LoadByRun implements workflow.EventStore.
func (s *Store) LoadByRun(ctx context.Context, runID domain.RunID) ([]workflow.Event, error) {
	rows, err := s.db.QueryContext(ctx, eventCols+` WHERE run_id=? ORDER BY seq ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

// LoadByTask returns the timeline across all runs of a task.
func (s *Store) LoadByTask(ctx context.Context, taskID domain.TaskID) ([]workflow.Event, error) {
	rows, err := s.db.QueryContext(ctx, eventCols+` WHERE task_id=? ORDER BY seq ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

const eventCols = `SELECT id,seq,task_id,run_id,type,actor_id,capability_used,payload,at FROM events`

func scanEvents(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]workflow.Event, error) {
	var out []workflow.Event
	for rows.Next() {
		var e workflow.Event
		var payload sql.NullString
		var at string
		if err := rows.Scan(&e.ID, &e.Seq, &e.TaskID, &e.RunID, &e.Type, &e.ActorID, &e.CapabilityUsed, &payload, &at); err != nil {
			return nil, err
		}
		if payload.Valid && payload.String != "" {
			e.Payload = []byte(payload.String)
		}
		e.At, _ = time.Parse(rfc, at)
		out = append(out, e)
	}
	return out, rows.Err()
}
