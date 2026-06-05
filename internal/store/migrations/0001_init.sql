-- Work / execution / authorization planes persisted to SQLite.
-- The events table is the append-only source of truth; tasks/workflow_runs
-- hold fast-query projections reconstructible from it.

CREATE TABLE IF NOT EXISTS actors (
    id            TEXT PRIMARY KEY,
    display_name  TEXT NOT NULL,
    kind          TEXT NOT NULL,
    identity_ref  TEXT NOT NULL DEFAULT '',
    roles         TEXT NOT NULL DEFAULT '[]',   -- JSON array
    delegated_from TEXT,
    capabilities  TEXT NOT NULL DEFAULT '[]'    -- JSON array
);

CREATE TABLE IF NOT EXISTS task_types (
    key                  TEXT PRIMARY KEY,
    display_name         TEXT NOT NULL,
    context_schema       TEXT NOT NULL DEFAULT '{}',
    default_workflow_key TEXT NOT NULL,
    default_workflow_ver INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL,
    intent          TEXT NOT NULL DEFAULT '',
    type            TEXT NOT NULL,
    status          TEXT NOT NULL,
    requester_id    TEXT NOT NULL,
    assignee_id     TEXT NOT NULL,
    parent_id       TEXT,
    workflow_run_id TEXT,
    context         TEXT NOT NULL DEFAULT '{}',
    priority        INTEGER,
    due_at          TEXT,
    origin          TEXT NOT NULL DEFAULT 'manual',
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workflow_runs (
    id             TEXT PRIMARY KEY,
    task_id        TEXT NOT NULL,
    definition_key TEXT NOT NULL,
    definition_ver INTEGER NOT NULL,
    status         TEXT NOT NULL,
    current_step   TEXT NOT NULL,
    waiting_on     TEXT  -- JSON or NULL
);

CREATE TABLE IF NOT EXISTS events (
    id              TEXT PRIMARY KEY,
    seq             INTEGER NOT NULL,
    task_id         TEXT NOT NULL,
    run_id          TEXT NOT NULL,
    type            TEXT NOT NULL,
    actor_id        TEXT NOT NULL DEFAULT '',
    capability_used TEXT NOT NULL DEFAULT '',
    payload         TEXT,
    at              TEXT NOT NULL,
    UNIQUE(run_id, seq)
);
CREATE INDEX IF NOT EXISTS idx_events_run ON events(run_id, seq);
CREATE INDEX IF NOT EXISTS idx_events_task ON events(task_id, seq);

CREATE TABLE IF NOT EXISTS approvals (
    id             TEXT PRIMARY KEY,
    target_task_id TEXT NOT NULL,
    target_step_id TEXT NOT NULL,
    approver_id    TEXT NOT NULL,
    decision       TEXT NOT NULL,
    conditions     TEXT,
    rationale      TEXT NOT NULL DEFAULT '',
    decided_at     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_approvals_task ON approvals(target_task_id);

CREATE TABLE IF NOT EXISTS task_links (
    id         TEXT PRIMARY KEY,
    source_id  TEXT NOT NULL,
    target_id  TEXT NOT NULL,
    link_type  TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_links_source ON task_links(source_id);
CREATE INDEX IF NOT EXISTS idx_links_target ON task_links(target_id);

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
