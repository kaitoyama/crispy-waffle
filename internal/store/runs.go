package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

func (s *Store) CreateRun(ctx context.Context, r workflow.WorkflowRun) error {
	return s.SaveRun(ctx, r)
}

// SaveRun implements workflow.RunStore (upsert).
func (s *Store) SaveRun(ctx context.Context, r workflow.WorkflowRun) error {
	var waiting any
	if r.WaitingOn != nil {
		b, _ := json.Marshal(r.WaitingOn)
		waiting = string(b)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workflow_runs(id,task_id,definition_key,definition_ver,status,current_step,waiting_on)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		  status=excluded.status, current_step=excluded.current_step, waiting_on=excluded.waiting_on`,
		r.ID, r.TaskID, r.DefinitionKey, r.DefinitionVer, r.Status, r.CurrentStep, waiting)
	return err
}

// ListWaitingApprovalRuns returns runs parked on an approval gate, for the
// approvals inbox.
func (s *Store) ListWaitingApprovalRuns(ctx context.Context) ([]workflow.WorkflowRun, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,task_id,definition_key,definition_ver,status,current_step,waiting_on
		FROM workflow_runs
		WHERE status='waiting' AND waiting_on IS NOT NULL AND json_extract(waiting_on,'$.kind')='approval'
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []workflow.WorkflowRun
	for rows.Next() {
		var r workflow.WorkflowRun
		var waiting sql.NullString
		if err := rows.Scan(&r.ID, &r.TaskID, &r.DefinitionKey, &r.DefinitionVer, &r.Status, &r.CurrentStep, &waiting); err != nil {
			return nil, err
		}
		if waiting.Valid && waiting.String != "" {
			var w workflow.WaitingOn
			if json.Unmarshal([]byte(waiting.String), &w) == nil {
				r.WaitingOn = &w
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LoadRun implements workflow.RunStore.
func (s *Store) LoadRun(ctx context.Context, id domain.RunID) (workflow.WorkflowRun, error) {
	var r workflow.WorkflowRun
	var waiting sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id,task_id,definition_key,definition_ver,status,current_step,waiting_on FROM workflow_runs WHERE id=?`, id).
		Scan(&r.ID, &r.TaskID, &r.DefinitionKey, &r.DefinitionVer, &r.Status, &r.CurrentStep, &waiting)
	if err != nil {
		return workflow.WorkflowRun{}, err
	}
	if waiting.Valid && waiting.String != "" {
		var w workflow.WaitingOn
		if json.Unmarshal([]byte(waiting.String), &w) == nil {
			r.WaitingOn = &w
		}
	}
	return r, nil
}
