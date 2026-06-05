// Maps task/run statuses and step kinds to badge colors and icons so the whole
// UI renders them consistently.

const STATUS_COLOR: Record<string, string> = {
  Drafting: "gray",
  ConfirmPre: "amber",
  Submitting: "blue",
  AwaitingApproval: "amber",
  AwaitingSettlement: "purple",
  Settling: "blue",
  Settled: "green",
  Rejected: "red",
  running: "blue",
  waiting: "amber",
  completed: "green",
  failed: "red",
  cancelled: "gray",
};

export function StatusBadge({ status }: { status: string }) {
  const color = STATUS_COLOR[status] ?? "gray";
  return <span className={`badge ${color}`}>{status}</span>;
}

export const STEP_KIND_ICON: Record<string, string> = {
  agent_action: "⚙",
  human_gate: "👤",
  system_action: "▶",
  sub_workflow: "⤳",
};

export const STEP_KIND_LABEL: Record<string, string> = {
  agent_action: "システム実行（プログラム/エージェント）",
  human_gate: "人間ゲート",
  system_action: "システム処理",
  sub_workflow: "サブワークフロー",
};
