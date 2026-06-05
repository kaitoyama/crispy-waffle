package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

func (s *Store) CreateApproval(ctx context.Context, a domain.ApprovalRecord) error {
	var cond any
	if len(a.Conditions) > 0 {
		cond = string(a.Conditions)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO approvals(id,target_task_id,target_step_id,approver_id,decision,conditions,rationale,decided_at)
		VALUES(?,?,?,?,?,?,?,?)`,
		a.ID, a.TargetTaskID, a.TargetStepID, a.ApproverID, a.Decision, cond, a.Rationale,
		a.DecidedAt.Format(rfc))
	return err
}

func (s *Store) ListApprovalsByTask(ctx context.Context, taskID domain.TaskID) ([]domain.ApprovalRecord, error) {
	rows, err := s.db.QueryContext(ctx, approvalCols+` WHERE target_task_id=? ORDER BY decided_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApprovals(rows)
}

const approvalCols = `SELECT id,target_task_id,target_step_id,approver_id,decision,conditions,rationale,decided_at FROM approvals`

func scanApprovals(rows *sql.Rows) ([]domain.ApprovalRecord, error) {
	var out []domain.ApprovalRecord
	for rows.Next() {
		var a domain.ApprovalRecord
		var cond sql.NullString
		var decided string
		if err := rows.Scan(&a.ID, &a.TargetTaskID, &a.TargetStepID, &a.ApproverID, &a.Decision, &cond, &a.Rationale, &decided); err != nil {
			return nil, err
		}
		if cond.Valid {
			a.Conditions = json.RawMessage(cond.String)
		}
		a.DecidedAt, _ = time.Parse(rfc, decided)
		out = append(out, a)
	}
	return out, rows.Err()
}
