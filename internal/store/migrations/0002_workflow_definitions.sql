-- User-defined workflow definitions, persisted so flows registered via the UI
-- survive restarts and can be loaded into the in-memory registry on boot.
-- Running instances pin (key, version), so editing a flow creates a new version
-- and never changes in-flight runs (docs/02 §11-6).

CREATE TABLE IF NOT EXISTS workflow_definitions (
    key          TEXT NOT NULL,
    version      INTEGER NOT NULL,
    display_name TEXT NOT NULL,
    entry_step   TEXT NOT NULL,
    steps        TEXT NOT NULL DEFAULT '{}',  -- JSON map[stepKey]StepDef
    created_at   TEXT NOT NULL,
    PRIMARY KEY (key, version)
);
