package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// UpsertDefinition persists a workflow definition (key, version).
func (s *Store) UpsertDefinition(ctx context.Context, def workflow.WorkflowDefinition) error {
	steps, err := json.Marshal(def.Steps)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO workflow_definitions(key,version,display_name,entry_step,steps,created_at)
		VALUES(?,?,?,?,?,?)
		ON CONFLICT(key,version) DO UPDATE SET
		  display_name=excluded.display_name, entry_step=excluded.entry_step, steps=excluded.steps`,
		def.Key, def.Version, def.DisplayName, def.EntryStep, string(steps),
		time.Now().UTC().Format(rfc))
	return err
}

// GetDefinition loads a single definition version.
func (s *Store) GetDefinition(ctx context.Context, key string, ver int) (workflow.WorkflowDefinition, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT key,version,display_name,entry_step,steps FROM workflow_definitions WHERE key=? AND version=?`, key, ver)
	return scanDefinition(row)
}

// MaxVersion returns the highest version for a key, or (0,false) if none.
func (s *Store) MaxVersion(ctx context.Context, key string) (int, bool, error) {
	var v sql.NullInt64
	if err := s.db.QueryRowContext(ctx,
		`SELECT MAX(version) FROM workflow_definitions WHERE key=?`, key).Scan(&v); err != nil {
		return 0, false, err
	}
	if !v.Valid {
		return 0, false, nil
	}
	return int(v.Int64), true, nil
}

// ListDefinitions returns every persisted definition.
func (s *Store) ListDefinitions(ctx context.Context) ([]workflow.WorkflowDefinition, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key,version,display_name,entry_step,steps FROM workflow_definitions ORDER BY key, version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []workflow.WorkflowDefinition
	for rows.Next() {
		d, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func scanDefinition(row scanner) (workflow.WorkflowDefinition, error) {
	var d workflow.WorkflowDefinition
	var steps string
	if err := row.Scan(&d.Key, &d.Version, &d.DisplayName, &d.EntryStep, &steps); err != nil {
		return workflow.WorkflowDefinition{}, err
	}
	if err := json.Unmarshal([]byte(steps), &d.Steps); err != nil {
		return workflow.WorkflowDefinition{}, err
	}
	return d, nil
}
