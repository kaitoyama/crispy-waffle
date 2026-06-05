import type { TaskDetail } from "../api/types";
import { StatusBadge, STEP_KIND_ICON, STEP_KIND_LABEL } from "./badges";
import { StateDiagram } from "./StateDiagram";

// Orders steps breadth-first from the entry step for a stable top-to-bottom list.
function orderedSteps(def: TaskDetail["definition"]): string[] {
  if (!def) return [];
  const order: string[] = [];
  const seen = new Set<string>();
  const queue = [def.entry_step];
  while (queue.length) {
    const k = queue.shift()!;
    if (seen.has(k) || !def.steps[k]) continue;
    seen.add(k);
    order.push(k);
    for (const t of def.steps[k].transitions ?? []) queue.push(t.to);
  }
  // include any unreferenced steps
  for (const k of Object.keys(def.steps)) if (!seen.has(k)) order.push(k);
  return order;
}

export function WorkflowStateView({ detail }: { detail: TaskDetail }) {
  const { run, definition, current_step, events } = detail;
  const completed = new Set(
    events
      .filter((e) => e.type === "step.completed")
      .map((e) => e.payload?.from)
      .filter(Boolean),
  );

  const waitingLabel = (() => {
    if (!run || run.status !== "waiting" || !run.waiting_on) return null;
    switch (run.waiting_on.kind) {
      case "approval":
        return "会計担当の承認待ち";
      case "elicitation":
        return "実行者の確認待ち（Yes/No）";
      case "trigger":
        return "実行トリガ待ち（精算する）";
      default:
        return run.waiting_on.kind;
    }
  })();

  return (
    <div className="panel">
      <h3>ワークフロー状態</h3>
      <div style={{ display: "flex", gap: 10, alignItems: "center", flexWrap: "wrap", marginBottom: 12 }}>
        <StatusBadge status={detail.task.status} />
        {run && <span className="badge gray">run: {run.status}</span>}
        {waitingLabel && <span className="badge amber">⏸ {waitingLabel}</span>}
      </div>

      {definition && (
        <div style={{ marginBottom: 16 }}>
          {orderedSteps(definition).map((key) => {
            const step = definition.steps[key];
            const isCurrent = current_step?.key === key && run?.status !== "completed";
            const isDone = completed.has(key) && !isCurrent;
            return (
              <div key={key} className={`stepcard ${isCurrent ? "current" : ""} ${isDone ? "done" : ""}`}>
                <div className={`kind-icon kind-${step.kind}`}>{STEP_KIND_ICON[step.kind]}</div>
                <div className="step-body">
                  <div className="step-title">
                    {step.title || key} {isDone && "✓"}
                  </div>
                  <div className="step-sub">
                    {STEP_KIND_LABEL[step.kind]}
                    {step.gate_kind ? ` · ${step.gate_kind}` : ""} · → {step.enters_state}
                  </div>
                </div>
                {isCurrent && <span className="badge blue">現在</span>}
              </div>
            );
          })}
        </div>
      )}

      {definition && (
        <details>
          <summary className="muted" style={{ cursor: "pointer" }}>状態機械図を表示</summary>
          <div style={{ marginTop: 10 }}>
            <StateDiagram def={definition} current={current_step?.key} />
          </div>
        </details>
      )}
    </div>
  );
}
