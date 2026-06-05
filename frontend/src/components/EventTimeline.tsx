import type { WorkflowEvent } from "../api/types";

const TYPE_LABEL: Record<string, string> = {
  "task.created": "タスク作成",
  "step.started": "ステップ開始",
  "tool.invoked": "ツール実行",
  "tool.denied": "ツール拒否",
  "elicitation.requested": "確認要求",
  "elicitation.answered": "確認応答",
  "trigger.fired": "トリガ発火",
  "approval.requested": "承認依頼",
  "approval.decided": "承認決定",
  "state.transitioned": "状態遷移",
  "step.completed": "ステップ完了",
  "step.failed": "ステップ失敗",
  "run.waiting": "待機開始",
  "run.completed": "実行完了",
  "run.failed": "実行失敗",
};

function summary(e: WorkflowEvent): string {
  const p = e.payload || {};
  switch (e.type) {
    case "tool.invoked":
      return `${p.tool}${p.output?.amount != null ? ` → ¥${p.output.amount}` : ""}`;
    case "state.transitioned":
      return `${p.from} → ${p.to}`;
    case "elicitation.answered":
      return p.yes ? "Yes" : "No";
    case "approval.decided":
      return `${p.decision}`;
    case "run.waiting":
      return `${p.kind} を待機`;
    default:
      return "";
  }
}

export function EventTimeline({ events }: { events: WorkflowEvent[] }) {
  if (!events.length) return <p className="muted">イベントはまだありません。</p>;
  const ordered = [...events].sort((a, b) => b.seq - a.seq);
  return (
    <ul className="timeline">
      {ordered.map((e) => (
        <li key={e.id}>
          <span className="dot" />
          <div className="ev-type">
            {TYPE_LABEL[e.type] ?? e.type}{" "}
            {summary(e) && <span className="kv">{summary(e)}</span>}
          </div>
          <div className="ev-meta">
            #{e.seq} · {e.actor_id || "system"} ·{" "}
            {new Date(e.at).toLocaleTimeString("ja-JP")}
            {e.capability_used && <> · <span className="ev-cap">{e.capability_used}</span></>}
          </div>
        </li>
      ))}
    </ul>
  );
}
