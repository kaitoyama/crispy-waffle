import { useState } from "react";
import type { TaskDetail } from "../api/types";
import { useTaskActions } from "../hooks/queries";

// Context-aware actions derived from available_actions (docs/07: elicitation
// confirm vs. approval vs. self-trigger "settle").
export function ActionBar({ detail }: { detail: TaskDetail }) {
  const actions = detail.available_actions;
  const a = useTaskActions(detail.task.id);
  const [err, setErr] = useState<string | null>(null);

  const run = async (fn: () => Promise<unknown>) => {
    setErr(null);
    try {
      await fn();
    } catch (e) {
      setErr((e as Error).message);
    }
  };

  if (!actions.length) return null;

  return (
    <div className="panel">
      <h3>アクション</h3>
      <div className="actions">
        {actions.includes("advance") && (
          <button className="btn-primary" onClick={() => run(() => a.advance.mutateAsync())}>
            ▶ システム実行を進める
          </button>
        )}
        {actions.includes("elicit") && (
          <>
            <span className="muted" style={{ alignSelf: "center" }}>
              この内容で事前申請しますか？
            </span>
            <button className="btn-primary" onClick={() => run(() => a.elicit.mutateAsync("yes"))}>
              はい（事前申請する）
            </button>
            <button onClick={() => run(() => a.elicit.mutateAsync("no"))}>いいえ（やり直す）</button>
          </>
        )}
        {actions.includes("awaiting_approval") && (
          <span className="badge amber">会計担当の承認を待っています（承認インボックスで処理）</span>
        )}
        {actions.includes("settle") && (
          <button className="btn-green" onClick={() => run(() => a.settle.mutateAsync())}>
            💴 精算する
          </button>
        )}
      </div>
      {err && <p className="err" style={{ marginBottom: 0 }}>{err}</p>}
    </div>
  );
}
