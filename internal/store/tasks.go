package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

const rfc = time.RFC3339Nano

// --- task types ---

func (s *Store) UpsertTaskType(ctx context.Context, t domain.TaskType) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO task_types(key,display_name,context_schema,default_workflow_key,default_workflow_ver)
		VALUES(?,?,?,?,?)
		ON CONFLICT(key) DO UPDATE SET
		  display_name=excluded.display_name, context_schema=excluded.context_schema,
		  default_workflow_key=excluded.default_workflow_key, default_workflow_ver=excluded.default_workflow_ver`,
		t.Key, t.DisplayName, string(t.ContextSchema), t.DefaultWorkflowKey, t.DefaultWorkflowVer)
	return err
}

func (s *Store) GetTaskType(ctx context.Context, key string) (domain.TaskType, error) {
	var t domain.TaskType
	var schema string
	err := s.db.QueryRowContext(ctx,
		`SELECT key,display_name,context_schema,default_workflow_key,default_workflow_ver FROM task_types WHERE key=?`, key).
		Scan(&t.Key, &t.DisplayName, &schema, &t.DefaultWorkflowKey, &t.DefaultWorkflowVer)
	if err != nil {
		return domain.TaskType{}, err
	}
	t.ContextSchema = json.RawMessage(schema)
	return t, nil
}

func (s *Store) ListTaskTypes(ctx context.Context) ([]domain.TaskType, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key,display_name,context_schema,default_workflow_key,default_workflow_ver FROM task_types ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TaskType
	for rows.Next() {
		var t domain.TaskType
		var schema string
		if err := rows.Scan(&t.Key, &t.DisplayName, &schema, &t.DefaultWorkflowKey, &t.DefaultWorkflowVer); err != nil {
			return nil, err
		}
		t.ContextSchema = json.RawMessage(schema)
		out = append(out, t)
	}
	return out, rows.Err()
}

// --- tasks ---

func (s *Store) CreateTask(ctx context.Context, t domain.Task) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks(id,title,intent,type,status,requester_id,assignee_id,parent_id,
		   workflow_run_id,context,priority,due_at,origin,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Title, t.Intent, t.Type, t.Status, t.RequesterID, t.AssigneeID,
		nullStr(t.ParentID), nullRun(t.WorkflowRunID), string(ctxOr(t.Context)),
		nullInt(t.Priority), nullTime(t.DueAt), t.Origin,
		t.CreatedAt.Format(rfc), t.UpdatedAt.Format(rfc))
	return err
}

// LoadTask implements workflow.TaskStore.
func (s *Store) LoadTask(ctx context.Context, id domain.TaskID) (domain.Task, error) {
	return s.GetTask(ctx, id)
}

func (s *Store) GetTask(ctx context.Context, id domain.TaskID) (domain.Task, error) {
	row := s.db.QueryRowContext(ctx, taskCols+` WHERE id=?`, id)
	return scanTask(row)
}

func (s *Store) ListTasks(ctx context.Context) ([]domain.Task, error) {
	rows, err := s.db.QueryContext(ctx, taskCols+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateTaskStatus implements workflow.TaskStore.
func (s *Store) UpdateTaskStatus(ctx context.Context, id domain.TaskID, status domain.TaskStatus) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status=?, updated_at=? WHERE id=?`,
		status, time.Now().UTC().Format(rfc), id)
	return err
}

const taskCols = `SELECT id,title,intent,type,status,requester_id,assignee_id,parent_id,
	workflow_run_id,context,priority,due_at,origin,created_at,updated_at FROM tasks`

func scanTask(row scanner) (domain.Task, error) {
	var t domain.Task
	var parent, run, due sql.NullString
	var prio sql.NullInt64
	var ctxStr, created, updated string
	if err := row.Scan(&t.ID, &t.Title, &t.Intent, &t.Type, &t.Status, &t.RequesterID, &t.AssigneeID,
		&parent, &run, &ctxStr, &prio, &due, &t.Origin, &created, &updated); err != nil {
		return domain.Task{}, err
	}
	t.Context = json.RawMessage(ctxStr)
	if parent.Valid {
		p := domain.TaskID(parent.String)
		t.ParentID = &p
	}
	if run.Valid {
		r := domain.RunID(run.String)
		t.WorkflowRunID = &r
	}
	if prio.Valid {
		p := int(prio.Int64)
		t.Priority = &p
	}
	if due.Valid {
		if dt, err := time.Parse(rfc, due.String); err == nil {
			t.DueAt = &dt
		}
	}
	t.CreatedAt, _ = time.Parse(rfc, created)
	t.UpdatedAt, _ = time.Parse(rfc, updated)
	return t, nil
}

func ctxOr(m json.RawMessage) json.RawMessage {
	if len(m) == 0 {
		return json.RawMessage("{}")
	}
	return m
}
func nullStr(p *domain.TaskID) any {
	if p == nil {
		return nil
	}
	return string(*p)
}
func nullRun(p *domain.RunID) any {
	if p == nil {
		return nil
	}
	return string(*p)
}
func nullInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullTime(p *time.Time) any {
	if p == nil {
		return nil
	}
	return p.Format(rfc)
}
