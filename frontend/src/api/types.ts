// Mirrors the Go DTOs returned by the backend.

export type TaskSummary = {
  id: string;
  title: string;
  type: string;
  status: string;
  assignee_id: string;
  updated_at: string;
};

export type Actor = {
  id: string;
  display_name: string;
  kind: "human" | "agent";
  roles: string[];
  delegated_from?: string;
  capabilities: { tool_key: string; scope?: Record<string, unknown> }[];
};

export type Tool = {
  key: string;
  display_name: string;
  side_effect_class: string;
  description: string;
  scope_dimensions?: string[];
};

export type WorkflowEvent = {
  id: string;
  seq: number;
  task_id: string;
  run_id: string;
  type: string;
  actor_id: string;
  capability_used?: string;
  payload?: any;
  at: string;
};

export type WaitingOn = { kind: string; ref: string };

export type Run = {
  id: string;
  task_id: string;
  definition_key: string;
  definition_ver: number;
  status: "running" | "waiting" | "completed" | "failed" | "cancelled";
  current_step: string;
  waiting_on?: WaitingOn;
};

export type StepDef = {
  key: string;
  kind: string;
  gate_kind?: string;
  title: string;
  enters_state: string;
  terminal?: boolean;
  transitions?: { to: string; guard?: string }[];
};

export type WorkflowDefinition = {
  key: string;
  version: number;
  display_name: string;
  entry_step: string;
  steps: Record<string, StepDef>;
};

export type StepView = {
  key: string;
  kind: string;
  gate_kind?: string;
  title: string;
  enters_state: string;
};

export type TaskLink = {
  id: string;
  source_id: string;
  target_id: string;
  link_type: string;
  created_by: string;
  created_at: string;
};

export type ApprovalRecord = {
  id: string;
  target_task_id: string;
  target_step_id: string;
  approver_id: string;
  decision: string;
  rationale: string;
  decided_at: string;
};

export type Task = {
  id: string;
  title: string;
  intent: string;
  type: string;
  status: string;
  requester_id: string;
  assignee_id: string;
  context: any;
  origin: string;
  created_at: string;
  updated_at: string;
};

export type TaskDetail = {
  task: Task;
  run?: Run;
  definition?: WorkflowDefinition;
  current_step?: StepView;
  available_actions: string[];
  events: WorkflowEvent[];
  links: TaskLink[];
  nodes: TaskSummary[];
  approvals: ApprovalRecord[];
};

export type ApprovalRequest = {
  task_id: string;
  run_id: string;
  step_key: string;
  title: string;
  type: string;
  requester_id: string;
  context: Record<string, any>;
};
